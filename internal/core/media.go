package core

import (
	"context"
	"io"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

// MediaOn reports whether a media bucket is configured.
func (c *Core) MediaOn() bool { return c.media != nil }

// Media opens a pack picture; not found when media is off or key isn't one.
func (c *Core) Media(ctx context.Context, key string) (io.ReadSeekCloser, domain.MediaInfo, error) {
	if c.media == nil || !domain.ValidMediaKey(key) {
		return nil, domain.MediaInfo{}, domain.ErrNotFound
	}
	return c.media.Get(ctx, key)
}
