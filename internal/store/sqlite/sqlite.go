// Package sqlite implements the domain repositories on SQLite (modernc.org/sqlite, no CGO).
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

	"github.com/kipit-labs/kipitfit/internal/domain"
	"github.com/kipit-labs/kipitfit/internal/ids"
	"github.com/kipit-labs/kipitfit/internal/sqlitedb"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	db *bun.DB
}

var (
	_ domain.UserRepository     = (*Store)(nil)
	_ domain.ExerciseRepository = (*Store)(nil)
	_ domain.LibraryRepository  = (*Store)(nil)
	_ domain.RoutineRepository  = (*Store)(nil)
	_ domain.WorkoutRepository  = (*Store)(nil)
)

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
	n, err := s.db.NewSelect().Model((*userRow)(nil)).Count(ctx)
	return n, mapErr(err)
}

func (s *Store) CreateUser(ctx context.Context, u domain.User) (domain.User, error) {
	return createUser(ctx, s.db, u)
}

func createUser(ctx context.Context, db bun.IDB, u domain.User) (domain.User, error) {
	u.ID, u.CreatedAt = ids.New(), now()
	if u.Unit == "" {
		u.Unit = domain.UnitKg
	}
	row := newUserRow(u)
	if _, err := db.NewInsert().Model(&row).Exec(ctx); err != nil {
		return domain.User{}, mapErr(err)
	}
	return u, nil
}

func (s *Store) GetUser(ctx context.Context, id string) (domain.User, error) {
	var u userRow
	err := s.db.NewSelect().Model(&u).Where("id = ?", id).Scan(ctx)
	return u.domain(), mapErr(err)
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	var u userRow
	err := s.db.NewSelect().Model(&u).Where("email = ?", email).Scan(ctx)
	return u.domain(), mapErr(err)
}

func (s *Store) UpdateUser(ctx context.Context, u domain.User) error {
	row := newUserRow(u)
	res, err := s.db.NewUpdate().Model(&row).Column("name", "language", "unit").WherePK().Exec(ctx)
	if err != nil {
		return mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) CreateSession(ctx context.Context, sess domain.Session) error {
	row := sessionRow{TokenHash: sess.TokenHash, UserID: sess.UserID, CreatedAt: now(),
		ExpiresAt: sess.ExpiresAt.UTC().Truncate(time.Microsecond)}
	_, err := s.db.NewInsert().Model(&row).Exec(ctx)
	return mapErr(err)
}

func (s *Store) GetSession(ctx context.Context, tokenHash string) (domain.Session, error) {
	var r sessionRow
	err := s.db.NewSelect().Model(&r).
		Where("token_hash = ?", tokenHash).Where("expires_at > ?", now()).Scan(ctx)
	return domain.Session{TokenHash: r.TokenHash, UserID: r.UserID, CreatedAt: r.CreatedAt, ExpiresAt: r.ExpiresAt}, mapErr(err)
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.NewDelete().Model((*sessionRow)(nil)).Where("token_hash = ?", tokenHash).Exec(ctx)
	return mapErr(err)
}

func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.db.NewDelete().Model((*sessionRow)(nil)).Where("expires_at <= ?", now()).Exec(ctx)
	return mapErr(err)
}

func (s *Store) CreateInvite(ctx context.Context, i domain.Invite) (domain.Invite, error) {
	row := inviteRow{ID: ids.New(), TokenHash: i.TokenHash, Email: i.Email, CreatedBy: i.CreatedBy, CreatedAt: now(),
		ExpiresAt: i.ExpiresAt.UTC().Truncate(time.Microsecond)}
	if _, err := s.db.NewInsert().Model(&row).Exec(ctx); err != nil {
		return domain.Invite{}, mapErr(err)
	}
	return row.domain(), nil
}

