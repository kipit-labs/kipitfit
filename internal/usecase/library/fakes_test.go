package library

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

// fakeLibrary is the library tables in memory: what was synced, and what the
// last pack import was asked to do.
type fakeLibrary struct {
	domain.ExerciseRepository // only ListExercises is used

	exercises []domain.Exercise
	version   string
	pack      *domain.Pack
	syncs     int
	// replaced and repoint are the last ReplaceLibrary's arguments.
	replaced []domain.Exercise
	repoint  map[string]string
}

func (f *fakeLibrary) ListExercises(_ context.Context, ownerID string) ([]domain.Exercise, error) {
	out := []domain.Exercise{}
	for _, e := range f.exercises {
		if e.OwnerID == "" || e.OwnerID == ownerID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeLibrary) SyncLibrary(_ context.Context, version string, exs []domain.Exercise) (bool, error) {
	if version == f.version {
		return false, nil
	}
	f.version, f.exercises, f.syncs = version, exs, f.syncs+1
	return true, nil
}

func (f *fakeLibrary) LibraryPack(context.Context) (domain.Pack, error) {
	if f.pack == nil {
		return domain.Pack{}, domain.ErrNotFound
	}
	return *f.pack, nil
}

func (f *fakeLibrary) ReplaceLibrary(_ context.Context, p domain.Pack, exs []domain.Exercise, repoint map[string]string) (domain.Replaced, error) {
	f.pack, f.replaced, f.repoint, f.exercises = &p, exs, repoint, exs
	return domain.Replaced{Exercises: len(exs), Repointed: len(repoint)}, nil
}

type fakeDataset []domain.Exercise

func (fakeDataset) Version() string                    { return "v1" }
func (d fakeDataset) Load() ([]domain.Exercise, error) { return d, nil }

// fakeMedia is a bucket in memory.
type fakeMedia struct {
	mu   sync.Mutex
	objs map[string]string
	puts int
}

func (m *fakeMedia) Stat(_ context.Context, key string) (domain.MediaInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.objs[key]
	if !ok {
		return domain.MediaInfo{}, domain.ErrNotFound
	}
	return domain.MediaInfo{Size: int64(len(o))}, nil
}

func (m *fakeMedia) Get(ctx context.Context, key string) (io.ReadSeekCloser, domain.MediaInfo, error) {
	info, err := m.Stat(ctx, key)
	if err != nil {
		return nil, info, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return nopCloser{bytes.NewReader([]byte(m.objs[key]))}, info, nil
}

func (m *fakeMedia) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objs[key] = string(b)
	m.puts++
	return nil
}

type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

func file(key string) domain.PackFile {
	return domain.PackFile{Key: key, ContentType: "image/webp", Size: 4,
		Open: func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("webp")), nil }}
}
