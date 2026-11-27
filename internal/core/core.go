// Package core holds kipitfit's logic. The API calls it; it reaches the
// database, the bucket and the password hasher only through domain ports.
package core

import (
	"log/slog"
	"sync"

	"github.com/kipit-labs/kipitfit/internal/config"
	"github.com/kipit-labs/kipitfit/internal/domain"
)

// Store is every repository core uses.
type Store interface {
	domain.UserRepository
	domain.ExerciseRepository
	domain.LibraryRepository
	domain.RoutineRepository
	domain.WorkoutRepository
}

type Core struct {
	cfg     config.Config
	store   Store
	hasher  domain.PasswordHasher
	dataset domain.ExerciseDataset
	log     *slog.Logger
	// media is nil when no bucket is configured.
	media domain.MediaStore
	// dummyHash keeps login timing the same whether or not the user exists.
	dummyHash func() string

	setupMu    sync.Mutex
	setupToken string
}

// New wires core; m is nil when media is off.
func New(cfg config.Config, st Store, h domain.PasswordHasher, ds domain.ExerciseDataset, m domain.MediaStore, log *slog.Logger) *Core {
	return &Core{cfg: cfg, store: st, hasher: h, dataset: ds, media: m, log: log,
		dummyHash: sync.OnceValue(func() string {
			hash, _ := h.Hash("kipitfit-dummy-password")
			return hash
		})}
}
