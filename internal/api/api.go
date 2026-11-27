// Package api exposes the core layer over JSON/HTTP.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/kipit-labs/kipitfit/internal/config"
	"github.com/kipit-labs/kipitfit/internal/core"
	"github.com/kipit-labs/kipitfit/internal/store"
)

// Codes the API itself answers with, next to core.Codes; each needs a
// translation under errors.<code> in web/src/locales.
var apiCodes = []string{"cross_origin", "too_many_attempts", "invalid_body", "not_found", "conflict", "invalid", "internal"}

type API struct {
	core    *core.Core
	cfg     config.Config
	log     *slog.Logger
	limiter *failLimiter
	handler http.Handler
}

func New(c *core.Core, cfg config.Config, log *slog.Logger) *API {
	a := &API{core: c, cfg: cfg, log: log, limiter: newFailLimiter(10, 15*time.Minute)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("GET /api/auth/state", a.authState)
	mux.HandleFunc("POST /api/auth/setup", a.setup)
	mux.HandleFunc("POST /api/auth/login", a.login)
	mux.HandleFunc("POST /api/auth/logout", a.logout)
	mux.HandleFunc("POST /api/invites/accept", a.acceptInvite)
	mux.HandleFunc("GET /api/me", a.me)
	mux.HandleFunc("PATCH /api/me", a.updateMe)
	mux.HandleFunc("GET /api/admin/invites", a.listInvites)
	mux.HandleFunc("POST /api/admin/invites", a.createInvite)
	mux.HandleFunc("DELETE /api/admin/invites/{id}", a.deleteInvite)
	mux.HandleFunc("GET /api/exercises", a.listExercises)
	mux.HandleFunc("POST /api/exercises", a.createExercise)
	mux.HandleFunc("GET /api/exercises/{id}", a.getExercise)
	mux.HandleFunc("PATCH /api/exercises/{id}", a.updateExercise)
	mux.HandleFunc("DELETE /api/exercises/{id}", a.deleteExercise)
	mux.HandleFunc("GET /api/exercises/{id}/progress", a.exerciseProgress)
	mux.HandleFunc("GET /api/routines", a.listRoutines)
	mux.HandleFunc("POST /api/routines", a.createRoutine)
	mux.HandleFunc("PUT /api/routines/order", a.reorder)
	mux.HandleFunc("PUT /api/routines/{id}", a.updateRoutine)
	mux.HandleFunc("DELETE /api/routines/{id}", a.deleteRoutine)
	mux.HandleFunc("GET /api/folders", a.listFolders)
	mux.HandleFunc("POST /api/folders", a.createFolder)
	mux.HandleFunc("PATCH /api/folders/{id}", a.renameFolder)
	mux.HandleFunc("DELETE /api/folders/{id}", a.deleteFolder)
	mux.HandleFunc("GET /api/workouts", a.listWorkouts)
	mux.HandleFunc("GET /api/workouts/{id}", a.getWorkout)
	mux.HandleFunc("PUT /api/workouts/{id}", a.putWorkout)
	mux.HandleFunc("DELETE /api/workouts/{id}", a.deleteWorkout)
	mux.HandleFunc("GET /api/stats", a.stats)
	mux.HandleFunc("GET /api/previous", a.previous)
	mux.HandleFunc("GET /api/bests", a.bests)
	mux.HandleFunc("GET /api/media/{key...}", a.media)
	mux.HandleFunc("GET /api/library", a.library)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "no such endpoint", nil)
	})
	a.handler = a.protect(mux)
	return a
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) { a.handler.ServeHTTP(w, r) }

type userKey struct{}

// user is the signed-in user protect put on the request.
func user(r *http.Request) store.User {
	u, _ := r.Context().Value(userKey{}).(store.User)
	return u
}

func withUser(ctx context.Context, u store.User) context.Context {
	return context.WithValue(ctx, userKey{}, u)
}

func (a *API) fail(w http.ResponseWriter, err error) {
	var ce *core.Error
	switch {
	case errors.As(err, &ce):
		writeError(w, status(ce.Kind), ce.Code, ce.Msg, ce.Params)
	case errors.Is(err, core.ErrInvalid):
		writeError(w, http.StatusBadRequest, "invalid", err.Error(), nil)
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "not found", nil)
	case errors.Is(err, store.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", "already exists", nil)
	default:
		a.log.Error("request failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "internal error", nil)
	}
}

func status(kind error) int {
	switch {
	case errors.Is(kind, core.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(kind, core.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(kind, core.ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusBadRequest
	}
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body: "+err.Error(), nil)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

type apiError struct {
	Error  string         `json:"error"`
	Code   string         `json:"code"`
	Params map[string]any `json:"params,omitempty"`
}

func writeError(w http.ResponseWriter, status int, code, msg string, params map[string]any) {
	writeJSON(w, status, apiError{Error: msg, Code: code, Params: params})
}
