package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

func (a *API) putWorkout(w http.ResponseWriter, r *http.Request) {
	var in domain.Workout
	if !decode(w, r, &in) {
		return
	}
	wo, err := a.workouts.PutWorkout(r.Context(), user(r), r.PathValue("id"), in)
	reply(a, w, http.StatusOK, wo, err)
}

func (a *API) listWorkouts(w http.ResponseWriter, r *http.Request) {
	page, err := a.workouts.ListWorkouts(r.Context(), user(r), r.URL.Query().Get("after"))
	reply(a, w, http.StatusOK, page, err)
}

func (a *API) getWorkout(w http.ResponseWriter, r *http.Request) {
	wo, err := a.workouts.GetWorkout(r.Context(), user(r), r.PathValue("id"))
	reply(a, w, http.StatusOK, wo, err)
}

func (a *API) deleteWorkout(w http.ResponseWriter, r *http.Request) {
	noContent(a, w, a.workouts.DeleteWorkout(r.Context(), user(r), r.PathValue("id")))
}

func (a *API) previous(w http.ResponseWriter, r *http.Request) {
	var exs []string
	if q := r.URL.Query().Get("exercise"); q != "" {
		exs = strings.Split(q, ",")
	}
	prev, err := a.workouts.PreviousSets(r.Context(), user(r), exs)
	reply(a, w, http.StatusOK, prev, err)
}

func (a *API) bests(w http.ResponseWriter, r *http.Request) {
	var exs []string
	if q := r.URL.Query().Get("exercise"); q != "" {
		exs = strings.Split(q, ",")
	}
	b, err := a.records.Bests(r.Context(), user(r), exs)
	reply(a, w, http.StatusOK, b, err)
}

func (a *API) stats(w http.ResponseWriter, r *http.Request) {
	// A from that doesn't parse is the zero time, which the use case refuses with its coded error.
	from, _ := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	st, err := a.workouts.Stats(r.Context(), user(r), from)
	reply(a, w, http.StatusOK, st, err)
}
