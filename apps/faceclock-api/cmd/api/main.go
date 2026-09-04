// Command api is the faceclock-api entry point: load config, connect
// dependencies, run migrations (dev only), serve HTTP, shut down gracefully
// (Fase 0 § 5.1, § 5.3).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/config"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/health"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/inference"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/platform/logger"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/platform/postgres"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/storage"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/version"
)

func main() {
	// config.Load() failing is exactly E1: report every missing/invalid
	// env var at once and exit non-zero, before any dependency is touched.
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "faceclock-api: fatal config error:", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel, os.Stdout)
	log.Info("starting faceclock-api",
		slog.String("version", version.Version),
		slog.String("commit", version.Commit),
		slog.String("app_env", cfg.AppEnv),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Connect(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns, log)
	if err != nil {
		log.Error("could not connect to postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()
	log.Info("connected to postgres")

	if cfg.AppEnv == "development" {
		if err := runMigrations(cfg.DatabaseURL, log); err != nil {
			log.Error("migration failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	store, err := storage.NewLocalStore(cfg.StorageLocalPath)
	if err != nil {
		log.Error("could not initialize storage", slog.String("error", err.Error()))
		os.Exit(1)
	}
	_ = store // wired into handlers starting Fase 3; kept here so main.go already proves it constructs.

	inferenceClient := inference.New(cfg.InferenceBaseURL, cfg.InferenceToken, cfg.InferenceTimeout)

	healthHandler := &health.Handler{DB: pool, Inference: inferenceClient}

	router := httpx.NewRouter(httpx.RouterDeps{
		Logger:             log,
		CORSAllowedOrigins: cfg.CORSAllowedOrigins,
		RequestTimeout:     cfg.RequestTimeout,
		MaxBodyBytes:       cfg.MaxBodyBytes,
		HealthzHandler:     healthHandler.Healthz,
		ReadyzHandler:      healthHandler.Readyz,
		VersionHandler:     version.Handler,
	})

	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("listening", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		log.Error("server error", slog.String("error", err.Error()))
		os.Exit(1)
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	// Fase 0 § 5.3: drain in-flight requests for up to 15s, then give up.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Warn("graceful shutdown deadline exceeded", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log.Info("shutdown complete")
}

func runMigrations(databaseURL string, log *slog.Logger) error {
	// golang-migrate's pgx/v5 driver wants a database/sql handle, not a
	// pgxpool.Pool — stdlib.OpenDB adapts a pgx connection config to one
	// without pulling in lib/pq as a second Postgres driver.
	connConfig, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("migrate: parse DATABASE_URL: %w", err)
	}
	sqlDB := stdlib.OpenDB(*connConfig)
	defer sqlDB.Close()

	driver, err := migratepgx.WithInstance(sqlDB, &migratepgx.Config{})
	if err != nil {
		return fmt.Errorf("migrate: open driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "pgx", driver)
	if err != nil {
		return fmt.Errorf("migrate: init: %w", err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("migrations: no change")
			return nil
		}
		// E5: a missing Postgres extension (e.g. `vector` not baked into
		// the image) surfaces here — make the image mismatch explicit
		// rather than a bare driver error.
		return fmt.Errorf("migrate up failed — if this mentions extension \"vector\", "+
			"confirm the postgres image is pgvector/pgvector:pg17, not a plain postgres image: %w", err)
	}

	log.Info("migrations applied")
	return nil
}
