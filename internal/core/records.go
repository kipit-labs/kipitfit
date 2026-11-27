package core

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/kipit-labs/kipitfit/internal/store"
)

// RecordKind names a best an exercise can have. Values are grams, reps,
// seconds or metres, as the kind says.
type RecordKind string

const (
	// RecordE1RM is the best estimated one-rep max (Epley), from a set of 1 to 12 reps.
	RecordE1RM RecordKind = "e1rm"
	// RecordWeight is the heaviest weight lifted (added weight for weighted bodyweight).
	RecordWeight RecordKind = "weight"
	// RecordSetVolume is the best weight × reps in one set.
	RecordSetVolume RecordKind = "set_volume"
	// RecordSessionVolume is the best weight × reps over one workout.
	RecordSessionVolume RecordKind = "session_volume"
	// RecordReps is the most reps in one set.
	RecordReps RecordKind = "reps"
	// RecordSessionReps is the most reps over one workout.
	RecordSessionReps RecordKind = "session_reps"
	// RecordDuration is the longest set.
	RecordDuration RecordKind = "duration"
	// RecordDistance is the longest distance in one set.
	RecordDistance RecordKind = "distance"
)

// e1rmMaxReps is the most reps a set may have to estimate a one-rep max; more
// reps say little about it, so such a set counts for rep records only.
const e1rmMaxReps = 12

// RecordKinds lists the records an exercise kind keeps, in display order.
// Assisted bodyweight keeps reps only: less assistance is progress, more reps too, and the two don't add up.
func RecordKinds(k store.Kind) []RecordKind {
	switch k {
	case store.KindWeightReps:
		return []RecordKind{RecordE1RM, RecordWeight, RecordSetVolume, RecordSessionVolume, RecordReps}
	case store.KindWeightedBodyweight:
		// No e1RM: the bodyweight isn't known until measurements (P7), so the added weight alone says little.
		return []RecordKind{RecordWeight, RecordSetVolume, RecordSessionVolume, RecordReps}
	case store.KindRepsOnly, store.KindAssistedBodyweight:
		return []RecordKind{RecordReps, RecordSessionReps}
	case store.KindDuration:
		return []RecordKind{RecordDuration}
	case store.KindDistanceDuration:
		return []RecordKind{RecordDistance, RecordDuration}
	}
	return nil
}

// E1RM is Epley's estimated one-rep max, w × (1 + reps/30), in grams rounded
// half up; ok is false outside 1 to 12 reps.
func E1RM(weightG, reps int) (int, bool) {
	if reps < 1 || reps > e1rmMaxReps || weightG <= 0 {
		return 0, false
	}
	return (weightG*(30+reps) + 15) / 30, true
}

// setValue is what one set counts for a per-set record kind; ok is false when
// it doesn't count (warm-ups never do).
func setValue(rk RecordKind, s store.LoggedSet) (int, bool) {
	if s.Type == store.SetWarmup {
		return 0, false
	}
	switch rk {
	case RecordE1RM:
		if s.WeightG != nil && s.Reps != nil {
			return E1RM(*s.WeightG, *s.Reps)
		}
	case RecordWeight:
		if s.WeightG != nil && *s.WeightG > 0 {
			return *s.WeightG, true
		}
	case RecordSetVolume, RecordSessionVolume:
		if s.WeightG != nil && s.Reps != nil && *s.WeightG > 0 && *s.Reps > 0 {
			return *s.WeightG * *s.Reps, true
		}
	case RecordReps, RecordSessionReps:
		if s.Reps != nil && *s.Reps > 0 {
			return *s.Reps, true
		}
	case RecordDuration:
		if s.DurationS != nil && *s.DurationS > 0 {
			return *s.DurationS, true
		}
	case RecordDistance:
		if s.DistanceM != nil && *s.DistanceM > 0 {
			return *s.DistanceM, true
		}
	}
	return 0, false
}

// summed are the kinds a workout's sets add up to, rather than its best set.
func summed(rk RecordKind) bool { return rk == RecordSessionVolume || rk == RecordSessionReps }

// Record is a best of an exercise and where it was set. Exercise and Set place
// the set in its workout (Set is -1 for a workout's total); its values are
// copied for display ("100 kg × 5").
type Record struct {
	ExerciseID string     `json:"exerciseId,omitempty"`
	Kind       RecordKind `json:"kind"`
	Value      int        `json:"value"`
	// Previous is the best a workout beat, 0 for current bests.
	Previous  int       `json:"previous,omitempty"`
	WorkoutID string    `json:"workoutId"`
	StartedAt time.Time `json:"startedAt"`
	Exercise  int       `json:"exercise"`
	Set       int       `json:"set"`
	WeightG   *int      `json:"weightG,omitempty"`
	Reps      *int      `json:"reps,omitempty"`
	DurationS *int      `json:"durationS,omitempty"`
	DistanceM *int      `json:"distanceM,omitempty"`
}

