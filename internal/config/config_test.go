package config

import (
	"net/netip"
	"slices"
	"testing"
)

func TestPrefixes(t *testing.T) {
	got := prefixes("10.42.0.0/16, 127.0.0.1,junk, ::1")
	want := []netip.Prefix{
		netip.MustParsePrefix("10.42.0.0/16"),
		netip.MustParsePrefix("127.0.0.1/32"),
		netip.MustParsePrefix("::1/128"),
	}
	if !slices.Equal(got, want) {
		t.Errorf("prefixes = %v, want %v", got, want)
	}
}

func TestLoad(t *testing.T) {
	t.Setenv("KIPITFIT_ADDR", ":9090")
	c, err := Load()
	if err != nil || c.AppURL != "http://localhost:9090" {
		t.Errorf("AppURL = %q (%v), want the listen port on localhost", c.AppURL, err)
	}
	t.Setenv("KIPITFIT_REAL_IP_HEADER", "CF-Connecting-IP")
	if _, err := Load(); err == nil {
		t.Error("a real IP header without trusted proxies was accepted")
	}
}
