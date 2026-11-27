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

	"github.com/kipit-labs/kipitfit/internal/domain"
)

//go:embed exercises.json
var data []byte

// rules is bumped when Kind changes, so the library syncs again even though
// the data did not.
const rules = "1"

// Dataset is the embedded library, the domain.ExerciseDataset.
type Dataset struct{}

// Version identifies the embedded data and mapping rules.
func (Dataset) Version() string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8]) + "-" + rules
}

// Load returns the library, each exercise with its Kind.
func (Dataset) Load() ([]domain.Exercise, error) {
	var rows []struct {
		Slug string `json:"slug"` // domain.Exercise keeps its own Slug out of JSON
		domain.Exercise
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	exs := make([]domain.Exercise, len(rows))
	for i, r := range rows {
		r.Exercise.Slug = r.Slug
		r.Kind = Kind(r.Exercise)
		exs[i] = r.Exercise
	}
	return exs, nil
}

// Kind guesses what a set of a library exercise records, from its category,
// equipment, force and name; the dataset has no such field.
func Kind(e domain.Exercise) domain.Kind {
	bodyweight := e.Equipment == "body_only" || e.Equipment == "none"
	switch {
	case e.Category == "stretching":
		return domain.KindDuration
	case e.Category == "cardio":
		return domain.KindDistanceDuration
	case strings.Contains(e.Name, "Assisted"):
		return domain.KindAssistedBodyweight
	// "Weighted Pull Ups" and "Weighted Bench Dip" list their belt or plate as "other".
	case strings.Contains(e.Name, "Weighted") && (bodyweight || e.Equipment == "other"):
		return domain.KindWeightedBodyweight
	case bodyweight && e.Force == "static": // planks, side bridges, isometric holds
		return domain.KindDuration
	case bodyweight:
		return domain.KindRepsOnly
	default:
		return domain.KindWeightReps
	}
}