func (s *Store) ListInvites(ctx context.Context) ([]domain.Invite, error) {
	var rows []inviteRow
	err := s.db.NewSelect().Model(&rows).Where("used_at IS NULL AND expires_at > ?", now()).Order("created_at DESC").Scan(ctx)
	is := make([]domain.Invite, len(rows))
	for i, r := range rows {
		is[i] = r.domain()
	}
	return is, mapErr(err)
}

func (s *Store) AcceptInvite(ctx context.Context, tokenHash string, u domain.User) (domain.User, error) {
	var created domain.User
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var i inviteRow
		res, err := tx.NewUpdate().Model(&i).Set("used_at = ?", now()).
			Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", tokenHash, now()).Returning("*").Exec(ctx)
		if err != nil {
			return mapErr(err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return domain.ErrNotFound
		}
		u.Email, u.IsAdmin = i.Email, false
		created, err = createUser(ctx, tx, u)
		return err
	})
	return created, err
}

func (s *Store) DeleteInvite(ctx context.Context, id string) error {
	res, err := s.db.NewDelete().Model((*inviteRow)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

const (
	libraryVersionKey = "exercise_library_version"
	libraryPackKey    = "exercise_library_pack"
)

func (s *Store) SyncLibrary(ctx context.Context, version string, exs []domain.Exercise) (bool, error) {
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
		rows := make([]exerciseRow, len(exs))
		t := now()
		for i, e := range exs {
			e.ID, e.OwnerID, e.CreatedAt, e.UpdatedAt = ids.New(), "", t, t
			rows[i] = newExerciseRow(e)
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

func (s *Store) ListExercises(ctx context.Context, ownerID string) ([]domain.Exercise, error) {
	var rows []exerciseRow
	err := s.db.NewSelect().Model(&rows).ExcludeColumn("instructions", "force", "level", "mechanic").
		Where("owner_id IS NULL OR owner_id = ?", ownerID).Where("archived_at IS NULL").
		OrderExpr("name COLLATE NOCASE").Scan(ctx)
	exs := make([]domain.Exercise, len(rows))
	for i, r := range rows {
		exs[i] = r.domain()
	}
	return exs, mapErr(err)
}

func (s *Store) GetExercise(ctx context.Context, id string) (domain.Exercise, error) {
	var r exerciseRow
	if err := s.db.NewSelect().Model(&r).Where("id = ?", id).Scan(ctx); err != nil {
		return domain.Exercise{}, mapErr(err)
	}
	return r.domain(), nil
}

func (s *Store) CreateExercise(ctx context.Context, e domain.Exercise) (domain.Exercise, error) {
	e.ID, e.Slug = ids.New(), ""
	e.CreatedAt = now()
	e.UpdatedAt = e.CreatedAt
	row := newExerciseRow(e)
	if _, err := s.db.NewInsert().Model(&row).Exec(ctx); err != nil {
		return domain.Exercise{}, mapErr(err)
	}
	return row.domain(), nil
}

func (s *Store) UpdateExercise(ctx context.Context, e domain.Exercise) (domain.Exercise, error) {
	e.UpdatedAt = now()
	row := newExerciseRow(e)
	res, err := s.db.NewUpdate().Model(&row).
		Column("name", "kind", "category", "equipment", "primary_muscles", "secondary_muscles", "updated_at").
		WherePK().Exec(ctx)
	if err != nil {
		return domain.Exercise{}, mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.Exercise{}, domain.ErrNotFound
	}
	e = row.domain()
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
			res, err = tx.NewUpdate().Model((*exerciseRow)(nil)).Where("id = ?", id).
				Set("archived_at = COALESCE(archived_at, ?)", now()).Exec(ctx)
		} else {
			res, err = tx.NewDelete().Model((*exerciseRow)(nil)).Where("id = ?", id).Exec(ctx)
		}
		return affected(res, err)
	})
}

func now() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	var se *sqlite.Error
	if errors.As(err, &se) {
		switch se.Code() {
		case sqlite3.SQLITE_CONSTRAINT_UNIQUE, sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY:
			return fmt.Errorf("%w: %w", domain.ErrConflict, err)
		}
	}
	return err
}
