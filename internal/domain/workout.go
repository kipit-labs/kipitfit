package domain

import (
	"context"
	"database/sql/driver"
	"fmt"
	"math"
	"strconv"
	"time"
)

type Workout struct {
	// ID is made on the phone when the workout starts.
	ID     string `json:"id"`
	UserID string `json:"-"`
	// RoutineID is the routine it started from, "" for none.
	RoutineID string            `json:"routineId,omitempty"`
	Title     string            `json:"title"`
	Notes     string            `json:"notes"`
	StartedAt time.Time         `json:"startedAt"`
	EndedAt   time.Time         `json:"endedAt"`
	VolumeG   int               `json:"volumeG"`
	CreatedAt time.Time         `json:"-"`
	UpdatedAt time.Time         `json:"-"`
	Exercises []WorkoutExercise `json:"exercises,omitempty"`
}

type WorkoutExercise struct {
	ID         string       `json:"-"`
	WorkoutID  string       `json:"-"`
	ExerciseID string       `json:"exerciseId"`
	Position   int          `json:"-"`
	Superset   string       `json:"superset,omitempty"`
	Notes      string       `json:"notes"`
	Exercise   *Exercise    `json:"exercise,omitempty"`
	Sets       []WorkoutSet `json:"sets"`
}

// WorkoutSet is a logged set; a value its exercise doesn't take is nil.
type WorkoutSet struct {
	WorkoutExerciseID string  `json:"-"`
	Position          int     `json:"-"`
	Type              SetType `json:"type"`
	WeightG           *int    `json:"weightG"`
	Reps              *int    `json:"reps"`
	RPE               *Tenths `json:"rpe"`
	DurationS         *int    `json:"durationS"`
	DistanceM         *int    `json:"distanceM"`
}

// LoggedSet is a set with where it was logged: its workout, and its place in
// it (Exercise is the workout exercise's position, Set the set's).
type LoggedSet struct {
	WorkoutID  string
	Title      string
	StartedAt  time.Time
	ExerciseID string
	Exercise   int
	Set        int
	Type       SetType
	WeightG    *int
	Reps       *int
	RPE        *Tenths
	DurationS  *int
	DistanceM  *int
}

// Tenths is a number with one decimal kept as an integer count of tenths
// (8.5 is 85), written as a plain number in JSON.
type Tenths int

func (t Tenths) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatFloat(float64(t)/10, 'f', -1, 64)), nil
}

func (t *Tenths) UnmarshalJSON(b []byte) error {
	f, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return err
	}
	*t = Tenths(math.Round(f * 10))
	return nil
}

// Value and Scan keep it an integer in the database (which would otherwise
// store the JSON).
func (t Tenths) Value() (driver.Value, error) { return int64(t), nil }

func (t *Tenths) Scan(src any) error {
	n, ok := src.(int64)
	if !ok {
		return fmt.Errorf("tenths: unexpected %T", src)
	}
	*t = Tenths(n)
	return nil
}

type WorkoutRepository interface {
	// PutWorkout creates the workout or replaces it (exercises and sets too),
	// in one transaction; ErrNotFound when the id is another user's.
	PutWorkout(ctx context.Context, w Workout) (Workout, error)
	// ListWorkouts returns up to limit of the user's workouts with their
	// exercises and sets, newest first, started before the workout after (""
	// for the newest).
	ListWorkouts(ctx context.Context, userID, after string, limit int) ([]Workout, error)
	// WorkoutsSince returns the user's workouts started at or after from, with
	// their exercises and sets, oldest first.
	WorkoutsSince(ctx context.Context, userID string, from time.Time) ([]Workout, error)
	GetWorkout(ctx context.Context, userID, id string) (Workout, error)
	DeleteWorkout(ctx context.Context, userID, id string) error
	// PreviousSets returns, per exercise, its sets in the user's latest workout
	// that has it; exercises never done are left out.
	PreviousSets(ctx context.Context, userID string, exerciseIDs []string) (map[string][]WorkoutSet, error)
	// ExerciseSets returns every set the user logged of the exercises, warm-ups
	// too, oldest workout first and in workout order: what records and
	// exercise history are worked out from.
	ExerciseSets(ctx context.Context, userID string, exerciseIDs []string) ([]LoggedSet, error)
}
