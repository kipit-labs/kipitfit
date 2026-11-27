package core

import (
	"context"
	"errors"

	"github.com/kipit-labs/kipitfit/internal/media"
	"github.com/kipit-labs/kipitfit/internal/store"
)

// MediaOn reports whether a media bucket is configured.
func (c *Core) MediaOn() bool { return c.media != nil }

// Media opens a pack picture; not found when media is off or key isn't one.
func (c *Core) Media(ctx context.Context, key string) (media.Object, media.Info, error) {
	if c.media == nil || !media.ValidKey(key) {
		return nil, media.Info{}, store.ErrNotFound
	}
	obj, info, err := c.media.Get(ctx, key)
	if errors.Is(err, media.ErrNotFound) {
		return nil, media.Info{}, store.ErrNotFound
	}
	return obj, info, err
}
