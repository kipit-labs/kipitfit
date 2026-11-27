package core

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/kipit-labs/kipitfit/internal/domain"
	"github.com/kipit-labs/kipitfit/internal/ids"
	"github.com/kipit-labs/kipitfit/internal/media"
)

func testPack() domain.ParsedPack {
	file := func(key string) domain.PackFile {
		return domain.PackFile{Key: key, ContentType: "image/webp", Size: 4,
			Open: func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("webp")), nil }}
	}
	ex := func(slug, name string, kind domain.Kind, files ...domain.PackFile) domain.PackExercise {
		return domain.PackExercise{Exercise: domain.Exercise{Slug: slug, Name: name, Kind: kind, Category: "strength",
			Equipment: "barbell", PrimaryMuscles: []string{"glutes"}}, Files: files}
	}
	return domain.ParsedPack{Pack: domain.Pack{Name: "test", Version: "1", Credit: "Exercise data by Test"}, Exercises: []domain.PackExercise{
		ex("test:deadlift", "barbell  deadlift", domain.KindWeightReps, file("packs/test/1/deadlift-start.webp"), file("packs/test/1/deadlift-peak.webp")),
		ex("test:plank", "Plank", domain.KindDuration, file("packs/test/1/plank-main.webp")),
		// Same name as a built-in one, other kind: its sets wouldn't fit.
		ex("test:situp", "3/4 Sit-Up", domain.KindWeightReps),
	}}
}

func TestImportPack(t *testing.T) {
	ctx := context.Background()
	c, me, _ := twoUsers(t)
	if _, err := c.ImportPack(ctx, testPack()); err == nil {
		t.Fatal("import with media off")
	}
	m := media.NewMemory()
	c.media = m

	builtin := map[string]domain.Exercise{}
	exs, _ := c.ListExercises(ctx, me)
	for _, e := range exs {
		builtin[e.Name] = e
	}
	dead, situp, squat := builtin["Barbell Deadlift"], builtin["3/4 Sit-Up"], builtin["Barbell Squat"]
	custom, err := c.CreateExercise(ctx, me, ExerciseInput{Name: ptr("Landmine Press"), Kind: ptr(domain.KindWeightReps), PrimaryMuscles: &[]string{"shoulders"}})
	if err != nil {
		t.Fatal(err)
	}
	r, err := c.CreateRoutine(ctx, me, domain.Routine{Name: "Pull", Exercises: []domain.RoutineExercise{{ExerciseID: dead.ID}, {ExerciseID: situp.ID}}})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 11, 20, 18, 0, 0, 0, time.UTC)
	wid := ids.New()
	_, err = c.PutWorkout(ctx, me, wid, domain.Workout{Title: "Pull", StartedAt: start, EndedAt: start.Add(time.Hour), Exercises: []domain.WorkoutExercise{
		{ExerciseID: dead.ID, Sets: []domain.WorkoutSet{{Type: domain.SetNormal, WeightG: ptr(100000), Reps: ptr(5)}}},
		{ExerciseID: situp.ID, Sets: []domain.WorkoutSet{{Type: domain.SetNormal, Reps: ptr(20)}}},
		{ExerciseID: custom.ID, Sets: []domain.WorkoutSet{{Type: domain.SetNormal, WeightG: ptr(20000), Reps: ptr(10)}}},
	}})
	if err != nil {
		t.Fatal(err)
	}

	res, err := c.ImportPack(ctx, testPack())
	if err != nil {
		t.Fatal(err)
	}
	// 2 rows re-pointed (routine + workout deadlift); the whole built-in library archived.
	if res.Uploaded != 3 || res.Skipped != 0 || res.Exercises != 3 || res.Repointed != 2 || res.Archived != 876 {
		t.Fatalf("import = %+v", res)
	}
	exs, _ = c.ListExercises(ctx, me)
	names := map[string]domain.Exercise{}
	for _, e := range exs {
		names[e.Name] = e
	}
	if len(exs) != 4 || names["Landmine Press"].ID != custom.ID || len(names["barbell  deadlift"].Media) != 2 {
		t.Fatalf("library after import: %+v", exs)
	}
	newDead := names["barbell  deadlift"]
	r, _ = c.store.GetRoutine(ctx, me.ID, r.ID)
	if r.Exercises[0].ExerciseID != newDead.ID || r.Exercises[1].ExerciseID != situp.ID || !r.Exercises[1].Exercise.Archived {
		t.Fatalf("routine after import: %+v", r.Exercises)
	}
	w, _ := c.GetWorkout(ctx, me, wid)
	if w.Exercises[0].ExerciseID != newDead.ID || w.Exercises[1].ExerciseID != situp.ID || w.Exercises[2].ExerciseID != custom.ID {
		t.Fatalf("workout after import: %+v", w.Exercises)
	}
	// A workout started before the import still saves on the old exercise.
	if old, err := c.GetExercise(ctx, me, squat.ID); err != nil || !old.Archived {
		t.Fatalf("old built-in = %+v, %v", old, err)
	}
	if _, err := c.PutWorkout(ctx, me, ids.New(), domain.Workout{Title: "Legs", StartedAt: start, EndedAt: start.Add(time.Hour),
		Exercises: []domain.WorkoutExercise{{ExerciseID: squat.ID, Sets: []domain.WorkoutSet{{Type: domain.SetNormal, WeightG: ptr(80000), Reps: ptr(5)}}}}}); err != nil {
		t.Fatalf("workout on an archived built-in: %v", err)
	}
	if l, _ := c.Library(ctx); l.Pack == nil || l.Pack.Credit != "Exercise data by Test" || !l.Media {
		t.Fatalf("library = %+v", l)
	}

	// The next start doesn't bring the built-in library back; a re-run skips what's uploaded.
	if err := c.SyncExercises(ctx); err != nil {
		t.Fatal(err)
	}
	res, err = c.ImportPack(ctx, testPack())
	if err != nil || res.Uploaded != 0 || res.Skipped != 3 || res.Repointed != 0 || res.Archived != 0 || m.Puts != 3 {
		t.Fatalf("re-run = %+v, %v (puts %d)", res, err, m.Puts)
	}
	if exs, _ = c.ListExercises(ctx, me); len(exs) != 4 || exs[0].ID == "" {
		t.Fatalf("library after re-run: %d", len(exs))
	}
	if again, _ := c.GetExercise(ctx, me, newDead.ID); again.Name != newDead.Name {
		t.Fatal("re-run changed the deadlift's id")
	}
}

func TestNormName(t *testing.T) {
	if normName("Pull-Up") != normName(" pull up ") || normName("Pull-Up") == normName("Push-Up") {
		t.Fatal("normName")
	}
}
