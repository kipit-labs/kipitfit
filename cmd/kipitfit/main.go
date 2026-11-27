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
	"github.com/kipit-labs/kipitfit/internal/core"
	"github.com/kipit-labs/kipitfit/internal/media"
	"github.com/kipit-labs/kipitfit/internal/store/sqlite"
	"github.com/kipit-labs/kipitfit/web"
)

// version is set at build time (-ldflags "-X main.version=...").
var version = "dev"

func main() {
	if err := serve(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// openMedia connects to the media bucket, or returns nil when none is set.
func openMedia(cfg config.Config) (media.Store, error) {
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
	c := core.New(cfg, st, m, log)
	if err := c.InitAuth(ctx); err != nil {
		return err
	}
	if err := c.SyncExercises(ctx); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", api.New(c, cfg, log))
	mux.Handle("/", web.Handler())
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.RealIP(cfg.RealIPHeader, cfg.TrustedProxies)(api.SecureHeaders(mux, cfg.CookieSecure)),
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
