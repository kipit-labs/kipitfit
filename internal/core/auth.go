package core

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

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
func (c *Core) InitAuth(ctx context.Context) error {
	n, err := c.store.CountUsers(ctx)
	if err != nil || n > 0 {
		return err
	}
	token := c.cfg.SetupToken
	if token == "" {
		token = randomToken(16)
	}
	c.setupMu.Lock()
	c.setupToken = token
	c.setupMu.Unlock()
	c.log.Warn("no admin account yet: open the app and use this setup token", "setup_token", token)
	return nil
}

func (c *Core) AuthState(ctx context.Context, sessionToken string) (AuthState, error) {
	n, err := c.store.CountUsers(ctx)
	if err != nil {
		return AuthState{}, err
	}
	if n == 0 {
		return AuthState{SetupRequired: true}, nil
	}
	u, err := c.Authenticate(ctx, sessionToken)
	if errors.Is(err, domain.ErrUnauthorized) {
		return AuthState{}, nil
	}
	if err != nil {
		return AuthState{}, err
	}
	return AuthState{User: &u}, nil
}

// Setup creates the first admin account and returns a session token.
func (c *Core) Setup(ctx context.Context, setupToken, email, name, password string) (domain.User, string, error) {
	c.setupMu.Lock()
	defer c.setupMu.Unlock()

	if n, err := c.store.CountUsers(ctx); err != nil {
		return domain.User{}, "", err
	} else if n > 0 {
		return domain.User{}, "", ErrSetupDone
	}
	if c.setupToken == "" || subtle.ConstantTimeCompare([]byte(strings.TrimSpace(setupToken)), []byte(c.setupToken)) != 1 {
		return domain.User{}, "", ErrWrongSetupToken
	}
	email, err := normalizeEmail(email)
	if err != nil {
		return domain.User{}, "", err
	}
	u, err := c.newUser(email, name, password)
	if err != nil {
		return domain.User{}, "", err
	}
	u.IsAdmin = true
	if u, err = c.store.CreateUser(ctx, u); err != nil {
		return domain.User{}, "", err
	}
	c.setupToken = ""
	token, err := c.newSession(ctx, u.ID)
	return u, token, err
}

// Login checks an email and password and returns the user and a new session token.
func (c *Core) Login(ctx context.Context, email, password string) (domain.User, string, error) {
	u, err := c.store.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, domain.ErrNotFound) {
		_ = c.hasher.Compare(c.dummyHash(), password)
		return domain.User{}, "", ErrWrongCredentials
	}
	if err != nil {
		return domain.User{}, "", err
	}
	if c.hasher.Compare(u.PasswordHash, password) != nil {
		return domain.User{}, "", ErrWrongCredentials
	}
	_ = c.store.DeleteExpiredSessions(ctx)
	token, err := c.newSession(ctx, u.ID)
	return u, token, err
}

// Authenticate returns the user a session token belongs to.
func (c *Core) Authenticate(ctx context.Context, sessionToken string) (domain.User, error) {
	if sessionToken == "" {
		return domain.User{}, ErrSignedOut
	}
	sess, err := c.store.GetSession(ctx, hashToken(sessionToken))
	if errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, ErrSignedOut
	}
	if err != nil {
		return domain.User{}, err
	}
	u, err := c.store.GetUser(ctx, sess.UserID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, ErrSignedOut
	}
	return u, err
}

func (c *Core) Logout(ctx context.Context, sessionToken string) error {
	if sessionToken == "" {
		return nil
	}
	return c.store.DeleteSession(ctx, hashToken(sessionToken))
}

func (c *Core) newSession(ctx context.Context, userID string) (string, error) {
	token := randomToken(32)
	err := c.store.CreateSession(ctx, domain.Session{
		TokenHash: hashToken(token),
		UserID:    userID,
		ExpiresAt: time.Now().Add(SessionTTL),
	})
	return token, err
}

// newUser checks name and password and hashes the password.
func (c *Core) newUser(email, name, password string) (domain.User, error) {
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > maxNameLength {
		return domain.User{}, ErrNameTooLong
	}
	if err := validatePassword(password); err != nil {
		return domain.User{}, err
	}
	hash, err := c.hasher.Hash(password)
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
