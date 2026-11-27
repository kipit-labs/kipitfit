package domain

import (
	"context"
	"io"
	"strings"
	"time"
)

// MediaInfo describes a stored picture.
type MediaInfo struct {
	Size        int64
	ContentType string
	ETag        string
	ModTime     time.Time
}

// MediaStore is the bucket exercise pictures live in. Objects are seekable, so
// the API can answer range requests.
type MediaStore interface {
	// Stat returns ErrNotFound when key isn't there.
	Stat(ctx context.Context, key string) (MediaInfo, error)
	Get(ctx context.Context, key string) (io.ReadSeekCloser, MediaInfo, error)
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
}

// ValidMediaKey reports whether key may be served: pack media only (never
// other things the bucket may hold later), no empty or dot segments.
func ValidMediaKey(key string) bool {
	if !strings.HasPrefix(key, "packs/") || len(key) > 512 {
		return false
	}
	for _, seg := range strings.Split(key, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
		for _, r := range seg {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != '.' {
				return false
			}
		}
	}
	return true
}

// ParsedPack is an exercise pack read from its vendor's ZIP: the exercises
// that replace the library, and their pictures.
type ParsedPack struct {
	Pack
	Exercises []PackExercise
}

// PackExercise is a library exercise (Slug "<vendor>:<id>") and its pictures
// in frame order; Media is filled in from Files on import.
type PackExercise struct {
	Exercise
	Files []PackFile
}

// PackFile is one picture of a pack exercise.
type PackFile struct {
	// Key is where it goes in the bucket: packs/<pack>/<version>/<name>.
	Key         string
	ContentType string
	Size        int64
	Open        func() (io.ReadCloser, error)
}
