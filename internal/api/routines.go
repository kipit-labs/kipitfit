package api

import (
	"net/http"

	"github.com/kipit-labs/kipitfit/internal/core"
	"github.com/kipit-labs/kipitfit/internal/domain"
)

func (a *API) listRoutines(w http.ResponseWriter, r *http.Request) {
	rs, err := a.core.ListRoutines(r.Context(), user(r))
	reply(a, w, http.StatusOK, rs, err)
}

func (a *API) createRoutine(w http.ResponseWriter, r *http.Request) {
	var in domain.Routine
	if !decode(w, r, &in) {
		return
	}
	rt, err := a.core.CreateRoutine(r.Context(), user(r), in)
	reply(a, w, http.StatusCreated, rt, err)
}

func (a *API) updateRoutine(w http.ResponseWriter, r *http.Request) {
	var in domain.Routine
	if !decode(w, r, &in) {
		return
	}
	rt, err := a.core.UpdateRoutine(r.Context(), user(r), r.PathValue("id"), in)
	reply(a, w, http.StatusOK, rt, err)
}

func (a *API) deleteRoutine(w http.ResponseWriter, r *http.Request) {
	noContent(a, w, a.core.DeleteRoutine(r.Context(), user(r), r.PathValue("id")))
}

func (a *API) reorder(w http.ResponseWriter, r *http.Request) {
	var in core.Order
	if !decode(w, r, &in) {
		return
	}
	noContent(a, w, a.core.Reorder(r.Context(), user(r), in))
}

func (a *API) listFolders(w http.ResponseWriter, r *http.Request) {
	fs, err := a.core.ListFolders(r.Context(), user(r))
	reply(a, w, http.StatusOK, fs, err)
}

func (a *API) createFolder(w http.ResponseWriter, r *http.Request) {
	var in core.FolderInput
	if !decode(w, r, &in) {
		return
	}
	f, err := a.core.CreateFolder(r.Context(), user(r), in)
	reply(a, w, http.StatusCreated, f, err)
}

func (a *API) renameFolder(w http.ResponseWriter, r *http.Request) {
	var in core.FolderInput
	if !decode(w, r, &in) {
		return
	}
	f, err := a.core.RenameFolder(r.Context(), user(r), r.PathValue("id"), in)
	reply(a, w, http.StatusOK, f, err)
}

func (a *API) deleteFolder(w http.ResponseWriter, r *http.Request) {
	noContent(a, w, a.core.DeleteFolder(r.Context(), user(r), r.PathValue("id")))
}

// reply writes v with status, or the error.
func reply(a *API, w http.ResponseWriter, status int, v any, err error) {
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, status, v)
}

func noContent(a *API, w http.ResponseWriter, err error) {
	if err != nil {
		a.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
