package api

import "net/http"

func (a *API) libraryInfo(w http.ResponseWriter, r *http.Request) {
	l, err := a.library.Library(r.Context())
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, l)
}
