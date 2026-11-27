package main

import (
	"net/http"
	"strings"
	"testing"
)

type access int

const (
	public access = iota
	member        // any signed-in user
	admin
)

// routeAccess is every API route and who may call it.
var routeAccess = map[string]access{
	"GET /api/health":                  public,
	"GET /api/auth/state":              public,
	"POST /api/auth/setup":             public,
	"POST /api/auth/login":             public,
	"POST /api/auth/logout":            public,
	"POST /api/invites/accept":         public,
	"GET /api/me":                      member,
	"PATCH /api/me":                    member,
	"GET /api/admin/invites":           admin,
	"POST /api/admin/invites":          admin,
	"DELETE /api/admin/invites/{id}":   admin,
	"GET /api/exercises":               member,
	"POST /api/exercises":              member,
	"GET /api/exercises/{id}":          member,
	"PATCH /api/exercises/{id}":        member,
	"DELETE /api/exercises/{id}":       member,
	"GET /api/exercises/{id}/progress": member,
	"GET /api/routines":                member,
	"POST /api/routines":               member,
	"PUT /api/routines/order":          member,
	"PUT /api/routines/{id}":           member,
	"DELETE /api/routines/{id}":        member,
	"GET /api/folders":                 member,
	"POST /api/folders":                member,
	"PATCH /api/folders/{id}":          member,
	"DELETE /api/folders/{id}":         member,
	"GET /api/workouts":                member,
	"GET /api/workouts/{id}":           member,
	"PUT /api/workouts/{id}":           member,
	"DELETE /api/workouts/{id}":        member,
	"GET /api/stats":                   member,
	"GET /api/previous":                member,
	"GET /api/bests":                   member,
	"GET /api/media/{key...}":          member,
	"GET /api/library":                 member,
}

func path(pattern string) string {
	return strings.NewReplacer("{id}", "01JDZ0000000000000000000ZZ", "{key...}", "packs/test/1/x.webp").Replace(pattern)
}

// Signed out, only public routes answer; members are refused admin routes.
func TestRoutesRefuseWhoMayNotCallThem(t *testing.T) {
	app := newTestApp(t, nil)
	adm := app.setUp(t)
	mem := app.join(t, adm, "member@kipitfit.test", "Member")
	anon := app.client(t)
	for route, who := range routeAccess {
		method, pattern, _ := strings.Cut(route, " ")
		if route == "POST /api/auth/logout" {
			continue // would end the session the other checks use
		}
		t.Run(route, func(t *testing.T) {
			r := anon.do(method, path(pattern), "{}", nil)
			if who == public && r.code() == "signed_out" || who != public && (r.status != http.StatusUnauthorized || r.code() != "signed_out") {
				t.Errorf("signed out: %d %s", r.status, r)
			}
			r = mem.do(method, path(pattern), "{}", nil)
			if refused := r.status == http.StatusForbidden && r.code() == "admin_only"; refused != (who == admin) {
				t.Errorf("member: %d %s", r.status, r)
			}
			if who != public && r.status == http.StatusUnauthorized {
				t.Errorf("member signed out: %s", r)
			}
		})
	}
}

// A mutation from another site is refused before it reaches any route, even
// with a valid session; reads and the app itself are served with the
// security headers.
func TestCrossSiteAndHeaders(t *testing.T) {
	app := newTestApp(t, nil)
	adm := app.setUp(t)
	req, _ := http.NewRequest("POST", app.url+"/api/admin/invites", strings.NewReader(`{"email":"x@kipitfit.test"}`))
	req.Header.Set("Origin", "https://evil.example")
	res, err := adm.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("cross-site invite: %d", res.StatusCode)
	}
	var pending []any
	adm.do("GET", "/api/admin/invites", nil, &pending)
	if len(pending) != 0 {
		t.Errorf("cross-site invite created: %v", pending)
	}

	want(t, "unknown endpoint", adm.do("GET", "/api/nope", nil, nil), http.StatusNotFound, "not_found")
	for _, p := range []string{"/", "/routines/01JDZ0000000000000000000ZZ", "/api/health"} {
		r := adm.do("GET", p, nil, nil)
		if r.status >= 300 || r.header.Get("X-Content-Type-Options") != "nosniff" || r.header.Get("Content-Security-Policy") == "" {
			t.Errorf("%s: %d %v", p, r.status, r.header)
		}
	}
}
