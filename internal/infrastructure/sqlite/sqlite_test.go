package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(context.Background(), filepath.Join(t.TempDir(), "kipitfit.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestUsers(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	u, err := s.CreateUser(ctx, domain.User{Email: "a@b.c", Name: "A", PasswordHash: "h", IsAdmin: true})
	if err != nil {
		t.Fatal(err)
	}
	if u.ID == "" || u.Unit != domain.UnitKg || u.CreatedAt.IsZero() {
		t.Fatalf("created user = %+v, want id, kg, created_at", u)
	}
	if _, err := s.CreateUser(ctx, domain.User{Email: "a@b.c", PasswordHash: "h"}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate email: err = %v, want ErrConflict", err)
	}
	u.Name, u.Language, u.Unit = "Al", "fr", domain.UnitLb
	if err := s.UpdateUser(ctx, u); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetUserByEmail(ctx, "a@b.c")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Al" || got.Language != "fr" || got.Unit != domain.UnitLb || !got.IsAdmin || !got.CreatedAt.Equal(u.CreatedAt) {
		t.Fatalf("read back %+v, want %+v", got, u)
	}
	if n, _ := s.CountUsers(ctx); n != 1 {
		t.Fatalf("CountUsers = %d, want 1", n)
	}
	if _, err := s.GetUser(ctx, "nope"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetUser(unknown): err = %v, want ErrNotFound", err)
	}
}

func TestSessions(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	u, _ := s.CreateUser(ctx, domain.User{Email: "a@b.c", PasswordHash: "h"})
	_ = s.CreateSession(ctx, domain.Session{TokenHash: "live", UserID: u.ID, ExpiresAt: time.Now().Add(time.Hour)})
	_ = s.CreateSession(ctx, domain.Session{TokenHash: "old", UserID: u.ID, ExpiresAt: time.Now().Add(-time.Hour)})
	if got, err := s.GetSession(ctx, "live"); err != nil || got.UserID != u.ID {
		t.Fatalf("GetSession(live) = %+v, %v", got, err)
	}
	if _, err := s.GetSession(ctx, "old"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expired session: err = %v, want ErrNotFound", err)
	}
	if err := s.DeleteSession(ctx, "live"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetSession(ctx, "live"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("deleted session: err = %v, want ErrNotFound", err)
	}
}

func TestInvites(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	admin, _ := s.CreateUser(ctx, domain.User{Email: "admin@b.c", PasswordHash: "h", IsAdmin: true})
	inv, err := s.CreateInvite(ctx, domain.Invite{TokenHash: "t1", Email: "new@b.c", CreatedBy: admin.ID, ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = s.CreateInvite(ctx, domain.Invite{TokenHash: "t2", Email: "late@b.c", CreatedBy: admin.ID, ExpiresAt: time.Now().Add(-time.Hour)})
	if list, _ := s.ListInvites(ctx); len(list) != 1 || list[0].ID != inv.ID {
		t.Fatalf("ListInvites = %+v, want only the live one", list)
	}

	u, err := s.AcceptInvite(ctx, "t1", domain.User{Email: "ignored@b.c", PasswordHash: "h", IsAdmin: true})
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "new@b.c" || u.IsAdmin {
		t.Fatalf("accepted user = %+v, want the invite's email, not admin", u)
	}
	if _, err := s.AcceptInvite(ctx, "t1", domain.User{PasswordHash: "h"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("reused invite: err = %v, want ErrNotFound", err)
	}
	if _, err := s.AcceptInvite(ctx, "t2", domain.User{PasswordHash: "h"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expired invite: err = %v, want ErrNotFound", err)
	}
	if list, _ := s.ListInvites(ctx); len(list) != 0 {
		t.Fatalf("ListInvites after accept = %+v, want none", list)
	}
	if err := s.DeleteInvite(ctx, "nope"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("DeleteInvite(unknown): err = %v, want ErrNotFound", err)
	}
}
