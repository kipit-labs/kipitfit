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

	// SyncLibrary upserts the library exercises by slug, in one transaction,
	// unless version is the one last synced; it never deletes a library row.
	// It reports whether it wrote anything.
	SyncLibrary(ctx context.Context, version string, exs []Exercise) (bool, error)
	// ListExercises returns the library and ownerID's custom exercises by
	// name, without instructions, force, level and mechanic, and without
	// archived ones.
	ListExercises(ctx context.Context, ownerID string) ([]Exercise, error)
	GetExercise(ctx context.Context, id string) (Exercise, error)
	// CreateExercise and UpdateExercise return ErrConflict when the owner
	// already has a custom exercise with that name (ASCII case ignored).
	CreateExercise(ctx context.Context, e Exercise) (Exercise, error)
	UpdateExercise(ctx context.Context, e Exercise) (Exercise, error)
	// DeleteExercise deletes the exercise, or archives it when a routine
	// uses it.
	DeleteExercise(ctx context.Context, id string) error

	// Folders and routines are always scoped to their user: someone else's is
	// ErrNotFound.
	ListFolders(ctx context.Context, userID string) ([]Folder, error)
	// CreateFolder puts the folder after the user's others.
	CreateFolder(ctx context.Context, f Folder) (Folder, error)
	RenameFolder(ctx context.Context, f Folder) error
	// DeleteFolder leaves its routines in no folder.
	DeleteFolder(ctx context.Context, userID, id string) error

	// ListRoutines returns the user's routines by position, each with its
	// exercises (archived ones too) and target sets in order.
	ListRoutines(ctx context.Context, userID string) ([]Routine, error)
	GetRoutine(ctx context.Context, userID, id string) (Routine, error)
	// CreateRoutine puts the routine after the user's others.
	CreateRoutine(ctx context.Context, r Routine) (Routine, error)
	// UpdateRoutine replaces the routine's fields, exercises and sets, in one
	// transaction; its position stays.
	UpdateRoutine(ctx context.Context, r Routine) (Routine, error)
	DeleteRoutine(ctx context.Context, userID, id string) error
	// Reorder sets the position of each listed folder and routine to its index,
	// in one transaction; ErrNotFound when one isn't the user's.
	Reorder(ctx context.Context, userID string, folders, routines []string) error
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

// Kind is what a set of the exercise records.
type Kind string

const (
	KindWeightReps         Kind = "weight_reps"
	KindRepsOnly           Kind = "reps_only"
	KindWeightedBodyweight Kind = "weighted_bodyweight"
	KindAssistedBodyweight Kind = "assisted_bodyweight"
	KindDuration           Kind = "duration"
	KindDistanceDuration   Kind = "distance_duration"
)

// Exercise is a library exercise (OwnerID "", Slug set) or a user's custom one.
// Category, equipment, muscles, force, level and mechanic are snake_case keys
// the web app translates.
type Exercise struct {
	bun.BaseModel `bun:"table:exercises,alias:ex" json:"-"`

	ID               string    `bun:"id,pk" json:"id"`
	OwnerID          string    `bun:"owner_id,nullzero" json:"-"`
	Slug             string    `bun:"slug,nullzero" json:"-"`
	Name             string    `bun:"name" json:"name"`
	Kind             Kind      `bun:"kind" json:"kind"`
	Category         string    `bun:"category" json:"category"`
	Equipment        string    `bun:"equipment" json:"equipment"`
	PrimaryMuscles   []string  `bun:"primary_muscles,type:json" json:"primaryMuscles"`
	SecondaryMuscles []string  `bun:"secondary_muscles,type:json" json:"secondaryMuscles"`
	Force            string    `bun:"force" json:"force,omitempty"`
	Level            string    `bun:"level" json:"level,omitempty"`
	Mechanic         string    `bun:"mechanic" json:"mechanic,omitempty"`
	Instructions     []string  `bun:"instructions,type:json" json:"instructions,omitempty"`
	CreatedAt        time.Time `bun:"created_at" json:"-"`
	UpdatedAt        time.Time `bun:"updated_at" json:"-"`
	ArchivedAt       time.Time `bun:"archived_at,nullzero" json:"-"`
	// Custom and Archived are set by the store: the exercise belongs to a
	// user; it was deleted while in use.
	Custom   bool `bun:"-" json:"custom"`
	Archived bool `bun:"-" json:"archived"`
}

type Folder struct {
	bun.BaseModel `bun:"table:routine_folders,alias:f" json:"-"`

	ID        string    `bun:"id,pk" json:"id"`
	UserID    string    `bun:"user_id" json:"-"`
	Name      string    `bun:"name" json:"name"`
	Position  int       `bun:"position" json:"-"`
	CreatedAt time.Time `bun:"created_at" json:"-"`
}

type Routine struct {
	bun.BaseModel `bun:"table:routines,alias:r" json:"-"`

	ID     string `bun:"id,pk" json:"id"`
	UserID string `bun:"user_id" json:"-"`
	// FolderID is "" when the routine is in no folder.
	FolderID  string            `bun:"folder_id,nullzero" json:"folderId,omitempty"`
	Name      string            `bun:"name" json:"name"`
	Notes     string            `bun:"notes" json:"notes"`
	Position  int               `bun:"position" json:"-"`
	CreatedAt time.Time         `bun:"created_at" json:"-"`
	UpdatedAt time.Time         `bun:"updated_at" json:"-"`
	Exercises []RoutineExercise `bun:"rel:has-many,join:id=routine_id" json:"exercises"`
}

type RoutineExercise struct {
	bun.BaseModel `bun:"table:routine_exercises,alias:re" json:"-"`

	ID         string `bun:"id,pk" json:"-"`
	RoutineID  string `bun:"routine_id" json:"-"`
	ExerciseID string `bun:"exercise_id" json:"exerciseId"`
	Position   int    `bun:"position" json:"-"`
	// Superset is the group letter shared by a superset's exercises, "" alone.
	Superset string `bun:"superset,nullzero" json:"superset,omitempty"`
	// RestS is the rest after each set, nil for the app default.
	RestS *int   `bun:"rest_s" json:"restS"`
	Notes string `bun:"notes" json:"notes"`
	// Exercise is filled in when reading, for its name and kind even archived.
	Exercise *Exercise    `bun:"rel:belongs-to,join:exercise_id=id" json:"exercise,omitempty"`
	Sets     []RoutineSet `bun:"rel:has-many,join:id=routine_exercise_id" json:"sets"`
}

type SetType string

const (
	SetNormal  SetType = "normal"
	SetWarmup  SetType = "warmup"
	SetDrop    SetType = "drop"
	SetFailure SetType = "failure"
)

// RoutineSet is a target set; each target is nil when not set.
type RoutineSet struct {
	bun.BaseModel `bun:"table:routine_sets,alias:rs" json:"-"`

	RoutineExerciseID string  `bun:"routine_exercise_id,pk" json:"-"`
	Position          int     `bun:"position,pk" json:"-"`
	Type              SetType `bun:"type" json:"type"`
	WeightG           *int    `bun:"weight_g" json:"weightG"`
	RepsMin           *int    `bun:"reps_min" json:"repsMin"`
	RepsMax           *int    `bun:"reps_max" json:"repsMax"`
	DurationS         *int    `bun:"duration_s" json:"durationS"`
	DistanceM         *int    `bun:"distance_m" json:"distanceM"`
}
