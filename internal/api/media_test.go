package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMedia(t *testing.T) {
	a, m := newAPIMedia(t)
	me, _ := twoSessions(t, a)
	ctx := context.Background()
	_ = m.Put(ctx, "packs/p/1/squat-start.webp", strings.NewReader("RIFFwebp"), 8, "image/webp")
	_ = m.Put(ctx, "packs/p/1/evil.html", strings.NewReader("<script>"), 8, "text/html")
	_ = m.Put(ctx, "users/u/private.jpg", strings.NewReader("jpg"), 3, "image/jpeg")

	if w := do(a, "GET", "/api/media/packs/p/1/squat-start.webp", "", "", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("signed out: %d", w.Code)
	}
	w := do(a, "GET", "/api/media/packs/p/1/squat-start.webp", "", me, "")
	if w.Code != http.StatusOK || w.Body.String() != "RIFFwebp" || w.Header().Get("Content-Type") != "image/webp" ||
		!strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("get: %d %v %q", w.Code, w.Header(), w.Body)
	}
	etag := w.Header().Get("ETag")
	r := httptest.NewRequest("GET", "/api/media/packs/p/1/squat-start.webp", nil)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: me})
	r.Header.Set("If-None-Match", etag)
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, r)
	if etag == "" || rec.Code != http.StatusNotModified {
		t.Fatalf("revalidate with %q: %d", etag, rec.Code)
	}
	if w := do(a, "GET", "/api/media/packs/p/1/evil.html", "", me, ""); w.Header().Get("Content-Type") != "application/octet-stream" {
		t.Fatalf("non-image served as %q", w.Header().Get("Content-Type"))
	}
	for _, path := range []string{"/api/media/packs/p/1/missing.webp", "/api/media/users/u/private.jpg", "/api/media/"} {
		if w := do(a, "GET", path, "", me, ""); w.Code != http.StatusNotFound {
			t.Errorf("%s: %d %s", path, w.Code, w.Body)
		}
	}
	// ServeMux cleans dot segments with a redirect, to a key that is refused above.
	if w := do(a, "GET", "/api/media/packs/p/../../users/u/private.jpg", "", me, ""); w.Code == http.StatusOK {
		t.Fatalf("dot segments served: %s", w.Body)
	}
}
