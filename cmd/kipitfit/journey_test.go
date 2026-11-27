package main

// End-to-end journeys: the real wire() over HTTP on a real SQLite database,
// the way the phone uses the API. Only the media bucket is in memory.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kipit-labs/kipitfit/internal/config"
	"github.com/kipit-labs/kipitfit/internal/core"
	"github.com/kipit-labs/kipitfit/internal/media"
	"github.com/kipit-labs/kipitfit/internal/store/sqlite"
)

const setupToken = "setup-token"

type testApp struct {
	url  string
	core *core.Core
}

// newTestApp serves wire() on a fresh database; m is nil for media off.
func newTestApp(t *testing.T, m media.Store) *testApp {
	t.Helper()
	ctx := context.Background()
	st, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "kipitfit.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	cfg := config.Config{SetupToken: setupToken, AppURL: "http://kipitfit.test"}
	h, c, err := wire(ctx, cfg, st, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &testApp{url: srv.URL, core: c}
}

// client is one browser: it keeps its session cookie.
type client struct {
	t    *testing.T
	base string
	http *http.Client
}

func (a *testApp) client(t *testing.T) *client {
	jar, _ := cookiejar.New(nil)
	return &client{t: t, base: a.url, http: &http.Client{Jar: jar}}
}

type response struct {
	status int
	body   []byte
	header http.Header
}

func (r response) String() string { return string(r.body) }

// code is the error code of an error response.
func (r response) code() string {
	var e struct{ Code string }
	_ = json.Unmarshal(r.body, &e)
	return e.Code
}

// do sends body (JSON-encoded unless a string) and decodes a 2xx answer into out.
func (c *client) do(method, path string, body, out any) response {
	c.t.Helper()
	var rd io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		rd = strings.NewReader(b)
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			c.t.Fatal(err)
		}
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, c.base+path, rd)
	if err != nil {
		c.t.Fatal(err)
	}
	if rd != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	r := response{status: res.StatusCode, body: raw, header: res.Header}
	if out != nil && res.StatusCode < 300 {
		if err := json.Unmarshal(raw, out); err != nil {
			c.t.Fatalf("%s %s: decode %v: %s", method, path, err, raw)
		}
	}
	return r
}

// want fails the test unless r has status (and, for an error, code).
func want(t *testing.T, what string, r response, status int, code ...string) {
	t.Helper()
	if r.status != status || len(code) > 0 && r.code() != code[0] {
		t.Fatalf("%s: %d %s, want %d %v", what, r.status, r, status, code)
	}
}

const password = "correct horse battery"

// setUp creates the first admin and returns their signed-in client.
func (a *testApp) setUp(t *testing.T) *client {
	t.Helper()
	admin := a.client(t)
	want(t, "setup", admin.do("POST", "/api/auth/setup",
		map[string]string{"setupToken": setupToken, "email": "admin@kipitfit.test", "name": "Admin", "password": password}, nil), http.StatusCreated)
	return admin
}

// invite has admin invite email and returns the invite's token.
func invite(t *testing.T, admin *client, email string) (id, token string) {
	t.Helper()
	var inv struct{ ID, Link string }
	want(t, "invite", admin.do("POST", "/api/admin/invites", map[string]string{"email": email}, &inv), http.StatusCreated)
	_, token, ok := strings.Cut(inv.Link, "token=")
	if !ok || !strings.HasPrefix(inv.Link, "http://kipitfit.test/") {
		t.Fatalf("invite link %q", inv.Link)
	}
	return inv.ID, token
}

// join invites email and accepts the invite in a new client.
func (a *testApp) join(t *testing.T, admin *client, email, name string) *client {
	t.Helper()
	_, token := invite(t, admin, email)
	c := a.client(t)
	want(t, "accept invite", c.do("POST", "/api/invites/accept", map[string]string{"token": token, "name": name, "password": password}, nil), http.StatusCreated)
	return c
}

type user struct {
	ID, Email, Name, Language, Unit string
	IsAdmin                         bool
}

