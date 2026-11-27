package sqlite

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"

	"github.com/kipit-labs/kipitfit/internal/domain"
	"github.com/kipit-labs/kipitfit/internal/ids"
)

func (s *Store) ListFolders(ctx context.Context, userID string) ([]domain.Folder, error) {
	var rows []folderRow
	err := s.db.NewSelect().Model(&rows).Where("user_id = ?", userID).Order("position", "id").Scan(ctx)
	fs := make([]domain.Folder, len(rows))
	for i, r := range rows {
		fs[i] = r.domain()
	}
	return fs, mapErr(err)
}

func (s *Store) CreateFolder(ctx context.Context, f domain.Folder) (domain.Folder, error) {
	row := folderRow{ID: ids.New(), UserID: f.UserID, Name: f.Name, CreatedAt: now()}
	_, err := s.db.NewInsert().Model(&row).
		Value("position", "(SELECT COALESCE(MAX(position) + 1, 0) FROM routine_folders WHERE user_id = ?)", f.UserID).
		Returning("position").Exec(ctx)
	return row.domain(), mapErr(err)
}

func (s *Store) RenameFolder(ctx context.Context, f domain.Folder) error {
	row := folderRow{ID: f.ID, UserID: f.UserID, Name: f.Name}
	res, err := s.db.NewUpdate().Model(&row).Column("name").Where("id = ? AND user_id = ?", f.ID, f.UserID).Exec(ctx)
	return affected(res, err)
}

func (s *Store) DeleteFolder(ctx context.Context, userID, id string) error {
	res, err := s.db.NewDelete().Model((*folderRow)(nil)).Where("id = ? AND user_id = ?", id, userID).Exec(ctx)
	return affected(res, err)
}

func (s *Store) ListRoutines(ctx context.Context, userID string) ([]domain.Routine, error) {
	var rows []routineRow
	err := routines(s.db, &rows).Where("r.user_id = ?", userID).Order("r.position", "r.id").Scan(ctx)
	rs := make([]domain.Routine, len(rows))
	for i, r := range rows {
		rs[i] = r.domain()
	}
	return rs, mapErr(err)
}

func (s *Store) GetRoutine(ctx context.Context, userID, id string) (domain.Routine, error) {
	return getRoutine(ctx, s.db, userID, id)
}

func getRoutine(ctx context.Context, db bun.IDB, userID, id string) (domain.Routine, error) {
	var r routineRow
	if err := routines(db, &r).Where("r.id = ? AND r.user_id = ?", id, userID).Scan(ctx); err != nil {
		return domain.Routine{}, mapErr(err)
	}
	return r.domain(), nil
}

// routines selects into model with each routine's exercises and sets in order.
func routines(db bun.IDB, model any) *bun.SelectQuery {
	return db.NewSelect().Model(model).ColumnExpr("r.*").
		ColumnExpr("(SELECT MAX(w.ended_at) FROM workouts AS w WHERE w.routine_id = r.id) AS last_done_at").
		Relation("Exercises", func(q *bun.SelectQuery) *bun.SelectQuery { return q.Order("re.position") }).
		Relation("Exercises.Exercise", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.ExcludeColumn("instructions", "force", "level", "mechanic", "created_at", "updated_at")
		}).
		Relation("Exercises.Sets", func(q *bun.SelectQuery) *bun.SelectQuery { return q.Order("rs.position") })
}

func (s *Store) CreateRoutine(ctx context.Context, r domain.Routine) (domain.Routine, error) {
	r.ID, r.CreatedAt = ids.New(), now()
	r.UpdatedAt = r.CreatedAt
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		row := newRoutineRow(r)
		_, err := tx.NewInsert().Model(&row).
			Value("position", "(SELECT COALESCE(MAX(position) + 1, 0) FROM routines WHERE user_id = ?)", r.UserID).
			Exec(ctx)
		if err != nil {
			return mapErr(err)
		}
		if err := insertExercises(ctx, tx, r); err != nil {
			return err
		}
		r, err = getRoutine(ctx, tx, r.UserID, r.ID)
		return err
	})
	return r, err
}

func (s *Store) UpdateRoutine(ctx context.Context, r domain.Routine) (domain.Routine, error) {
	r.UpdatedAt = now()
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		row := newRoutineRow(r)
		res, err := tx.NewUpdate().Model(&row).Column("folder_id", "name", "notes", "updated_at").
			Where("id = ? AND user_id = ?", r.ID, r.UserID).Exec(ctx)
		if err := affected(res, err); err != nil {
			return err
		}
		// The sets go with their exercises (ON DELETE CASCADE).
		if _, err := tx.NewDelete().Model((*routineExerciseRow)(nil)).Where("routine_id = ?", r.ID).Exec(ctx); err != nil {
			return err
		}
		if err := insertExercises(ctx, tx, r); err != nil {
			return err
		}
		r, err = getRoutine(ctx, tx, r.UserID, r.ID)
		return err
	})
	return r, err
}

// insertExercises writes r's exercises and sets at their positions in r.
func insertExercises(ctx context.Context, tx bun.Tx, r domain.Routine) error {
	if len(r.Exercises) == 0 {
		return nil
	}
	exs := make([]routineExerciseRow, len(r.Exercises))
	sets := []routineSetRow{}
	for i, re := range r.Exercises {
		exs[i] = routineExerciseRow{ID: ids.New(), RoutineID: r.ID, ExerciseID: re.ExerciseID, Position: i,
			Superset: re.Superset, RestS: re.RestS, Notes: re.Notes}
		for j, set := range re.Sets {
			sets = append(sets, routineSetRow{RoutineExerciseID: exs[i].ID, Position: j, Type: set.Type, WeightG: set.WeightG,
				RepsMin: set.RepsMin, RepsMax: set.RepsMax, DurationS: set.DurationS, DistanceM: set.DistanceM})
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

func (s *Store) DeleteRoutine(ctx context.Context, userID, id string) error {
	res, err := s.db.NewDelete().Model((*routineRow)(nil)).Where("id = ? AND user_id = ?", id, userID).Exec(ctx)
	return affected(res, err)
}

func (s *Store) Reorder(ctx context.Context, userID string, folders, routines []string) error {
	return s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		for _, l := range []struct {
			model any
			ids   []string
		}{{(*folderRow)(nil), folders}, {(*routineRow)(nil), routines}} {
			for i, id := range l.ids {
				res, err := tx.NewUpdate().Model(l.model).Set("position = ?", i).
					Where("id = ? AND user_id = ?", id, userID).Exec(ctx)
				if err := affected(res, err); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// affected maps an update or delete that matched no row to ErrNotFound.
func affected(res sql.Result, err error) error {
	if err != nil {
		return mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
