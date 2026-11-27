// Package sqlite implements store.Store on SQLite (modernc.org/sqlite, no CGO).
package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/kipit-labs/kipitfit/internal/ids"
	"github.com/kipit-labs/kipitfit/internal/sqlitedb"
	"github.com/kipit-labs/kipitfit/internal/store"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	db *bun.DB
}

var _ store.Store = (*Store)(nil)

// Open opens (or creates) the database at path and applies pending migrations.
func Open(ctx context.Context, path string) (*Store, error) {
	sub, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return nil, err
	}
	sqldb, err := sqlitedb.Open(ctx, path, sub)
	if err != nil {
		return nil, err
	}
	return &Store{db: bun.NewDB(sqldb, sqlitedialect.New())}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	n, err := s.db.NewSelect().Model((*store.User)(nil)).Count(ctx)
	return n, mapErr(err)
}

func (s *Store) CreateUser(ctx context.Context, u store.User) (store.User, error) {
	return createUser(ctx, s.db, u)
}

func createUser(ctx context.Context, db bun.IDB, u store.User) (store.User, error) {
	u.ID, u.CreatedAt = ids.New(), now()
	if u.Unit == "" {
		u.Unit = store.UnitKg
	}
	if _, err := db.NewInsert().Model(&u).Exec(ctx); err != nil {
		return store.User{}, mapErr(err)
	}
	return u, nil
}

func (s *Store) GetUser(ctx context.Context, id string) (store.User, error) {
	var u store.User
	err := s.db.NewSelect().Model(&u).Where("id = ?", id).Scan(ctx)
	return u, mapErr(err)
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (store.User, error) {
	var u store.User
	err := s.db.NewSelect().Model(&u).Where("email = ?", email).Scan(ctx)
	return u, mapErr(err)
}

func (s *Store) UpdateUser(ctx context.Context, u store.User) error {
	res, err := s.db.NewUpdate().Model(&u).Column("name", "language", "unit").WherePK().Exec(ctx)
	if err != nil {
		return mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *Store) CreateSession(ctx context.Context, sess store.Session) error {
	sess.CreatedAt = now()
	sess.ExpiresAt = sess.ExpiresAt.UTC().Truncate(time.Microsecond)
	_, err := s.db.NewInsert().Model(&sess).Exec(ctx)
	return mapErr(err)
}

func (s *Store) GetSession(ctx context.Context, tokenHash string) (store.Session, error) {
	var sess store.Session
	err := s.db.NewSelect().Model(&sess).
		Where("token_hash = ?", tokenHash).Where("expires_at > ?", now()).Scan(ctx)
	return sess, mapErr(err)
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.NewDelete().Model((*store.Session)(nil)).Where("token_hash = ?", tokenHash).Exec(ctx)
	return mapErr(err)
}

func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.db.NewDelete().Model((*store.Session)(nil)).Where("expires_at <= ?", now()).Exec(ctx)
	return mapErr(err)
}

func (s *Store) CreateInvite(ctx context.Context, i store.Invite) (store.Invite, error) {
	i.ID, i.CreatedAt, i.UsedAt = ids.New(), now(), nil
	i.ExpiresAt = i.ExpiresAt.UTC().Truncate(time.Microsecond)
	if _, err := s.db.NewInsert().Model(&i).Exec(ctx); err != nil {
		return store.Invite{}, mapErr(err)
	}
	return i, nil
}

func (s *Store) ListInvites(ctx context.Context) ([]store.Invite, error) {
	is := []store.Invite{}
	err := s.db.NewSelect().Model(&is).Where("used_at IS NULL AND expires_at > ?", now()).Order("created_at DESC").Scan(ctx)
	return is, mapErr(err)
}

func (s *Store) AcceptInvite(ctx context.Context, tokenHash string, u store.User) (store.User, error) {
	var created store.User
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var i store.Invite
		res, err := tx.NewUpdate().Model(&i).Set("used_at = ?", now()).
			Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", tokenHash, now()).Returning("*").Exec(ctx)
		if err != nil {
			return mapErr(err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return store.ErrNotFound
		}
		u.Email, u.IsAdmin = i.Email, false
		created, err = createUser(ctx, tx, u)
		return err
	})
	return created, err
}

func (s *Store) DeleteInvite(ctx context.Context, id string) error {
	res, err := s.db.NewDelete().Model((*store.Invite)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return store.ErrNotFound
	}
	return nil
}

const (
	libraryVersionKey = "exercise_library_version"
	libraryPackKey    = "exercise_library_pack"
)