// RepRecord is the heaviest weight lifted for at least Reps reps.
type RepRecord struct {
	Reps      int       `json:"reps"`
	WeightG   int       `json:"weightG"`
	WorkoutID string    `json:"workoutId"`
	StartedAt time.Time `json:"startedAt"`
}

// Session is an exercise in one workout: its sets, its values for each record
// kind (the chart's points) and the records it beat.
type Session struct {
	WorkoutID string             `json:"workoutId"`
	Title     string             `json:"title"`
	StartedAt time.Time          `json:"startedAt"`
	Sets      []store.WorkoutSet `json:"sets"`
	Best      map[RecordKind]int `json:"best"`
	Records   []Record           `json:"records"`
	rows      []store.LoggedSet
}

// history is everything known about one exercise, worked out from its sets.
type history struct {
	sessions []Session
	bests    map[RecordKind]Record
	reps     []RepRecord
}

// analyze replays an exercise's sets in order (rows of one exercise, oldest
// workout first). A workout beats a record when its best for a kind is above
// every earlier workout's; the first workout with a value sets none, else every
// first workout would be all records. Within a workout only its best set holds
// the record, the earliest on a tie.
func analyze(k store.Kind, rows []store.LoggedSet) history {
	kinds := RecordKinds(k)
	h := history{bests: map[RecordKind]Record{}}
	for start := 0; start < len(rows); {
		end := start
		for end < len(rows) && rows[end].WorkoutID == rows[start].WorkoutID {
			end++
		}
		s := Session{WorkoutID: rows[start].WorkoutID, Title: rows[start].Title, StartedAt: rows[start].StartedAt,
			Best: map[RecordKind]int{}, Records: []Record{}, rows: rows[start:end]}
		for _, r := range s.rows {
			s.Sets = append(s.Sets, store.WorkoutSet{Type: r.Type, WeightG: r.WeightG, Reps: r.Reps, RPE: r.RPE, DurationS: r.DurationS, DistanceM: r.DistanceM})
		}
		for _, rk := range kinds {
			best, ok := sessionBest(rk, s.rows)
			if !ok {
				continue
			}
			s.Best[rk] = best.Value
			prev, had := h.bests[rk]
			if had && best.Value > prev.Value {
				beat := best
				beat.Previous = prev.Value
				s.Records = append(s.Records, beat)
			}
			if !had || best.Value > prev.Value {
				h.bests[rk] = best
			}
		}
		h.sessions = append(h.sessions, s)
		start = end
	}
	if slices.Contains(kinds, RecordWeight) {
		h.reps = repRecords(rows)
	}
	return h
}

// sessionBest is one workout's value for a record kind: its best set, or the
// sum of its sets for a workout total.
func sessionBest(rk RecordKind, rows []store.LoggedSet) (Record, bool) {
	var best Record
	found := false
	for _, r := range rows {
		v, ok := setValue(rk, r)
		if !ok {
			continue
		}
		if summed(rk) {
			if !found {
				best = record(rk, 0, r)
				best.Set = -1
				best.WeightG, best.Reps, best.DurationS, best.DistanceM = nil, nil, nil, nil
			}
			best.Value += v
			found = true
		} else if !found || v > best.Value {
			best, found = record(rk, v, r), true
		}
	}
	return best, found
}

func record(rk RecordKind, v int, r store.LoggedSet) Record {
	return Record{Kind: rk, Value: v, WorkoutID: r.WorkoutID, StartedAt: r.StartedAt, Exercise: r.Exercise, Set: r.Set,
		WeightG: r.WeightG, Reps: r.Reps, DurationS: r.DurationS, DistanceM: r.DistanceM}
}

// repRecords are the heaviest weights lifted for at least 1 to 12 reps, the
// first time each was reached; reps never reached are left out.
func repRecords(rows []store.LoggedSet) []RepRecord {
	out := []RepRecord{}
	for n := 1; n <= e1rmMaxReps; n++ {
		var best *RepRecord
		for _, r := range rows {
			if r.Type == store.SetWarmup || r.WeightG == nil || r.Reps == nil || *r.WeightG <= 0 || *r.Reps < n {
				continue
			}
			if best == nil || *r.WeightG > best.WeightG {
				best = &RepRecord{Reps: n, WeightG: *r.WeightG, WorkoutID: r.WorkoutID, StartedAt: r.StartedAt}
			}
		}
		if best != nil {
			out = append(out, *best)
		}
	}
	return out
}

