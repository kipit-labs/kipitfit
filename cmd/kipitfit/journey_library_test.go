package main

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kipit-labs/kipitfit/internal/ids"
	"github.com/kipit-labs/kipitfit/internal/media"
	"github.com/kipit-labs/kipitfit/internal/pack"
	"github.com/kipit-labs/kipitfit/internal/store"
)

type library struct {
	Pack  *store.Pack
	Media bool
}

// A synthetic pack: third-party packs are never committed.
func testPack(squat exercise) pack.Pack {
	file := func(key string) pack.File {
		return pack.File{Key: key, ContentType: "image/webp", Size: 8,
			Open: func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("RIFFwebp")), nil }}
	}
	return pack.Pack{
		Pack: store.Pack{Name: "test", Version: "1", Credit: "Exercise data by Test", CreditURL: "https://example.com"},
		Exercises: []pack.Exercise{{
			Exercise: store.Exercise{Slug: "test:squat", Name: "barbell squat", Kind: store.KindWeightReps, Category: "strength",
				Equipment: "barbell", PrimaryMuscles: squat.PrimaryMuscles},
			Files: []pack.File{file("packs/test/1/squat-start.webp"), file("packs/test/1/squat-end.webp")},
		}},
	}
}

// Importing a pack makes it the library, with its pictures served from the
// bucket; routines and workouts move to the pack's exercise of the same name.
func TestJourneyPackImport(t *testing.T) {
	m := media.NewMemory()
	app := newTestApp(t, m)
	me := app.setUp(t)
	squat := libraryExercise(t, me, "Barbell Squat")
	var lib library
	me.do("GET", "/api/library", nil, &lib)
	if lib.Pack != nil || !lib.Media {
		t.Fatalf("library before import = %+v", lib)
	}

	var routine struct{ ID string }
	want(t, "routine", me.do("POST", "/api/routines", map[string]any{"name": "Legs",
		"exercises": []map[string]any{{"exerciseId": squat.ID, "sets": []map[string]any{{"type": "normal"}}}}}, &routine), http.StatusCreated)
	wid := ids.New()
	finish(t, me, wid, routine.ID, squat.ID, time.Now().UTC().AddDate(0, 0, -1), set(100000, 5))

	res, err := app.core.ImportPack(context.Background(), testPack(squat))
	if err != nil || res.Uploaded != 2 || res.Repointed != 2 {
		t.Fatalf("import = %+v, %v", res, err)
	}

	me.do("GET", "/api/library", nil, &lib)
	if lib.Pack == nil || lib.Pack.Name != "test" || lib.Pack.Credit != "Exercise data by Test" {
		t.Fatalf("library after import = %+v", lib)
	}
	var exs []exercise
	me.do("GET", "/api/exercises", nil, &exs)
	if len(exs) != 1 || len(exs[0].Media) != 2 {
		t.Fatalf("exercises after import = %+v", exs)
	}
	packSquat := exs[0]

	r := me.do("GET", "/api/media/"+packSquat.Media[0], nil, nil)
	if r.status != http.StatusOK || r.String() != "RIFFwebp" || r.header.Get("Content-Type") != "image/webp" {
		t.Fatalf("picture: %d %v %s", r.status, r.header, r)
	}
	want(t, "picture signed out", app.client(t).do("GET", "/api/media/"+packSquat.Media[0], nil, nil), http.StatusUnauthorized, "signed_out")

	var list struct {
		Routines []struct {
			Exercises []struct{ ExerciseID string }
		}
	}
	me.do("GET", "/api/routines", nil, &list)
	if list.Routines[0].Exercises[0].ExerciseID != packSquat.ID {
		t.Errorf("routine still on the old exercise: %+v", list)
	}
	var w struct {
		Exercises []struct{ ExerciseID string }
	}
	me.do("GET", "/api/workouts/"+wid, nil, &w)
	if w.Exercises[0].ExerciseID != packSquat.ID {
		t.Errorf("workout still on the old exercise: %+v", w)
	}
	var bests map[string]map[string]int
	me.do("GET", "/api/bests?exercise="+packSquat.ID, nil, &bests)
	if bests[packSquat.ID]["weight"] != 100000 {
		t.Errorf("records lost in the move: %v", bests)
	}

	// A re-run resumes: pictures already in the bucket are skipped.
	res, err = app.core.ImportPack(context.Background(), testPack(squat))
	if err != nil || res.Uploaded != 0 || res.Skipped != 2 {
		t.Errorf("re-import = %+v, %v", res, err)
	}
}

// Without a bucket the app still works, on the built-in library.
func TestJourneyMediaOff(t *testing.T) {
	app := newTestApp(t, nil)
	me := app.setUp(t)
	var lib library
	me.do("GET", "/api/library", nil, &lib)
	if lib.Pack != nil || lib.Media {
		t.Fatalf("library = %+v", lib)
	}
	want(t, "picture", me.do("GET", "/api/media/packs/test/1/squat-start.webp", nil, nil), http.StatusNotFound)
	if _, err := app.core.ImportPack(context.Background(), testPack(libraryExercise(t, me, "Barbell Squat"))); err == nil {
		t.Error("pack imported with media off")
	}
}
