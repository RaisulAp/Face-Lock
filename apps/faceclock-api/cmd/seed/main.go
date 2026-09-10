package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/seeder"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Error("DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	seedEmail := os.Getenv("SEED_ADMIN_EMAIL")
	seedPassword := os.Getenv("SEED_ADMIN_PASSWORD")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		logger.Error("Failed to parse DATABASE_URL", "error", err)
		os.Exit(1)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}

	err = seeder.Run(ctx, pool, seeder.Options{
		AppEnv:            appEnv,
		SeedAdminEmail:    seedEmail,
		SeedAdminPassword: seedPassword,
		Logger:            logger,
	})
	if err != nil {
		logger.Error("Seeder execution failed", "error", err)
		os.Exit(1)
	}

	fmt.Println("Seeder completed successfully.")
}
