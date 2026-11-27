// Package pack reads exercise packs: an exercise list with its pictures, from
// a vendor's ZIP. kipitfit ships no pack (their licences forbid republishing
// them); each deployment imports its own.
package pack

import (
	"archive/zip"
	"io"
	"mime"
	"path"
	"strings"

	"github.com/kipit-labs/kipitfit/internal/store"
)

// File is one picture of an exercise, in the ZIP.
type File struct {
	// Key is where it goes in the bucket: packs/<pack>/<version>/<name>.
	Key         string
	ContentType string
	Size        int64
	Open        func() (io.ReadCloser, error)
}

// Exercise is a library exercise (Slug "<vendor>:<id>") and its pictures in
// frame order; Media is filled in from Files on import.
type Exercise struct {
	store.Exercise
	Files []File
}

type Pack struct {
	store.Pack
	Exercises []Exercise
}

// Adapter reads one vendor's ZIP. URL is where to fetch it when no file is
// given ("" when it must be bought).
type Adapter struct {
	URL  string
	Read func(z *zip.Reader) (Pack, error)
}

// Adapters by pack name. Paid RepDB and ExerciseDB adapters will be written
// from their preview bundles; slugs stay "<vendor>:<id>" so moving from the
// free to the paid RepDB keeps each exercise's id.
var Adapters = map[string]Adapter{
	"repdb-free": {URL: "https://cdn.repdb.co/repdb-assets/site/repdb-free.zip", Read: readRepDBFree},
}

// file finds name in z and describes it for the bucket.
func file(z *zip.Reader, name, key string) (File, bool) {
	for _, f := range z.File {
		if f.Name == name {
			ct := mime.TypeByExtension(path.Ext(name))
			if ct == "" {
				ct = "application/octet-stream"
			}
			return File{Key: key, ContentType: ct, Size: int64(f.UncompressedSize64), Open: f.Open}, true
		}
	}
	return File{}, false
}

// keySafe keeps a version or id usable in a bucket key (see media.ValidKey).
func keySafe(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			return r
		}
		return -1
	}, s)
}
