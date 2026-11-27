package domain

import "testing"

func TestValidMediaKey(t *testing.T) {
	for key, want := range map[string]bool{
		"packs/repdb-free/20261114T121104Z/squat-start.webp": true,
		"packs/a/b.webp":    true,
		"packs/../secret":   false,
		"packs/a/./b":       false,
		"packs//a":          false,
		"packs/a/":          false,
		"users/1/photo.jpg": false,
		"packs/a b.webp":    false,
		"packs/a%2F..%2Fb":  false,
		"":                  false,
		"/packs/a.webp":     false,
		"packs/é.webp":      false,
	} {
		if got := ValidMediaKey(key); got != want {
			t.Errorf("ValidMediaKey(%q) = %v, want %v", key, got, want)
		}
	}
}
