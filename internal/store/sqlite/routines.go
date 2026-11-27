package sqlite

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"

	"github.com/kipit-labs/kipitfit/internal/ids"
	"github.com/kipit-labs/kipitfit/internal/store"
)

func (s *Store) ListFolders(ctx context.Context, userID string) ([]store.Folder, error) {
	fs := []store.Folder{}
	err := s.db.NewSelect().Model(&fs).Where("user_id = ?", userID).Order("position", "id").Scan(ctx)
	return fs, mapErr(err)
}

func (s *Store) CreateFolder(ctx context.Context, f store.Folder) (store.Folder, error) {
	f.ID, f.CreatedAt = ids.New(), now()
	_, err := s.db.NewInsert().Model(&f).
		Value("position", "(SELECT COALESCE(MAX(position) + 1, 0) FROM routine_folders WHERE user_id = ?)", f.UserID).
		Returning("position").Exec(ctx)
	return f, mapErr(err)
}

func (s *Store) RenameFolder(ctx context.Context, f store.Folder) error {
	res, err := s.db.NewUpdate().Model(&f).Column("name").Where("id = ? AND user_id = ?", f.ID, f.UserID).Exec(ctx)
	return affected(res, err)
}

func (s *Store) DeleteFolder(ctx context.Context, userID, id string) error {
	res, err := s.db.NewDelete().Model((*store.Folder)(nil)).Where("id = ? AND user_id = ?", id, userID).Exec(ctx)
	return affected(res, err)
}

func (s *Store) ListRoutines(ctx context.Context, userID string) ([]store.Routine, error) {
	rs := []store.Routine{}
	err := routines(s.db, &rs).Where("r.user_id = ?", userID).Order("r.position", "r.id").Scan(ctx)
	fillRoutines(rs)
	return rs, mapErr(err)
}

func (s *Store) GetRoutine(ctx context.Context, userID, id string) (store.Routine, error) {
	return getRoutine(ctx, s.db, userID, id)
}

func getRoutine(ctx context.Context, db bun.IDB, userID, id string) (store.Routine, error) {
	var r store.Routine
	if err := routines(db, &r).Where("r.id = ? AND r.user_id = ?", id, userID).Scan(ctx); err != nil {
		return store.Routine{}, mapErr(err)
	}
	rs := []store.Routine{r}
	fillRoutines(rs)
	return rs[0], nil
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

func fillRoutines(rs []store.Routine) {
	for i := range rs {
		for j := range rs[i].Exercises {
			re := &rs[i].Exercises[j]
			if re.Exercise != nil {
				flags(re.Exercise)
			}
			if re.Sets == nil {
				re.Sets = []store.RoutineSet{}
			}
		}
		if rs[i].Exercises == nil {
			rs[i].Exercises = []store.RoutineExercise{}
		}
	}
}

func (s *Store) CreateRoutine(ctx context.Context, r store.Routine) (store.Routine, error) {
	r.ID, r.CreatedAt = ids.New(), now()
	r.UpdatedAt = r.CreatedAt
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().Model(&r).
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

func (s *Store) UpdateRoutine(ctx context.Context, r store.Routine) (store.Routine, error) {
	r.UpdatedAt = now()
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		res, err := tx.NewUpdate().Model(&r).Column("folder_id", "name", "notes", "updated_at").
			Where("id = ? AND user_id = ?", r.ID, r.UserID).Exec(ctx)
		if err := affected(res, err); err != nil {
			return err
		}
		// The sets go with their exercises (ON DELETE CASCADE).
		if _, err := tx.NewDelete().Model((*store.RoutineExercise)(nil)).Where("routine_id = ?", r.ID).Exec(ctx); err != nil {
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
func insertExercises(ctx context.Context, tx bun.Tx, r store.Routine) error {
	if len(r.Exercises) == 0 {
		return nil
	}
	sets := []store.RoutineSet{}
	for i := range r.Exercises {
		re := &r.Exercises[i]
		re.ID, re.RoutineID, re.Position = ids.New(), r.ID, i
		for j, set := range re.Sets {
			set.RoutineExerciseID, set.Position = re.ID, j
			sets = append(sets, set)
		}
	}
	if _, err := tx.NewInsert().Model(&r.Exercises).Exec(ctx); err != nil {
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
	res, err := s.db.NewDelete().Model((*store.Routine)(nil)).Where("id = ? AND user_id = ?", id, userID).Exec(ctx)
	return affected(res, err)
}

func (s *Store) Reorder(ctx context.Context, userID string, folders, routines []string) error {
	return s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		for _, l := range []struct {
			model any
			ids   []string
		}{{(*store.Folder)(nil), folders}, {(*store.Routine)(nil), routines}} {
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
		return store.ErrNotFound
	}
	return nil
}
