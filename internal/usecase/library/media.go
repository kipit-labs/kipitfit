package library

import (
	"context"
	"io"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

// MediaOn reports whether a media bucket is configured.
func (s *Service) MediaOn() bool { return s.media != nil }

// Media opens a pack picture; not found when media is off or key isn't one.
func (s *Service) Media(ctx context.Context, key string) (io.ReadSeekCloser, domain.MediaInfo, error) {
	if s.media == nil || !domain.ValidMediaKey(key) {
		return nil, domain.MediaInfo{}, domain.ErrNotFound
	}
	return s.media.Get(ctx, key)
}