func (s *Store) SyncLibrary(ctx context.Context, version string, exs []store.Exercise) (bool, error) {
	wrote := false
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var current string
		err := tx.NewSelect().Table("meta").Column("value").Where("key = ?", libraryVersionKey).Scan(ctx, &current)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if current == version {
			return nil
		}
		rows := make([]store.Exercise, len(exs))
		t := now()
		for i, e := range exs {
			e.ID, e.OwnerID, e.CreatedAt, e.UpdatedAt = ids.New(), "", t, t
			rows[i] = nonNilLists(e)
		}
		// An existing slug keeps its id (workouts will point at it) and gets the new fields.
		_, err = tx.NewInsert().Model(&rows).
			On("CONFLICT (slug) DO UPDATE").
			Set("name = EXCLUDED.name, kind = EXCLUDED.kind, category = EXCLUDED.category, equipment = EXCLUDED.equipment").
			Set("primary_muscles = EXCLUDED.primary_muscles, secondary_muscles = EXCLUDED.secondary_muscles").
			Set("force = EXCLUDED.force, level = EXCLUDED.level, mechanic = EXCLUDED.mechanic").
			Set("instructions = EXCLUDED.instructions, updated_at = EXCLUDED.updated_at").
			Exec(ctx)
		if err != nil {
			return mapErr(err)
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO meta (key, value) VALUES (?, ?) ON CONFLICT (key) DO UPDATE SET value = excluded.value",
			libraryVersionKey, version)
		wrote = err == nil
		return err
	})
	return wrote, err
}

func (s *Store) ListExercises(ctx context.Context, ownerID string) ([]store.Exercise, error) {
	exs := []store.Exercise{}
	err := s.db.NewSelect().Model(&exs).ExcludeColumn("instructions", "force", "level", "mechanic").
		Where("owner_id IS NULL OR owner_id = ?", ownerID).Where("archived_at IS NULL").
		OrderExpr("name COLLATE NOCASE").Scan(ctx)
	for i := range exs {
		flags(&exs[i])
	}
	return exs, mapErr(err)
}

func (s *Store) GetExercise(ctx context.Context, id string) (store.Exercise, error) {
	var e store.Exercise
	if err := s.db.NewSelect().Model(&e).Where("id = ?", id).Scan(ctx); err != nil {
		return store.Exercise{}, mapErr(err)
	}
	flags(&e)
	return e, nil
}

func flags(e *store.Exercise) {
	e.Custom, e.Archived = e.OwnerID != "", !e.ArchivedAt.IsZero()
}

func (s *Store) CreateExercise(ctx context.Context, e store.Exercise) (store.Exercise, error) {
	e.ID, e.Slug, e.Custom = ids.New(), "", true
	e.CreatedAt = now()
	e.UpdatedAt = e.CreatedAt
	e = nonNilLists(e)
	if _, err := s.db.NewInsert().Model(&e).Exec(ctx); err != nil {
		return store.Exercise{}, mapErr(err)
	}
	return e, nil
}

func (s *Store) UpdateExercise(ctx context.Context, e store.Exercise) (store.Exercise, error) {
	e.UpdatedAt = now()
	e = nonNilLists(e)
	res, err := s.db.NewUpdate().Model(&e).
		Column("name", "kind", "category", "equipment", "primary_muscles", "secondary_muscles", "updated_at").
		WherePK().Exec(ctx)
	if err != nil {
		return store.Exercise{}, mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return store.Exercise{}, store.ErrNotFound
	}
	return e, nil
}

func (s *Store) DeleteExercise(ctx context.Context, id string) error {
	return s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var used bool
		err := tx.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM routine_exercises WHERE exercise_id = ?) "+
			"OR EXISTS (SELECT 1 FROM workout_exercises WHERE exercise_id = ?)", id, id).Scan(&used)
		if err != nil {
			return err
		}
		var res sql.Result
		if used {
			// Archiving twice keeps the first date.
			res, err = tx.NewUpdate().Model((*store.Exercise)(nil)).Where("id = ?", id).
				Set("archived_at = COALESCE(archived_at, ?)", now()).Exec(ctx)
		} else {
			res, err = tx.NewDelete().Model((*store.Exercise)(nil)).Where("id = ?", id).Exec(ctx)
		}
		return affected(res, err)
	})
}

// nonNilLists keeps JSON columns arrays: a nil slice would be stored as null.
func nonNilLists(e store.Exercise) store.Exercise {
	for _, l := range []*[]string{&e.PrimaryMuscles, &e.SecondaryMuscles, &e.Instructions, &e.Media} {
		if *l == nil {
			*l = []string{}
		}
	}
	return e
}

func now() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return store.ErrNotFound
	}
	var se *sqlite.Error
	if errors.As(err, &se) {
		switch se.Code() {
		case sqlite3.SQLITE_CONSTRAINT_UNIQUE, sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY:
			return fmt.Errorf("%w: %w", store.ErrConflict, err)
		}
	}
	return err
}
