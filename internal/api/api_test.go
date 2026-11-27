package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kipit-labs/kipitfit/internal/config"
	"github.com/kipit-labs/kipitfit/internal/domain"
	"github.com/kipit-labs/kipitfit/internal/exercisedb"
	"github.com/kipit-labs/kipitfit/internal/infrastructure/security"
	"github.com/kipit-labs/kipitfit/internal/media"
	"github.com/kipit-labs/kipitfit/internal/store/sqlite"
	"github.com/kipit-labs/kipitfit/internal/usecase/auth"
	"github.com/kipit-labs/kipitfit/internal/usecase/exercises"
	"github.com/kipit-labs/kipitfit/internal/usecase/library"
	"github.com/kipit-labs/kipitfit/internal/usecase/records"
	"github.com/kipit-labs/kipitfit/internal/usecase/routines"
	"github.com/kipit-labs/kipitfit/internal/usecase/workouts"
)

func newAPI(t *testing.T) *API {
	t.Helper()
	a, _ := newAPIMedia(t)
	return a
}

// newAPIMedia is newAPI with an in-memory media bucket.
func newAPIMedia(t *testing.T) (*API, *media.Memory) {
	t.Helper()
	st, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "kipitfit.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	cfg := config.Config{SetupToken: "setup", AppURL: "http://example.com"}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	m := media.NewMemory()
	authSvc := auth.NewService(st, security.NewBcryptHasher(0), cfg.SetupToken, cfg.AppURL, log)
	if err := authSvc.InitAuth(context.Background()); err != nil {
		t.Fatal(err)
	}
	ex := exercises.NewService(st)
	rec := records.NewService(st, ex)
	return New(Services{Auth: authSvc, Exercises: ex, Records: rec, Library: library.NewService(st, st, exercisedb.Dataset{}, m, log),
		Routines: routines.NewService(st, ex), Workouts: workouts.NewService(st, st, ex, rec)}, cfg, log), m
}

// do sends a request as from http://example.com (httptest's host).
func do(a *API, method, path, body, cookie, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	if cookie != "" {
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: cookie})
	}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}

func code(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var e apiError
	_ = json.Unmarshal(w.Body.Bytes(), &e)
	return e.Code
}

func session(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			return c.Value
		}
	}
	t.Fatalf("no session cookie: %d %s", w.Code, w.Body)
	return ""
}

func TestSameOrigin(t *testing.T) {
	a := newAPI(t)
	login := `{"email":"a@b.c","password":"whatever12"}`
	if w := do(a, "POST", "/api/auth/login", login, "", "https://evil.example"); w.Code != http.StatusForbidden || code(t, w) != "cross_origin" {
		t.Fatalf("cross-origin POST: %d %s", w.Code, w.Body)
	}
	if w := do(a, "POST", "/api/auth/login", login, "", "http://example.com"); w.Code == http.StatusForbidden {
		t.Fatalf("same-origin POST refused: %s", w.Body)
	}
	if w := do(a, "POST", "/api/auth/login", login, "", ""); w.Code == http.StatusForbidden {
		t.Fatalf("POST without Origin refused: %s", w.Body)
	}
	if w := do(a, "GET", "/api/health", "", "", "https://evil.example"); w.Code != http.StatusNoContent {
		t.Fatalf("cross-origin GET: %d", w.Code)
	}
}

