package api

import (
	"net/http"

	"github.com/kipit-labs/kipitfit/internal/usecase/auth"
)

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, user(r))
}

func (a *API) updateMe(w http.ResponseWriter, r *http.Request) {
	var p auth.Profile
	if !decode(w, r, &p) {
		return
	}
	u, err := a.auth.UpdateMe(r.Context(), user(r), p)
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (a *API) listInvites(w http.ResponseWriter, r *http.Request) {
	list, err := a.auth.ListInvites(r.Context(), user(r))
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (a *API) createInvite(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Email string `json:"email"`
	}
	if !decode(w, r, &b) {
		return
	}
	i, err := a.auth.CreateInvite(r.Context(), user(r), b.Email)
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, i)
}

func (a *API) deleteInvite(w http.ResponseWriter, r *http.Request) {
	if err := a.auth.DeleteInvite(r.Context(), user(r), r.PathValue("id")); err != nil {
		a.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
