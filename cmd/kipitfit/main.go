package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kipit-labs/kipitfit/internal/api"
	"github.com/kipit-labs/kipitfit/internal/config"
	"github.com/kipit-labs/kipitfit/internal/domain"
	"github.com/kipit-labs/kipitfit/internal/exercisedb"
	"github.com/kipit-labs/kipitfit/internal/infrastructure/security"
	"github.com/kipit-labs/kipitfit/internal/media"
	"github.com/kipit-labs/kipitfit/internal/store/sqlite"
	"github.com/kipit-labs/kipitfit/internal/usecase/auth"
	"github.com/kipit-labs/kipitfit/internal/usecase/exercises"
	librarysvc "github.com/kipit-labs/kipitfit/internal/usecase/library"
	"github.com/kipit-labs/kipitfit/internal/usecase/records"
	"github.com/kipit-labs/kipitfit/internal/usecase/routines"
	"github.com/kipit-labs/kipitfit/internal/usecase/workouts"
	"github.com/kipit-labs/kipitfit/web"
)

// version is set at build time (-ldflags "-X main.version=...").
var version = "dev"

func main() {
	run := serve
	if len(os.Args) > 1 && os.Args[1] == "pack" {
		run = func() error { return packCmd(os.Args[2:]) }
	}
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// openMedia connects to the media bucket, or returns nil when none is set.
func openMedia(cfg config.Config) (domain.MediaStore, error) {
	if cfg.S3Endpoint == "" {
		return nil, nil
	}
	return media.NewS3(media.Config{Endpoint: cfg.S3Endpoint, Bucket: cfg.S3Bucket,
		AccessKey: cfg.S3AccessKeyID, SecretKey: cfg.S3SecretAccessKey, Region: cfg.S3Region})
}

func serve() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.Version = version

	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		level = slog.LevelInfo
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return err
	}
	st, err := sqlite.Open(ctx, cfg.DBPath())
	if err != nil {
		return err
	}
	defer st.Close()

	m, err := openMedia(cfg)
	if err != nil {
		return err
	}
	if m == nil {
		log.Info("media off: no KIPITFIT_S3_ENDPOINT, exercises show equipment icons")
	}
	handler, _, err := wire(ctx, cfg, st, m, log)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	errc := make(chan error, 1)
	go func() {
		log.Info("kipitfit listening", "addr", cfg.Addr, "version", version)
		errc <- srv.ListenAndServe()
	}()
	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return srv.Shutdown(shutdown)
}

// wire builds the app on an open store: the use cases, the API and the web
// app behind the real-IP and security-header middleware. m is nil when media
// is off. It returns the library too, for the pack import.
func wire(ctx context.Context, cfg config.Config, st *sqlite.Store, m domain.MediaStore, log *slog.Logger) (http.Handler, *librarysvc.Service, error) {
	authSvc := auth.NewService(st, security.NewBcryptHasher(0), cfg.SetupToken, cfg.AppURL, log)
	if err := authSvc.InitAuth(ctx); err != nil {
		return nil, nil, err
	}
	lib := librarysvc.NewService(st, st, exercisedb.Dataset{}, m, log)
	if err := lib.SyncExercises(ctx); err != nil {
		return nil, nil, err
	}
	ex := exercises.NewService(st)
	rec := records.NewService(st, ex)
	mux := http.NewServeMux()
	mux.Handle("/api/", api.New(api.Services{
		Auth:      authSvc,
		Exercises: ex,
		Library:   lib,
		Records:   rec,
		Routines:  routines.NewService(st, ex),
		Workouts:  workouts.NewService(st, st, ex, rec),
	}, cfg, log))
	mux.Handle("/", web.Handler())
	return api.RealIP(cfg.RealIPHeader, cfg.TrustedProxies)(api.SecureHeaders(mux, cfg.CookieSecure)), lib, nil
}
