package core

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/kipit-labs/kipitfit/internal/store"
)

const (
	maxTitle     = 100
	maxNotes     = 1000
	maxExercises = 50
	maxSets      = 50
	// Bounds of a set value, far above any real one: 1 t, 10 000 reps, 24 h, 1000 km.
	maxWeightG   = 1_000_000
	maxReps      = 10_000
	maxDurationS = 24 * 3600
	maxDistanceM = 1_000_000
	maxRestS     = 3600
	// More folders or routines than one reorder can list.
	maxOrder = 1000
)

var SetTypes = []store.SetType{store.SetNormal, store.SetWarmup, store.SetDrop, store.SetFailure}

var (
	ErrTitle            = coded(ErrInvalid, "title_invalid", fmt.Sprintf("the name must be 1 to %d characters", maxTitle))
	ErrNotesTooLong     = coded(ErrInvalid, "notes_too_long", fmt.Sprintf("notes can be at most %d characters", maxNotes))
	ErrTooManyExercises = coded(ErrInvalid, "too_many_exercises", fmt.Sprintf("at most %d exercises", maxExercises))
	ErrTooManySets      = coded(ErrInvalid, "too_many_sets", fmt.Sprintf("at most %d sets per exercise", maxSets))
	ErrSetType          = coded(ErrInvalid, "set_type_invalid", "a set type must be normal, warmup, drop or failure")
	ErrSetValue         = coded(ErrInvalid, "set_value_invalid", "a set has a value out of range or that its exercise doesn't take")
	ErrSuperset         = coded(ErrInvalid, "superset_invalid", "a superset is one letter, A to Z")
)

func init() {
	ErrTitle.Params = map[string]any{"max": maxTitle}
	ErrNotesTooLong.Params = map[string]any{"max": maxNotes}
	ErrTooManyExercises.Params = map[string]any{"max": maxExercises}
	ErrTooManySets.Params = map[string]any{"max": maxSets}
}

// Routines are me's folders and routines, each in order.
type Routines struct {
	Folders  []store.Folder  `json:"folders"`
	Routines []store.Routine `json:"routines"`
}

func (c *Core) ListRoutines(ctx context.Context, me store.User) (Routines, error) {
	fs, err := c.store.ListFolders(ctx, me.ID)
	if err != nil {
		return Routines{}, err
	}
	rs, err := c.store.ListRoutines(ctx, me.ID)
	return Routines{Folders: fs, Routines: rs}, err
}

// CreateRoutine saves in as a new routine of me's, after the others.
func (c *Core) CreateRoutine(ctx context.Context, me store.User, in store.Routine) (store.Routine, error) {
	if err := c.checkRoutine(ctx, me, &in); err != nil {
		return store.Routine{}, err
	}
	return c.store.CreateRoutine(ctx, in)
}

// UpdateRoutine replaces routine id with in, exercises and sets included.
func (c *Core) UpdateRoutine(ctx context.Context, me store.User, id string, in store.Routine) (store.Routine, error) {
	if err := c.checkRoutine(ctx, me, &in); err != nil {
		return store.Routine{}, err
	}
	in.ID = id
	return c.store.UpdateRoutine(ctx, in)
}

func (c *Core) DeleteRoutine(ctx context.Context, me store.User, id string) error {
	return c.store.DeleteRoutine(ctx, me.ID, id)
}

