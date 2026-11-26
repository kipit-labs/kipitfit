package sqlite

import (
	"context"
	"errors"
	"testing"

	"github.com/kipit-labs/kipitfit/internal/store"
)

func ptr[T any](v T) *T { return &v }

func TestRoutines(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	a, _ := s.CreateUser(ctx, store.User{Email: "a@b.c", PasswordHash: "h"})
	b, _ := s.CreateUser(ctx, store.User{Email: "b@b.c", PasswordHash: "h"})
	if _, err := s.SyncLibrary(ctx, "v1", []store.Exercise{libraryRow("bench", "Bench Press"), libraryRow("row", "Row")}); err != nil {
		t.Fatal(err)
	}
	lib, _ := s.ListExercises(ctx, a.ID)
	bench, row := lib[0].ID, lib[1].ID

	push, err := s.CreateFolder(ctx, store.Folder{UserID: a.ID, Name: "Push"})
	if err != nil {
		t.Fatal(err)
	}
	pull, _ := s.CreateFolder(ctx, store.Folder{UserID: a.ID, Name: "Pull"})
	if push.Position != 0 || pull.Position != 1 {
		t.Errorf("folder positions = %d, %d", push.Position, pull.Position)
	}

	r, err := s.CreateRoutine(ctx, store.Routine{UserID: a.ID, FolderID: push.ID, Name: "Push A", Exercises: []store.RoutineExercise{
		{ExerciseID: bench, Superset: "A", RestS: ptr(90), Sets: []store.RoutineSet{
			{Type: store.SetWarmup, WeightG: ptr(40000), RepsMin: ptr(10), RepsMax: ptr(10)},
			{Type: store.SetNormal, WeightG: ptr(80000), RepsMin: ptr(6), RepsMax: ptr(8)},
		}},
		{ExerciseID: row, Superset: "A"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Exercises) != 2 || r.Exercises[0].Exercise.Name != "Bench Press" || len(r.Exercises[0].Sets) != 2 ||
		r.Exercises[0].Sets[0].Type != store.SetWarmup || *r.Exercises[0].Sets[1].WeightG != 80000 ||
		*r.Exercises[0].RestS != 90 || r.Exercises[1].RestS != nil || len(r.Exercises[1].Sets) != 0 {
		t.Fatalf("created = %+v", r)
	}
	if r.Exercises[0].Exercise.Instructions != nil {
		t.Error("a routine's exercises leave instructions out")
	}
	r2, _ := s.CreateRoutine(ctx, store.Routine{UserID: a.ID, Name: "Empty"})
	if r2.Position != 1 || len(r2.Exercises) != 0 {
		t.Errorf("second routine = %+v", r2)
	}

	// Replace: one exercise, the folder gone.
	r.FolderID, r.Name = "", "Push B"
	r.Exercises = []store.RoutineExercise{{ExerciseID: row, Notes: "slow", Sets: []store.RoutineSet{{Type: store.SetFailure}}}}
	if r, err = s.UpdateRoutine(ctx, r); err != nil {
		t.Fatal(err)
	}
	if r.Name != "Push B" || r.FolderID != "" || len(r.Exercises) != 1 || r.Exercises[0].Notes != "slow" ||
		r.Exercises[0].Superset != "" || r.Exercises[0].Sets[0].WeightG != nil {
		t.Fatalf("updated = %+v", r)
	}

	// Someone else's: not found, and untouched.
	if _, err := s.GetRoutine(ctx, b.ID, r.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("b gets a's routine: %v", err)
	}
	if _, err := s.UpdateRoutine(ctx, store.Routine{ID: r.ID, UserID: b.ID, Name: "x"}); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("b updates a's routine: %v", err)
	}
	if err := s.DeleteRoutine(ctx, b.ID, r.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("b deletes a's routine: %v", err)
	}
	if err := s.RenameFolder(ctx, store.Folder{ID: push.ID, UserID: b.ID, Name: "x"}); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("b renames a's folder: %v", err)
	}
	if err := s.Reorder(ctx, b.ID, nil, []string{r.ID}); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("b reorders a's routine: %v", err)
	}
	if got, _ := s.ListRoutines(ctx, b.ID); len(got) != 0 {
		t.Errorf("b lists %d routines", len(got))
	}

	// Reorder, then deleting a folder keeps its routines.
	if err := s.Reorder(ctx, a.ID, []string{pull.ID, push.ID}, []string{r2.ID, r.ID}); err != nil {
		t.Fatal(err)
	}
	r2.FolderID = pull.ID
	if _, err := s.UpdateRoutine(ctx, r2); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteFolder(ctx, a.ID, pull.ID); err != nil {
		t.Fatal(err)
	}
	fs, _ := s.ListFolders(ctx, a.ID)
	rs, _ := s.ListRoutines(ctx, a.ID)
	if len(fs) != 1 || fs[0].ID != push.ID || len(rs) != 2 || rs[0].ID != r2.ID || rs[0].FolderID != "" {
		t.Errorf("after reorder and folder delete: folders %+v, routines %+v", fs, rs)
	}

	if err := s.DeleteRoutine(ctx, a.ID, r.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetRoutine(ctx, a.ID, r.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("deleted routine: %v", err)
	}
}

func TestDeleteUsedExerciseArchives(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	a, _ := s.CreateUser(ctx, store.User{Email: "a@b.c", PasswordHash: "h"})
	mine := store.Exercise{OwnerID: a.ID, Name: "Landmine Press", Kind: store.KindWeightReps, Category: "strength",
		Equipment: "barbell", PrimaryMuscles: []string{"shoulders"}}
	used, _ := s.CreateExercise(ctx, mine)
	r, err := s.CreateRoutine(ctx, store.Routine{UserID: a.ID, Name: "R", Exercises: []store.RoutineExercise{{ExerciseID: used.ID}}})
	if err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteExercise(ctx, used.ID); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetExercise(ctx, used.ID)
	if err != nil || !got.Archived {
		t.Fatalf("used exercise after delete = %+v, %v", got, err)
	}
	if list, _ := s.ListExercises(ctx, a.ID); len(list) != 0 {
		t.Errorf("archived exercise listed: %+v", list)
	}
	if r, _ = s.GetRoutine(ctx, a.ID, r.ID); !r.Exercises[0].Exercise.Archived || r.Exercises[0].Exercise.Name != "Landmine Press" {
		t.Errorf("routine's archived exercise = %+v", r.Exercises[0].Exercise)
	}
	// The name is free again.
	if _, err := s.CreateExercise(ctx, mine); err != nil {
		t.Errorf("reuse an archived name: %v", err)
	}
	// Once no routine uses it, a delete really deletes.
	if err := s.DeleteRoutine(ctx, a.ID, r.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteExercise(ctx, used.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetExercise(ctx, used.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("unused archived exercise after delete: %v", err)
	}
}
