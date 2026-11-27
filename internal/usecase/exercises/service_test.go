package exercises

import (
	"context"
	"errors"
	"testing"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

func TestExercises(t *testing.T) {
	ctx := context.Background()
	c := NewService(newFakeExercises("Bench Press", "Deadlift", "Squat"))
	admin, friend := domain.User{ID: "admin"}, domain.User{ID: "friend"}

	library, err := c.ListExercises(ctx, admin)
	if err != nil || len(library) != 3 {
		t.Fatalf("library: %d, %v", len(library), err)
	}

	in := ExerciseInput{Name: ptr("  Landmine   Press "), Kind: ptr(domain.KindWeightReps), Equipment: ptr("barbell"),
		PrimaryMuscles: &[]string{"shoulders", "shoulders"}}
	e, err := c.CreateExercise(ctx, admin, in)
	if err != nil {
		t.Fatal(err)
	}
	if e.Name != "Landmine Press" || e.Category != "strength" || len(e.PrimaryMuscles) != 1 || !e.Custom {
		t.Errorf("created %+v", e)
	}
	if _, err := c.CreateExercise(ctx, admin, in); !errors.Is(err, ErrExerciseNameTaken) {
		t.Errorf("duplicate: %v", err)
	}

	for name, bad := range map[string]struct {
		in   ExerciseInput
		want error
	}{
		"no name":      {ExerciseInput{Kind: in.Kind, PrimaryMuscles: in.PrimaryMuscles}, ErrExerciseName},
		"blank name":   {ExerciseInput{Name: ptr("   "), Kind: in.Kind, PrimaryMuscles: in.PrimaryMuscles}, ErrExerciseName},
		"long name":    {ExerciseInput{Name: ptr(string(make([]rune, 101))), Kind: in.Kind}, ErrExerciseName},
		"no kind":      {ExerciseInput{Name: ptr("X"), PrimaryMuscles: in.PrimaryMuscles}, ErrExerciseField},
		"bad kind":     {ExerciseInput{Name: ptr("X"), Kind: ptr(domain.Kind("lift")), PrimaryMuscles: in.PrimaryMuscles}, ErrExerciseField},
		"bad category": {ExerciseInput{Name: ptr("X"), Kind: in.Kind, Category: ptr("yoga"), PrimaryMuscles: in.PrimaryMuscles}, ErrExerciseField},
		"bad muscle":   {ExerciseInput{Name: ptr("X"), Kind: in.Kind, PrimaryMuscles: &[]string{"heart"}}, ErrExerciseField},
		"no muscle":    {ExerciseInput{Name: ptr("X"), Kind: in.Kind, PrimaryMuscles: &[]string{}}, ErrPrimaryMuscleEmpty},
	} {
		if _, err := c.CreateExercise(ctx, admin, bad.in); !errors.Is(err, bad.want) {
			t.Errorf("%s: err = %v, want %v", name, err, bad.want)
		}
	}

	// The friend can't see, change or delete the admin's exercise, and can't change the library.
	if _, err := c.GetExercise(ctx, friend, e.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("friend get: %v", err)
	}
	if _, err := c.UpdateExercise(ctx, friend, e.ID, ExerciseInput{Name: ptr("Mine")}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("friend update: %v", err)
	}
	if err := c.DeleteExercise(ctx, friend, e.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("friend delete: %v", err)
	}
	if _, err := c.UpdateExercise(ctx, friend, library[0].ID, ExerciseInput{Name: ptr("Mine")}); !errors.Is(err, ErrLibraryReadonly) {
		t.Errorf("library update: %v", err)
	}
	if err := c.DeleteExercise(ctx, admin, library[0].ID); !errors.Is(err, ErrLibraryReadonly) {
		t.Errorf("library delete: %v", err)
	}
	if list, _ := c.ListExercises(ctx, friend); len(list) != 3 {
		t.Errorf("friend sees %d, want the library only", len(list))
	}

	e, err = c.UpdateExercise(ctx, admin, e.ID, ExerciseInput{SecondaryMuscles: &[]string{"triceps"}, Kind: ptr(domain.KindWeightedBodyweight)})
	if err != nil || e.SecondaryMuscles[0] != "triceps" || e.Kind != domain.KindWeightedBodyweight || e.Name != "Landmine Press" {
		t.Errorf("update = %+v, %v", e, err)
	}
	if _, err := c.UpdateExercise(ctx, admin, e.ID, ExerciseInput{PrimaryMuscles: &[]string{}}); !errors.Is(err, ErrPrimaryMuscleEmpty) {
		t.Errorf("clear primary: %v", err)
	}
	if err := c.DeleteExercise(ctx, admin, e.ID); err != nil {
		t.Fatal(err)
	}
}