// A fresh instance needs the setup token for its first admin, who then
// invites people; invites work once, can be revoked, and sessions end on
// sign-out.
func TestJourneyAccounts(t *testing.T) {
	app := newTestApp(t, nil)
	anon := app.client(t)

	var state struct {
		SetupRequired bool
		User          *user
	}
	anon.do("GET", "/api/auth/state", nil, &state)
	if !state.SetupRequired || state.User != nil {
		t.Fatalf("fresh state = %+v", state)
	}
	want(t, "setup with a wrong token", anon.do("POST", "/api/auth/setup",
		map[string]string{"setupToken": "guess", "email": "x@kipitfit.test", "name": "X", "password": password}, nil), http.StatusUnauthorized, "wrong_setup_token")

	admin := app.setUp(t)
	want(t, "second setup", anon.do("POST", "/api/auth/setup",
		map[string]string{"setupToken": setupToken, "email": "x@kipitfit.test", "name": "X", "password": password}, nil), http.StatusBadRequest, "setup_done")
	admin.do("GET", "/api/auth/state", nil, &state)
	if state.SetupRequired || state.User == nil || !state.User.IsAdmin {
		t.Fatalf("admin state = %+v", state)
	}

	// An invite works once.
	_, token := invite(t, admin, "ana@kipitfit.test")
	ana := app.client(t)
	want(t, "short password", ana.do("POST", "/api/invites/accept", map[string]string{"token": token, "name": "Ana", "password": "short"}, nil),
		http.StatusBadRequest, "password_too_short")
	want(t, "accept", ana.do("POST", "/api/invites/accept", map[string]string{"token": token, "name": "Ana", "password": password}, nil), http.StatusCreated)
	want(t, "accept twice", app.client(t).do("POST", "/api/invites/accept", map[string]string{"token": token, "name": "Eve", "password": password}, nil),
		http.StatusBadRequest, "invalid_invite")

	var me user
	ana.do("GET", "/api/me", nil, &me)
	if me.Email != "ana@kipitfit.test" || me.Name != "Ana" || me.IsAdmin || me.Unit != "kg" {
		t.Fatalf("me = %+v", me)
	}
	want(t, "profile", ana.do("PATCH", "/api/me", map[string]string{"unit": "lb", "language": "fr"}, &me), http.StatusOK)
	if me.Unit != "lb" || me.Language != "fr" {
		t.Errorf("profile = %+v", me)
	}
	want(t, "invite as a member", ana.do("POST", "/api/admin/invites", map[string]string{"email": "bo@kipitfit.test"}, nil), http.StatusForbidden, "admin_only")

	// A revoked invite no longer lets anyone in.
	id, token := invite(t, admin, "bo@kipitfit.test")
	var pending []struct{ ID, Email string }
	admin.do("GET", "/api/admin/invites", nil, &pending)
	if len(pending) != 1 || pending[0].ID != id {
		t.Fatalf("pending invites = %+v", pending)
	}
	want(t, "revoke", admin.do("DELETE", "/api/admin/invites/"+id, nil, nil), http.StatusNoContent)
	want(t, "accept revoked", app.client(t).do("POST", "/api/invites/accept", map[string]string{"token": token, "name": "Bo", "password": password}, nil),
		http.StatusBadRequest, "invalid_invite")

	// Sign out, then back in.
	want(t, "logout", ana.do("POST", "/api/auth/logout", nil, nil), http.StatusNoContent)
	want(t, "me signed out", ana.do("GET", "/api/me", nil, nil), http.StatusUnauthorized, "signed_out")
	want(t, "wrong password", ana.do("POST", "/api/auth/login", map[string]string{"email": "ana@kipitfit.test", "password": "not the password"}, nil),
		http.StatusUnauthorized, "invalid_credentials")
	want(t, "login", ana.do("POST", "/api/auth/login", map[string]string{"email": "ANA@kipitfit.test", "password": password}, nil), http.StatusOK)
	want(t, "me", ana.do("GET", "/api/me", nil, &me), http.StatusOK)
	if me.Unit != "lb" {
		t.Errorf("profile lost across sessions: %+v", me)
	}
}
