package workouts

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/kipit-labs/kipitfit/internal/domain"
	"github.com/kipit-labs/kipitfit/internal/usecase/exercises"
	"github.com/kipit-labs/kipitfit/internal/usecase/records"
)

func ptr[T any](v T) *T { return &v }

// fakeExercises is a fixed set of exercises.
type fakeExercises struct {
	domain.ExerciseRepository // only GetExercise is used
	byID                      map[string]domain.Exercise
}

func (f fakeExercises) GetExercise(_ context.Context, id string) (domain.Exercise, error) {
	e, ok := f.byID[id]
	if !ok {
		return domain.Exercise{}, domain.ErrNotFound
	}
	return e, nil
}

// fakeRoutines holds routines by id.
type fakeRoutines struct {
	domain.RoutineRepository // only GetRoutine is used
	byID                     map[string]domain.Routine
}

func (f fakeRoutines) GetRoutine(_ context.Context, userID, id string) (domain.Routine, error) {
	r, ok := f.byID[id]
	if !ok || r.UserID != userID {
		return domain.Routine{}, domain.ErrNotFound
	}
	return r, nil
}

// fakeWorkouts keeps workouts in memory like the SQLite store: scoped to
// their user, each exercise filled in on reading.
type fakeWorkouts struct {
	exercises fakeExercises
	byID      map[string]domain.Workout
}

func (f *fakeWorkouts) read(w domain.Workout) domain.Workout {
	exs := make([]domain.WorkoutExercise, len(w.Exercises))
	for i, we := range w.Exercises {
		e := f.exercises.byID[we.ExerciseID]
		we.Exercise = &e
		exs[i] = we
	}
	w.Exercises = exs
	return w
}

func (f *fakeWorkouts) PutWorkout(_ context.Context, w domain.Workout) (domain.Workout, error) {
	if old, ok := f.byID[w.ID]; ok && old.UserID != w.UserID {
		return domain.Workout{}, domain.ErrNotFound
	}
	f.byID[w.ID] = w
	return f.read(w), nil
}

// mine is the user's workouts, oldest first.
func (f *fakeWorkouts) mine(userID string) []domain.Workout {
	out := []domain.Workout{}
	for _, w := range f.byID {
		if w.UserID == userID {
			out = append(out, f.read(w))
		}
	}
	slices.SortFunc(out, func(a, b domain.Workout) int {
		return cmp.Or(a.StartedAt.Compare(b.StartedAt), cmp.Compare(a.ID, b.ID))
	})
	return out
}

func (f *fakeWorkouts) ListWorkouts(_ context.Context, userID, after string, limit int) ([]domain.Workout, error) {
	ws := f.mine(userID)
	slices.Reverse(ws)
	if after != "" {
		i := slices.IndexFunc(ws, func(w domain.Workout) bool { return w.ID == after })
		ws = ws[i+1:]
	}
	return ws[:min(limit, len(ws))], nil
}

func (f *fakeWorkouts) WorkoutsSince(_ context.Context, userID string, from time.Time) ([]domain.Workout, error) {
	return slices.DeleteFunc(f.mine(userID), func(w domain.Workout) bool { return w.StartedAt.Before(from) }), nil
}

func (f *fakeWorkouts) GetWorkout(_ context.Context, userID, id string) (domain.Workout, error) {
	w, ok := f.byID[id]
	if !ok || w.UserID != userID {
		return domain.Workout{}, domain.ErrNotFound
	}
	return f.read(w), nil
}

func (f *fakeWorkouts) DeleteWorkout(_ context.Context, userID, id string) error {
	if w, ok := f.byID[id]; !ok || w.UserID != userID {
		return domain.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

func (f *fakeWorkouts) PreviousSets(_ context.Context, userID string, exerciseIDs []string) (map[string][]domain.WorkoutSet, error) {
	out := map[string][]domain.WorkoutSet{}
	for _, w := range f.mine(userID) { // a later workout overwrites an earlier one
		for _, we := range w.Exercises {
			if slices.Contains(exerciseIDs, we.ExerciseID) {
				out[we.ExerciseID] = we.Sets
			}
		}
	}
	return out, nil
}

func (f *fakeWorkouts) ExerciseSets(_ context.Context, userID string, exerciseIDs []string) ([]domain.LoggedSet, error) {
	out := []domain.LoggedSet{}
	for _, w := range f.mine(userID) {
		for i, we := range w.Exercises {
			if !slices.Contains(exerciseIDs, we.ExerciseID) {
				continue
			}
			for j, s := range we.Sets {
				out = append(out, domain.LoggedSet{WorkoutID: w.ID, Title: w.Title, StartedAt: w.StartedAt, ExerciseID: we.ExerciseID,
					Exercise: i, Set: j, Type: s.Type, WeightG: s.WeightG, Reps: s.Reps, RPE: s.RPE, DurationS: s.DurationS, DistanceM: s.DistanceM})
			}
		}
	}
	return out, nil
}

// env is the workouts of two users, me and friend, with an exercise of each
// kind and a routine of mine.
type env struct {
	*Service
	routines fakeRoutines
	store    fakeExercises
	records  *records.Service
	me       domain.User
	friend   domain.User
	byKind   map[domain.Kind]domain.Exercise
}

func newEnv() env {
	exs := fakeExercises{byID: map[string]domain.Exercise{}}
	byKind := map[domain.Kind]domain.Exercise{}
	for _, k := range domain.Kinds {
		e := domain.Exercise{ID: "ex-" + string(k), Name: string(k), Kind: k, PrimaryMuscles: []string{"chest", "triceps"}}
		exs.byID[e.ID], byKind[k] = e, e
	}
	ex := exercises.NewService(exs)
	repo := &fakeWorkouts{exercises: exs, byID: map[string]domain.Workout{}}
	rec := records.NewService(repo, ex)
	me := domain.User{ID: "me"}
	routines := fakeRoutines{byID: map[string]domain.Routine{"push": {ID: "push", UserID: me.ID, Name: "Push"}}}
	return env{Service: NewService(repo, routines, ex, rec), routines: routines, store: exs, records: rec, me: me,
		friend: domain.User{ID: "friend"}, byKind: byKind}
}
