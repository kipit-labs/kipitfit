// Package store is kipitfit's persistence port: core talks to it only through
// Store, so SQL stays in one place and can move to Postgres later.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

type Store interface {
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

type Unit string

const (
	UnitKg Unit = "kg"
	UnitLb Unit = "lb"
)

type User struct {
	bun.BaseModel `bun:"table:users,alias:u" json:"-"`

	ID           string `bun:"id,pk" json:"id"`
	Email        string `bun:"email" json:"email"`
	Name         string `bun:"name" json:"name"`
	PasswordHash string `bun:"password_hash" json:"-"`
	IsAdmin      bool   `bun:"is_admin" json:"isAdmin"`
	// Language is "en", "fr", or "" until picked.
	Language  string    `bun:"language" json:"language"`
	Unit      Unit      `bun:"unit" json:"unit"`
	CreatedAt time.Time `bun:"created_at" json:"createdAt"`
}

type Session struct {
	bun.BaseModel `bun:"table:sessions,alias:session" json:"-"`

	TokenHash string    `bun:"token_hash,pk"`
	UserID    string    `bun:"user_id"`
	CreatedAt time.Time `bun:"created_at"`
	ExpiresAt time.Time `bun:"expires_at"`
}

type Invite struct {
	bun.BaseModel `bun:"table:invites,alias:invite" json:"-"`

	ID        string     `bun:"id,pk" json:"id"`
	TokenHash string     `bun:"token_hash" json:"-"`
	Email     string     `bun:"email" json:"email"`
	CreatedBy string     `bun:"created_by" json:"createdBy"`
	CreatedAt time.Time  `bun:"created_at" json:"createdAt"`
	ExpiresAt time.Time  `bun:"expires_at" json:"expiresAt"`
	UsedAt    *time.Time `bun:"used_at" json:"usedAt,omitempty"`
}
