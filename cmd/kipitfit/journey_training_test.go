package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/kipit-labs/kipitfit/internal/ids"
)

type exercise struct {
	ID, Name, Kind   string
	PrimaryMuscles   []string
	Media            []string
	Custom, Archived bool
}

// libraryExercise finds a built-in exercise by name.
func libraryExercise(t *testing.T, c *client, name string) exercise {
	t.Helper()
	var exs []exercise
	want(t, "exercises", c.do("GET", "/api/exercises", nil, &exs), http.StatusOK)
	for _, e := range exs {
		if e.Name == name {
			return e
		}
	}
	t.Fatalf("no %q among %d exercises", name, len(exs))
	return exercise{}
}

type record struct {
	ExerciseID, Kind string
	Value            int
	Previous         *int
}

type workout struct {
	ID, Title, RoutineID string
	VolumeG              int
	Records              []record
}

// finish saves a workout of one exercise's sets the way the phone does at
// the end of a live workout: PUT under the ULID it made at the start.
func finish(t *testing.T, c *client, id, routineID, exerciseID string, at time.Time, sets ...map[string]any) workout {
	t.Helper()
	var w workout
	want(t, "finish workout", c.do("PUT", "/api/workouts/"+id, map[string]any{
		"title": "Legs", "routineId": routineID, "startedAt": at, "endedAt": at.Add(time.Hour),
		"exercises": []map[string]any{{"exerciseId": exerciseID, "sets": sets}},
	}, &w), http.StatusOK)
	return w
}

func set(weightG, reps int) map[string]any {
	return map[string]any{"type": "normal", "weightG": weightG, "reps": reps}
}

// Plan a routine in a folder, train it twice, and see the records, the
// exercise's progress and the stats follow; a deleted workout takes its
// records with it. Nothing of it shows to another member.
func TestJourneyTraining(t *testing.T) {
	app := newTestApp(t, nil)
	me := app.setUp(t)
	other := app.join(t, me, "other@kipitfit.test", "Other")
	squat := libraryExercise(t, me, "Barbell Squat")

	var folder struct{ ID string }
	want(t, "folder", me.do("POST", "/api/folders", map[string]string{"name": "Legs"}, &folder), http.StatusCreated)
	var routine struct{ ID string }
	want(t, "routine", me.do("POST", "/api/routines", map[string]any{"name": "Leg day", "folderId": folder.ID,
		"exercises": []map[string]any{{"exerciseId": squat.ID, "restS": 120, "sets": []map[string]any{
			{"type": "warmup", "weightG": 40000, "repsMin": 10, "repsMax": 10},
			{"type": "normal", "weightG": 100000, "repsMin": 5, "repsMax": 8},
		}}}}, &routine), http.StatusCreated)

	type routines struct {
		Folders  []struct{ ID string }
		Routines []struct {
			ID, FolderID string
			LastDoneAt   *time.Time
		}
	}
	var list routines
	me.do("GET", "/api/routines", nil, &list)
	if len(list.Routines) != 1 || list.Routines[0].FolderID != folder.ID || list.Routines[0].LastDoneAt != nil {
		t.Fatalf("routines = %+v", list)
	}

	// Starting it: nothing done before.
	var previous map[string][]map[string]any
	me.do("GET", "/api/previous?exercise="+squat.ID, nil, &previous)
	if len(previous[squat.ID]) != 0 {
		t.Fatalf("previous before any workout = %v", previous)
	}

	day1 := time.Now().UTC().AddDate(0, 0, -7).Truncate(time.Second)
	first := ids.New()
	w := finish(t, me, first, routine.ID, squat.ID, day1, map[string]any{"type": "warmup", "weightG": 40000, "reps": 10}, set(100000, 5), set(100000, 5))
	if w.VolumeG != 1000000 {
		t.Errorf("volume = %d, want the working sets only", w.VolumeG)
	}
	// A retried finish saves once.
	finish(t, me, first, routine.ID, squat.ID, day1, map[string]any{"type": "warmup", "weightG": 40000, "reps": 10}, set(100000, 5), set(100000, 5))
	var page struct{ Workouts []workout }
	me.do("GET", "/api/workouts", nil, &page)
	if len(page.Workouts) != 1 {
		t.Fatalf("workouts after a retried finish = %d", len(page.Workouts))
	}
	me.do("GET", "/api/routines", nil, &list)
	if list.Routines[0].LastDoneAt == nil {
		t.Errorf("routine not marked done: %+v", list.Routines[0])
	}

	// Next time, the app shows last time's sets; a heavier session beats the record.
	me.do("GET", "/api/previous?exercise="+squat.ID, nil, &previous)
	if len(previous[squat.ID]) != 3 {
		t.Fatalf("previous = %v", previous)
	}
	second := ids.New()
	w = finish(t, me, second, routine.ID, squat.ID, day1.AddDate(0, 0, 3), set(110000, 5))
	if !hasRecord(w.Records, "weight", 110000, 100000) {
		t.Errorf("records of the heavier workout = %+v", w.Records)
	}
	var bests map[string]map[string]int
	me.do("GET", "/api/bests?exercise="+squat.ID, nil, &bests)
	if bests[squat.ID]["weight"] != 110000 {
		t.Errorf("bests = %v", bests)
	}
	var progress struct {
		Records  []record
		Sessions []struct{ WorkoutID string }
	}
	me.do("GET", "/api/exercises/"+squat.ID+"/progress", nil, &progress)
	if len(progress.Sessions) != 2 || progress.Sessions[0].WorkoutID != second {
		t.Errorf("progress sessions = %+v", progress.Sessions)
	}
	var stats []struct {
		ID      string
		Sets    int
		Muscles map[string]int
	}
	from := url.QueryEscape(day1.AddDate(0, 0, -1).Format(time.RFC3339))
	me.do("GET", "/api/stats?from="+from, nil, &stats)
	if len(stats) != 2 || stats[0].ID != first || stats[0].Muscles[squat.PrimaryMuscles[0]] != 2 {
		t.Errorf("stats = %+v", stats)
	}

	// None of it is anyone else's business.
	for _, path := range []string{"/api/workouts/" + first, "/api/routines/" + routine.ID, "/api/folders/" + folder.ID} {
		method := map[bool]string{true: "GET", false: "DELETE"}[strings.HasPrefix(path, "/api/workouts")]
		want(t, "other "+method+" "+path, other.do(method, path, nil, nil), http.StatusNotFound)
	}
	var theirs struct{ Workouts []workout }
	other.do("GET", "/api/workouts", nil, &theirs)
	var theirBests map[string]map[string]int
	other.do("GET", "/api/bests?exercise="+squat.ID, nil, &theirBests)
	if len(theirs.Workouts) != 0 || len(theirBests) != 0 {
		t.Errorf("other sees workouts %+v, bests %v", theirs.Workouts, theirBests)
	}

	// Deleting the heavier workout gives the record back to the first one.
	want(t, "delete workout", me.do("DELETE", "/api/workouts/"+second, nil, nil), http.StatusNoContent)
	me.do("GET", "/api/bests?exercise="+squat.ID, nil, &bests)
	if bests[squat.ID]["weight"] != 100000 {
		t.Errorf("bests after delete = %v", bests)
	}
	want(t, "delete routine", me.do("DELETE", "/api/routines/"+routine.ID, nil, nil), http.StatusNoContent)
	want(t, "delete folder", me.do("DELETE", "/api/folders/"+folder.ID, nil, nil), http.StatusNoContent)
	me.do("GET", "/api/workouts/"+first, nil, &w)
	if w.ID != first {
		t.Errorf("workout lost with its routine: %+v", w)
	}
}

