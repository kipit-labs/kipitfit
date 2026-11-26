// Package config reads kipitfit's settings from KIPITFIT_* environment variables.
package config

import (
	"errors"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	// Addr is the HTTP listen address for the app and API.
	Addr string
	// DataDir holds the SQLite database. Must be on a local disk (never NFS/SMB).
	DataDir string
	// LogLevel is one of debug, info, warn, error.
	LogLevel string
	// SetupToken, if set, replaces the random token printed on first start
	// that is required to create the admin account.
	SetupToken string
	// AppURL is the public URL, used in invite links.
	AppURL string
	// CookieSecure marks the session cookie Secure and turns on HSTS: set it
	// behind HTTPS.
	CookieSecure bool
	// RealIPHeader is the header a reverse proxy puts the client IP in, e.g.
	// CF-Connecting-IP; read only on requests from TrustedProxies.
	RealIPHeader   string
	TrustedProxies []netip.Prefix

	// Version is the running build (set by main, not the environment).
	Version string
}

func Load() (Config, error) {
	addr := env("KIPITFIT_ADDR", ":8080")
	_, port, _ := net.SplitHostPort(addr)
	c := Config{
		Addr:           addr,
		DataDir:        env("KIPITFIT_DATA_DIR", "/data"),
		LogLevel:       env("KIPITFIT_LOG_LEVEL", "info"),
		SetupToken:     env("KIPITFIT_SETUP_TOKEN", ""),
		AppURL:         strings.TrimRight(env("KIPITFIT_APP_URL", "http://localhost:"+port), "/"),
		CookieSecure:   env("KIPITFIT_COOKIE_SECURE", "false") == "true",
		RealIPHeader:   strings.TrimSpace(env("KIPITFIT_REAL_IP_HEADER", "")),
		TrustedProxies: prefixes(env("KIPITFIT_TRUSTED_PROXIES", "")),
	}
	if c.RealIPHeader != "" && len(c.TrustedProxies) == 0 {
		return c, errors.New("KIPITFIT_REAL_IP_HEADER needs KIPITFIT_TRUSTED_PROXIES: the proxies allowed to set it")
	}
	return c, nil
}

func (c Config) DBPath() string {
	return filepath.Join(c.DataDir, "kipitfit.db")
}

// prefixes parses a comma-separated list of CIDRs or addresses, skipping
// invalid entries.
func prefixes(list string) []netip.Prefix {
	var out []netip.Prefix
	for _, f := range strings.Split(list, ",") {
		f = strings.TrimSpace(f)
		if p, err := netip.ParsePrefix(f); err == nil {
			out = append(out, p.Masked())
		} else if a, err := netip.ParseAddr(f); err == nil {
			out = append(out, netip.PrefixFrom(a.Unmap(), a.Unmap().BitLen()))
		}
	}
	return out
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
