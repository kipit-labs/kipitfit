package exercises

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

func ptr[T any](v T) *T { return &v }

// fakeExercises keeps exercises in memory, like the SQLite store: custom
// names are unique per owner (ASCII case ignored), lists are sorted by name.
type fakeExercises struct {
	byID map[string]domain.Exercise
	n    int
}

// newFakeExercises holds a library of the given names, all weight × reps.
func newFakeExercises(library ...string) *fakeExercises {
	f := &fakeExercises{byID: map[string]domain.Exercise{}}
	for _, name := range library {
		f.n++
		id := fmt.Sprintf("lib-%d", f.n)
		f.byID[id] = domain.Exercise{ID: id, Slug: id, Name: name, Kind: domain.KindWeightReps, Category: "strength",
			Equipment: "barbell", PrimaryMuscles: []string{"chest"}, SecondaryMuscles: []string{}}
	}
	return f
}

func (f *fakeExercises) ListExercises(_ context.Context, ownerID string) ([]domain.Exercise, error) {
	out := []domain.Exercise{}
	for _, e := range f.byID {
		if (e.OwnerID == "" || e.OwnerID == ownerID) && !e.Archived {
			out = append(out, e)
		}
	}
	slices.SortFunc(out, func(a, b domain.Exercise) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return out, nil
}

func (f *fakeExercises) GetExercise(_ context.Context, id string) (domain.Exercise, error) {
	e, ok := f.byID[id]
	if !ok {
		return domain.Exercise{}, domain.ErrNotFound
	}
	return e, nil
}

func (f *fakeExercises) taken(e domain.Exercise) bool {
	for _, o := range f.byID {
		if o.ID != e.ID && o.OwnerID == e.OwnerID && strings.EqualFold(o.Name, e.Name) {
			return true
		}
	}
	return false
}

func (f *fakeExercises) CreateExercise(_ context.Context, e domain.Exercise) (domain.Exercise, error) {
	if f.taken(e) {
		return domain.Exercise{}, domain.ErrConflict
	}
	f.n++
	e.ID, e.Custom = fmt.Sprintf("custom-%d", f.n), true
	f.byID[e.ID] = e
	return e, nil
}

func (f *fakeExercises) UpdateExercise(_ context.Context, e domain.Exercise) (domain.Exercise, error) {
	if _, ok := f.byID[e.ID]; !ok {
		return domain.Exercise{}, domain.ErrNotFound
	}
	if f.taken(e) {
		return domain.Exercise{}, domain.ErrConflict
	}
	f.byID[e.ID] = e
	return e, nil
}

func (f *fakeExercises) DeleteExercise(_ context.Context, id string) error {
	if _, ok := f.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}
