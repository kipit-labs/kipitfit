package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// twoSessions sets up an admin, invites a friend, and returns both sessions.
func twoSessions(t *testing.T, a *API) (admin, friend string) {
	t.Helper()
	w := do(a, "POST", "/api/auth/setup", `{"setupToken":"setup","email":"admin@b.c","name":"Admin","password":"correct horse"}`, "", "")
	admin = session(t, w)
	w = do(a, "POST", "/api/admin/invites", `{"email":"friend@b.c"}`, admin, "")
	var inv struct{ Link string }
	_ = json.Unmarshal(w.Body.Bytes(), &inv)
	token := inv.Link[strings.Index(inv.Link, "token=")+len("token="):]
	w = do(a, "POST", "/api/invites/accept", `{"token":"`+token+`","name":"Friend","password":"correct horse"}`, "", "")
	return admin, session(t, w)
}

func TestRoutinesAPI(t *testing.T) {
	a := newAPI(t)
	me, friend := twoSessions(t, a)
	var ex, folder, routine struct{ ID string }
	w := do(a, "POST", "/api/exercises", `{"name":"Landmine Press","kind":"weight_reps","primaryMuscles":["shoulders"]}`, me, "")
	_ = json.Unmarshal(w.Body.Bytes(), &ex)
	w = do(a, "POST", "/api/folders", `{"name":"Push"}`, me, "")
	if w.Code != http.StatusCreated {
		t.Fatalf("create folder: %d %s", w.Code, w.Body)
	}
	_ = json.Unmarshal(w.Body.Bytes(), &folder)

	body := `{"name":"Push A","folderId":"` + folder.ID + `","exercises":[{"exerciseId":"` + ex.ID +
		`","restS":90,"sets":[{"type":"warmup","weightG":20000,"repsMin":10,"repsMax":10},{"type":"normal","weightG":102058}]}]}`
	w = do(a, "POST", "/api/routines", body, me, "")
	if w.Code != http.StatusCreated || !strings.Contains(w.Body.String(), `"name":"Landmine Press"`) {
		t.Fatalf("create routine: %d %s", w.Code, w.Body)
	}
	_ = json.Unmarshal(w.Body.Bytes(), &routine)

	w = do(a, "GET", "/api/routines", "", me, "")
	var list struct {
		Folders  []struct{ ID, Name string }
		Routines []struct {
			ID, FolderID string
			Exercises    []struct {
				RestS *int
				Sets  []struct {
					Type    string
					WeightG *int
				}
			}
		}
	}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil || len(list.Folders) != 1 || len(list.Routines) != 1 ||
		list.Routines[0].FolderID != folder.ID || *list.Routines[0].Exercises[0].Sets[1].WeightG != 102058 {
		t.Fatalf("list: %v %s", err, w.Body)
	}

	if w := do(a, "POST", "/api/routines", `{"name":""}`, me, ""); w.Code != http.StatusBadRequest || code(t, w) != "title_invalid" {
		t.Errorf("blank name: %d %s", w.Code, w.Body)
	}
	if w := do(a, "POST", "/api/routines", `{"name":"R","exercises":[{"exerciseId":"`+ex.ID+`","sets":[{"type":"giant"}]}]}`, me, ""); code(t, w) != "set_type_invalid" {
		t.Errorf("bad set type: %d %s", w.Code, w.Body)
	}

	// Someone else's routine, folder and exercise answer 404.
	for _, req := range []struct{ method, path, body string }{
		{"PUT", "/api/routines/" + routine.ID, `{"name":"Mine"}`},
		{"DELETE", "/api/routines/" + routine.ID, ""},
		{"PATCH", "/api/folders/" + folder.ID, `{"name":"Mine"}`},
		{"DELETE", "/api/folders/" + folder.ID, ""},
		{"PUT", "/api/routines/order", `{"routines":["` + routine.ID + `"]}`},
		{"POST", "/api/routines", `{"name":"R","exercises":[{"exerciseId":"` + ex.ID + `"}]}`},
	} {
		if w := do(a, req.method, req.path, req.body, friend, ""); w.Code != http.StatusNotFound {
			t.Errorf("friend %s %s: %d %s", req.method, req.path, w.Code, w.Body)
		}
	}
	if w := do(a, "GET", "/api/routines", "", friend, ""); w.Body.String() != `{"folders":[],"routines":[]}`+"\n" {
		t.Errorf("friend list: %s", w.Body)
	}

	// Delete the used exercise: archived, still readable, gone from the library.
	if w := do(a, "DELETE", "/api/exercises/"+ex.ID, "", me, ""); w.Code != http.StatusNoContent {
		t.Fatalf("delete used exercise: %d %s", w.Code, w.Body)
	}
	if w := do(a, "GET", "/api/exercises/"+ex.ID, "", me, ""); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"archived":true`) {
		t.Errorf("get archived: %d %s", w.Code, w.Body)
	}
	if w := do(a, "GET", "/api/exercises", "", me, ""); strings.Contains(w.Body.String(), ex.ID) {
		t.Errorf("archived exercise listed: %s", w.Body)
	}

	if w := do(a, "PUT", "/api/routines/order", `{"folders":["`+folder.ID+`"],"routines":["`+routine.ID+`"]}`, me, ""); w.Code != http.StatusNoContent {
		t.Errorf("reorder: %d %s", w.Code, w.Body)
	}
	if w := do(a, "PATCH", "/api/folders/"+folder.ID, `{"name":"Push day"}`, me, ""); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Push day") {
		t.Errorf("rename folder: %d %s", w.Code, w.Body)
	}
	if w := do(a, "PUT", "/api/routines/"+routine.ID, `{"name":"Push B"}`, me, ""); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"exercises":[]`) {
		t.Errorf("replace routine: %d %s", w.Code, w.Body)
	}
	if w := do(a, "DELETE", "/api/routines/"+routine.ID, "", me, ""); w.Code != http.StatusNoContent {
		t.Errorf("delete routine: %d %s", w.Code, w.Body)
	}
	if w := do(a, "DELETE", "/api/folders/"+folder.ID, "", me, ""); w.Code != http.StatusNoContent {
		t.Errorf("delete folder: %d %s", w.Code, w.Body)
	}
}

