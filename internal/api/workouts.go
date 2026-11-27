package api

import (
	"net/http"
	"strings"

	"github.com/kipit-labs/kipitfit/internal/store"
)

func (a *API) putWorkout(w http.ResponseWriter, r *http.Request) {
	var in store.Workout
	if !decode(w, r, &in) {
		return
	}
	wo, err := a.core.PutWorkout(r.Context(), user(r), r.PathValue("id"), in)
	reply(a, w, http.StatusOK, wo, err)
}

func (a *API) listWorkouts(w http.ResponseWriter, r *http.Request) {
	page, err := a.core.ListWorkouts(r.Context(), user(r), r.URL.Query().Get("after"))
	reply(a, w, http.StatusOK, page, err)
}

func (a *API) getWorkout(w http.ResponseWriter, r *http.Request) {
	wo, err := a.core.GetWorkout(r.Context(), user(r), r.PathValue("id"))
	reply(a, w, http.StatusOK, wo, err)
}

func (a *API) deleteWorkout(w http.ResponseWriter, r *http.Request) {
	noContent(a, w, a.core.DeleteWorkout(r.Context(), user(r), r.PathValue("id")))
}

func (a *API) previous(w http.ResponseWriter, r *http.Request) {
	var exs []string
	if q := r.URL.Query().Get("exercise"); q != "" {
		exs = strings.Split(q, ",")
	}
	prev, err := a.core.PreviousSets(r.Context(), user(r), exs)
	reply(a, w, http.StatusOK, prev, err)
}

func (a *API) bests(w http.ResponseWriter, r *http.Request) {
	var exs []string
	if q := r.URL.Query().Get("exercise"); q != "" {
		exs = strings.Split(q, ",")
	}
	b, err := a.core.Bests(r.Context(), user(r), exs)
	reply(a, w, http.StatusOK, b, err)
}
