package routines

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kipit-labs/kipitfit/internal/domain"
	"github.com/kipit-labs/kipitfit/internal/usecase/exercises"
)

func TestRoutines(t *testing.T) {
	ctx := context.Background()
	c, ex, store, lift, plank := newService()
	me, friend := domain.User{ID: "me"}, domain.User{ID: "friend"}

	folder, err := c.CreateFolder(ctx, me, FolderInput{Name: "  Push  day "})
	if err != nil || folder.Name != "Push day" {
		t.Fatalf("folder = %+v, %v", folder, err)
	}
	in := domain.Routine{Name: " Push  A ", FolderID: folder.ID, Exercises: []domain.RoutineExercise{
		{ExerciseID: lift.ID, Superset: "A", RestS: ptr(90), Sets: []domain.RoutineSet{
			{Type: domain.SetWarmup, WeightG: ptr(20000), RepsMin: ptr(10), RepsMax: ptr(10)},
			{Type: domain.SetNormal, WeightG: ptr(60000), RepsMin: ptr(8), RepsMax: ptr(12)},
		}},
		{ExerciseID: plank.ID, Superset: "A", Sets: []domain.RoutineSet{{Type: domain.SetNormal, DurationS: ptr(60)}}},
	}}
	r, err := c.CreateRoutine(ctx, me, in)
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "Push A" || r.UserID != me.ID || len(r.Exercises) != 2 || r.Exercises[1].Exercise.Kind != domain.KindDuration {
		t.Fatalf("created %+v", r)
	}

	list, err := c.ListRoutines(ctx, me)
	if err != nil || len(list.Folders) != 1 || len(list.Routines) != 1 {
		t.Fatalf("list = %+v, %v", list, err)
	}
	if list, _ := c.ListRoutines(ctx, friend); len(list.Folders) != 0 || len(list.Routines) != 0 {
		t.Errorf("friend sees %+v", list)
	}

	long := strings.Repeat("x", 101)
	sets := make([]domain.RoutineSet, 51)
	for i := range sets {
		sets[i].Type = domain.SetNormal
	}
	exs := make([]domain.RoutineExercise, 51)
	for i := range exs {
		exs[i].ExerciseID = lift.ID
	}
	one := func(s domain.RoutineSet) []domain.RoutineExercise {
		return []domain.RoutineExercise{{ExerciseID: lift.ID, Sets: []domain.RoutineSet{s}}}
	}
	for name, bad := range map[string]struct {
		in   domain.Routine
		want error
	}{
		"blank name":       {domain.Routine{Name: "  "}, domain.ErrTitle},
		"long name":        {domain.Routine{Name: long}, domain.ErrTitle},
		"long notes":       {domain.Routine{Name: "R", Notes: strings.Repeat("x", 1001)}, domain.ErrNotesTooLong},
		"51 exercises":     {domain.Routine{Name: "R", Exercises: exs}, domain.ErrTooManyExercises},
		"51 sets":          {domain.Routine{Name: "R", Exercises: []domain.RoutineExercise{{ExerciseID: lift.ID, Sets: sets}}}, domain.ErrTooManySets},
		"bad set type":     {domain.Routine{Name: "R", Exercises: one(domain.RoutineSet{Type: "giant"})}, domain.ErrSetType},
		"negative weight":  {domain.Routine{Name: "R", Exercises: one(domain.RoutineSet{Type: domain.SetNormal, WeightG: ptr(-1)})}, domain.ErrSetValue},
		"reps min > max":   {domain.Routine{Name: "R", Exercises: one(domain.RoutineSet{Type: domain.SetNormal, RepsMin: ptr(9), RepsMax: ptr(8)})}, domain.ErrSetValue},
		"duration on lift": {domain.Routine{Name: "R", Exercises: one(domain.RoutineSet{Type: domain.SetNormal, DurationS: ptr(30)})}, domain.ErrSetValue},
		"weight on plank": {domain.Routine{Name: "R", Exercises: []domain.RoutineExercise{{ExerciseID: plank.ID,
			Sets: []domain.RoutineSet{{Type: domain.SetNormal, WeightG: ptr(1000)}}}}}, domain.ErrSetValue},
		"bad superset":       {domain.Routine{Name: "R", Exercises: []domain.RoutineExercise{{ExerciseID: lift.ID, Superset: "a"}}}, domain.ErrSuperset},
		"long rest":          {domain.Routine{Name: "R", Exercises: []domain.RoutineExercise{{ExerciseID: lift.ID, RestS: ptr(3601)}}}, domain.ErrSetValue},
		"unknown exercise":   {domain.Routine{Name: "R", Exercises: []domain.RoutineExercise{{ExerciseID: "nope"}}}, domain.ErrNotFound},
		"someone's folder":   {domain.Routine{Name: "R", FolderID: "nope"}, domain.ErrNotFound},
		"long exercise note": {domain.Routine{Name: "R", Exercises: []domain.RoutineExercise{{ExerciseID: lift.ID, Notes: strings.Repeat("x", 1001)}}}, domain.ErrNotesTooLong},
	} {
		if _, err := c.CreateRoutine(ctx, me, bad.in); !errors.Is(err, bad.want) {
			t.Errorf("%s: err = %v, want %v", name, err, bad.want)
		}
	}

	// The friend can't use my folder, my custom exercise, or touch my routine.
	mine, err := ex.CreateExercise(ctx, me, exercises.ExerciseInput{Name: ptr("Landmine Press"), Kind: ptr(domain.KindWeightReps), PrimaryMuscles: &[]string{"shoulders"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.CreateRoutine(ctx, friend, domain.Routine{Name: "R", FolderID: folder.ID}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("friend uses my folder: %v", err)
	}
	if _, err := c.CreateRoutine(ctx, friend, domain.Routine{Name: "R", Exercises: []domain.RoutineExercise{{ExerciseID: mine.ID}}}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("friend uses my exercise: %v", err)
	}
	if _, err := c.UpdateRoutine(ctx, friend, r.ID, domain.Routine{Name: "Mine now"}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("friend updates my routine: %v", err)
	}
	if err := c.DeleteRoutine(ctx, friend, r.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("friend deletes my routine: %v", err)
	}
	if _, err := c.RenameFolder(ctx, friend, folder.ID, FolderInput{Name: "X"}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("friend renames my folder: %v", err)
	}
	if err := c.DeleteFolder(ctx, friend, folder.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("friend deletes my folder: %v", err)
	}

	// Deleting my custom exercise while a routine uses it archives it (the store's job, see
	// sqlite.TestDeleteUsedExerciseArchives): the routine still saves with it.
	r.Exercises = append(r.Exercises, domain.RoutineExercise{ExerciseID: mine.ID})
	if r, err = c.UpdateRoutine(ctx, me, r.ID, r); err != nil {
		t.Fatal(err)
	}
	mine.Archived = true
	store.byID[mine.ID] = mine
	if r, err = c.UpdateRoutine(ctx, me, r.ID, r); err != nil || !r.Exercises[2].Exercise.Archived {
		t.Errorf("save with an archived exercise = %+v, %v", r.Exercises, err)
	}

	if err := c.Reorder(ctx, me, Order{Routines: make([]string, 1001)}); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("huge reorder: %v", err)
	}
	if err := c.DeleteFolder(ctx, me, folder.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.ListRoutines(ctx, me); len(got.Routines) != 1 || got.Routines[0].FolderID != "" {
		t.Errorf("after folder delete: %+v", got)
	}
}
