package routines

import (
	"context"
	"fmt"
	"slices"
	"unicode/utf8"

	"github.com/kipit-labs/kipitfit/internal/domain"
	"github.com/kipit-labs/kipitfit/internal/usecase/exercises"
)

const (
	maxRestS = 3600
	// More folders or routines than one reorder can list.
	maxOrder = 1000
)

// Service keeps each user's routines and their folders.
type Service struct {
	repo      domain.RoutineRepository
	exercises *exercises.Service
}

func NewService(repo domain.RoutineRepository, ex *exercises.Service) *Service {
	return &Service{repo: repo, exercises: ex}
}

// Routines are me's folders and routines, each in order.
type Routines struct {
	Folders  []domain.Folder  `json:"folders"`
	Routines []domain.Routine `json:"routines"`
}

func (s *Service) ListRoutines(ctx context.Context, me domain.User) (Routines, error) {
	fs, err := s.repo.ListFolders(ctx, me.ID)
	if err != nil {
		return Routines{}, err
	}
	rs, err := s.repo.ListRoutines(ctx, me.ID)
	return Routines{Folders: fs, Routines: rs}, err
}

// CreateRoutine saves in as a new routine of me's, after the others.
func (s *Service) CreateRoutine(ctx context.Context, me domain.User, in domain.Routine) (domain.Routine, error) {
	if err := s.checkRoutine(ctx, me, &in); err != nil {
		return domain.Routine{}, err
	}
	return s.repo.CreateRoutine(ctx, in)
}

// UpdateRoutine replaces routine id with in, exercises and sets included.
func (s *Service) UpdateRoutine(ctx context.Context, me domain.User, id string, in domain.Routine) (domain.Routine, error) {
	if err := s.checkRoutine(ctx, me, &in); err != nil {
		return domain.Routine{}, err
	}
	in.ID = id
	return s.repo.UpdateRoutine(ctx, in)
}

func (s *Service) DeleteRoutine(ctx context.Context, me domain.User, id string) error {
	return s.repo.DeleteRoutine(ctx, me.ID, id)
}

// checkRoutine validates and normalises r as me's; its folder and exercises
// must be ones me sees.
func (s *Service) checkRoutine(ctx context.Context, me domain.User, r *domain.Routine) error {
	r.UserID = me.ID
	var err error
	if r.Name, err = domain.Title(r.Name); err != nil {
		return err
	}
	if utf8.RuneCountInString(r.Notes) > domain.MaxNotes {
		return domain.ErrNotesTooLong
	}
	if r.FolderID != "" {
		fs, err := s.repo.ListFolders(ctx, me.ID)
		if err != nil {
			return err
		}
		if !slices.ContainsFunc(fs, func(f domain.Folder) bool { return f.ID == r.FolderID }) {
			return domain.ErrNotFound
		}
	}
	if len(r.Exercises) > domain.MaxExercises {
		return domain.ErrTooManyExercises
	}
	for i := range r.Exercises {
		re := &r.Exercises[i]
		// Archived exercises stay valid, so a routine using one can still be saved.
		e, err := s.exercises.GetExercise(ctx, me, re.ExerciseID)
		if err != nil {
			return err
		}
		re.Exercise = nil
		if !domain.ValidSuperset(re.Superset) {
			return domain.ErrSuperset
		}
		if re.RestS != nil && (*re.RestS < 0 || *re.RestS > maxRestS) {
			return domain.ErrSetValue
		}
		if utf8.RuneCountInString(re.Notes) > domain.MaxNotes {
			return domain.ErrNotesTooLong
		}
		if len(re.Sets) > domain.MaxSets {
			return domain.ErrTooManySets
		}
		for _, s := range re.Sets {
			if err := checkTarget(e.Kind, s); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkTarget(k domain.Kind, s domain.RoutineSet) error {
	if !slices.Contains(domain.SetTypes, s.Type) {
		return domain.ErrSetType
	}
	f := k.Fields()
	ok := domain.InRange(s.WeightG, f.Weight, domain.MaxWeightG) && domain.InRange(s.RepsMin, f.Reps, domain.MaxReps) &&
		domain.InRange(s.RepsMax, f.Reps, domain.MaxReps) && domain.InRange(s.DurationS, f.Duration, domain.MaxDurationS) &&
		domain.InRange(s.DistanceM, f.Distance, domain.MaxDistanceM)
	if !ok || (s.RepsMin != nil && s.RepsMax != nil && *s.RepsMin > *s.RepsMax) {
		return domain.ErrSetValue
	}
	return nil
}

// FolderInput names a folder.
type FolderInput struct {
	Name string `json:"name"`
}

func (s *Service) ListFolders(ctx context.Context, me domain.User) ([]domain.Folder, error) {
	return s.repo.ListFolders(ctx, me.ID)
}

func (s *Service) CreateFolder(ctx context.Context, me domain.User, in FolderInput) (domain.Folder, error) {
	name, err := domain.Title(in.Name)
	if err != nil {
		return domain.Folder{}, err
	}
	return s.repo.CreateFolder(ctx, domain.Folder{UserID: me.ID, Name: name})
}

func (s *Service) RenameFolder(ctx context.Context, me domain.User, id string, in FolderInput) (domain.Folder, error) {
	name, err := domain.Title(in.Name)
	if err != nil {
		return domain.Folder{}, err
	}
	f := domain.Folder{ID: id, UserID: me.ID, Name: name}
	return f, s.repo.RenameFolder(ctx, f)
}

// DeleteFolder deletes the folder; its routines stay, in no folder.
func (s *Service) DeleteFolder(ctx context.Context, me domain.User, id string) error {
	return s.repo.DeleteFolder(ctx, me.ID, id)
}

// Order lists folders and routines in their new order; the ones left out keep
// their position.
type Order struct {
	Folders  []string `json:"folders"`
	Routines []string `json:"routines"`
}

func (s *Service) Reorder(ctx context.Context, me domain.User, in Order) error {
	if len(in.Folders) > maxOrder || len(in.Routines) > maxOrder {
		return fmt.Errorf("%w: at most %d ids", domain.ErrInvalid, maxOrder)
	}
	return s.repo.Reorder(ctx, me.ID, in.Folders, in.Routines)
}
