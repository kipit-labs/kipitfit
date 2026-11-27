package workouts

import (
	"context"
	"testing"
	"time"

	"github.com/kipit-labs/kipitfit/internal/domain"
	"github.com/kipit-labs/kipitfit/internal/ids"
	"github.com/kipit-labs/kipitfit/internal/usecase/records"
)

func TestRecordsInWorkouts(t *testing.T) {
	ctx := context.Background()
	c := newEnv()
	me, friend, lift := c.me, c.friend, c.byKind[domain.KindWeightReps]
	start := time.Date(2026, 3, 2, 18, 0, 0, 0, time.UTC)
	put := func(id string, day int, sets ...[2]int) records.WorkoutView {
		t.Helper()
		ss := []domain.WorkoutSet{}
		for _, s := range sets {
			ss = append(ss, domain.WorkoutSet{Type: domain.SetNormal, WeightG: ptr(s[0]), Reps: ptr(s[1])})
		}
		at := start.AddDate(0, 0, day)
		w, err := c.PutWorkout(ctx, me, id, domain.Workout{Title: "Push", StartedAt: at, EndedAt: at.Add(time.Hour),
			Exercises: []domain.WorkoutExercise{{ExerciseID: lift.ID, Sets: ss}}})
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	first, second, third := ids.New(), ids.New(), ids.New()
	if w := put(first, 0, [2]int{100000, 5}); len(w.Records) != 0 {
		t.Fatalf("first workout beat %+v", w.Records)
	}
	w := put(second, 7, [2]int{90000, 5}, [2]int{105000, 5})
	kinds := map[records.RecordKind]records.Record{}
	for _, r := range w.Records {
		kinds[r.Kind] = r
	}
	if r := kinds[records.RecordWeight]; r.Value != 105000 || r.Previous != 100000 || r.Set != 1 || r.Exercise != 0 || r.ExerciseID != lift.ID {
		t.Errorf("weight record = %+v", r)
	}
	if r := kinds[records.RecordSessionVolume]; r.Value != 90000*5+105000*5 || r.Set != -1 {
		t.Errorf("session volume record = %+v", r)
	}
	if _, ok := kinds[records.RecordReps]; ok {
		t.Error("5 reps again is no reps record")
	}

	// A later workout's records are worked out against what came before it.
	put(third, 14, [2]int{110000, 5})
	if got, _ := c.GetWorkout(ctx, me, third); len(got.Records) == 0 {
		t.Error("third workout beat nothing")
	}
	// Editing the second down takes its records away and gives them to the third.
	put(second, 7, [2]int{90000, 5})
	if got, _ := c.GetWorkout(ctx, me, second); len(got.Records) != 0 {
		t.Errorf("edited workout still beats %+v", got.Records)
	}
	// Deleting the first leaves the third as the second workout ever: it beats the edited second.
	if err := c.DeleteWorkout(ctx, me, first); err != nil {
		t.Fatal(err)
	}
	p, err := c.records.ExerciseProgress(ctx, me, lift.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Sessions) != 2 || p.Sessions[0].WorkoutID != third || len(p.Sessions[0].Records) == 0 || len(p.Sessions[1].Records) != 0 {
		t.Fatalf("progress after delete = %+v", p.Sessions)
	}
	if p.Sessions[0].Best[records.RecordE1RM] != 128333 || len(p.Sessions[0].Sets) != 1 {
		t.Errorf("third session = %+v", p.Sessions[0])
	}
	if len(p.RepRecords) != 5 || p.RepRecords[0].WeightG != 110000 || p.RepRecords[4].Reps != 5 {
		t.Errorf("rep records = %+v", p.RepRecords)
	}
	if len(p.Records) != len(records.RecordKinds(domain.KindWeightReps)) || p.Records[0].Kind != records.RecordE1RM {
		t.Errorf("records = %+v", p.Records)
	}

	b, err := c.records.Bests(ctx, me, []string{lift.ID, "never-done"})
	if err != nil || b[lift.ID][records.RecordWeight] != 110000 || len(b) != 1 {
		t.Errorf("bests = %+v, %v", b, err)
	}
	// Someone else sees nothing of mine.
	if b, _ := c.records.Bests(ctx, friend, []string{lift.ID}); len(b) != 0 {
		t.Errorf("friend's bests = %+v", b)
	}
	page, err := c.ListWorkouts(ctx, me, "")
	if err != nil || len(page.Workouts) != 2 || len(page.Workouts[0].Records) == 0 || len(page.Workouts[0].Exercises) != 1 {
		t.Errorf("history = %+v, %v", page, err)
	}
}
