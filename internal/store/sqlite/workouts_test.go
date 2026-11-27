package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kipit-labs/kipitfit/internal/ids"
	"github.com/kipit-labs/kipitfit/internal/store"
)

func TestWorkouts(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	a, _ := s.CreateUser(ctx, store.User{Email: "a@b.c", PasswordHash: "h"})
	b, _ := s.CreateUser(ctx, store.User{Email: "b@b.c", PasswordHash: "h"})
	if _, err := s.SyncLibrary(ctx, "v1", []store.Exercise{libraryRow("bench", "Bench Press"), libraryRow("row", "Row")}); err != nil {
		t.Fatal(err)
	}
	lib, _ := s.ListExercises(ctx, a.ID)
	bench, row := lib[0].ID, lib[1].ID
	r, _ := s.CreateRoutine(ctx, store.Routine{UserID: a.ID, Name: "Push"})

	start := time.Date(2026, 11, 20, 18, 0, 0, 0, time.UTC)
	workout := func(id string, at time.Time, weight int) store.Workout {
		rpe := store.Tenths(85)
		return store.Workout{ID: id, UserID: a.ID, RoutineID: r.ID, Title: "Push", StartedAt: at, EndedAt: at.Add(time.Hour), VolumeG: 1,
			Exercises: []store.WorkoutExercise{{ExerciseID: bench, Sets: []store.WorkoutSet{
				{Type: store.SetWarmup, WeightG: ptr(20000), Reps: ptr(10)},
				{Type: store.SetNormal, WeightG: ptr(weight), Reps: ptr(5), RPE: &rpe},
			}}}}
	}
	first, second := ids.New(), ids.New()
	w, err := s.PutWorkout(ctx, workout(first, start, 80000))
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Exercises) != 1 || w.Exercises[0].Exercise.Name != "Bench Press" || *w.Exercises[0].Sets[1].RPE != 85 {
		t.Fatalf("put = %+v", w)
	}
	// The same id again replaces it: still one workout.
	if _, err := s.PutWorkout(ctx, workout(first, start, 82500)); err != nil {
		t.Fatal(err)
	}
	if ws, _ := s.ListWorkouts(ctx, a.ID, "", 20); len(ws) != 1 {
		t.Fatalf("after a retry: %d workouts", len(ws))
	}
	// Someone else's id is not found, and stays theirs.
	if _, err := s.PutWorkout(ctx, store.Workout{ID: first, UserID: b.ID, Title: "x", StartedAt: start, EndedAt: start}); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("b puts a's workout: %v", err)
	}
	if _, err := s.GetWorkout(ctx, b.ID, first); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("b gets a's workout: %v", err)
	}

	if _, err := s.PutWorkout(ctx, workout(second, start.Add(48*time.Hour), 85000)); err != nil {
		t.Fatal(err)
	}
	ws, _ := s.ListWorkouts(ctx, a.ID, "", 1)
	if len(ws) != 1 || ws[0].ID != second || len(ws[0].Exercises) != 1 || len(ws[0].Exercises[0].Sets) != 2 {
		t.Fatalf("page 1 = %+v", ws)
	}
	ws, _ = s.ListWorkouts(ctx, a.ID, ws[0].ID, 1)
	if len(ws) != 1 || ws[0].ID != first {
		t.Fatalf("page 2 = %+v", ws)
	}
	if ws, _ = s.ListWorkouts(ctx, a.ID, first, 1); len(ws) != 0 {
		t.Fatalf("page 3 = %+v", ws)
	}

	since, err := s.WorkoutsSince(ctx, a.ID, start.Add(time.Hour))
	if err != nil || len(since) != 1 || since[0].ID != second || since[0].Exercises[0].Exercise.Name != "Bench Press" {
		t.Fatalf("since = %+v, %v", since, err)
	}
	rows, err := s.ExerciseSets(ctx, a.ID, []string{bench, row})
	if err != nil || len(rows) != 4 || rows[0].WorkoutID != first || rows[3].WorkoutID != second ||
		rows[1].Set != 1 || *rows[1].WeightG != 82500 || !rows[0].StartedAt.Equal(start) || rows[0].Title != "Push" || rows[0].Type != store.SetWarmup {
		t.Fatalf("exercise sets = %+v, %v", rows, err)
	}
	if rows, _ := s.ExerciseSets(ctx, b.ID, []string{bench}); len(rows) != 0 {
		t.Errorf("b sees a's sets: %+v", rows)
	}

	prev, err := s.PreviousSets(ctx, a.ID, []string{bench, row})
	if err != nil || len(prev) != 1 || *prev[bench][1].WeightG != 85000 {
		t.Fatalf("previous = %+v, %v", prev, err)
	}
	if prev, _ := s.PreviousSets(ctx, b.ID, []string{bench}); len(prev) != 0 {
		t.Errorf("b's previous = %+v", prev)
	}

	got, _ := s.GetRoutine(ctx, a.ID, r.ID)
	if got.LastDoneAt == nil || !got.LastDoneAt.Equal(start.Add(49*time.Hour)) {
		t.Errorf("last done = %v", got.LastDoneAt)
	}

	// An exercise a workout uses is archived on delete.
	mine, _ := s.CreateExercise(ctx, store.Exercise{OwnerID: a.ID, Name: "Mine", Kind: store.KindRepsOnly, Category: "strength",
		Equipment: "none", PrimaryMuscles: []string{"chest"}})
	w = workout(ids.New(), start.Add(-time.Hour), 1)
	w.Exercises = append(w.Exercises, store.WorkoutExercise{ExerciseID: mine.ID, Sets: []store.WorkoutSet{{Type: store.SetNormal, Reps: ptr(3)}}})
	if _, err := s.PutWorkout(ctx, w); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteExercise(ctx, mine.ID); err != nil {
		t.Fatal(err)
	}
	if e, err := s.GetExercise(ctx, mine.ID); err != nil || !e.Archived {
		t.Errorf("exercise used by a workout after delete = %+v, %v", e, err)
	}

	if err := s.DeleteWorkout(ctx, b.ID, first); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("b deletes a's workout: %v", err)
	}
	if err := s.DeleteWorkout(ctx, a.ID, first); err != nil {
		t.Fatal(err)
	}
	// Deleting the routine keeps its workouts.
	if err := s.DeleteRoutine(ctx, a.ID, r.ID); err != nil {
		t.Fatal(err)
	}
	if w, err := s.GetWorkout(ctx, a.ID, second); err != nil || w.RoutineID != "" {
		t.Errorf("workout after its routine is deleted = %+v, %v", w, err)
	}
}
