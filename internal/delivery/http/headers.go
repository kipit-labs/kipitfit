package http

import (
	"net/http"
	"net/netip"
	"strings"
)

// csp allows only the app's own assets. Styles need 'unsafe-inline': Radix
// and sonner inject <style> elements at runtime.
const csp = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; font-src 'self' data:; connect-src 'self'; manifest-src 'self'; " +
	"object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"

// SecureHeaders sets browser hardening headers on every response, and HSTS
// when the app is served over HTTPS (secure cookies on).
func SecureHeaders(next http.Handler, https bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", csp)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if https {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}

// RealIP sets RemoteAddr to the client IP a trusted reverse proxy reports
// in header, so the login limiter sees the real client. The header is only
// read when the connection comes from one of trusted, so clients cannot pick
// their own IP. For X-Forwarded-For, the address taken is the rightmost one
// that is not itself a trusted proxy. Without a header, RemoteAddr stays the
// socket address.
func RealIP(header string, trusted []netip.Prefix) func(http.Handler) http.Handler {
	header = http.CanonicalHeaderKey(header)
	return func(next http.Handler) http.Handler {
		if header == "" {
			return next
		}
		isTrusted := func(a netip.Addr) bool {
			a = a.Unmap()
			for _, p := range trusted {
				if p.Contains(a) {
					return true
				}
			}
			return false
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if peer, err := netip.ParseAddrPort(r.RemoteAddr); err == nil && isTrusted(peer.Addr()) {
				if ip, ok := headerIP(r.Header.Values(header), header == "X-Forwarded-For", isTrusted); ok {
					r.RemoteAddr = ip.String()
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// headerIP reads the client IP from a header's values. A list header is
// walked from the right, past the trusted proxies that appended to it.
func headerIP(values []string, list bool, trusted func(netip.Addr) bool) (netip.Addr, bool) {
	if !list {
		if len(values) != 1 {
			return netip.Addr{}, false
		}
		a, err := netip.ParseAddr(strings.TrimSpace(values[0]))
		return a.Unmap(), err == nil
	}
	var hops []string
	for _, v := range values {
		hops = append(hops, strings.Split(v, ",")...)
	}
	for i := len(hops) - 1; i >= 0; i-- {
		a, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			return netip.Addr{}, false
		}
		if !trusted(a) {
			return a.Unmap(), true
		}
	}
	return netip.Addr{}, false
}
