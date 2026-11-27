package api

import (
	"net/http"
	"strings"
)

// media streams a pack picture from the bucket. Keys carry the pack and its
// version, so a key's content never changes and phones may keep it a year.
func (a *API) media(w http.ResponseWriter, r *http.Request) {
	obj, info, err := a.core.Media(r.Context(), r.PathValue("key"))
	if err != nil {
		a.fail(w, err)
		return
	}
	defer obj.Close()
	h := w.Header()
	// Only pictures and clips: anything else the bucket holds must not render as a page on our origin.
	ct := info.ContentType
	if !strings.HasPrefix(ct, "image/") && !strings.HasPrefix(ct, "video/") || ct == "image/svg+xml" {
		ct = "application/octet-stream"
	}
	h.Set("Content-Type", ct)
	h.Set("Cache-Control", "private, max-age=31536000, immutable")
	if info.ETag != "" {
		h.Set("ETag", `"`+strings.Trim(info.ETag, `"`)+`"`)
	}
	http.ServeContent(w, r, "", info.ModTime, obj)
}
