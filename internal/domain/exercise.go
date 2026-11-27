package domain

import (
	"context"
	"time"
)

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

// The values exercises take; the web app translates each (exercises.<field>.<value>).
var (
	Kinds = []Kind{KindWeightReps, KindRepsOnly, KindWeightedBodyweight,
		KindAssistedBodyweight, KindDuration, KindDistanceDuration}
	Categories = []string{"strength", "powerlifting", "olympic_weightlifting", "strongman", "plyometrics", "stretching", "cardio"}
	Equipment  = []string{"barbell", "dumbbell", "kettlebells", "machine", "cable", "ez_curl_bar", "bands",
		"medicine_ball", "exercise_ball", "foam_roll", "body_only", "other", "none"}
	Muscles = []string{"abdominals", "abductors", "adductors", "biceps", "calves", "chest", "forearms", "glutes",
		"hamstrings", "hip_flexors", "lats", "lower_back", "middle_back", "neck", "quadriceps", "shoulders", "traps", "triceps"}
	Levels = []string{"beginner", "intermediate", "expert"}
)

// Exercise is a library exercise (OwnerID "", Slug set) or a user's custom one.
// Category, equipment, muscles, force, level and mechanic are snake_case keys
// the web app translates.
type Exercise struct {
	ID               string   `json:"id"`
	OwnerID          string   `json:"-"`
	Slug             string   `json:"-"`
	Name             string   `json:"name"`
	Kind             Kind     `json:"kind"`
	Category         string   `json:"category"`
	Equipment        string   `json:"equipment"`
	PrimaryMuscles   []string `json:"primaryMuscles"`
	SecondaryMuscles []string `json:"secondaryMuscles"`
	Force            string   `json:"force,omitempty"`
	Level            string   `json:"level,omitempty"`
	Mechanic         string   `json:"mechanic,omitempty"`
	Instructions     []string `json:"instructions,omitempty"`
	// Media are the bucket keys of a pack exercise's pictures, in frame order.
	Media      []string  `json:"media"`
	CreatedAt  time.Time `json:"-"`
	UpdatedAt  time.Time `json:"-"`
	ArchivedAt time.Time `json:"-"`
	// Custom and Archived are set by the store: the exercise belongs to a
	// user; it was deleted while in use.
	Custom   bool `json:"custom"`
	Archived bool `json:"archived"`
}

// Pack is where the library came from, and the credit its licence asks for.
type Pack struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Credit    string `json:"credit"`
	CreditURL string `json:"creditUrl"`
}

// Replaced counts what ReplaceLibrary did.
type Replaced struct {
	Exercises, Repointed, Archived int
}

type ExerciseRepository interface {
	// ListExercises returns the library and ownerID's custom exercises by
	// name, without instructions, force, level and mechanic, and without
	// archived ones.
	ListExercises(ctx context.Context, ownerID string) ([]Exercise, error)
	GetExercise(ctx context.Context, id string) (Exercise, error)
	// CreateExercise and UpdateExercise return ErrConflict when the owner
	// already has a custom exercise with that name (ASCII case ignored).
	CreateExercise(ctx context.Context, e Exercise) (Exercise, error)
	UpdateExercise(ctx context.Context, e Exercise) (Exercise, error)
	// DeleteExercise deletes the exercise, or archives it when a routine or a
	// workout uses it.
	DeleteExercise(ctx context.Context, id string) error
}

// LibraryRepository keeps where the library exercises came from.
type LibraryRepository interface {
	// SyncLibrary upserts the library exercises by slug, in one transaction,
	// unless version is the one last synced; it never deletes a library row.
	// It reports whether it wrote anything.
	SyncLibrary(ctx context.Context, version string, exs []Exercise) (bool, error)
	// LibraryPack returns the pack the library was imported from, ErrNotFound
	// while it is the built-in one.
	LibraryPack(ctx context.Context) (Pack, error)
	// ReplaceLibrary makes exs the library, in one transaction: it upserts
	// them by slug, re-points routines and workouts from each old slug in
	// repoint to its new one, archives the other library exercises, and
	// records p.
	ReplaceLibrary(ctx context.Context, p Pack, exs []Exercise, repoint map[string]string) (Replaced, error)
}

// ExerciseDataset is the built-in library shipped in the binary.
type ExerciseDataset interface {
	// Version changes whenever the data, or how it is read, does.
	Version() string
	Load() ([]Exercise, error)
}
