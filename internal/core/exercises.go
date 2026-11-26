package core

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kipit-labs/kipitfit/internal/exercisedb"
	"github.com/kipit-labs/kipitfit/internal/store"
)

const maxExerciseName = 100

// The values exercises take; the web app translates each (exercises.<field>.<value>).
var (
	Kinds = []store.Kind{store.KindWeightReps, store.KindRepsOnly, store.KindWeightedBodyweight,
		store.KindAssistedBodyweight, store.KindDuration, store.KindDistanceDuration}
	Categories = []string{"strength", "powerlifting", "olympic_weightlifting", "strongman", "plyometrics", "stretching", "cardio"}
	Equipment  = []string{"barbell", "dumbbell", "kettlebells", "machine", "cable", "ez_curl_bar", "bands",
		"medicine_ball", "exercise_ball", "foam_roll", "body_only", "other", "none"}
	Muscles = []string{"abdominals", "abductors", "adductors", "biceps", "calves", "chest", "forearms", "glutes",
		"hamstrings", "lats", "lower_back", "middle_back", "neck", "quadriceps", "shoulders", "traps", "triceps"}
	Levels = []string{"beginner", "intermediate", "expert"}
)

var (
	ErrLibraryReadonly    = coded(ErrForbidden, "library_exercise_readonly", "library exercises can't be changed: create your own instead")
	ErrExerciseNameTaken  = coded(ErrConflict, "exercise_name_taken", "you already have an exercise with this name")
	ErrExerciseName       = coded(ErrInvalid, "exercise_name_invalid", "the name must be 1 to 100 characters")
	ErrExerciseField      = coded(ErrInvalid, "exercise_field_invalid", "an exercise field has a value the app doesn't know")
	ErrPrimaryMuscleEmpty = coded(ErrInvalid, "primary_muscle_required", "pick at least one primary muscle")
)

func init() { ErrExerciseName.Params = map[string]any{"max": maxExerciseName} }

// SyncExercises brings the library in line with the embedded dataset; a
// no-op when it already is.
func (c *Core) SyncExercises(ctx context.Context) error {
	start := time.Now()
	exs, err := exercisedb.Load()
	if err != nil {
		return err
	}
	wrote, err := c.store.SyncLibrary(ctx, exercisedb.Version(), exs)
	if err != nil {
		return err
	}
	if wrote {
		c.log.Info("exercise library synced", "exercises", len(exs), "version", exercisedb.Version(), "took", time.Since(start))
	}
	return nil
}

// ListExercises returns the library and me's custom exercises, compact.
func (c *Core) ListExercises(ctx context.Context, me store.User) ([]store.Exercise, error) {
	return c.store.ListExercises(ctx, me.ID)
}

// GetExercise returns a library exercise or one of me's; someone else's is
// not found.
func (c *Core) GetExercise(ctx context.Context, me store.User, id string) (store.Exercise, error) {
	e, err := c.store.GetExercise(ctx, id)
	if err != nil {
		return store.Exercise{}, err
	}
	if e.Custom && e.OwnerID != me.ID {
		return store.Exercise{}, store.ErrNotFound
	}
	return e, nil
}

// ExerciseInput is a custom exercise to create, or the fields of one to
// change (nil fields stay).
type ExerciseInput struct {
	Name             *string     `json:"name"`
	Kind             *store.Kind `json:"kind"`
	Category         *string     `json:"category"`
	Equipment        *string     `json:"equipment"`
	PrimaryMuscles   *[]string   `json:"primaryMuscles"`
	SecondaryMuscles *[]string   `json:"secondaryMuscles"`
}

func (c *Core) CreateExercise(ctx context.Context, me store.User, in ExerciseInput) (store.Exercise, error) {
	e := store.Exercise{OwnerID: me.ID, Category: "strength", Equipment: "none", SecondaryMuscles: []string{}}
	if err := apply(&e, in); err != nil {
		return store.Exercise{}, err
	}
	if in.Name == nil {
		return store.Exercise{}, ErrExerciseName
	}
	if in.Kind == nil {
		return store.Exercise{}, ErrExerciseField
	}
	e, err := c.store.CreateExercise(ctx, e)
	return e, nameTaken(err)
}

func (c *Core) UpdateExercise(ctx context.Context, me store.User, id string, in ExerciseInput) (store.Exercise, error) {
	e, err := c.ownExercise(ctx, me, id)
	if err != nil {
		return store.Exercise{}, err
	}
	if err := apply(&e, in); err != nil {
		return store.Exercise{}, err
	}
	e, err = c.store.UpdateExercise(ctx, e)
	return e, nameTaken(err)
}

func (c *Core) DeleteExercise(ctx context.Context, me store.User, id string) error {
	if _, err := c.ownExercise(ctx, me, id); err != nil {
		return err
	}
	return c.store.DeleteExercise(ctx, id)
}

// ownExercise is me's custom exercise id, for a change.
func (c *Core) ownExercise(ctx context.Context, me store.User, id string) (store.Exercise, error) {
	e, err := c.GetExercise(ctx, me, id)
	if err != nil {
		return store.Exercise{}, err
	}
	if !e.Custom {
		return store.Exercise{}, ErrLibraryReadonly
	}
	return e, nil
}

// apply validates in and copies its fields onto e.
func apply(e *store.Exercise, in ExerciseInput) error {
	if in.Name != nil {
		e.Name = strings.Join(strings.Fields(*in.Name), " ")
		if n := utf8.RuneCountInString(e.Name); n == 0 || n > maxExerciseName {
			return ErrExerciseName
		}
	}
	if in.Kind != nil {
		if !slices.Contains(Kinds, *in.Kind) {
			return ErrExerciseField
		}
		e.Kind = *in.Kind
	}
	for _, f := range []struct {
		in      *string
		out     *string
		allowed []string
	}{{in.Category, &e.Category, Categories}, {in.Equipment, &e.Equipment, Equipment}} {
		if f.in != nil {
			if !slices.Contains(f.allowed, *f.in) {
				return ErrExerciseField
			}
			*f.out = *f.in
		}
	}
	for _, f := range []struct{ in, out *[]string }{{in.PrimaryMuscles, &e.PrimaryMuscles}, {in.SecondaryMuscles, &e.SecondaryMuscles}} {
		if f.in != nil {
			ms := slices.Compact(slices.Sorted(slices.Values(*f.in)))
			for _, m := range ms {
				if !slices.Contains(Muscles, m) {
					return ErrExerciseField
				}
			}
			*f.out = ms
		}
	}
	if len(e.PrimaryMuscles) == 0 {
		return ErrPrimaryMuscleEmpty
	}
	return nil
}

func nameTaken(err error) error {
	if errors.Is(err, store.ErrConflict) {
		return ErrExerciseNameTaken
	}
	return err
}
