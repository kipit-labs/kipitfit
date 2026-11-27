package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"unicode"

	"golang.org/x/sync/errgroup"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

// Library is what the Me tab shows about the exercise library.
type Library struct {
	// Pack is nil for the built-in library.
	Pack  *domain.Pack `json:"pack"`
	Media bool         `json:"media"`
}

func (c *Core) Library(ctx context.Context) (Library, error) {
	l := Library{Media: c.MediaOn()}
	p, err := c.store.LibraryPack(ctx)
	if err == nil {
		l.Pack = &p
	} else if !errors.Is(err, domain.ErrNotFound) {
		return Library{}, err
	}
	return l, nil
}

// Imported reports what ImportPack did.
type Imported struct {
	domain.Replaced
	Uploaded, Skipped int
}

// ImportPack uploads the pack's pictures (skipping those already in the
// bucket, so a re-run resumes) and makes its exercises the library. Routines
// and workouts on an old library exercise move to the pack exercise with the
// same name and kind; the old exercises are archived (hidden, still valid),
// so nothing that uses them breaks.
func (c *Core) ImportPack(ctx context.Context, p domain.ParsedPack) (Imported, error) {
	if c.media == nil {
		return Imported{}, errors.New("media is off: set KIPITFIT_S3_ENDPOINT and the other KIPITFIT_S3_* settings")
	}
	if len(p.Exercises) == 0 {
		return Imported{}, errors.New("the pack has no exercises")
	}
	var res Imported
	exs := make([]domain.Exercise, len(p.Exercises))
	var files []domain.PackFile
	for i, e := range p.Exercises {
		e.Media = []string{}
		for _, f := range e.Files {
			if !domain.ValidMediaKey(f.Key) {
				return Imported{}, fmt.Errorf("pack media key %q isn't servable", f.Key)
			}
			e.Media = append(e.Media, f.Key)
			files = append(files, f)
		}
		exs[i] = e.Exercise
	}
	var uploaded, skipped atomic.Int64
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(8)
	for _, f := range files {
		g.Go(func() error {
			if _, err := c.media.Stat(gctx, f.Key); err == nil {
				skipped.Add(1)
				return nil
			} else if !errors.Is(err, domain.ErrNotFound) {
				return fmt.Errorf("%s: %w", f.Key, err)
			}
			r, err := f.Open()
			if err != nil {
				return err
			}
			defer r.Close()
			if err := c.media.Put(gctx, f.Key, r, f.Size, f.ContentType); err != nil {
				return fmt.Errorf("upload %s: %w", f.Key, err)
			}
			uploaded.Add(1)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return Imported{}, err
	}
	res.Uploaded, res.Skipped = int(uploaded.Load()), int(skipped.Load())

	repoint, err := c.repoint(ctx, exs)
	if err != nil {
		return Imported{}, err
	}
	res.Replaced, err = c.store.ReplaceLibrary(ctx, p.Pack, exs, repoint)
	return res, err
}

// repoint matches each current library exercise the pack doesn't have to the
// pack exercise with the same name (case, spaces and punctuation ignored) and
// kind, so its routine targets and logged sets still fit.
func (c *Core) repoint(ctx context.Context, exs []domain.Exercise) (map[string]string, error) {
	current, err := c.store.ListExercises(ctx, "") // no owner: the library only
	if err != nil {
		return nil, err
	}
	bySlug := map[string]bool{}
	byName := map[string]domain.Exercise{}
	for _, e := range exs {
		bySlug[e.Slug] = true
		if _, dup := byName[normName(e.Name)]; !dup {
			byName[normName(e.Name)] = e
		}
	}
	m := map[string]string{}
	for _, e := range current {
		if e.Slug == "" || bySlug[e.Slug] {
			continue
		}
		if to, ok := byName[normName(e.Name)]; ok && to.Kind == e.Kind {
			m[e.Slug] = to.Slug
		}
	}
	return m, nil
}

// normName keeps lower-case letters and digits: "Pull-Up" and "pull up" match.
func normName(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
}