// checkRoutine validates and normalises r as me's; its folder and exercises
// must be ones me sees.
func (c *Core) checkRoutine(ctx context.Context, me store.User, r *store.Routine) error {
	r.UserID = me.ID
	var err error
	if r.Name, err = title(r.Name); err != nil {
		return err
	}
	if utf8.RuneCountInString(r.Notes) > maxNotes {
		return ErrNotesTooLong
	}
	if r.FolderID != "" {
		fs, err := c.store.ListFolders(ctx, me.ID)
		if err != nil {
			return err
		}
		if !slices.ContainsFunc(fs, func(f store.Folder) bool { return f.ID == r.FolderID }) {
			return store.ErrNotFound
		}
	}
	if len(r.Exercises) > maxExercises {
		return ErrTooManyExercises
	}
	for i := range r.Exercises {
		re := &r.Exercises[i]
		// Archived exercises stay valid, so a routine using one can still be saved.
		e, err := c.GetExercise(ctx, me, re.ExerciseID)
		if err != nil {
			return err
		}
		re.Exercise = nil
		if re.Superset != "" && (len(re.Superset) != 1 || re.Superset[0] < 'A' || re.Superset[0] > 'Z') {
			return ErrSuperset
		}
		if re.RestS != nil && (*re.RestS < 0 || *re.RestS > maxRestS) {
			return ErrSetValue
		}
		if utf8.RuneCountInString(re.Notes) > maxNotes {
			return ErrNotesTooLong
		}
		if len(re.Sets) > maxSets {
			return ErrTooManySets
		}
		for _, s := range re.Sets {
			if err := checkTarget(e.Kind, s); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkTarget(k store.Kind, s store.RoutineSet) error {
	if !slices.Contains(SetTypes, s.Type) {
		return ErrSetType
	}
	f := fieldsOf(k)
	ok := value(s.WeightG, f.weight, maxWeightG) && value(s.RepsMin, f.reps, maxReps) &&
		value(s.RepsMax, f.reps, maxReps) && value(s.DurationS, f.duration, maxDurationS) &&
		value(s.DistanceM, f.distance, maxDistanceM)
	if !ok || (s.RepsMin != nil && s.RepsMax != nil && *s.RepsMin > *s.RepsMax) {
		return ErrSetValue
	}
	return nil
}

// fields are the values a set of an exercise kind takes.
type fields struct{ weight, reps, duration, distance bool }

func fieldsOf(k store.Kind) fields {
	switch k {
	case store.KindWeightReps, store.KindWeightedBodyweight, store.KindAssistedBodyweight:
		return fields{weight: true, reps: true}
	case store.KindRepsOnly:
		return fields{reps: true}
	case store.KindDuration:
		return fields{duration: true}
	case store.KindDistanceDuration:
		return fields{duration: true, distance: true}
	}
	return fields{}
}

// value reports whether v is unset, or allowed and within 0..maxV.
func value(v *int, allowed bool, maxV int) bool {
	return v == nil || (allowed && *v >= 0 && *v <= maxV)
}

// title is a routine, folder or workout name with its spaces tidied.
func title(s string) (string, error) {
	s = strings.Join(strings.Fields(s), " ")
	if n := utf8.RuneCountInString(s); n == 0 || n > maxTitle {
		return "", ErrTitle
	}
	return s, nil
}

// FolderInput names a folder.
type FolderInput struct {
	Name string `json:"name"`
}

func (c *Core) ListFolders(ctx context.Context, me store.User) ([]store.Folder, error) {
	return c.store.ListFolders(ctx, me.ID)
}

func (c *Core) CreateFolder(ctx context.Context, me store.User, in FolderInput) (store.Folder, error) {
	name, err := title(in.Name)
	if err != nil {
		return store.Folder{}, err
	}
	return c.store.CreateFolder(ctx, store.Folder{UserID: me.ID, Name: name})
}

func (c *Core) RenameFolder(ctx context.Context, me store.User, id string, in FolderInput) (store.Folder, error) {
	name, err := title(in.Name)
	if err != nil {
		return store.Folder{}, err
	}
	f := store.Folder{ID: id, UserID: me.ID, Name: name}
	return f, c.store.RenameFolder(ctx, f)
}

// DeleteFolder deletes the folder; its routines stay, in no folder.
func (c *Core) DeleteFolder(ctx context.Context, me store.User, id string) error {
	return c.store.DeleteFolder(ctx, me.ID, id)
}

// Order lists folders and routines in their new order; the ones left out keep
// their position.
type Order struct {
	Folders  []string `json:"folders"`
	Routines []string `json:"routines"`
}

func (c *Core) Reorder(ctx context.Context, me store.User, in Order) error {
	if len(in.Folders) > maxOrder || len(in.Routines) > maxOrder {
		return fmt.Errorf("%w: at most %d ids", ErrInvalid, maxOrder)
	}
	return c.store.Reorder(ctx, me.ID, in.Folders, in.Routines)
}
