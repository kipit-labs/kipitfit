package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kipit-labs/kipitfit/internal/ids"
	"github.com/kipit-labs/kipitfit/internal/store"
)

func TestPutWorkout(t *testing.T) {
	ctx := context.Background()
	c, me, friend := twoUsers(t)
	lift := byKind(t, c, me, store.KindWeightReps)
	dip := byKind(t, c, me, store.KindWeightedBodyweight)
	pullup := byKind(t, c, me, store.KindAssistedBodyweight)
	plank := byKind(t, c, me, store.KindDuration)
	r, _ := c.CreateRoutine(ctx, me, store.Routine{Name: "Push"})

	start := time.Date(2026, 11, 20, 18, 0, 0, 0, time.UTC)
	rpe := store.Tenths(85)
	in := store.Workout{Title: " Push ", RoutineID: r.ID, StartedAt: start, EndedAt: start.Add(time.Hour), Exercises: []store.WorkoutExercise{
		{ExerciseID: lift.ID, Superset: "A", Sets: []store.WorkoutSet{
			{Type: store.SetWarmup, WeightG: ptr(20000), Reps: ptr(10)}, // left out of volume
			{Type: store.SetNormal, WeightG: ptr(102058), Reps: ptr(5), RPE: &rpe},
			{Type: store.SetDrop, WeightG: ptr(60000), Reps: ptr(8)},
		}},
		{ExerciseID: dip.ID, Superset: "A", Sets: []store.WorkoutSet{{Type: store.SetFailure, WeightG: ptr(10000), Reps: ptr(6)}, {Type: store.SetNormal, Reps: ptr(10)}}},
		{ExerciseID: pullup.ID, Sets: []store.WorkoutSet{{Type: store.SetNormal, WeightG: ptr(20000), Reps: ptr(8)}}},
		{ExerciseID: plank.ID, Sets: []store.WorkoutSet{{Type: store.SetNormal, DurationS: ptr(60)}}},
	}}
	id := ids.New()
	w, err := c.PutWorkout(ctx, me, id, in)
	if err != nil {
		t.Fatal(err)
	}
	want := 102058*5 + 60000*8 + 10000*6
	if w.VolumeG != want || w.Title != "Push" || w.RoutineID != r.ID {
		t.Fatalf("volume %d want %d; %+v", w.VolumeG, want, w)
	}
	// The retry of a finish that did land: same id, one workout, same volume.
	in.VolumeG = 1
	if w, err = c.PutWorkout(ctx, me, id, in); err != nil || w.VolumeG != want {
		t.Fatalf("retry = %d, %v", w.VolumeG, err)
	}
	if page, _ := c.ListWorkouts(ctx, me, ""); len(page.Workouts) != 1 || page.Next != "" {
		t.Fatalf("history after a retry = %+v", page)
	}
	if _, err := c.PutWorkout(ctx, friend, id, in); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("friend puts my workout id: %v", err)
	}

	one := func(e store.Exercise, s store.WorkoutSet) store.Workout {
		return store.Workout{Title: "W", StartedAt: start, EndedAt: start.Add(time.Hour),
			Exercises: []store.WorkoutExercise{{ExerciseID: e.ID, Sets: []store.WorkoutSet{s}}}}
	}
	n := store.SetNormal
	for name, bad := range map[string]struct {
		id   string
		in   store.Workout
		want error
	}{
		"bad id":          {"nope", one(lift, store.WorkoutSet{Type: n, WeightG: ptr(1), Reps: ptr(1)}), ErrWorkoutID},
		"ends first":      {ids.New(), store.Workout{Title: "W", StartedAt: start, EndedAt: start.Add(-time.Second)}, ErrWorkoutTimes},
		"two days":        {ids.New(), store.Workout{Title: "W", StartedAt: start, EndedAt: start.Add(48 * time.Hour)}, ErrWorkoutTimes},
		"no title":        {ids.New(), store.Workout{StartedAt: start, EndedAt: start}, ErrTitle},
		"no sets":         {ids.New(), store.Workout{Title: "W", StartedAt: start, EndedAt: start, Exercises: []store.WorkoutExercise{{ExerciseID: lift.ID}}}, ErrWorkoutEmpty},
		"lift no weight":  {ids.New(), one(lift, store.WorkoutSet{Type: n, Reps: ptr(5)}), ErrSetMissing},
		"lift no reps":    {ids.New(), one(lift, store.WorkoutSet{Type: n, WeightG: ptr(5)}), ErrSetMissing},
		"plank with reps": {ids.New(), one(plank, store.WorkoutSet{Type: n, DurationS: ptr(30), Reps: ptr(3)}), ErrSetValue},
		"plank no time":   {ids.New(), one(plank, store.WorkoutSet{Type: n}), ErrSetMissing},
		"rpe 5.5":         {ids.New(), one(lift, store.WorkoutSet{Type: n, WeightG: ptr(1), Reps: ptr(1), RPE: ptr(store.Tenths(55))}), ErrRPE},
		"rpe 8.3":         {ids.New(), one(lift, store.WorkoutSet{Type: n, WeightG: ptr(1), Reps: ptr(1), RPE: ptr(store.Tenths(83))}), ErrRPE},
		"bad type":        {ids.New(), one(lift, store.WorkoutSet{Type: "x", WeightG: ptr(1), Reps: ptr(1)}), ErrSetType},
		"unknown ex":      {ids.New(), one(store.Exercise{ID: "nope"}, store.WorkoutSet{Type: n}), store.ErrNotFound},
	} {
		if _, err := c.PutWorkout(ctx, me, bad.id, bad.in); !errors.Is(err, bad.want) {
			t.Errorf("%s: err = %v, want %v", name, err, bad.want)
		}
	}

	// A finish queued while its routine got deleted still saves, without the routine.
	if err := c.DeleteRoutine(ctx, me, r.ID); err != nil {
		t.Fatal(err)
	}
	if w, err := c.PutWorkout(ctx, me, ids.New(), in); err != nil || w.RoutineID != "" {
		t.Errorf("finish of a deleted routine = %+v, %v", w.RoutineID, err)
	}

	prev, err := c.PreviousSets(ctx, me, []string{lift.ID, plank.ID})
	if err != nil || len(prev[lift.ID]) != 3 || *prev[plank.ID][0].DurationS != 60 {
		t.Errorf("previous = %+v, %v", prev, err)
	}
}

func TestListWorkoutsPages(t *testing.T) {
	ctx := context.Background()
	c, me, _ := twoUsers(t)
	plank := byKind(t, c, me, store.KindDuration)
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	for i := range 21 {
		at := start.Add(time.Duration(i) * 24 * time.Hour)
		_, err := c.PutWorkout(ctx, me, ids.New(), store.Workout{Title: "W", StartedAt: at, EndedAt: at.Add(time.Hour),
			Exercises: []store.WorkoutExercise{{ExerciseID: plank.ID, Sets: []store.WorkoutSet{{Type: store.SetNormal, DurationS: ptr(30)}}}}})
		if err != nil {
			t.Fatal(err)
		}
	}
	page, err := c.ListWorkouts(ctx, me, "")
	if err != nil || len(page.Workouts) != 20 || page.Next == "" || !page.Workouts[0].StartedAt.Equal(start.Add(20*24*time.Hour)) {
		t.Fatalf("page 1: %d, next %q, %v", len(page.Workouts), page.Next, err)
	}
	page, _ = c.ListWorkouts(ctx, me, page.Next)
	if len(page.Workouts) != 1 || page.Next != "" || !page.Workouts[0].StartedAt.Equal(start) {
		t.Fatalf("page 2: %+v", page)
	}
}
