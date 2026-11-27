package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/kipit-labs/kipitfit/internal/config"
	"github.com/kipit-labs/kipitfit/internal/core"
	"github.com/kipit-labs/kipitfit/internal/exercisedb"
	"github.com/kipit-labs/kipitfit/internal/infrastructure/security"
	"github.com/kipit-labs/kipitfit/internal/pack"
	"github.com/kipit-labs/kipitfit/internal/store/sqlite"
)

const packUsage = `usage: kipitfit pack import [--sha256 HEX] <pack> [ZIP file or URL]

Uploads the pack's pictures to the media bucket (KIPITFIT_S3_*) and makes its
exercises the library. Safe to run again: pictures already uploaded are
skipped. Packs: `

// packCmd runs `kipitfit pack ...`.
func packCmd(args []string) error {
	names := make([]string, 0, len(pack.Adapters))
	for n := range pack.Adapters {
		names = append(names, n)
	}
	slices.Sort(names)
	fs := flag.NewFlagSet("pack import", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprintln(os.Stderr, packUsage+strings.Join(names, ", ")) }
	want := fs.String("sha256", "", "refuse the ZIP unless its SHA-256 is this")
	if len(args) == 0 || args[0] != "import" {
		fs.Usage()
		return errors.New("unknown pack command")
	}
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() < 1 || fs.NArg() > 2 {
		fs.Usage()
		return errors.New("expected a pack name and an optional ZIP")
	}
	ad, ok := pack.Adapters[fs.Arg(0)]
	if !ok {
		return fmt.Errorf("unknown pack %q (known: %s)", fs.Arg(0), strings.Join(names, ", "))
	}
	src := fs.Arg(1)
	if src == "" {
		src = ad.URL
	}
	if src == "" {
		return fmt.Errorf("%s is bought from its vendor: give the path or URL of your ZIP", fs.Arg(0))
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx := context.Background()
	m, err := openMedia(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return err
	}

	path, sum, cleanup, err := fetch(ctx, src, cfg.DataDir)
	if err != nil {
		return err
	}
	defer cleanup()
	fmt.Printf("sha256 %s  %s\n", sum, src)
	if *want != "" && !strings.EqualFold(*want, sum) {
		return fmt.Errorf("SHA-256 mismatch: want %s", *want)
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("%s: %w", src, err)
	}
	defer zr.Close()
	p, err := ad.Read(&zr.Reader)
	if err != nil {
		return err
	}

	st, err := sqlite.Open(ctx, cfg.DBPath())
	if err != nil {
		return err
	}
	defer st.Close()
	start := time.Now()
	res, err := core.New(cfg, st, security.NewBcryptHasher(0), exercisedb.Dataset{}, m, log).ImportPack(ctx, p)
	if err != nil {
		return err
	}
	fmt.Printf("%s %s: %d exercises; pictures %d uploaded, %d already there; %d routine and workout entries moved to the new exercises; "+
		"%d old exercises archived; took %s\n",
		p.Name, p.Version, res.Exercises, res.Uploaded, res.Skipped, res.Repointed, res.Archived, time.Since(start).Round(time.Second))
	if p.Credit != "" {
		fmt.Printf("Licence: show %q, linked to %s (kipitfit shows it on the Me tab).\n", p.Credit, p.CreditURL)
	}
	return nil
}

// fetch returns a local path to src, downloading a URL into dir, and the ZIP's SHA-256.
func fetch(ctx context.Context, src, dir string) (path, sum string, cleanup func(), err error) {
	cleanup = func() {}
	h := sha256.New()
	if !strings.HasPrefix(src, "https://") && !strings.HasPrefix(src, "http://") {
		f, err := os.Open(src)
		if err != nil {
			return "", "", cleanup, err
		}
		defer f.Close()
		if _, err := io.Copy(h, f); err != nil {
			return "", "", cleanup, err
		}
		return src, hex.EncodeToString(h.Sum(nil)), cleanup, nil
	}
	cl := &http.Client{Timeout: 30 * time.Minute}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		return "", "", cleanup, err
	}
	resp, err := cl.Do(req)
	if err != nil {
		return "", "", cleanup, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", cleanup, fmt.Errorf("download %s: %s", src, resp.Status)
	}
	// Into the data dir, on the volume: the image may have no writable /tmp.
	f, err := os.CreateTemp(dir, "pack-*.zip")
	if err != nil {
		return "", "", cleanup, err
	}
	cleanup = func() { _ = os.Remove(f.Name()) }
	_, err = io.Copy(io.MultiWriter(f, h), resp.Body)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		cleanup()
		return "", "", func() {}, err
	}
	return f.Name(), hex.EncodeToString(h.Sum(nil)), cleanup, nil
}
