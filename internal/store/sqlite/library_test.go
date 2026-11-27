package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kipit-labs/kipitfit/internal/domain"
	"github.com/kipit-labs/kipitfit/internal/ids"
)

// A pack import re-points routines and workouts, archives the rest of the old
// library without breaking what uses it, and keeps ids across re-runs.
func TestReplaceLibrary(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	me, _ := s.CreateUser(ctx, domain.User{Email: "a@b.c", PasswordHash: "h"})
	if _, err := s.LibraryPack(ctx); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("pack of the built-in library: %v", err)
	}
	if _, err := s.SyncLibrary(ctx, "v1", []domain.Exercise{libraryRow("dead", "Deadlift"), libraryRow("situp", "Sit-Up"),
		libraryRow("squat", "Squat")}); err != nil {
		t.Fatal(err)
	}
	lib, _ := s.ListExercises(ctx, me.ID)
	dead, situp := lib[0], lib[1]
	custom, err := s.CreateExercise(ctx, domain.Exercise{OwnerID: me.ID, Name: "Landmine Press", Kind: domain.KindWeightReps})
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.CreateRoutine(ctx, domain.Routine{UserID: me.ID, Name: "Pull", Exercises: []domain.RoutineExercise{
		{ExerciseID: dead.ID}, {ExerciseID: situp.ID}}})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 11, 20, 18, 0, 0, 0, time.UTC)
	w, err := s.PutWorkout(ctx, domain.Workout{ID: ids.New(), UserID: me.ID, Title: "Pull", StartedAt: start, EndedAt: start.Add(time.Hour),
		Exercises: []domain.WorkoutExercise{{ExerciseID: dead.ID}, {ExerciseID: custom.ID}}})
	if err != nil {
		t.Fatal(err)
	}

	pack := domain.Pack{Name: "test", Version: "1", Credit: "Exercise data by Test"}
	packDead := libraryRow("test:dead", "deadlift")
	packDead.Media = []string{"packs/test/1/dead.webp"}
	exs := []domain.Exercise{packDead, libraryRow("test:plank", "Plank")}
	res, err := s.ReplaceLibrary(ctx, pack, exs, map[string]string{"dead": "test:dead"})
	// The routine and the workout deadlift move; the three built-in ones are archived.
	if err != nil || res.Exercises != 2 || res.Repointed != 2 || res.Archived != 3 {
		t.Fatalf("replace = %+v, %v", res, err)
	}
	list, _ := s.ListExercises(ctx, me.ID)
	if len(list) != 3 || list[0].Name != "deadlift" || len(list[0].Media) != 1 || list[1].ID != custom.ID {
		t.Fatalf("library after import = %+v", list)
	}
	newDead := list[0]
	if r, _ = s.GetRoutine(ctx, me.ID, r.ID); r.Exercises[0].ExerciseID != newDead.ID || r.Exercises[1].ExerciseID != situp.ID ||
		!r.Exercises[1].Exercise.Archived {
		t.Fatalf("routine after import = %+v", r.Exercises)
	}
	if w, _ = s.GetWorkout(ctx, me.ID, w.ID); w.Exercises[0].ExerciseID != newDead.ID || w.Exercises[1].ExerciseID != custom.ID {
		t.Fatalf("workout after import = %+v", w.Exercises)
	}
	if old, err := s.GetExercise(ctx, situp.ID); err != nil || !old.Archived {
		t.Fatalf("old built-in = %+v, %v", old, err)
	}
	if p, err := s.LibraryPack(ctx); err != nil || p != pack {
		t.Fatalf("pack = %+v, %v", p, err)
	}

	// A re-run changes nothing and keeps the ids; an exercise archived by an earlier import comes back.
	res, err = s.ReplaceLibrary(ctx, pack, append(exs, libraryRow("situp", "Sit-Up")), nil)
	if err != nil || res.Repointed != 0 || res.Archived != 0 {
		t.Fatalf("re-run = %+v, %v", res, err)
	}
	if again, _ := s.GetExercise(ctx, newDead.ID); again.Name != "deadlift" {
		t.Fatal("re-run changed the deadlift's id")
	}
	if back, _ := s.GetExercise(ctx, situp.ID); back.Archived {
		t.Error("a re-imported exercise stays archived")
	}
}
