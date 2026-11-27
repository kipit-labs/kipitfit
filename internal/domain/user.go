package domain

import (
	"context"
	"time"
)

type Unit string

const (
	UnitKg Unit = "kg"
	UnitLb Unit = "lb"
)

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	PasswordHash string `json:"-"`
	IsAdmin      bool   `json:"isAdmin"`
	// Language is "en", "fr", or "" until picked.
	Language  string    `json:"language"`
	Unit      Unit      `json:"unit"`
	CreatedAt time.Time `json:"createdAt"`
}

type Session struct {
	TokenHash string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type Invite struct {
	ID        string     `json:"id"`
	TokenHash string     `json:"-"`
	Email     string     `json:"email"`
	CreatedBy string     `json:"createdBy"`
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt time.Time  `json:"expiresAt"`
	UsedAt    *time.Time `json:"usedAt,omitempty"`
}

// UserRepository keeps accounts, their sessions and the invites to join.
type UserRepository interface {
	CountUsers(ctx context.Context) (int, error)
	CreateUser(ctx context.Context, u User) (User, error)
	GetUser(ctx context.Context, id string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	// UpdateUser saves u's name, language and unit.
	UpdateUser(ctx context.Context, u User) error

	CreateSession(ctx context.Context, s Session) error
	// GetSession returns ErrNotFound for unknown or expired sessions.
	GetSession(ctx context.Context, tokenHash string) (Session, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	DeleteExpiredSessions(ctx context.Context) error

	CreateInvite(ctx context.Context, i Invite) (Invite, error)
	// ListInvites returns the unused, unexpired invites, newest first.
	ListInvites(ctx context.Context) ([]Invite, error)
	// AcceptInvite marks the invite used and creates u with its email, in one
	// transaction; ErrNotFound for unknown, used or expired invites.
	AcceptInvite(ctx context.Context, tokenHash string, u User) (User, error)
	DeleteInvite(ctx context.Context, id string) error
}

// PasswordHasher hashes passwords for storage and checks them.
type PasswordHasher interface {
	Hash(password string) (string, error)
	// Compare returns an error unless password is the one hash was made from.
	Compare(hash, password string) error
}
