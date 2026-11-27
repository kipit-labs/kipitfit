package auth

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/url"
	"testing"
	"time"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

const pw = "correct horse"

func newService(t *testing.T) (*Service, *fakeUsers) {
	t.Helper()
	users := newFakeUsers()
	c := NewService(users, fakeHasher{}, "setup", "https://fit.example", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := c.InitAuth(context.Background()); err != nil {
		t.Fatal(err)
	}
	return c, users
}

func newAdmin(t *testing.T, c *Service) domain.User {
	t.Helper()
	u, _, err := c.Setup(context.Background(), "setup", "Admin@Example.com ", "Admin", pw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestSetup(t *testing.T) {
	ctx := context.Background()
	c, _ := newService(t)
	if _, _, err := c.Setup(ctx, "wrong", "a@b.c", "", pw); !errors.Is(err, ErrWrongSetupToken) {
		t.Fatalf("wrong token: err = %v", err)
	}
	if _, _, err := c.Setup(ctx, "setup", "a@b.c", "", "short"); !errors.Is(err, errPasswordShort) {
		t.Fatalf("short password: err = %v", err)
	}
	if _, _, err := c.Setup(ctx, "setup", "not an email", "", pw); !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("bad email: err = %v", err)
	}
	u := newAdmin(t, c)
	if !u.IsAdmin || u.Email != "admin@example.com" {
		t.Fatalf("admin = %+v", u)
	}
	if _, _, err := c.Setup(ctx, "setup", "b@b.c", "", pw); !errors.Is(err, ErrSetupDone) {
		t.Fatalf("second setup: err = %v, want ErrSetupDone", err)
	}
	if st, _ := c.AuthState(ctx, ""); st.SetupRequired || st.User != nil {
		t.Fatalf("state after setup = %+v", st)
	}
}

func TestLogin(t *testing.T) {
	ctx := context.Background()
	c, _ := newService(t)
	newAdmin(t, c)
	if _, _, err := c.Login(ctx, "admin@example.com", "wrong password"); !errors.Is(err, ErrWrongCredentials) {
		t.Fatalf("bad password: err = %v", err)
	}
	if _, _, err := c.Login(ctx, "nobody@example.com", pw); !errors.Is(err, ErrWrongCredentials) {
		t.Fatalf("unknown user: err = %v", err)
	}
	u, token, err := c.Login(ctx, "ADMIN@example.com", pw)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := c.Authenticate(ctx, token); err != nil || got.ID != u.ID {
		t.Fatalf("Authenticate = %+v, %v", got, err)
	}
	if err := c.Logout(ctx, token); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Authenticate(ctx, token); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("after logout: err = %v", err)
	}
}

func inviteToken(t *testing.T, link string) string {
	t.Helper()
	u, err := url.Parse(link)
	if err != nil || u.Host != "fit.example" || u.Path != "/invite" {
		t.Fatalf("link = %q", link)
	}
	return u.Query().Get("token")
}

func TestInvites(t *testing.T) {
	ctx := context.Background()
	c, st := newService(t)
	admin := newAdmin(t, c)

	if _, err := c.CreateInvite(ctx, admin, admin.Email); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("invite an existing user: err = %v", err)
	}
	inv, err := c.CreateInvite(ctx, admin, "Friend@Example.com")
	if err != nil {
		t.Fatal(err)
	}
	token := inviteToken(t, inv.Link)
	if _, _, err := c.AcceptInvite(ctx, "bogus", "Friend", pw); !errors.Is(err, ErrInvalidInvite) {
		t.Fatalf("bogus token: err = %v", err)
	}
	friend, session, err := c.AcceptInvite(ctx, token, "Friend", pw)
	if err != nil {
		t.Fatal(err)
	}
	if friend.Email != "friend@example.com" || friend.IsAdmin || session == "" {
		t.Fatalf("friend = %+v", friend)
	}
	if _, _, err := c.AcceptInvite(ctx, token, "Again", pw); !errors.Is(err, ErrInvalidInvite) {
		t.Fatalf("used invite: err = %v", err)
	}

	if _, err := c.CreateInvite(ctx, friend, "x@example.com"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("non-admin invite: err = %v", err)
	}
	if _, err := c.ListInvites(ctx, friend); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("non-admin list: err = %v", err)
	}

	// An expired invite is refused.
	if _, err := st.CreateInvite(ctx, domain.Invite{TokenHash: hashToken("late"), Email: "late@example.com", CreatedBy: admin.ID, ExpiresAt: time.Now().Add(-time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.AcceptInvite(ctx, "late", "Late", pw); !errors.Is(err, ErrInvalidInvite) {
		t.Fatalf("expired invite: err = %v", err)
	}
}

func TestUpdateMe(t *testing.T) {
	ctx := context.Background()
	c, _ := newService(t)
	admin := newAdmin(t, c)
	bad := "de"
	if _, err := c.UpdateMe(ctx, admin, Profile{Language: &bad}); !errors.Is(err, ErrInvalidLanguage) {
		t.Fatalf("language de: err = %v", err)
	}
	fr, lb := "fr", domain.UnitLb
	u, err := c.UpdateMe(ctx, admin, Profile{Language: &fr, Unit: &lb})
	if err != nil || u.Language != "fr" || u.Unit != lb || u.Name != "Admin" {
		t.Fatalf("UpdateMe = %+v, %v", u, err)
	}
}