func TestWorkoutsAPI(t *testing.T) {
	a := newAPI(t)
	me, friend := twoSessions(t, a)
	var ex struct{ ID string }
	w := do(a, "POST", "/api/exercises", `{"name":"Landmine Press","kind":"weight_reps","primaryMuscles":["shoulders"]}`, me, "")
	_ = json.Unmarshal(w.Body.Bytes(), &ex)
	id := "01JDZ0000000000000000000AA"
	body := `{"title":"Push","startedAt":"2026-11-20T18:00:00Z","endedAt":"2026-11-20T19:00:00Z","exercises":[{"exerciseId":"` + ex.ID +
		`","sets":[{"type":"normal","weightG":102058,"reps":5,"rpe":8.5}]}]}`
	for range 2 {
		if w := do(a, "PUT", "/api/workouts/"+id, body, me, ""); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"rpe":8.5`) ||
			!strings.Contains(w.Body.String(), `"volumeG":510290`) {
			t.Fatalf("put: %d %s", w.Code, w.Body)
		}
	}
	w = do(a, "GET", "/api/workouts", "", me, "")
	if strings.Count(w.Body.String(), `"id":"`+id) != 1 {
		t.Errorf("list after two puts: %s", w.Body)
	}
	if w := do(a, "PUT", "/api/workouts/"+id, body, friend, ""); w.Code != http.StatusNotFound {
		t.Errorf("friend puts my id: %d %s", w.Code, w.Body)
	}
	if w := do(a, "GET", "/api/workouts/"+id, "", friend, ""); w.Code != http.StatusNotFound {
		t.Errorf("friend gets my workout: %d", w.Code)
	}
	if w := do(a, "PUT", "/api/workouts/"+id, strings.Replace(body, "8.5", "11", 1), me, ""); code(t, w) != "rpe_invalid" {
		t.Errorf("rpe 11: %d %s", w.Code, w.Body)
	}
	if w := do(a, "GET", "/api/previous?exercise="+ex.ID+",nope", "", me, ""); !strings.Contains(w.Body.String(), `"`+ex.ID+`":[{"type":"normal","weightG":102058`) {
		t.Errorf("previous: %s", w.Body)
	}
	if w := do(a, "DELETE", "/api/workouts/"+id, "", friend, ""); w.Code != http.StatusNotFound {
		t.Errorf("friend deletes my workout: %d", w.Code)
	}
	if w := do(a, "DELETE", "/api/workouts/"+id, "", me, ""); w.Code != http.StatusNoContent {
		t.Errorf("delete: %d %s", w.Code, w.Body)
	}
}