func TestAuth(t *testing.T) {
	a := newAPI(t)
	for _, route := range []string{"GET /api/me", "GET /api/admin/invites", "GET /api/nope"} {
		method, path, _ := strings.Cut(route, " ")
		if w := do(a, method, path, "", "", ""); w.Code != http.StatusUnauthorized || code(t, w) != "signed_out" {
			t.Errorf("%s signed out: %d %s", route, w.Code, w.Body)
		}
	}
	if w := do(a, "GET", "/api/me", "", "forged", ""); w.Code != http.StatusUnauthorized {
		t.Errorf("forged session: %d", w.Code)
	}

	w := do(a, "POST", "/api/auth/setup", `{"setupToken":"setup","email":"admin@b.c","name":"Admin","password":"correct horse"}`, "", "")
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: %d %s", w.Code, w.Body)
	}
	admin := session(t, w)
	if w := do(a, "GET", "/api/me", "", admin, ""); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"isAdmin":true`) {
		t.Fatalf("me: %d %s", w.Code, w.Body)
	}
	if strings.Contains(w.Body.String(), "password") {
		t.Fatalf("me leaks the password hash: %s", w.Body)
	}

	w = do(a, "POST", "/api/admin/invites", `{"email":"friend@b.c"}`, admin, "")
	if w.Code != http.StatusCreated {
		t.Fatalf("invite: %d %s", w.Code, w.Body)
	}
	var inv struct{ Link string }
	_ = json.Unmarshal(w.Body.Bytes(), &inv)
	token := inv.Link[strings.Index(inv.Link, "token=")+len("token="):]
	w = do(a, "POST", "/api/invites/accept", `{"token":"`+token+`","name":"Friend","password":"correct horse"}`, "", "")
	if w.Code != http.StatusCreated {
		t.Fatalf("accept: %d %s", w.Code, w.Body)
	}
	friend := session(t, w)
	if w := do(a, "GET", "/api/admin/invites", "", friend, ""); w.Code != http.StatusForbidden || code(t, w) != "admin_only" {
		t.Fatalf("non-admin list invites: %d %s", w.Code, w.Body)
	}
	if w := do(a, "PATCH", "/api/me", `{"unit":"st"}`, friend, ""); w.Code != http.StatusBadRequest || code(t, w) != "invalid_unit" {
		t.Fatalf("bad unit: %d %s", w.Code, w.Body)
	}
	if w := do(a, "PATCH", "/api/me", `{"bogus":1}`, friend, ""); code(t, w) != "invalid_body" {
		t.Fatalf("unknown field: %d %s", w.Code, w.Body)
	}
}

func TestLoginLimiter(t *testing.T) {
	a := newAPI(t)
	var w *httptest.ResponseRecorder
	for range 11 {
		w = do(a, "POST", "/api/auth/login", `{"email":"a@b.c","password":"wrong password"}`, "", "")
	}
	if w.Code != http.StatusTooManyRequests || code(t, w) != "too_many_attempts" {
		t.Fatalf("11th failed login: %d %s", w.Code, w.Body)
	}
}

// TestErrorCodesTranslated checks every error code the API can answer with
// has a message in each web catalog.
func TestErrorCodesTranslated(t *testing.T) {
	for _, lang := range []string{"en", "fr"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "web", "src", "locales", lang+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var cat struct{ Errors map[string]string }
		if err := json.Unmarshal(raw, &cat); err != nil {
			t.Fatalf("%s.json: %v", lang, err)
		}
		for _, c := range append(append([]string{}, domain.Codes...), apiCodes...) {
			if cat.Errors[c] == "" {
				t.Errorf("%s.json: missing errors.%s", lang, c)
			}
		}
	}
}

// TestExerciseLabelsTranslated checks every exercise enum value the API can
// return has a label in each web catalog; the web looks them up dynamically.
func TestExerciseLabelsTranslated(t *testing.T) {
	enums := map[string][]string{"categories": domain.Categories, "equipment": domain.Equipment, "muscles": domain.Muscles, "levels": domain.Levels}
	for _, k := range domain.Kinds {
		enums["kinds"] = append(enums["kinds"], string(k))
	}
	for _, lang := range []string{"en", "fr"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "web", "src", "locales", lang+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var cat struct{ Exercises map[string]json.RawMessage }
		if err := json.Unmarshal(raw, &cat); err != nil {
			t.Fatalf("%s.json: %v", lang, err)
		}
		for group, values := range enums {
			var labels map[string]string
			_ = json.Unmarshal(cat.Exercises[group], &labels)
			for _, v := range values {
				if labels[v] == "" {
					t.Errorf("%s.json: missing exercises.%s.%s", lang, group, v)
				}
			}
		}
	}
}

func TestRealIP(t *testing.T) {
	cluster := []netip.Prefix{netip.MustParsePrefix("10.42.0.0/16")}
	tests := []struct {
		name    string
		header  string
		remote  string
		headers map[string][]string
		want    string
	}{
		{"no header configured keeps the socket", "", "1.2.3.4:5", map[string][]string{"CF-Connecting-IP": {"6.6.6.6"}}, "1.2.3.4:5"},
		{"trusted proxy's header", "CF-Connecting-IP", "10.42.0.5:1234", map[string][]string{"CF-Connecting-IP": {"1.1.1.1"}}, "1.1.1.1"},
		{"untrusted peer cannot pick its IP", "CF-Connecting-IP", "8.8.8.8:1234", map[string][]string{"CF-Connecting-IP": {"1.1.1.1"}}, "8.8.8.8:1234"},
		{"invalid header keeps the socket", "CF-Connecting-IP", "10.42.0.5:1234", map[string][]string{"CF-Connecting-IP": {"nope"}}, "10.42.0.5:1234"},
		{"repeated single-value header rejected", "X-Real-IP", "10.42.0.5:1234", map[string][]string{"X-Real-IP": {"1.1.1.1", "2.2.2.2"}}, "10.42.0.5:1234"},
		{"forwarded-for takes the hop before the proxies", "X-Forwarded-For", "10.42.0.5:1234",
			map[string][]string{"X-Forwarded-For": {"6.6.6.6, 1.1.1.1, 10.42.0.9"}}, "1.1.1.1"},
		{"forwarded-for with only proxies", "X-Forwarded-For", "10.42.0.5:1234",
			map[string][]string{"X-Forwarded-For": {"10.42.0.8"}}, "10.42.0.5:1234"},
		{"forwarded-for with garbage", "X-Forwarded-For", "10.42.0.5:1234",
			map[string][]string{"X-Forwarded-For": {"1.1.1.1, junk"}}, "10.42.0.5:1234"},
		{"IPv4-mapped peer is trusted", "CF-Connecting-IP", "[::ffff:10.42.0.5]:1234", map[string][]string{"CF-Connecting-IP": {"1.1.1.1"}}, "1.1.1.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tt.remote
			for k, vs := range tt.headers {
				for _, v := range vs {
					r.Header.Add(k, v)
				}
			}
			var got string
			RealIP(tt.header, cluster)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = r.RemoteAddr
			})).ServeHTTP(httptest.NewRecorder(), r)
			if got != tt.want {
				t.Errorf("RemoteAddr = %q, want %q", got, tt.want)
			}
		})
	}
}
