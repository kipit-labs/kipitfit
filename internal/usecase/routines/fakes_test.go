package routines

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/kipit-labs/kipitfit/internal/domain"
	"github.com/kipit-labs/kipitfit/internal/usecase/exercises"
)

func ptr[T any](v T) *T { return &v }

// fakeExercises keeps exercises in memory; custom names are unique per owner.
type fakeExercises struct {
	byID map[string]domain.Exercise
	n    int
}

func (f *fakeExercises) id() string {
	f.n++
	return fmt.Sprint("ex-", f.n)
}

func (f *fakeExercises) ListExercises(_ context.Context, ownerID string) ([]domain.Exercise, error) {
	out := []domain.Exercise{}
	for _, e := range f.byID {
		if (e.OwnerID == "" || e.OwnerID == ownerID) && !e.Archived {
			out = append(out, e)
		}
	}
	slices.SortFunc(out, func(a, b domain.Exercise) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

func (f *fakeExercises) GetExercise(_ context.Context, id string) (domain.Exercise, error) {
	e, ok := f.byID[id]
	if !ok {
		return domain.Exercise{}, domain.ErrNotFound
	}
	return e, nil
}

func (f *fakeExercises) CreateExercise(_ context.Context, e domain.Exercise) (domain.Exercise, error) {
	for _, o := range f.byID {
		if o.OwnerID == e.OwnerID && strings.EqualFold(o.Name, e.Name) {
			return domain.Exercise{}, domain.ErrConflict
		}
	}
	e.ID, e.Custom = f.id(), true
	f.byID[e.ID] = e
	return e, nil
}

func (f *fakeExercises) UpdateExercise(_ context.Context, e domain.Exercise) (domain.Exercise, error) {
	f.byID[e.ID] = e
	return e, nil
}

func (f *fakeExercises) DeleteExercise(_ context.Context, id string) error {
	delete(f.byID, id)
	return nil
}

// fakeRoutines keeps folders and routines in memory, scoped to their user
// like the SQLite store, and fills in each routine exercise on reading.
type fakeRoutines struct {
	exercises *fakeExercises
	folders   map[string]domain.Folder
	routines  map[string]domain.Routine
	n         int
}

func (f *fakeRoutines) id() string {
	f.n++
	return fmt.Sprint("r-", f.n)
}

func (f *fakeRoutines) ListFolders(_ context.Context, userID string) ([]domain.Folder, error) {
	out := []domain.Folder{}
	for _, fo := range f.folders {
		if fo.UserID == userID {
			out = append(out, fo)
		}
	}
	slices.SortFunc(out, func(a, b domain.Folder) int { return a.Position - b.Position })
	return out, nil
}

func (f *fakeRoutines) CreateFolder(ctx context.Context, fo domain.Folder) (domain.Folder, error) {
	mine, _ := f.ListFolders(ctx, fo.UserID)
	fo.ID, fo.Position = f.id(), len(mine)
	f.folders[fo.ID] = fo
	return fo, nil
}

func (f *fakeRoutines) RenameFolder(_ context.Context, fo domain.Folder) error {
	old, ok := f.folders[fo.ID]
	if !ok || old.UserID != fo.UserID {
		return domain.ErrNotFound
	}
	old.Name = fo.Name
	f.folders[fo.ID] = old
	return nil
}

func (f *fakeRoutines) DeleteFolder(_ context.Context, userID, id string) error {
	if fo, ok := f.folders[id]; !ok || fo.UserID != userID {
		return domain.ErrNotFound
	}
	delete(f.folders, id)
	for rid, r := range f.routines {
		if r.FolderID == id {
			r.FolderID = ""
			f.routines[rid] = r
		}
	}
	return nil
}

// read is r as the store returns it: each exercise filled in.
func (f *fakeRoutines) read(r domain.Routine) domain.Routine {
	exs := make([]domain.RoutineExercise, len(r.Exercises))
	for i, re := range r.Exercises {
		e := f.exercises.byID[re.ExerciseID]
		re.Exercise = &e
		exs[i] = re
	}
	r.Exercises = exs
	return r
}

func (f *fakeRoutines) ListRoutines(_ context.Context, userID string) ([]domain.Routine, error) {
	out := []domain.Routine{}
	for _, r := range f.routines {
		if r.UserID == userID {
			out = append(out, f.read(r))
		}
	}
	slices.SortFunc(out, func(a, b domain.Routine) int { return a.Position - b.Position })
	return out, nil
}

func (f *fakeRoutines) GetRoutine(_ context.Context, userID, id string) (domain.Routine, error) {
	r, ok := f.routines[id]
	if !ok || r.UserID != userID {
		return domain.Routine{}, domain.ErrNotFound
	}
	return f.read(r), nil
}

func (f *fakeRoutines) CreateRoutine(ctx context.Context, r domain.Routine) (domain.Routine, error) {
	mine, _ := f.ListRoutines(ctx, r.UserID)
	r.ID, r.Position = f.id(), len(mine)
	f.routines[r.ID] = r
	return f.read(r), nil
}

func (f *fakeRoutines) UpdateRoutine(_ context.Context, r domain.Routine) (domain.Routine, error) {
	old, ok := f.routines[r.ID]
	if !ok || old.UserID != r.UserID {
		return domain.Routine{}, domain.ErrNotFound
	}
	r.Position = old.Position
	f.routines[r.ID] = r
	return f.read(r), nil
}

func (f *fakeRoutines) DeleteRoutine(_ context.Context, userID, id string) error {
	if r, ok := f.routines[id]; !ok || r.UserID != userID {
		return domain.ErrNotFound
	}
	delete(f.routines, id)
	return nil
}

func (f *fakeRoutines) Reorder(_ context.Context, userID string, folders, routines []string) error {
	for i, id := range folders {
		fo, ok := f.folders[id]
		if !ok || fo.UserID != userID {
			return domain.ErrNotFound
		}
		fo.Position = i
		f.folders[id] = fo
	}
	for i, id := range routines {
		r, ok := f.routines[id]
		if !ok || r.UserID != userID {
			return domain.ErrNotFound
		}
		r.Position = i
		f.routines[id] = r
	}
	return nil
}

// newService is the routines of two users, me and friend, over a library
// with a lift and a plank.
func newService() (s *Service, ex *exercises.Service, store *fakeExercises, lift, plank domain.Exercise) {
	store = &fakeExercises{byID: map[string]domain.Exercise{}}
	lift = domain.Exercise{ID: "lift", Name: "Bench Press", Kind: domain.KindWeightReps, PrimaryMuscles: []string{"chest"}}
	plank = domain.Exercise{ID: "plank", Name: "Plank", Kind: domain.KindDuration, PrimaryMuscles: []string{"abdominals"}}
	store.byID[lift.ID], store.byID[plank.ID] = lift, plank
	ex = exercises.NewService(store)
	routines := &fakeRoutines{exercises: store, folders: map[string]domain.Folder{}, routines: map[string]domain.Routine{}}
	return NewService(routines, ex), ex, store, lift, plank
}
