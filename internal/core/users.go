package core

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

const InviteTTL = 7 * 24 * time.Hour

// Profile is a change to the signed-in user's own settings; nil fields stay.
type Profile struct {
	Name     *string      `json:"name"`
	Language *string      `json:"language"`
	Unit     *domain.Unit `json:"unit"`
}

func (c *Core) UpdateMe(ctx context.Context, me domain.User, p Profile) (domain.User, error) {
	if p.Name != nil {
		me.Name = strings.TrimSpace(*p.Name)
		if utf8.RuneCountInString(me.Name) > maxNameLength {
			return domain.User{}, ErrNameTooLong
		}
	}
	if p.Language != nil {
		if l := *p.Language; l != "" && l != "en" && l != "fr" {
			return domain.User{}, ErrInvalidLanguage
		}
		me.Language = *p.Language
	}
	if p.Unit != nil {
		if *p.Unit != domain.UnitKg && *p.Unit != domain.UnitLb {
			return domain.User{}, ErrInvalidUnit
		}
		me.Unit = *p.Unit
	}
	return me, c.store.UpdateUser(ctx, me)
}

// InviteLink is a new invite with the link to hand to the person; the link
// can't be shown again (only its hash is stored).
type InviteLink struct {
	domain.Invite
	Link string `json:"link"`
}

func (c *Core) CreateInvite(ctx context.Context, me domain.User, email string) (InviteLink, error) {
	if !me.IsAdmin {
		return InviteLink{}, ErrNotAdmin
	}
	email, err := normalizeEmail(email)
	if err != nil {
		return InviteLink{}, err
	}
	if _, err := c.store.GetUserByEmail(ctx, email); err == nil {
		return InviteLink{}, ErrEmailTaken
	} else if !errors.Is(err, domain.ErrNotFound) {
		return InviteLink{}, err
	}
	token := randomToken(32)
	i, err := c.store.CreateInvite(ctx, domain.Invite{
		TokenHash: hashToken(token),
		Email:     email,
		CreatedBy: me.ID,
		ExpiresAt: time.Now().Add(InviteTTL),
	})
	if err != nil {
		return InviteLink{}, err
	}
	return InviteLink{Invite: i, Link: c.cfg.AppURL + "/invite?token=" + url.QueryEscape(token)}, nil
}

func (c *Core) ListInvites(ctx context.Context, me domain.User) ([]domain.Invite, error) {
	if !me.IsAdmin {
		return nil, ErrNotAdmin
	}
	return c.store.ListInvites(ctx)
}

func (c *Core) DeleteInvite(ctx context.Context, me domain.User, id string) error {
	if !me.IsAdmin {
		return ErrNotAdmin
	}
	return c.store.DeleteInvite(ctx, id)
}

// AcceptInvite creates the invited account and returns it with a session token.
func (c *Core) AcceptInvite(ctx context.Context, token, name, password string) (domain.User, string, error) {
	u, err := c.newUser("", name, password)
	if err != nil {
		return domain.User{}, "", err
	}
	u, err = c.store.AcceptInvite(ctx, hashToken(token), u)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return domain.User{}, "", ErrInvalidInvite
	case errors.Is(err, domain.ErrConflict): // the email got an account since
		return domain.User{}, "", ErrEmailTaken
	case err != nil:
		return domain.User{}, "", err
	}
	session, err := c.newSession(ctx, u.ID)
	return u, session, err
}
