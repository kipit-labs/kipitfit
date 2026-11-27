package auth

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

// fakeUsers keeps accounts, sessions and invites in memory, like the SQLite
// store: emails are unique, expired sessions and used or expired invites are
// not found.
type fakeUsers struct {
	users    map[string]domain.User
	sessions map[string]domain.Session
	invites  map[string]domain.Invite
	n        int
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{users: map[string]domain.User{}, sessions: map[string]domain.Session{}, invites: map[string]domain.Invite{}}
}

func (f *fakeUsers) id() string {
	f.n++
	return fmt.Sprint(f.n)
}

func (f *fakeUsers) CountUsers(context.Context) (int, error) { return len(f.users), nil }

func (f *fakeUsers) CreateUser(_ context.Context, u domain.User) (domain.User, error) {
	if _, err := f.GetUserByEmail(context.Background(), u.Email); err == nil {
		return domain.User{}, domain.ErrConflict
	}
	u.ID, u.CreatedAt = f.id(), time.Now()
	if u.Unit == "" {
		u.Unit = domain.UnitKg
	}
	f.users[u.ID] = u
	return u, nil
}

func (f *fakeUsers) GetUser(_ context.Context, id string) (domain.User, error) {
	u, ok := f.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeUsers) GetUserByEmail(_ context.Context, email string) (domain.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

func (f *fakeUsers) UpdateUser(_ context.Context, u domain.User) error {
	old, ok := f.users[u.ID]
	if !ok {
		return domain.ErrNotFound
	}
	old.Name, old.Language, old.Unit = u.Name, u.Language, u.Unit
	f.users[u.ID] = old
	return nil
}

func (f *fakeUsers) CreateSession(_ context.Context, s domain.Session) error {
	f.sessions[s.TokenHash] = s
	return nil
}

func (f *fakeUsers) GetSession(_ context.Context, tokenHash string) (domain.Session, error) {
	s, ok := f.sessions[tokenHash]
	if !ok || !s.ExpiresAt.After(time.Now()) {
		return domain.Session{}, domain.ErrNotFound
	}
	return s, nil
}

func (f *fakeUsers) DeleteSession(_ context.Context, tokenHash string) error {
	delete(f.sessions, tokenHash)
	return nil
}

func (f *fakeUsers) DeleteExpiredSessions(context.Context) error {
	for h, s := range f.sessions {
		if !s.ExpiresAt.After(time.Now()) {
			delete(f.sessions, h)
		}
	}
	return nil
}

func (f *fakeUsers) CreateInvite(_ context.Context, i domain.Invite) (domain.Invite, error) {
	i.ID, i.CreatedAt, i.UsedAt = f.id(), time.Now(), nil
	f.invites[i.ID] = i
	return i, nil
}

func (f *fakeUsers) ListInvites(context.Context) ([]domain.Invite, error) {
	out := []domain.Invite{}
	for _, i := range f.invites {
		if i.UsedAt == nil && i.ExpiresAt.After(time.Now()) {
			out = append(out, i)
		}
	}
	slices.SortFunc(out, func(a, b domain.Invite) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return out, nil
}

func (f *fakeUsers) AcceptInvite(ctx context.Context, tokenHash string, u domain.User) (domain.User, error) {
	for id, i := range f.invites {
		if i.TokenHash != tokenHash || i.UsedAt != nil || !i.ExpiresAt.After(time.Now()) {
			continue
		}
		u.Email, u.IsAdmin = i.Email, false
		created, err := f.CreateUser(ctx, u)
		if err != nil {
			return domain.User{}, err
		}
		now := time.Now()
		i.UsedAt = &now
		f.invites[id] = i
		return created, nil
	}
	return domain.User{}, domain.ErrNotFound
}

func (f *fakeUsers) DeleteInvite(_ context.Context, id string) error {
	if _, ok := f.invites[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.invites, id)
	return nil
}

// fakeHasher "hashes" by prefixing: fast, and enough to tell passwords apart.
type fakeHasher struct{}

func (fakeHasher) Hash(password string) (string, error) { return "hashed:" + password, nil }

func (fakeHasher) Compare(hash, password string) error {
	if !strings.HasPrefix(hash, "hashed:") || hash != "hashed:"+password {
		return errors.New("wrong password")
	}
	return nil
}
