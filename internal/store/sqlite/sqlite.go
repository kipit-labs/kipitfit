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
			return fmt.Errorf("%w: %v", store.ErrConflict, err)
		}
	}
	return err
}
