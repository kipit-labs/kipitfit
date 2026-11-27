// Package core holds kipitfit's logic. The API calls it; it reaches the
// database only through store.Store.
package core

import (
	"log/slog"
	"sync"

	"github.com/kipit-labs/kipitfit/internal/config"
	"github.com/kipit-labs/kipitfit/internal/media"
	"github.com/kipit-labs/kipitfit/internal/store"
)

type Core struct {
	cfg   config.Config
	store store.Store
	log   *slog.Logger
	// media is nil when no bucket is configured.
	media media.Store

	setupMu    sync.Mutex
	setupToken string
}

// New wires core; m is nil when media is off.
func New(cfg config.Config, st store.Store, m media.Store, log *slog.Logger) *Core {
	return &Core{cfg: cfg, store: st, media: m, log: log}
}