func hasRecord(rs []record, kind string, value, previous int) bool {
	for _, r := range rs {
		if r.Kind == kind && r.Value == value && r.Previous != nil && *r.Previous == previous {
			return true
		}
	}
	return false
}

// Your own exercise: create, rename, use, delete. One in use is archived, so
// the routine that has it keeps working; another member can't see it.
func TestJourneyCustomExercise(t *testing.T) {
	app := newTestApp(t, nil)
	me := app.setUp(t)
	other := app.join(t, me, "other@kipitfit.test", "Other")

	var ex exercise
	want(t, "create", me.do("POST", "/api/exercises", map[string]any{"name": "Landmine Press", "kind": "weight_reps", "primaryMuscles": []string{"shoulders"}}, &ex),
		http.StatusCreated)
	if !ex.Custom {
		t.Errorf("own exercise not custom: %+v", ex)
	}
	want(t, "duplicate name", me.do("POST", "/api/exercises", map[string]any{"name": "landmine press", "kind": "weight_reps", "primaryMuscles": []string{"shoulders"}}, nil),
		http.StatusConflict)
	want(t, "rename", me.do("PATCH", "/api/exercises/"+ex.ID, map[string]any{"name": "Landmine Press (one arm)"}, &ex), http.StatusOK)
	want(t, "edit library exercise", me.do("PATCH", "/api/exercises/"+libraryExercise(t, me, "Barbell Squat").ID, map[string]any{"name": "Mine"}, nil),
		http.StatusForbidden, "library_exercise_readonly")
	want(t, "other reads it", other.do("GET", "/api/exercises/"+ex.ID, nil, nil), http.StatusNotFound)

	var routine struct{ ID string }
	want(t, "routine", me.do("POST", "/api/routines", map[string]any{"name": "Push",
		"exercises": []map[string]any{{"exerciseId": ex.ID, "sets": []map[string]any{{"type": "normal"}}}}}, &routine), http.StatusCreated)
	want(t, "delete in use", me.do("DELETE", "/api/exercises/"+ex.ID, nil, nil), http.StatusNoContent)
	want(t, "get archived", me.do("GET", "/api/exercises/"+ex.ID, nil, &ex), http.StatusOK)
	if !ex.Archived {
		t.Errorf("deleted exercise in use not archived: %+v", ex)
	}
	var list struct {
		Routines []struct {
			Exercises []struct{ Exercise *exercise }
		}
	}
	me.do("GET", "/api/routines", nil, &list)
	if len(list.Routines) != 1 || len(list.Routines[0].Exercises) != 1 || list.Routines[0].Exercises[0].Exercise.ID != ex.ID {
		t.Errorf("routine lost its archived exercise: %+v", list)
	}

	var unused exercise
	me.do("POST", "/api/exercises", map[string]any{"name": "Cable Fly", "kind": "weight_reps", "primaryMuscles": []string{"chest"}}, &unused)
	want(t, "delete unused", me.do("DELETE", "/api/exercises/"+unused.ID, nil, nil), http.StatusNoContent)
	want(t, "get deleted", me.do("GET", "/api/exercises/"+unused.ID, nil, nil), http.StatusNotFound)
}
