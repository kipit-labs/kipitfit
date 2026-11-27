// Package pack reads exercise packs: an exercise list with its pictures, from
// a vendor's ZIP. kipitfit ships no pack (their licences forbid republishing
// them); each deployment imports its own.
package pack

import (
	"archive/zip"
	"mime"
	"path"
	"strings"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

// Adapter reads one vendor's ZIP. URL is where to fetch it when no file is
// given ("" when it must be bought).
type Adapter struct {
	URL  string
	Read func(z *zip.Reader) (domain.ParsedPack, error)
}

// Adapters by pack name. Paid RepDB and ExerciseDB adapters will be written
// from their preview bundles; slugs stay "<vendor>:<id>" so moving from the
// free to the paid RepDB keeps each exercise's id.
var Adapters = map[string]Adapter{
	"repdb-free": {URL: "https://cdn.repdb.co/repdb-assets/site/repdb-free.zip", Read: readRepDBFree},
}

// file finds name in z and describes it for the bucket.
func file(z *zip.Reader, name, key string) (domain.PackFile, bool) {
	for _, f := range z.File {
		if f.Name == name {
			ct := mime.TypeByExtension(path.Ext(name))
			if ct == "" {
				ct = "application/octet-stream"
			}
			return domain.PackFile{Key: key, ContentType: ct, Size: int64(f.UncompressedSize64), Open: f.Open}, true
		}
	}
	return domain.PackFile{}, false
}

// keySafe keeps a version or id usable in a bucket key (see domain.ValidMediaKey).
func keySafe(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			return r
		}
		return -1
	}, s)
}
