package domain

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Bounds shared by routines and workouts.
const (
	MaxTitle     = 100
	MaxNotes     = 1000
	MaxExercises = 50
	MaxSets      = 50
	// Bounds of a set value, far above any real one: 1 t, 10 000 reps, 24 h, 1000 km.
	MaxWeightG   = 1_000_000
	MaxReps      = 10_000
	MaxDurationS = 24 * 3600
	MaxDistanceM = 1_000_000
)

var (
	ErrTitle            = Coded(ErrInvalid, "title_invalid", fmt.Sprintf("the name must be 1 to %d characters", MaxTitle))
	ErrNotesTooLong     = Coded(ErrInvalid, "notes_too_long", fmt.Sprintf("notes can be at most %d characters", MaxNotes))
	ErrTooManyExercises = Coded(ErrInvalid, "too_many_exercises", fmt.Sprintf("at most %d exercises", MaxExercises))
	ErrTooManySets      = Coded(ErrInvalid, "too_many_sets", fmt.Sprintf("at most %d sets per exercise", MaxSets))
	ErrSetType          = Coded(ErrInvalid, "set_type_invalid", "a set type must be normal, warmup, drop or failure")
	ErrSetValue         = Coded(ErrInvalid, "set_value_invalid", "a set has a value out of range or that its exercise doesn't take")
	ErrSuperset         = Coded(ErrInvalid, "superset_invalid", "a superset is one letter, A to Z")
)

func init() {
	ErrTitle.Params = map[string]any{"max": MaxTitle}
	ErrNotesTooLong.Params = map[string]any{"max": MaxNotes}
	ErrTooManyExercises.Params = map[string]any{"max": MaxExercises}
	ErrTooManySets.Params = map[string]any{"max": MaxSets}
}

// Title is a routine, folder or workout name with its spaces tidied.
func Title(s string) (string, error) {
	s = strings.Join(strings.Fields(s), " ")
	if n := utf8.RuneCountInString(s); n == 0 || n > MaxTitle {
		return "", ErrTitle
	}
	return s, nil
}

// ValidSuperset reports whether s is a superset letter, A to Z, or "" for none.
func ValidSuperset(s string) bool {
	return s == "" || len(s) == 1 && s[0] >= 'A' && s[0] <= 'Z'
}

// SetFields are the values a set of an exercise kind takes.
type SetFields struct{ Weight, Reps, Duration, Distance bool }

func (k Kind) Fields() SetFields {
	switch k {
	case KindWeightReps, KindWeightedBodyweight, KindAssistedBodyweight:
		return SetFields{Weight: true, Reps: true}
	case KindRepsOnly:
		return SetFields{Reps: true}
	case KindDuration:
		return SetFields{Duration: true}
	case KindDistanceDuration:
		return SetFields{Duration: true, Distance: true}
	}
	return SetFields{}
}

// InRange reports whether v is unset, or allowed and within 0..maxV.
func InRange(v *int, allowed bool, maxV int) bool {
	return v == nil || (allowed && *v >= 0 && *v <= maxV)
}
