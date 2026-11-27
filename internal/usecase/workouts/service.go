package workouts

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"unicode/utf8"

	"github.com/kipit-labs/kipitfit/internal/domain"
	"github.com/kipit-labs/kipitfit/internal/usecase/exercises"
	"github.com/kipit-labs/kipitfit/internal/usecase/records"
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

// Service keeps each user's finished workouts.
type Service struct {
	repo      domain.WorkoutRepository
	routines  domain.RoutineRepository
	exercises *exercises.Service
	records   *records.Service
}

func NewService(repo domain.WorkoutRepository, routines domain.RoutineRepository, ex *exercises.Service, rec *records.Service) *Service {
	return &Service{repo: repo, routines: routines, exercises: ex, records: rec}
}

var ulidRE = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

// PutWorkout saves a finished workout under its phone-made id: created the
// first time, replaced after, so a finish sent twice saves once.
// The answer carries the records it beat.
func (s *Service) PutWorkout(ctx context.Context, me domain.User, id string, w domain.Workout) (records.WorkoutView, error) {
	if !ulidRE.MatchString(id) {
		return records.WorkoutView{}, ErrWorkoutID
	}
	w.ID, w.UserID = id, me.ID
	var err error
	if w.Title, err = domain.Title(w.Title); err != nil {
		return records.WorkoutView{}, err
	}
	if utf8.RuneCountInString(w.Notes) > domain.MaxNotes {
		return records.WorkoutView{}, domain.ErrNotesTooLong
	}
	if w.StartedAt.IsZero() || w.EndedAt.Before(w.StartedAt) || w.EndedAt.Sub(w.StartedAt).Hours() > maxWorkoutHours {
		return records.WorkoutView{}, ErrWorkoutTimes
	}
	w.StartedAt, w.EndedAt = w.StartedAt.UTC(), w.EndedAt.UTC()
	if w.RoutineID != "" {
		// The routine may have been deleted while the finish waited for signal: save anyway.
		if _, err := s.routines.GetRoutine(ctx, me.ID, w.RoutineID); errors.Is(err, domain.ErrNotFound) {
			w.RoutineID = ""
		} else if err != nil {
			return records.WorkoutView{}, err
		}
	}
	if len(w.Exercises) > domain.MaxExercises {
		return records.WorkoutView{}, domain.ErrTooManyExercises
	}
	w.VolumeG = 0
	sets := 0
	for i := range w.Exercises {
		we := &w.Exercises[i]
		e, err := s.exercises.GetExercise(ctx, me, we.ExerciseID)
		if err != nil {
			return records.WorkoutView{}, err
		}
		we.Exercise = nil
		if !domain.ValidSuperset(we.Superset) {
			return records.WorkoutView{}, domain.ErrSuperset
		}
		if utf8.RuneCountInString(we.Notes) > domain.MaxNotes {
			return records.WorkoutView{}, domain.ErrNotesTooLong
		}
		if len(we.Sets) > domain.MaxSets {
			return records.WorkoutView{}, domain.ErrTooManySets
		}
		for _, set := range we.Sets {
			if err := checkSet(e.Kind, set); err != nil {
				return records.WorkoutView{}, err
			}
			w.VolumeG += volume(e.Kind, set)
		}
		sets += len(we.Sets)
	}
	if sets == 0 {
		return records.WorkoutView{}, ErrWorkoutEmpty
	}
	saved, err := s.repo.PutWorkout(ctx, w)
	if err != nil {
		return records.WorkoutView{}, err
	}
	return s.workoutView(ctx, me, saved)
}

// checkSet validates a logged set: the values its exercise takes, the ones it
// needs (the weight of a bodyweight exercise is optional), and the RPE.
func checkSet(k domain.Kind, s domain.WorkoutSet) error {
	if !slices.Contains(domain.SetTypes, s.Type) {
		return domain.ErrSetType
	}
	f := k.Fields()
	if !domain.InRange(s.WeightG, f.Weight, domain.MaxWeightG) || !domain.InRange(s.Reps, f.Reps, domain.MaxReps) ||
		!domain.InRange(s.DurationS, f.Duration, domain.MaxDurationS) || !domain.InRange(s.DistanceM, f.Distance, domain.MaxDistanceM) {
		return domain.ErrSetValue
	}
	if (k == domain.KindWeightReps && s.WeightG == nil) || (f.Reps && s.Reps == nil) ||
		(f.Duration && s.DurationS == nil) || (f.Distance && s.DistanceM == nil) {
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
	Workouts []records.WorkoutView `json:"workouts"`
	Next     string                `json:"next,omitempty"`
}

func (s *Service) ListWorkouts(ctx context.Context, me domain.User, after string) (Workouts, error) {
	ws, err := s.repo.ListWorkouts(ctx, me.ID, after, workoutsPage+1)
	if err != nil {
		return Workouts{}, err
	}
	page := Workouts{}
	if len(ws) > workoutsPage {
		ws = ws[:workoutsPage]
		page.Next = ws[workoutsPage-1].ID
	}
	page.Workouts, err = s.records.WithRecords(ctx, me, ws)
	return page, err
}

func (s *Service) GetWorkout(ctx context.Context, me domain.User, id string) (records.WorkoutView, error) {
	w, err := s.repo.GetWorkout(ctx, me.ID, id)
	if err != nil {
		return records.WorkoutView{}, err
	}
	return s.workoutView(ctx, me, w)
}

func (s *Service) workoutView(ctx context.Context, me domain.User, w domain.Workout) (records.WorkoutView, error) {
	vs, err := s.records.WithRecords(ctx, me, []domain.Workout{w})
	if err != nil {
		return records.WorkoutView{}, err
	}
	return vs[0], nil
}

func (s *Service) DeleteWorkout(ctx context.Context, me domain.User, id string) error {
	return s.repo.DeleteWorkout(ctx, me.ID, id)
}

// PreviousSets returns the sets of each exercise in me's latest workout with it.
func (s *Service) PreviousSets(ctx context.Context, me domain.User, exerciseIDs []string) (map[string][]domain.WorkoutSet, error) {
	if len(exerciseIDs) > domain.MaxExercises {
		return nil, fmt.Errorf("%w: at most %d exercises", domain.ErrInvalid, domain.MaxExercises)
	}
	return s.repo.PreviousSets(ctx, me.ID, exerciseIDs)
}
