// Package core holds kipitfit's logic. The API calls it; it reaches the
// database only through store.Store.
package core

import (
	"log/slog"
	"sync"

	"github.com/kipit-labs/kipitfit/internal/config"
	"github.com/kipit-labs/kipitfit/internal/store"
)

type Core struct {
	cfg   config.Config
	store store.Store
	log   *slog.Logger

	setupMu    sync.Mutex
	setupToken string
}

func New(cfg config.Config, st store.Store, log *slog.Logger) *Core {
	return &Core{cfg: cfg, store: st, log: log}
}
