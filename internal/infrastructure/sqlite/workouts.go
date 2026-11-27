package sqlite

import (
	"context"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/kipit-labs/kipitfit/internal/domain"
	"github.com/kipit-labs/kipitfit/internal/ids"
)

func (s *Store) PutWorkout(ctx context.Context, w domain.Workout) (domain.Workout, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var owner string
		err := tx.NewSelect().Table("workouts").Column("user_id").Where("id = ?", w.ID).Scan(ctx, &owner)
		switch {
		case err == nil && owner != w.UserID:
			return domain.ErrNotFound
		case err == nil:
			// A retry or an edit: replace it, keeping when it was first saved.
			w.UpdatedAt = now()
			row := newWorkoutRow(w)
			_, err = tx.NewUpdate().Model(&row).ExcludeColumn("id", "user_id", "created_at").WherePK().Exec(ctx)
			if err == nil {
				_, err = tx.NewDelete().Model((*workoutExerciseRow)(nil)).Where("workout_id = ?", w.ID).Exec(ctx)
			}
		case errors.Is(mapErr(err), domain.ErrNotFound):
			w.CreatedAt = now()
			w.UpdatedAt = w.CreatedAt
			row := newWorkoutRow(w)
			_, err = tx.NewInsert().Model(&row).Exec(ctx)
		}
		if err != nil {
			return mapErr(err)
		}
		if err := insertWorkoutExercises(ctx, tx, w); err != nil {
			return err
		}
		w, err = getWorkout(ctx, tx, w.UserID, w.ID)
		return err
	})
	return w, err
}

// insertWorkoutExercises writes w's exercises and sets at their positions in w.
func insertWorkoutExercises(ctx context.Context, tx bun.Tx, w domain.Workout) error {
	if len(w.Exercises) == 0 {
		return nil
	}
	exs := make([]workoutExerciseRow, len(w.Exercises))
	sets := []workoutSetRow{}
	for i, we := range w.Exercises {
		exs[i] = workoutExerciseRow{ID: ids.New(), WorkoutID: w.ID, ExerciseID: we.ExerciseID, Position: i,
			Superset: we.Superset, Notes: we.Notes}
		for j, set := range we.Sets {
			sets = append(sets, workoutSetRow{WorkoutExerciseID: exs[i].ID, Position: j, Type: set.Type, WeightG: set.WeightG,
				Reps: set.Reps, RPE: set.RPE, DurationS: set.DurationS, DistanceM: set.DistanceM})
		}
	}
	if _, err := tx.NewInsert().Model(&exs).Exec(ctx); err != nil {
		return mapErr(err)
	}
	if len(sets) > 0 {
		if _, err := tx.NewInsert().Model(&sets).Exec(ctx); err != nil {
			return mapErr(err)
		}
	}
	return nil
}

func (s *Store) ListWorkouts(ctx context.Context, userID, after string, limit int) ([]domain.Workout, error) {
	var rows []workoutRow
	q := withExercises(s.db.NewSelect().Model(&rows)).Where("w.user_id = ?", userID).OrderExpr("w.started_at DESC, w.id DESC").Limit(limit)
	if after != "" {
		q = q.Where("(w.started_at, w.id) < (SELECT a.started_at, a.id FROM workouts AS a WHERE a.id = ? AND a.user_id = ?)", after, userID)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, mapErr(err)
	}
	return workouts(rows), nil
}

func (s *Store) WorkoutsSince(ctx context.Context, userID string, from time.Time) ([]domain.Workout, error) {
	var rows []workoutRow
	err := withExercises(s.db.NewSelect().Model(&rows)).Where("w.user_id = ? AND w.started_at >= ?", userID, from.UTC()).
		OrderExpr("w.started_at, w.id").Scan(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	return workouts(rows), nil
}

func workouts(rows []workoutRow) []domain.Workout {
	ws := make([]domain.Workout, len(rows))
	for i, r := range rows {
		ws[i] = r.domain()
	}
	return ws
}

func (s *Store) GetWorkout(ctx context.Context, userID, id string) (domain.Workout, error) {
	return getWorkout(ctx, s.db, userID, id)
}

func getWorkout(ctx context.Context, db bun.IDB, userID, id string) (domain.Workout, error) {
	var w workoutRow
	if err := withExercises(db.NewSelect().Model(&w)).Where("w.id = ? AND w.user_id = ?", id, userID).Scan(ctx); err != nil {
		return domain.Workout{}, mapErr(err)
	}
	return w.domain(), nil
}

// withExercises loads a workout query's exercises (archived ones too) and sets, in order.
func withExercises(q *bun.SelectQuery) *bun.SelectQuery {
	return q.Relation("Exercises", func(q *bun.SelectQuery) *bun.SelectQuery { return q.Order("we.position") }).
		Relation("Exercises.Exercise", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.ExcludeColumn("instructions", "force", "level", "mechanic", "created_at", "updated_at")
		}).
		Relation("Exercises.Sets", func(q *bun.SelectQuery) *bun.SelectQuery { return q.Order("s.position") })
}

func (s *Store) DeleteWorkout(ctx context.Context, userID, id string) error {
	res, err := s.db.NewDelete().Model((*workoutRow)(nil)).Where("id = ? AND user_id = ?", id, userID).Exec(ctx)
	return affected(res, err)
}

func (s *Store) PreviousSets(ctx context.Context, userID string, exerciseIDs []string) (map[string][]domain.WorkoutSet, error) {
	out := map[string][]domain.WorkoutSet{}
	for _, ex := range exerciseIDs {
		var weID string
		err := s.db.NewSelect().TableExpr("workout_exercises AS we").TableExpr("workouts AS w").ColumnExpr("we.id").
			Where("we.workout_id = w.id AND w.user_id = ? AND we.exercise_id = ?", userID, ex).
			OrderExpr("w.started_at DESC, we.position").Limit(1).Scan(ctx, &weID)
		if errors.Is(mapErr(err), domain.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var rows []workoutSetRow
		if err := s.db.NewSelect().Model(&rows).Where("workout_exercise_id = ?", weID).Order("position").Scan(ctx); err != nil {
			return nil, err
		}
		sets := make([]domain.WorkoutSet, len(rows))
		for i, r := range rows {
			sets[i] = r.domain()
		}
		out[ex] = sets
	}
	return out, nil
}

func (s *Store) ExerciseSets(ctx context.Context, userID string, exerciseIDs []string) ([]domain.LoggedSet, error) {
	if len(exerciseIDs) == 0 {
		return []domain.LoggedSet{}, nil
	}
	var rows []loggedSetRow
	err := s.db.NewSelect().TableExpr("sets AS s").
		Join("JOIN workout_exercises AS we ON we.id = s.workout_exercise_id").
		Join("JOIN workouts AS w ON w.id = we.workout_id").
		ColumnExpr("w.id AS workout_id, w.title, w.started_at, we.exercise_id, we.position AS exercise").
		ColumnExpr(`s.position AS "set", s.type, s.weight_g, s.reps, s.rpe, s.duration_s, s.distance_m`).
		Where("w.user_id = ? AND we.exercise_id IN (?)", userID, bun.List(exerciseIDs)).
		OrderExpr("w.started_at, w.id, we.position, s.position").
		Scan(ctx, &rows)
	out := make([]domain.LoggedSet, len(rows))
	for i, r := range rows {
		out[i] = domain.LoggedSet(r)
	}
	return out, mapErr(err)
}
