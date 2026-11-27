package domain

import (
	"context"
	"time"
)

type Folder struct {
	ID        string    `json:"id"`
	UserID    string    `json:"-"`
	Name      string    `json:"name"`
	Position  int       `json:"-"`
	CreatedAt time.Time `json:"-"`
}

type Routine struct {
	ID     string `json:"id"`
	UserID string `json:"-"`
	// FolderID is "" when the routine is in no folder.
	FolderID  string    `json:"folderId,omitempty"`
	Name      string    `json:"name"`
	Notes     string    `json:"notes"`
	Position  int       `json:"-"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
	// LastDoneAt is the end of its latest workout, nil if never done.
	LastDoneAt *time.Time        `json:"lastDoneAt"`
	Exercises  []RoutineExercise `json:"exercises"`
}

type RoutineExercise struct {
	ID         string `json:"-"`
	RoutineID  string `json:"-"`
	ExerciseID string `json:"exerciseId"`
	Position   int    `json:"-"`
	// Superset is the group letter shared by a superset's exercises, "" alone.
	Superset string `json:"superset,omitempty"`
	// RestS is the rest after each set, nil for the app default.
	RestS *int   `json:"restS"`
	Notes string `json:"notes"`
	// Exercise is filled in when reading, for its name and kind even archived.
	Exercise *Exercise    `json:"exercise,omitempty"`
	Sets     []RoutineSet `json:"sets"`
}

type SetType string

const (
	SetNormal  SetType = "normal"
	SetWarmup  SetType = "warmup"
	SetDrop    SetType = "drop"
	SetFailure SetType = "failure"
)

var SetTypes = []SetType{SetNormal, SetWarmup, SetDrop, SetFailure}

// RoutineSet is a target set; each target is nil when not set.
type RoutineSet struct {
	RoutineExerciseID string  `json:"-"`
	Position          int     `json:"-"`
	Type              SetType `json:"type"`
	WeightG           *int    `json:"weightG"`
	RepsMin           *int    `json:"repsMin"`
	RepsMax           *int    `json:"repsMax"`
	DurationS         *int    `json:"durationS"`
	DistanceM         *int    `json:"distanceM"`
}

// RoutineRepository keeps routines and their folders, always scoped to their
// user: someone else's is ErrNotFound.
type RoutineRepository interface {
	ListFolders(ctx context.Context, userID string) ([]Folder, error)
	// CreateFolder puts the folder after the user's others.
	CreateFolder(ctx context.Context, f Folder) (Folder, error)
	RenameFolder(ctx context.Context, f Folder) error
	// DeleteFolder leaves its routines in no folder.
	DeleteFolder(ctx context.Context, userID, id string) error

	// ListRoutines returns the user's routines by position, each with its
	// exercises (archived ones too) and target sets in order, and when it was
	// last done.
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
