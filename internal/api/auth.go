package api

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/kipit-labs/kipitfit/internal/core"
)

const sessionCookie = "kipitfit_session"

// publicRoutes are reachable without a session.
var publicRoutes = map[string]bool{
	"GET /api/health":          true,
	"GET /api/auth/state":      true,
	"POST /api/auth/setup":     true,
	"POST /api/auth/login":     true,
	"POST /api/auth/logout":    true,
	"POST /api/invites/accept": true, // the invite token is the credential
}

// protect refuses cross-origin mutations and requires a session on every
// route but publicRoutes.
func (a *API) protect(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			writeError(w, http.StatusForbidden, "cross_origin", "cross-origin request refused", nil)
			return
		}
		if _, pattern := mux.Handler(r); publicRoutes[pattern] {
			mux.ServeHTTP(w, r)
			return
		}
		u, err := a.core.Authenticate(r.Context(), sessionToken(r))
		if err != nil {
			a.fail(w, err)
			return
		}
		mux.ServeHTTP(w, r.WithContext(withUser(r.Context(), u)))
	})
}

// sameOrigin rejects state-changing requests whose Origin doesn't match the
// host. SameSite=Lax cookies already block most CSRF; this closes the rest.
func sameOrigin(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // non-browser client
	}
	u, err := url.Parse(origin)
	return err == nil && u.Host == r.Host
}

func sessionToken(r *http.Request) string {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return ""
	}
	return c.Value
}

func (a *API) setSessionCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   a.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *API) authState(w http.ResponseWriter, r *http.Request) {
	st, err := a.core.AuthState(r.Context(), sessionToken(r))
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// limited runs a credential check under the per-IP failure limiter: check
// returns the session token to set, and its failures count against the IP.
func (a *API) limited(w http.ResponseWriter, r *http.Request, body any, check func() (any, string, error), status int) {
	ip := clientIP(r)
	if !a.limiter.allow(r.Context(), ip) {
		writeError(w, http.StatusTooManyRequests, "too_many_attempts", "too many attempts, try again later", nil)
		return
	}
	if !decode(w, r, body) {
		return
	}
	v, token, err := check()
	if err != nil {
		a.limiter.fail(r.Context(), ip)
		a.fail(w, err)
		return
	}
	a.limiter.reset(ip)
	a.setSessionCookie(w, token, int(core.SessionTTL.Seconds()))
	writeJSON(w, status, v)
}

func (a *API) setup(w http.ResponseWriter, r *http.Request) {
	var b struct {
		SetupToken string `json:"setupToken"`
		Email      string `json:"email"`
		Name       string `json:"name"`
		Password   string `json:"password"`
	}
	a.limited(w, r, &b, func() (any, string, error) {
		return a.core.Setup(r.Context(), b.SetupToken, b.Email, b.Name, b.Password)
	}, http.StatusCreated)
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	a.limited(w, r, &b, func() (any, string, error) {
		return a.core.Login(r.Context(), b.Email, b.Password)
	}, http.StatusOK)
}

func (a *API) acceptInvite(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Token    string `json:"token"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	a.limited(w, r, &b, func() (any, string, error) {
		return a.core.AcceptInvite(r.Context(), b.Token, b.Name, b.Password)
	}, http.StatusCreated)
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	if err := a.core.Logout(r.Context(), sessionToken(r)); err != nil {
		a.fail(w, err)
		return
	}
	a.setSessionCookie(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

// clientIP is the address failures count against; RealIP already replaced
// RemoteAddr with the client's when a trusted proxy sent the request.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// maxTrackedIPs caps how many addresses a limiter remembers: entries only
// expire when their IP comes back, so a caller rotating addresses (IPv6)
// would otherwise grow the map forever.
const maxTrackedIPs = 10_000

// failLimiter blocks an IP after too many failed attempts within a window.
// Attempts still running count too: concurrent requests would otherwise all
// pass the check before any of them fails.
type failLimiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	fails  map[string][]time.Time
	busy   map[string]map[context.Context]bool // attempts running, by request
}

func newFailLimiter(maxFails int, window time.Duration) *failLimiter {
	return &failLimiter{max: maxFails, window: window, fails: map[string][]time.Time{}, busy: map[string]map[context.Context]bool{}}
}

// allow reports whether ip may make an attempt, which then counts until it
// fails or ctx (the request's) ends.
func (l *failLimiter) allow(ctx context.Context, ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.recent(ip))+len(l.busy[ip]) >= l.max {
		return false
	}
	if l.busy[ip] == nil {
		l.busy[ip] = map[context.Context]bool{}
	}
	l.busy[ip][ctx] = true
	context.AfterFunc(ctx, func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.done(ctx, ip)
	})
	return true
}

// fail records ctx's attempt as failed.
func (l *failLimiter) fail(ctx context.Context, ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.done(ctx, ip)
	if _, ok := l.fails[ip]; !ok && len(l.fails) >= maxTrackedIPs {
		l.evict()
	}
	l.fails[ip] = append(l.recent(ip), time.Now())
}

// done stops counting ctx's attempt as running; callers hold mu.
func (l *failLimiter) done(ctx context.Context, ip string) {
	if delete(l.busy[ip], ctx); len(l.busy[ip]) == 0 {
		delete(l.busy, ip)
	}
}

// evict drops every expired entry, then arbitrary ones if the map is still
// full; callers hold mu.
func (l *failLimiter) evict() {
	for ip := range l.fails {
		l.recent(ip)
	}
	for ip := range l.fails {
		if len(l.fails) < maxTrackedIPs {
			break
		}
		delete(l.fails, ip)
	}
}

func (l *failLimiter) reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.fails, ip)
}

// recent drops expired entries; callers hold mu.
func (l *failLimiter) recent(ip string) []time.Time {
	cutoff := time.Now().Add(-l.window)
	ts := l.fails[ip]
	i := 0
	for i < len(ts) && ts[i].Before(cutoff) {
		i++
	}
	ts = ts[i:]
	if len(ts) == 0 {
		delete(l.fails, ip)
	} else {
		l.fails[ip] = ts
	}
	return ts
}
