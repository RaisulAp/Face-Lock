// Package postgres wraps pgxpool with the retry/backoff and health-check
// behavior Fase 0 requires (§ 5.1 cold start, § 6 E2/E3).
package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// connectBackoff is the exact sequence from Fase 0 § 5.1: "retry 5×,
// backoff 1s/2s/4s/8s/16s → gagal ⇒ exit(1)".
var connectBackoff = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	16 * time.Second,
}

// Connect builds a pgxpool.Pool and retries with the locked backoff
// schedule until a connection succeeds or the schedule is exhausted.
func Connect(ctx context.Context, databaseURL string, maxConns int32, log *slog.Logger) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse DATABASE_URL: %w", err)
	}
	cfg.MaxConns = maxConns
	cfg.HealthCheckPeriod = 30 * time.Second

	var pool *pgxpool.Pool
	var lastErr error

	attempts := len(connectBackoff) + 1 // first try + 5 retries
	for attempt := 1; attempt <= attempts; attempt++ {
		pool, lastErr = pgxpool.NewWithConfig(ctx, cfg)
		if lastErr == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			lastErr = pool.Ping(pingCtx)
			cancel()
			if lastErr == nil {
				return pool, nil
			}
			pool.Close()
		}

		if attempt > len(connectBackoff) {
			break
		}
		wait := connectBackoff[attempt-1]
		log.Warn("postgres: connect attempt failed, retrying",
			slog.Int("attempt", attempt),
			slog.String("wait", wait.String()),
			slog.String("error", lastErr.Error()),
		)
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("postgres: could not connect after %d attempts: %w", attempts, lastErr)
}

// Ping is used by GET /readyz with its own 2-second budget (Fase 0 § 4).
func Ping(ctx context.Context, pool *pgxpool.Pool) (time.Duration, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err := pool.Ping(ctx)
	return time.Since(start), err
}
