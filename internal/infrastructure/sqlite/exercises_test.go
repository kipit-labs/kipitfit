package sqlite

import (
	"context"
	"errors"
	"testing"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

func libraryRow(slug, name string) domain.Exercise {
	return domain.Exercise{Slug: slug, Name: name, Kind: domain.KindWeightReps, Category: "strength", Equipment: "barbell",
		PrimaryMuscles: []string{"chest"}, Instructions: []string{"Lie down.", "Press."}}
}

func TestSyncLibrary(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	v1 := []domain.Exercise{libraryRow("bench", "Bench Press"), libraryRow("squat", "Squat")}
	if wrote, err := s.SyncLibrary(ctx, "v1", v1); err != nil || !wrote {
		t.Fatalf("first sync: wrote=%v err=%v", wrote, err)
	}
	if wrote, err := s.SyncLibrary(ctx, "v1", v1); err != nil || wrote {
		t.Fatalf("same version: wrote=%v err=%v", wrote, err)
	}
	before, err := s.ListExercises(ctx, "")
	if err != nil || len(before) != 2 {
		t.Fatalf("list: %v %v", before, err)
	}

	// v2 renames bench and drops squat: same id, squat kept.
	if _, err := s.SyncLibrary(ctx, "v2", []domain.Exercise{libraryRow("bench", "Bench Press (Barbell)")}); err != nil {
		t.Fatal(err)
	}
	after, err := s.ListExercises(ctx, "")
	if err != nil || len(after) != 2 {
		t.Fatalf("after v2: %v %v", after, err)
	}
	if after[0].Name != "Bench Press (Barbell)" || after[0].ID != before[0].ID || after[0].Custom {
		t.Errorf("bench after v2 = %+v, before %+v", after[0], before[0])
	}
	if after[0].Instructions != nil {
		t.Error("list should leave instructions out")
	}
	full, err := s.GetExercise(ctx, after[0].ID)
	if err != nil || len(full.Instructions) != 2 || full.PrimaryMuscles[0] != "chest" {
		t.Errorf("get = %+v, %v", full, err)
	}
}

func TestCustomExercises(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	a, _ := s.CreateUser(ctx, domain.User{Email: "a@b.c", PasswordHash: "h"})
	b, _ := s.CreateUser(ctx, domain.User{Email: "b@b.c", PasswordHash: "h"})
	if _, err := s.SyncLibrary(ctx, "v1", []domain.Exercise{libraryRow("bench", "Bench Press")}); err != nil {
		t.Fatal(err)
	}
	mine := domain.Exercise{OwnerID: a.ID, Name: "Landmine Press", Kind: domain.KindWeightReps, Category: "strength",
		Equipment: "barbell", PrimaryMuscles: []string{"shoulders"}}
	e, err := s.CreateExercise(ctx, mine)
	if err != nil || !e.Custom || e.SecondaryMuscles == nil {
		t.Fatalf("create = %+v, %v", e, err)
	}
	mine.Name = "landmine PRESS"
	if _, err := s.CreateExercise(ctx, mine); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("same name, other case: err = %v", err)
	}
	mine.OwnerID = b.ID
	if _, err := s.CreateExercise(ctx, mine); err != nil {
		t.Errorf("same name, other owner: %v", err)
	}
	// Library names don't block custom ones.
	if _, err := s.CreateExercise(ctx, domain.Exercise{OwnerID: a.ID, Name: "Bench Press", Kind: domain.KindRepsOnly,
		Category: "strength", Equipment: "none", PrimaryMuscles: []string{"chest"}}); err != nil {
		t.Errorf("custom named like a library one: %v", err)
	}

	list, _ := s.ListExercises(ctx, a.ID)
	if len(list) != 3 {
		t.Errorf("a sees %d exercises, want library + 2 own", len(list))
	}
	list, _ = s.ListExercises(ctx, b.ID)
	if len(list) != 2 {
		t.Errorf("b sees %d exercises, want library + 1 own", len(list))
	}

	e.Name, e.PrimaryMuscles = "Landmine Press, one arm", []string{"shoulders", "triceps"}
	if _, err := s.UpdateExercise(ctx, e); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetExercise(ctx, e.ID)
	if got.Name != e.Name || len(got.PrimaryMuscles) != 2 {
		t.Errorf("after update: %+v", got)
	}
	if err := s.DeleteExercise(ctx, e.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteExercise(ctx, e.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
	if _, err := s.UpdateExercise(ctx, e); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("update deleted: %v", err)
	}
}
