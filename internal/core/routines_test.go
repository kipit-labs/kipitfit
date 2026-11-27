package core

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

// twoUsers is an admin and an invited friend on a core with the library synced.
func twoUsers(t *testing.T) (*Core, domain.User, domain.User) {
	t.Helper()
	ctx := context.Background()
	c, _ := newCore(t)
	if err := c.SyncExercises(ctx); err != nil {
		t.Fatal(err)
	}
	admin := newAdmin(t, c)
	link, err := c.CreateInvite(ctx, admin, "friend@example.com")
	if err != nil {
		t.Fatal(err)
	}
	friend, _, err := c.AcceptInvite(ctx, inviteToken(t, link.Link), "Friend", pw)
	if err != nil {
		t.Fatal(err)
	}
	return c, admin, friend
}

// byKind is a library exercise of kind k.
func byKind(t *testing.T, c *Core, me domain.User, k domain.Kind) domain.Exercise {
	t.Helper()
	exs, _ := c.ListExercises(context.Background(), me)
	for _, e := range exs {
		if e.Kind == k {
			return e
		}
	}
	t.Fatalf("no %s exercise", k)
	return domain.Exercise{}
}

func TestRoutines(t *testing.T) {
	ctx := context.Background()
	c, me, friend := twoUsers(t)
	lift := byKind(t, c, me, domain.KindWeightReps)
	plank := byKind(t, c, me, domain.KindDuration)

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
		"blank name":       {domain.Routine{Name: "  "}, ErrTitle},
		"long name":        {domain.Routine{Name: long}, ErrTitle},
		"long notes":       {domain.Routine{Name: "R", Notes: strings.Repeat("x", 1001)}, ErrNotesTooLong},
		"51 exercises":     {domain.Routine{Name: "R", Exercises: exs}, ErrTooManyExercises},
		"51 sets":          {domain.Routine{Name: "R", Exercises: []domain.RoutineExercise{{ExerciseID: lift.ID, Sets: sets}}}, ErrTooManySets},
		"bad set type":     {domain.Routine{Name: "R", Exercises: one(domain.RoutineSet{Type: "giant"})}, ErrSetType},
		"negative weight":  {domain.Routine{Name: "R", Exercises: one(domain.RoutineSet{Type: domain.SetNormal, WeightG: ptr(-1)})}, ErrSetValue},
		"reps min > max":   {domain.Routine{Name: "R", Exercises: one(domain.RoutineSet{Type: domain.SetNormal, RepsMin: ptr(9), RepsMax: ptr(8)})}, ErrSetValue},
		"duration on lift": {domain.Routine{Name: "R", Exercises: one(domain.RoutineSet{Type: domain.SetNormal, DurationS: ptr(30)})}, ErrSetValue},
		"weight on plank": {domain.Routine{Name: "R", Exercises: []domain.RoutineExercise{{ExerciseID: plank.ID,
			Sets: []domain.RoutineSet{{Type: domain.SetNormal, WeightG: ptr(1000)}}}}}, ErrSetValue},
		"bad superset":       {domain.Routine{Name: "R", Exercises: []domain.RoutineExercise{{ExerciseID: lift.ID, Superset: "a"}}}, ErrSuperset},
		"long rest":          {domain.Routine{Name: "R", Exercises: []domain.RoutineExercise{{ExerciseID: lift.ID, RestS: ptr(3601)}}}, ErrSetValue},
		"unknown exercise":   {domain.Routine{Name: "R", Exercises: []domain.RoutineExercise{{ExerciseID: "nope"}}}, domain.ErrNotFound},
		"someone's folder":   {domain.Routine{Name: "R", FolderID: "nope"}, domain.ErrNotFound},
		"long exercise note": {domain.Routine{Name: "R", Exercises: []domain.RoutineExercise{{ExerciseID: lift.ID, Notes: strings.Repeat("x", 1001)}}}, ErrNotesTooLong},
	} {
		if _, err := c.CreateRoutine(ctx, me, bad.in); !errors.Is(err, bad.want) {
			t.Errorf("%s: err = %v, want %v", name, err, bad.want)
		}
	}

	// The friend can't use my folder, my custom exercise, or touch my routine.
	mine, err := c.CreateExercise(ctx, me, ExerciseInput{Name: ptr("Landmine Press"), Kind: ptr(domain.KindWeightReps), PrimaryMuscles: &[]string{"shoulders"}})
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

	// Deleting my custom exercise while a routine uses it archives it.
	r.Exercises = append(r.Exercises, domain.RoutineExercise{ExerciseID: mine.ID})
	if r, err = c.UpdateRoutine(ctx, me, r.ID, r); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteExercise(ctx, me, mine.ID); err != nil {
		t.Fatal(err)
	}
	if got, err := c.GetExercise(ctx, me, mine.ID); err != nil || !got.Archived {
		t.Errorf("used exercise after delete = %+v, %v", got, err)
	}
	// The routine still saves with its archived exercise.
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
