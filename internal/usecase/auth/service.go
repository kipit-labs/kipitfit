package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/mail"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

// Service signs people up and in: the first admin, invites, sessions, and
// each user's own settings.
type Service struct {
	users           domain.UserRepository
	hasher          domain.PasswordHasher
	configuredToken string
	appURL          string
	log             *slog.Logger
	// dummyHash keeps login timing the same whether or not the user exists.
	dummyHash func() string

	setupMu    sync.Mutex
	setupToken string
}

// NewService builds the service. setupToken replaces the random token the
// first admin needs when set; appURL starts invite links.
func NewService(users domain.UserRepository, hasher domain.PasswordHasher, setupToken, appURL string, log *slog.Logger) *Service {
	return &Service{users: users, hasher: hasher, configuredToken: setupToken, appURL: appURL, log: log,
		dummyHash: sync.OnceValue(func() string {
			hash, _ := hasher.Hash("kipitfit-dummy-password")
			return hash
		})}
}

const (
	SessionTTL        = 30 * 24 * time.Hour
	minPasswordLength = 10
	// bcrypt ignores everything past 72 bytes; refuse rather than truncate.
	maxPasswordBytes = 72
	maxNameLength    = 80
)

type AuthState struct {
	SetupRequired bool         `json:"setupRequired"`
	User          *domain.User `json:"user,omitempty"`
}

// InitAuth creates the one-time token required to create the first admin,
// so whoever reaches a fresh instance first can't claim it without access to
// the server logs.
func (s *Service) InitAuth(ctx context.Context) error {
	n, err := s.users.CountUsers(ctx)
	if err != nil || n > 0 {
		return err
	}
	token := s.configuredToken
	if token == "" {
		token = randomToken(16)
	}
	s.setupMu.Lock()
	s.setupToken = token
	s.setupMu.Unlock()
	s.log.Warn("no admin account yet: open the app and use this setup token", "setup_token", token)
	return nil
}

func (s *Service) AuthState(ctx context.Context, sessionToken string) (AuthState, error) {
	n, err := s.users.CountUsers(ctx)
	if err != nil {
		return AuthState{}, err
	}
	if n == 0 {
		return AuthState{SetupRequired: true}, nil
	}
	u, err := s.Authenticate(ctx, sessionToken)
	if errors.Is(err, domain.ErrUnauthorized) {
		return AuthState{}, nil
	}
	if err != nil {
		return AuthState{}, err
	}
	return AuthState{User: &u}, nil
}

// Setup creates the first admin account and returns a session token.
func (s *Service) Setup(ctx context.Context, setupToken, email, name, password string) (domain.User, string, error) {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()

	if n, err := s.users.CountUsers(ctx); err != nil {
		return domain.User{}, "", err
	} else if n > 0 {
		return domain.User{}, "", ErrSetupDone
	}
	if s.setupToken == "" || subtle.ConstantTimeCompare([]byte(strings.TrimSpace(setupToken)), []byte(s.setupToken)) != 1 {
		return domain.User{}, "", ErrWrongSetupToken
	}
	email, err := normalizeEmail(email)
	if err != nil {
		return domain.User{}, "", err
	}
	u, err := s.newUser(email, name, password)
	if err != nil {
		return domain.User{}, "", err
	}
	u.IsAdmin = true
	if u, err = s.users.CreateUser(ctx, u); err != nil {
		return domain.User{}, "", err
	}
	s.setupToken = ""
	token, err := s.newSession(ctx, u.ID)
	return u, token, err
}

// Login checks an email and password and returns the user and a new session token.
func (s *Service) Login(ctx context.Context, email, password string) (domain.User, string, error) {
	u, err := s.users.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, domain.ErrNotFound) {
		_ = s.hasher.Compare(s.dummyHash(), password)
		return domain.User{}, "", ErrWrongCredentials
	}
	if err != nil {
		return domain.User{}, "", err
	}
	if s.hasher.Compare(u.PasswordHash, password) != nil {
		return domain.User{}, "", ErrWrongCredentials
	}
	_ = s.users.DeleteExpiredSessions(ctx)
	token, err := s.newSession(ctx, u.ID)
	return u, token, err
}

// Authenticate returns the user a session token belongs to.
func (s *Service) Authenticate(ctx context.Context, sessionToken string) (domain.User, error) {
	if sessionToken == "" {
		return domain.User{}, ErrSignedOut
	}
	sess, err := s.users.GetSession(ctx, hashToken(sessionToken))
	if errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, ErrSignedOut
	}
	if err != nil {
		return domain.User{}, err
	}
	u, err := s.users.GetUser(ctx, sess.UserID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, ErrSignedOut
	}
	return u, err
}

func (s *Service) Logout(ctx context.Context, sessionToken string) error {
	if sessionToken == "" {
		return nil
	}
	return s.users.DeleteSession(ctx, hashToken(sessionToken))
}

func (s *Service) newSession(ctx context.Context, userID string) (string, error) {
	token := randomToken(32)
	err := s.users.CreateSession(ctx, domain.Session{
		TokenHash: hashToken(token),
		UserID:    userID,
		ExpiresAt: time.Now().Add(SessionTTL),
	})
	return token, err
}

// newUser checks name and password and hashes the password.
func (s *Service) newUser(email, name, password string) (domain.User, error) {
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > maxNameLength {
		return domain.User{}, ErrNameTooLong
	}
	if err := validatePassword(password); err != nil {
		return domain.User{}, err
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return domain.User{}, err
	}
	return domain.User{Email: email, Name: name, PasswordHash: hash}, nil
}

func validatePassword(p string) error {
	if utf8.RuneCountInString(p) < minPasswordLength {
		return errPasswordShort
	}
	if len(p) > maxPasswordBytes {
		return errPasswordLong
	}
	return nil
}

// normalizeEmail lowercases a bare address ("a@b.c"), refusing anything else.
func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if a, err := mail.ParseAddress(email); err != nil || a.Address != email {
		return "", ErrInvalidEmail
	}
	return email, nil
}

func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b) // never fails (crypto/rand panics on failure since Go 1.24)
	return base64.RawURLEncoding.EncodeToString(b)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