// currentBests lists h's bests in the kind's order.
func (h history) currentBests(k store.Kind) []Record {
	out := []Record{}
	for _, rk := range RecordKinds(k) {
		if r, ok := h.bests[rk]; ok {
			out = append(out, r)
		}
	}
	return out
}

// byExercise splits sets by exercise, keeping their order.
func byExercise(rows []store.LoggedSet) map[string][]store.LoggedSet {
	out := map[string][]store.LoggedSet{}
	for _, r := range rows {
		out[r.ExerciseID] = append(out[r.ExerciseID], r)
	}
	return out
}

// ExerciseProgress is an exercise's records and history, for its page.
type ExerciseProgress struct {
	Records    []Record    `json:"records"`
	RepRecords []RepRecord `json:"repRecords"`
	// Sessions are newest first.
	Sessions []Session `json:"sessions"`
}

// ExerciseProgress works out me's records and history of an exercise, archived
// ones included.
// shortcut: replays every set of the exercise on each call (ms for years of training); keep a records table if it shows.
func (c *Core) ExerciseProgress(ctx context.Context, me store.User, id string) (ExerciseProgress, error) {
	e, err := c.GetExercise(ctx, me, id)
	if err != nil {
		return ExerciseProgress{}, err
	}
	rows, err := c.store.ExerciseSets(ctx, me.ID, []string{id})
	if err != nil {
		return ExerciseProgress{}, err
	}
	h := analyze(e.Kind, rows)
	slices.Reverse(h.sessions)
	if h.sessions == nil {
		h.sessions = []Session{}
	}
	return ExerciseProgress{Records: h.currentBests(e.Kind), RepRecords: h.reps, Sessions: h.sessions}, nil
}

// Bests returns me's current best of each record kind per exercise, for the
// phone to spot records while logging; exercises never done are left out.
func (c *Core) Bests(ctx context.Context, me store.User, exerciseIDs []string) (map[string]map[RecordKind]int, error) {
	if len(exerciseIDs) > maxExercises {
		return nil, fmt.Errorf("%w: at most %d exercises", ErrInvalid, maxExercises)
	}
	h, err := c.histories(ctx, me, exerciseIDs)
	if err != nil {
		return nil, err
	}
	out := map[string]map[RecordKind]int{}
	for id, hh := range h {
		if len(hh.bests) == 0 {
			continue
		}
		out[id] = map[RecordKind]int{}
		for rk, r := range hh.bests {
			out[id][rk] = r.Value
		}
	}
	return out, nil
}

// histories analyses each of the exercises me can see; unknown ones are skipped.
func (c *Core) histories(ctx context.Context, me store.User, exerciseIDs []string) (map[string]history, error) {
	rows, err := c.store.ExerciseSets(ctx, me.ID, exerciseIDs)
	if err != nil {
		return nil, err
	}
	out := map[string]history{}
	for id, rs := range byExercise(rows) {
		e, err := c.store.GetExercise(ctx, id)
		if err != nil {
			return nil, err
		}
		out[id] = analyze(e.Kind, rs)
	}
	return out, nil
}

// WorkoutView is a workout with the records it beat.
type WorkoutView struct {
	store.Workout
	Records []Record `json:"records"`
}

// withRecords adds to each workout the records it beat.
func (c *Core) withRecords(ctx context.Context, me store.User, ws []store.Workout) ([]WorkoutView, error) {
	var exIDs []string
	for _, w := range ws {
		for _, we := range w.Exercises {
			if !slices.Contains(exIDs, we.ExerciseID) {
				exIDs = append(exIDs, we.ExerciseID)
			}
		}
	}
	h, err := c.histories(ctx, me, exIDs)
	if err != nil {
		return nil, err
	}
	beaten := map[string][]Record{}
	for _, id := range exIDs {
		for _, s := range h[id].sessions {
			for _, r := range s.Records {
				r.ExerciseID = id
				beaten[s.WorkoutID] = append(beaten[s.WorkoutID], r)
			}
		}
	}
	out := make([]WorkoutView, len(ws))
	for i, w := range ws {
		rs := beaten[w.ID]
		// In workout order: exercise, then set (a workout total after its sets).
		slices.SortStableFunc(rs, func(a, b Record) int {
			if a.Exercise != b.Exercise {
				return a.Exercise - b.Exercise
			}
			return setOrder(a.Set) - setOrder(b.Set)
		})
		if rs == nil {
			rs = []Record{}
		}
		out[i] = WorkoutView{Workout: w, Records: rs}
	}
	return out, nil
}

func setOrder(set int) int {
	if set < 0 {
		return 1 << 30
	}
	return set
}
