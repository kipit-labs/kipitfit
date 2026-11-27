// Package exercisedb is kipitfit's exercise library: free-exercise-db
// (public domain, see NOTICE.md), trimmed to the fields kipitfit uses and
// embedded in the binary.
package exercisedb

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/kipit-labs/kipitfit/internal/store"
)

//go:embed exercises.json
var data []byte

// rules is bumped when Kind changes, so the library syncs again even though
// the data did not.
const rules = "1"

// Version identifies the embedded data and mapping rules.
func Version() string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8]) + "-" + rules
}

// Load returns the library, each exercise with its Kind.
func Load() ([]store.Exercise, error) {
	var rows []struct {
		Slug string `json:"slug"` // store.Exercise keeps its own Slug out of JSON
		store.Exercise
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	exs := make([]store.Exercise, len(rows))
	for i, r := range rows {
		r.Exercise.Slug = r.Slug
		r.Kind = Kind(r.Exercise)
		exs[i] = r.Exercise
	}
	return exs, nil
}

// Kind guesses what a set of a library exercise records, from its category,
// equipment, force and name; the dataset has no such field.
func Kind(e store.Exercise) store.Kind {
	bodyweight := e.Equipment == "body_only" || e.Equipment == "none"
	switch {
	case e.Category == "stretching":
		return store.KindDuration
	case e.Category == "cardio":
		return store.KindDistanceDuration
	case strings.Contains(e.Name, "Assisted"):
		return store.KindAssistedBodyweight
	// "Weighted Pull Ups" and "Weighted Bench Dip" list their belt or plate as "other".
	case strings.Contains(e.Name, "Weighted") && (bodyweight || e.Equipment == "other"):
		return store.KindWeightedBodyweight
	case bodyweight && e.Force == "static": // planks, side bridges, isometric holds
		return store.KindDuration
	case bodyweight:
		return store.KindRepsOnly
	default:
		return store.KindWeightReps
	}
}
