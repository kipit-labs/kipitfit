package api

import (
	"net/http"

	"github.com/kipit-labs/kipitfit/internal/core"
)

func (a *API) listExercises(w http.ResponseWriter, r *http.Request) {
	exs, err := a.core.ListExercises(r.Context(), user(r))
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, exs)
}

func (a *API) getExercise(w http.ResponseWriter, r *http.Request) {
	e, err := a.core.GetExercise(r.Context(), user(r), r.PathValue("id"))
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (a *API) createExercise(w http.ResponseWriter, r *http.Request) {
	var in core.ExerciseInput
	if !decode(w, r, &in) {
		return
	}
	e, err := a.core.CreateExercise(r.Context(), user(r), in)
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (a *API) updateExercise(w http.ResponseWriter, r *http.Request) {
	var in core.ExerciseInput
	if !decode(w, r, &in) {
		return
	}
	e, err := a.core.UpdateExercise(r.Context(), user(r), r.PathValue("id"), in)
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (a *API) deleteExercise(w http.ResponseWriter, r *http.Request) {
	if err := a.core.DeleteExercise(r.Context(), user(r), r.PathValue("id")); err != nil {
		a.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) exerciseProgress(w http.ResponseWriter, r *http.Request) {
	p, err := a.core.ExerciseProgress(r.Context(), user(r), r.PathValue("id"))
	reply(a, w, http.StatusOK, p, err)
}
