package sqlite

import (
	"time"

	"github.com/uptrace/bun"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

// The tables as bun models; each converts to and from its domain entity.

type userRow struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID           string `bun:"id,pk"`
	Email        string
	Name         string
	PasswordHash string
	IsAdmin      bool
	Language     string
	Unit         domain.Unit
	CreatedAt    time.Time
}

func newUserRow(u domain.User) userRow {
	return userRow{ID: u.ID, Email: u.Email, Name: u.Name, PasswordHash: u.PasswordHash, IsAdmin: u.IsAdmin,
		Language: u.Language, Unit: u.Unit, CreatedAt: u.CreatedAt}
}

func (r userRow) domain() domain.User {
	return domain.User{ID: r.ID, Email: r.Email, Name: r.Name, PasswordHash: r.PasswordHash, IsAdmin: r.IsAdmin,
		Language: r.Language, Unit: r.Unit, CreatedAt: r.CreatedAt}
}

type sessionRow struct {
	bun.BaseModel `bun:"table:sessions,alias:session"`

	TokenHash string `bun:"token_hash,pk"`
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type inviteRow struct {
	bun.BaseModel `bun:"table:invites,alias:invite"`

	ID        string `bun:"id,pk"`
	TokenHash string
	Email     string
	CreatedBy string
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}

func (r inviteRow) domain() domain.Invite {
	return domain.Invite{ID: r.ID, TokenHash: r.TokenHash, Email: r.Email, CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt, ExpiresAt: r.ExpiresAt, UsedAt: r.UsedAt}
}

type exerciseRow struct {
	bun.BaseModel `bun:"table:exercises,alias:ex"`

	ID               string `bun:"id,pk"`
	OwnerID          string `bun:"owner_id,nullzero"`
	Slug             string `bun:"slug,nullzero"`
	Name             string
	Kind             domain.Kind
	Category         string
	Equipment        string
	PrimaryMuscles   []string `bun:"primary_muscles,type:json"`
	SecondaryMuscles []string `bun:"secondary_muscles,type:json"`
	Force            string
	Level            string
	Mechanic         string
	Instructions     []string `bun:"instructions,type:json"`
	Media            []string `bun:"media,type:json"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ArchivedAt       time.Time `bun:"archived_at,nullzero"`
}

// newExerciseRow keeps JSON columns arrays: a nil slice would be stored as null.
func newExerciseRow(e domain.Exercise) exerciseRow {
	for _, l := range []*[]string{&e.PrimaryMuscles, &e.SecondaryMuscles, &e.Instructions, &e.Media} {
		if *l == nil {
			*l = []string{}
		}
	}
	return exerciseRow{ID: e.ID, OwnerID: e.OwnerID, Slug: e.Slug, Name: e.Name, Kind: e.Kind, Category: e.Category,
		Equipment: e.Equipment, PrimaryMuscles: e.PrimaryMuscles, SecondaryMuscles: e.SecondaryMuscles, Force: e.Force,
		Level: e.Level, Mechanic: e.Mechanic, Instructions: e.Instructions, Media: e.Media,
		CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt, ArchivedAt: e.ArchivedAt}
}

// domain also sets the flags the store owns: custom, archived.
func (r exerciseRow) domain() domain.Exercise {
	return domain.Exercise{ID: r.ID, OwnerID: r.OwnerID, Slug: r.Slug, Name: r.Name, Kind: r.Kind, Category: r.Category,
		Equipment: r.Equipment, PrimaryMuscles: r.PrimaryMuscles, SecondaryMuscles: r.SecondaryMuscles, Force: r.Force,
		Level: r.Level, Mechanic: r.Mechanic, Instructions: r.Instructions, Media: r.Media,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, ArchivedAt: r.ArchivedAt,
		Custom: r.OwnerID != "", Archived: !r.ArchivedAt.IsZero()}
}

func exerciseRef(r *exerciseRow) *domain.Exercise {
	if r == nil {
		return nil
	}
	e := r.domain()
	return &e
}

type folderRow struct {
	bun.BaseModel `bun:"table:routine_folders,alias:f"`

	ID        string `bun:"id,pk"`
	UserID    string
	Name      string
	Position  int
	CreatedAt time.Time
}

func (r folderRow) domain() domain.Folder {
	return domain.Folder{ID: r.ID, UserID: r.UserID, Name: r.Name, Position: r.Position, CreatedAt: r.CreatedAt}
}

type routineRow struct {
	bun.BaseModel `bun:"table:routines,alias:r"`

	ID         string `bun:"id,pk"`
	UserID     string
	FolderID   string `bun:"folder_id,nullzero"`
	Name       string
	Notes      string
	Position   int
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LastDoneAt *time.Time           `bun:"last_done_at,scanonly"`
	Exercises  []routineExerciseRow `bun:"rel:has-many,join:id=routine_id"`
}

type routineExerciseRow struct {
	bun.BaseModel `bun:"table:routine_exercises,alias:re"`

	ID         string `bun:"id,pk"`
	RoutineID  string
	ExerciseID string
	Position   int
	Superset   string `bun:"superset,nullzero"`
	RestS      *int   `bun:"rest_s"`
	Notes      string
	Exercise   *exerciseRow    `bun:"rel:belongs-to,join:exercise_id=id"`
	Sets       []routineSetRow `bun:"rel:has-many,join:id=routine_exercise_id"`
}

type routineSetRow struct {
	bun.BaseModel `bun:"table:routine_sets,alias:rs"`

	RoutineExerciseID string `bun:"routine_exercise_id,pk"`
	Position          int    `bun:"position,pk"`
	Type              domain.SetType
	WeightG           *int `bun:"weight_g"`
	RepsMin           *int `bun:"reps_min"`
	RepsMax           *int `bun:"reps_max"`
	DurationS         *int `bun:"duration_s"`
	DistanceM         *int `bun:"distance_m"`
}

func newRoutineRow(r domain.Routine) routineRow {
	return routineRow{ID: r.ID, UserID: r.UserID, FolderID: r.FolderID, Name: r.Name, Notes: r.Notes,
		Position: r.Position, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

// domain loads empty lists rather than nil, for the JSON.
func (r routineRow) domain() domain.Routine {
	out := domain.Routine{ID: r.ID, UserID: r.UserID, FolderID: r.FolderID, Name: r.Name, Notes: r.Notes,
		Position: r.Position, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, LastDoneAt: r.LastDoneAt,
		Exercises: make([]domain.RoutineExercise, len(r.Exercises))}
	for i, re := range r.Exercises {
		sets := make([]domain.RoutineSet, len(re.Sets))
		for j, s := range re.Sets {
			sets[j] = domain.RoutineSet{RoutineExerciseID: s.RoutineExerciseID, Position: s.Position, Type: s.Type,
				WeightG: s.WeightG, RepsMin: s.RepsMin, RepsMax: s.RepsMax, DurationS: s.DurationS, DistanceM: s.DistanceM}
		}
		out.Exercises[i] = domain.RoutineExercise{ID: re.ID, RoutineID: re.RoutineID, ExerciseID: re.ExerciseID,
			Position: re.Position, Superset: re.Superset, RestS: re.RestS, Notes: re.Notes,
			Exercise: exerciseRef(re.Exercise), Sets: sets}
	}
	return out
}

type workoutRow struct {
	bun.BaseModel `bun:"table:workouts,alias:w"`

	ID        string `bun:"id,pk"`
	UserID    string
	RoutineID string `bun:"routine_id,nullzero"`
	Title     string
	Notes     string
	StartedAt time.Time
	EndedAt   time.Time
	VolumeG   int `bun:"volume_g"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Exercises []workoutExerciseRow `bun:"rel:has-many,join:id=workout_id"`
}

type workoutExerciseRow struct {
	bun.BaseModel `bun:"table:workout_exercises,alias:we"`

	ID         string `bun:"id,pk"`
	WorkoutID  string
	ExerciseID string
	Position   int
	Superset   string `bun:"superset,nullzero"`
	Notes      string
	Exercise   *exerciseRow    `bun:"rel:belongs-to,join:exercise_id=id"`
	Sets       []workoutSetRow `bun:"rel:has-many,join:id=workout_exercise_id"`
}

type workoutSetRow struct {
	bun.BaseModel `bun:"table:sets,alias:s"`

	WorkoutExerciseID string `bun:"workout_exercise_id,pk"`
	Position          int    `bun:"position,pk"`
	Type              domain.SetType
	WeightG           *int           `bun:"weight_g"`
	Reps              *int           `bun:"reps"`
	RPE               *domain.Tenths `bun:"rpe"`
	DurationS         *int           `bun:"duration_s"`
	DistanceM         *int           `bun:"distance_m"`
}

func newWorkoutRow(w domain.Workout) workoutRow {
	return workoutRow{ID: w.ID, UserID: w.UserID, RoutineID: w.RoutineID, Title: w.Title, Notes: w.Notes,
		StartedAt: w.StartedAt, EndedAt: w.EndedAt, VolumeG: w.VolumeG, CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt}
}

func (s workoutSetRow) domain() domain.WorkoutSet {
	return domain.WorkoutSet{WorkoutExerciseID: s.WorkoutExerciseID, Position: s.Position, Type: s.Type,
		WeightG: s.WeightG, Reps: s.Reps, RPE: s.RPE, DurationS: s.DurationS, DistanceM: s.DistanceM}
}

// domain loads empty lists rather than nil, for the JSON.
func (w workoutRow) domain() domain.Workout {
	out := domain.Workout{ID: w.ID, UserID: w.UserID, RoutineID: w.RoutineID, Title: w.Title, Notes: w.Notes,
		StartedAt: w.StartedAt, EndedAt: w.EndedAt, VolumeG: w.VolumeG, CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt,
		Exercises: make([]domain.WorkoutExercise, len(w.Exercises))}
	for i, we := range w.Exercises {
		sets := make([]domain.WorkoutSet, len(we.Sets))
		for j, s := range we.Sets {
			sets[j] = s.domain()
		}
		out.Exercises[i] = domain.WorkoutExercise{ID: we.ID, WorkoutID: we.WorkoutID, ExerciseID: we.ExerciseID,
			Position: we.Position, Superset: we.Superset, Notes: we.Notes, Exercise: exerciseRef(we.Exercise), Sets: sets}
	}
	return out
}

// loggedSetRow is what ExerciseSets scans; it converts to domain.LoggedSet as is.
type loggedSetRow struct {
	WorkoutID  string `bun:"workout_id"`
	Title      string `bun:"title"`
	StartedAt  time.Time
	ExerciseID string         `bun:"exercise_id"`
	Exercise   int            `bun:"exercise"`
	Set        int            `bun:"set"`
	Type       domain.SetType `bun:"type"`
	WeightG    *int           `bun:"weight_g"`
	Reps       *int           `bun:"reps"`
	RPE        *domain.Tenths `bun:"rpe"`
	DurationS  *int           `bun:"duration_s"`
	DistanceM  *int           `bun:"distance_m"`
}
