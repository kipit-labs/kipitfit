package workouts

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kipit-labs/kipitfit/internal/domain"
	"github.com/kipit-labs/kipitfit/internal/ids"
)

func TestStats(t *testing.T) {
	ctx := context.Background()
	c := newEnv()
	me, friend, lift := c.me, c.friend, c.byKind[domain.KindWeightReps]
	at := time.Now().UTC().Truncate(time.Second).Add(-48 * time.Hour)
	put := func(start time.Time, sets ...domain.WorkoutSet) {
		t.Helper()
		if _, err := c.PutWorkout(ctx, me, ids.New(), domain.Workout{Title: "Push", StartedAt: start, EndedAt: start.Add(time.Hour),
			Exercises: []domain.WorkoutExercise{{ExerciseID: lift.ID, Sets: sets}}}); err != nil {
			t.Fatal(err)
		}
	}
	put(at.AddDate(0, 0, -30), domain.WorkoutSet{Type: domain.SetNormal, WeightG: ptr(50000), Reps: ptr(5)})
	put(at,
		domain.WorkoutSet{Type: domain.SetWarmup, WeightG: ptr(40000), Reps: ptr(10)},
		domain.WorkoutSet{Type: domain.SetNormal, WeightG: ptr(100000), Reps: ptr(5)},
		domain.WorkoutSet{Type: domain.SetNormal, WeightG: ptr(100000), Reps: ptr(5)})

	st, err := c.Stats(ctx, me, at.AddDate(0, 0, -7))
	if err != nil || len(st) != 1 {
		t.Fatalf("stats = %+v, %v", st, err)
	}
	s := st[0]
	if s.Sets != 3 || s.VolumeG != 1000000 || !s.EndedAt.Equal(at.Add(time.Hour)) {
		t.Errorf("workout = %+v", s)
	}
	// Warm-ups don't count for muscles; each primary muscle gets the set.
	for _, m := range lift.PrimaryMuscles {
		if s.Muscles[m] != 2 {
			t.Errorf("%s: %d sets, want 2 (%v)", m, s.Muscles[m], s.Muscles)
		}
	}
	if st, _ := c.Stats(ctx, me, at.AddDate(0, 0, -60)); len(st) != 2 || !st[0].StartedAt.Before(st[1].StartedAt) {
		t.Errorf("60 days = %+v, want both, oldest first", st)
	}
	if st, _ := c.Stats(ctx, friend, at.AddDate(0, 0, -60)); len(st) != 0 {
		t.Errorf("friend sees %+v", st)
	}
	for _, from := range []time.Time{{}, time.Now().AddDate(-2, 0, 0)} {
		if _, err := c.Stats(ctx, me, from); !errors.Is(err, ErrStatsFrom) {
			t.Errorf("from %v: %v", from, err)
		}
	}
}
