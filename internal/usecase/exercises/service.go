package exercises

import (
	"context"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

const maxExerciseName = 100

var (
	ErrLibraryReadonly    = domain.Coded(domain.ErrForbidden, "library_exercise_readonly", "library exercises can't be changed: create your own instead")
	ErrExerciseNameTaken  = domain.Coded(domain.ErrConflict, "exercise_name_taken", "you already have an exercise with this name")
	ErrExerciseName       = domain.Coded(domain.ErrInvalid, "exercise_name_invalid", "the name must be 1 to 100 characters")
	ErrExerciseField      = domain.Coded(domain.ErrInvalid, "exercise_field_invalid", "an exercise field has a value the app doesn't know")
	ErrPrimaryMuscleEmpty = domain.Coded(domain.ErrInvalid, "primary_muscle_required", "pick at least one primary muscle")
)

func init() { ErrExerciseName.Params = map[string]any{"max": maxExerciseName} }

// Service runs the exercise list: the library, and each user's own exercises.
type Service struct {
	repo domain.ExerciseRepository
}

func NewService(repo domain.ExerciseRepository) *Service {
	return &Service{repo: repo}
}

// ListExercises returns the library and me's custom exercises, compact.
func (s *Service) ListExercises(ctx context.Context, me domain.User) ([]domain.Exercise, error) {
	return s.repo.ListExercises(ctx, me.ID)
}

// GetExercise returns a library exercise or one of me's; someone else's is
// not found.
func (s *Service) GetExercise(ctx context.Context, me domain.User, id string) (domain.Exercise, error) {
	e, err := s.repo.GetExercise(ctx, id)
	if err != nil {
		return domain.Exercise{}, err
	}
	if e.Custom && e.OwnerID != me.ID {
		return domain.Exercise{}, domain.ErrNotFound
	}
	return e, nil
}

// ExerciseInput is a custom exercise to create, or the fields of one to
// change (nil fields stay).
type ExerciseInput struct {
	Name             *string      `json:"name"`
	Kind             *domain.Kind `json:"kind"`
	Category         *string      `json:"category"`
	Equipment        *string      `json:"equipment"`
	PrimaryMuscles   *[]string    `json:"primaryMuscles"`
	SecondaryMuscles *[]string    `json:"secondaryMuscles"`
}

func (s *Service) CreateExercise(ctx context.Context, me domain.User, in ExerciseInput) (domain.Exercise, error) {
	e := domain.Exercise{OwnerID: me.ID, Category: "strength", Equipment: "none", SecondaryMuscles: []string{}}
	if err := apply(&e, in); err != nil {
		return domain.Exercise{}, err
	}
	if in.Name == nil {
		return domain.Exercise{}, ErrExerciseName
	}
	if in.Kind == nil {
		return domain.Exercise{}, ErrExerciseField
	}
	e, err := s.repo.CreateExercise(ctx, e)
	return e, nameTaken(err)
}

func (s *Service) UpdateExercise(ctx context.Context, me domain.User, id string, in ExerciseInput) (domain.Exercise, error) {
	e, err := s.ownExercise(ctx, me, id)
	if err != nil {
		return domain.Exercise{}, err
	}
	if err := apply(&e, in); err != nil {
		return domain.Exercise{}, err
	}
	e, err = s.repo.UpdateExercise(ctx, e)
	return e, nameTaken(err)
}

func (s *Service) DeleteExercise(ctx context.Context, me domain.User, id string) error {
	if _, err := s.ownExercise(ctx, me, id); err != nil {
		return err
	}
	return s.repo.DeleteExercise(ctx, id)
}

// ownExercise is me's custom exercise id, for a change.
func (s *Service) ownExercise(ctx context.Context, me domain.User, id string) (domain.Exercise, error) {
	e, err := s.GetExercise(ctx, me, id)
	if err != nil {
		return domain.Exercise{}, err
	}
	if !e.Custom {
		return domain.Exercise{}, ErrLibraryReadonly
	}
	return e, nil
}

// apply validates in and copies its fields onto e.
func apply(e *domain.Exercise, in ExerciseInput) error {
	if in.Name != nil {
		e.Name = strings.Join(strings.Fields(*in.Name), " ")
		if n := utf8.RuneCountInString(e.Name); n == 0 || n > maxExerciseName {
			return ErrExerciseName
		}
	}
	if in.Kind != nil {
		if !slices.Contains(domain.Kinds, *in.Kind) {
			return ErrExerciseField
		}
		e.Kind = *in.Kind
	}
	for _, f := range []struct {
		in      *string
		out     *string
		allowed []string
	}{{in.Category, &e.Category, domain.Categories}, {in.Equipment, &e.Equipment, domain.Equipment}} {
		if f.in != nil {
			if !slices.Contains(f.allowed, *f.in) {
				return ErrExerciseField
			}
			*f.out = *f.in
		}
	}
	for _, f := range []struct{ in, out *[]string }{{in.PrimaryMuscles, &e.PrimaryMuscles}, {in.SecondaryMuscles, &e.SecondaryMuscles}} {
		if f.in != nil {
			ms := slices.Compact(slices.Sorted(slices.Values(*f.in)))
			for _, m := range ms {
				if !slices.Contains(domain.Muscles, m) {
					return ErrExerciseField
				}
			}
			*f.out = ms
		}
	}
	if len(e.PrimaryMuscles) == 0 {
		return ErrPrimaryMuscleEmpty
	}
	return nil
}

func nameTaken(err error) error {
	if errors.Is(err, domain.ErrConflict) {
		return ErrExerciseNameTaken
	}
	return err
}
