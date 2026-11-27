package sqlite

import (
	"context"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/kipit-labs/kipitfit/internal/ids"
	"github.com/kipit-labs/kipitfit/internal/store"
)

func (s *Store) PutWorkout(ctx context.Context, w store.Workout) (store.Workout, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var owner string
		err := tx.NewSelect().Table("workouts").Column("user_id").Where("id = ?", w.ID).Scan(ctx, &owner)
		switch {
		case err == nil && owner != w.UserID:
			return store.ErrNotFound
		case err == nil:
			// A retry or an edit: replace it, keeping when it was first saved.
			w.UpdatedAt = now()
			_, err = tx.NewUpdate().Model(&w).ExcludeColumn("id", "user_id", "created_at").WherePK().Exec(ctx)
			if err == nil {
				_, err = tx.NewDelete().Model((*store.WorkoutExercise)(nil)).Where("workout_id = ?", w.ID).Exec(ctx)
			}
		case errors.Is(mapErr(err), store.ErrNotFound):
			w.CreatedAt = now()
			w.UpdatedAt = w.CreatedAt
			_, err = tx.NewInsert().Model(&w).Exec(ctx)
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
func insertWorkoutExercises(ctx context.Context, tx bun.Tx, w store.Workout) error {
	if len(w.Exercises) == 0 {
		return nil
	}
	sets := []store.WorkoutSet{}
	for i := range w.Exercises {
		we := &w.Exercises[i]
		we.ID, we.WorkoutID, we.Position = ids.New(), w.ID, i
		for j, set := range we.Sets {
			set.WorkoutExerciseID, set.Position = we.ID, j
			sets = append(sets, set)
		}
	}
	if _, err := tx.NewInsert().Model(&w.Exercises).Exec(ctx); err != nil {
		return mapErr(err)
	}
	if len(sets) > 0 {
		if _, err := tx.NewInsert().Model(&sets).Exec(ctx); err != nil {
			return mapErr(err)
		}
	}
	return nil
}

func (s *Store) ListWorkouts(ctx context.Context, userID, after string, limit int) ([]store.Workout, error) {
	ws := []store.Workout{}
	q := withExercises(s.db.NewSelect().Model(&ws)).Where("w.user_id = ?", userID).OrderExpr("w.started_at DESC, w.id DESC").Limit(limit)
	if after != "" {
		q = q.Where("(w.started_at, w.id) < (SELECT a.started_at, a.id FROM workouts AS a WHERE a.id = ? AND a.user_id = ?)", after, userID)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, mapErr(err)
	}
	for i := range ws {
		fill(&ws[i])
	}
	return ws, nil
}

func (s *Store) WorkoutsSince(ctx context.Context, userID string, from time.Time) ([]store.Workout, error) {
	ws := []store.Workout{}
	err := withExercises(s.db.NewSelect().Model(&ws)).Where("w.user_id = ? AND w.started_at >= ?", userID, from.UTC()).
		OrderExpr("w.started_at, w.id").Scan(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	for i := range ws {
		fill(&ws[i])
	}
	return ws, nil
}

func (s *Store) GetWorkout(ctx context.Context, userID, id string) (store.Workout, error) {
	return getWorkout(ctx, s.db, userID, id)
}

func getWorkout(ctx context.Context, db bun.IDB, userID, id string) (store.Workout, error) {
	var w store.Workout
	if err := withExercises(db.NewSelect().Model(&w)).Where("w.id = ? AND w.user_id = ?", id, userID).Scan(ctx); err != nil {
		return store.Workout{}, mapErr(err)
	}
	fill(&w)
	return w, nil
}

// withExercises loads a workout query's exercises (archived ones too) and sets, in order.
func withExercises(q *bun.SelectQuery) *bun.SelectQuery {
	return q.Relation("Exercises", func(q *bun.SelectQuery) *bun.SelectQuery { return q.Order("we.position") }).
		Relation("Exercises.Exercise", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.ExcludeColumn("instructions", "force", "level", "mechanic", "created_at", "updated_at")
		}).
		Relation("Exercises.Sets", func(q *bun.SelectQuery) *bun.SelectQuery { return q.Order("s.position") })
}

// fill sets what a loaded workout's JSON needs: exercise flags and empty lists rather than null.
func fill(w *store.Workout) {
	for i := range w.Exercises {
		if e := w.Exercises[i].Exercise; e != nil {
			flags(e)
		}
		if w.Exercises[i].Sets == nil {
			w.Exercises[i].Sets = []store.WorkoutSet{}
		}
	}
	if w.Exercises == nil {
		w.Exercises = []store.WorkoutExercise{}
	}
}

func (s *Store) DeleteWorkout(ctx context.Context, userID, id string) error {
	res, err := s.db.NewDelete().Model((*store.Workout)(nil)).Where("id = ? AND user_id = ?", id, userID).Exec(ctx)
	return affected(res, err)
}

func (s *Store) PreviousSets(ctx context.Context, userID string, exerciseIDs []string) (map[string][]store.WorkoutSet, error) {
	out := map[string][]store.WorkoutSet{}
	for _, ex := range exerciseIDs {
		var weID string
		err := s.db.NewSelect().TableExpr("workout_exercises AS we").TableExpr("workouts AS w").ColumnExpr("we.id").
			Where("we.workout_id = w.id AND w.user_id = ? AND we.exercise_id = ?", userID, ex).
			OrderExpr("w.started_at DESC, we.position").Limit(1).Scan(ctx, &weID)
		if errors.Is(mapErr(err), store.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		sets := []store.WorkoutSet{}
		if err := s.db.NewSelect().Model(&sets).Where("workout_exercise_id = ?", weID).Order("position").Scan(ctx); err != nil {
			return nil, err
		}
		out[ex] = sets
	}
	return out, nil
}

func (s *Store) ExerciseSets(ctx context.Context, userID string, exerciseIDs []string) ([]store.LoggedSet, error) {
	out := []store.LoggedSet{}
	if len(exerciseIDs) == 0 {
		return out, nil
	}
	err := s.db.NewSelect().TableExpr("sets AS s").
		Join("JOIN workout_exercises AS we ON we.id = s.workout_exercise_id").
		Join("JOIN workouts AS w ON w.id = we.workout_id").
		ColumnExpr("w.id AS workout_id, w.title, w.started_at, we.exercise_id, we.position AS exercise").
		ColumnExpr(`s.position AS "set", s.type, s.weight_g, s.reps, s.rpe, s.duration_s, s.distance_m`).
		Where("w.user_id = ? AND we.exercise_id IN (?)", userID, bun.In(exerciseIDs)).
		OrderExpr("w.started_at, w.id, we.position, s.position").
		Scan(ctx, &out)
	return out, mapErr(err)
}
