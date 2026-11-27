package core

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"unicode/utf8"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

const (
	workoutsPage = 20
	// A workout longer than this is a phone left running, not a session.
	maxWorkoutHours = 24
)

var (
	ErrWorkoutID    = domain.Coded(domain.ErrInvalid, "workout_id_invalid", "a workout id is a ULID made on the phone")
	ErrWorkoutTimes = domain.Coded(domain.ErrInvalid, "workout_times_invalid", "a workout ends after it starts, within a day")
	ErrWorkoutEmpty = domain.Coded(domain.ErrInvalid, "workout_empty", "log at least one set")
	ErrSetMissing   = domain.Coded(domain.ErrInvalid, "set_value_missing", "a set is missing a value its exercise needs")
	ErrRPE          = domain.Coded(domain.ErrInvalid, "rpe_invalid", "RPE goes from 6 to 10 in steps of 0.5")
)

var ulidRE = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

// PutWorkout saves a finished workout under its phone-made id: created the
// first time, replaced after, so a finish sent twice saves once.
// The answer carries the records it beat.
func (c *Core) PutWorkout(ctx context.Context, me domain.User, id string, w domain.Workout) (WorkoutView, error) {
	if !ulidRE.MatchString(id) {
		return WorkoutView{}, ErrWorkoutID
	}
	w.ID, w.UserID = id, me.ID
	var err error
	if w.Title, err = title(w.Title); err != nil {
		return WorkoutView{}, err
	}
	if utf8.RuneCountInString(w.Notes) > maxNotes {
		return WorkoutView{}, ErrNotesTooLong
	}
	if w.StartedAt.IsZero() || w.EndedAt.Before(w.StartedAt) || w.EndedAt.Sub(w.StartedAt).Hours() > maxWorkoutHours {
		return WorkoutView{}, ErrWorkoutTimes
	}
	w.StartedAt, w.EndedAt = w.StartedAt.UTC(), w.EndedAt.UTC()
	if w.RoutineID != "" {
		// The routine may have been deleted while the finish waited for signal: save anyway.
		if _, err := c.store.GetRoutine(ctx, me.ID, w.RoutineID); errors.Is(err, domain.ErrNotFound) {
			w.RoutineID = ""
		} else if err != nil {
			return WorkoutView{}, err
		}
	}
	if len(w.Exercises) > maxExercises {
		return WorkoutView{}, ErrTooManyExercises
	}
	w.VolumeG = 0
	sets := 0
	for i := range w.Exercises {
		we := &w.Exercises[i]
		e, err := c.GetExercise(ctx, me, we.ExerciseID)
		if err != nil {
			return WorkoutView{}, err
		}
		we.Exercise = nil
		if we.Superset != "" && (len(we.Superset) != 1 || we.Superset[0] < 'A' || we.Superset[0] > 'Z') {
			return WorkoutView{}, ErrSuperset
		}
		if utf8.RuneCountInString(we.Notes) > maxNotes {
			return WorkoutView{}, ErrNotesTooLong
		}
		if len(we.Sets) > maxSets {
			return WorkoutView{}, ErrTooManySets
		}
		for _, s := range we.Sets {
			if err := checkSet(e.Kind, s); err != nil {
				return WorkoutView{}, err
			}
			w.VolumeG += volume(e.Kind, s)
		}
		sets += len(we.Sets)
	}
	if sets == 0 {
		return WorkoutView{}, ErrWorkoutEmpty
	}
	saved, err := c.store.PutWorkout(ctx, w)
	if err != nil {
		return WorkoutView{}, err
	}
	return c.workoutView(ctx, me, saved)
}

// checkSet validates a logged set: the values its exercise takes, the ones it
// needs (the weight of a bodyweight exercise is optional), and the RPE.
func checkSet(k domain.Kind, s domain.WorkoutSet) error {
	if !slices.Contains(domain.SetTypes, s.Type) {
		return ErrSetType
	}
	f := fieldsOf(k)
	if !value(s.WeightG, f.weight, maxWeightG) || !value(s.Reps, f.reps, maxReps) ||
		!value(s.DurationS, f.duration, maxDurationS) || !value(s.DistanceM, f.distance, maxDistanceM) {
		return ErrSetValue
	}
	if (k == domain.KindWeightReps && s.WeightG == nil) || (f.reps && s.Reps == nil) ||
		(f.duration && s.DurationS == nil) || (f.distance && s.DistanceM == nil) {
		return ErrSetMissing
	}
	if s.RPE != nil && (*s.RPE < 60 || *s.RPE > 100 || *s.RPE%5 != 0) {
		return ErrRPE
	}
	return nil
}

// volume is the weight moved by a set that isn't a warm-up: the added weight
// for a weighted bodyweight exercise, nothing for an assisted one (P7 counts
// the bodyweight).
func volume(k domain.Kind, s domain.WorkoutSet) int {
	if s.Type == domain.SetWarmup || s.WeightG == nil || s.Reps == nil ||
		(k != domain.KindWeightReps && k != domain.KindWeightedBodyweight) {
		return 0
	}
	return *s.WeightG * *s.Reps
}

// Workouts is a page of history, newest first; Next is the cursor of the
// next page, "" on the last.
type Workouts struct {
	Workouts []WorkoutView `json:"workouts"`
	Next     string        `json:"next,omitempty"`
}

func (c *Core) ListWorkouts(ctx context.Context, me domain.User, after string) (Workouts, error) {
	ws, err := c.store.ListWorkouts(ctx, me.ID, after, workoutsPage+1)
	if err != nil {
		return Workouts{}, err
	}
	page := Workouts{}
	if len(ws) > workoutsPage {
		ws = ws[:workoutsPage]
		page.Next = ws[workoutsPage-1].ID
	}
	page.Workouts, err = c.withRecords(ctx, me, ws)
	return page, err
}

func (c *Core) GetWorkout(ctx context.Context, me domain.User, id string) (WorkoutView, error) {
	w, err := c.store.GetWorkout(ctx, me.ID, id)
	if err != nil {
		return WorkoutView{}, err
	}
	return c.workoutView(ctx, me, w)
}

func (c *Core) workoutView(ctx context.Context, me domain.User, w domain.Workout) (WorkoutView, error) {
	vs, err := c.withRecords(ctx, me, []domain.Workout{w})
	if err != nil {
		return WorkoutView{}, err
	}
	return vs[0], nil
}

func (c *Core) DeleteWorkout(ctx context.Context, me domain.User, id string) error {
	return c.store.DeleteWorkout(ctx, me.ID, id)
}

// PreviousSets returns the sets of each exercise in me's latest workout with it.
func (c *Core) PreviousSets(ctx context.Context, me domain.User, exerciseIDs []string) (map[string][]domain.WorkoutSet, error) {
	if len(exerciseIDs) > maxExercises {
		return nil, fmt.Errorf("%w: at most %d exercises", domain.ErrInvalid, maxExercises)
	}
	return c.store.PreviousSets(ctx, me.ID, exerciseIDs)
}
