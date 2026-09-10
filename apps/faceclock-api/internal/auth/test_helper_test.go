package auth

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/config"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/settings"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://faceclock:devpassword123@localhost:5434/faceclock?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skipping db test, cannot connect to %s: %v", dsn, err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skipping db test, ping failed: %v", err)
	}
	return pool
}

func setupTestServices(t *testing.T, pool *pgxpool.Pool) (*Service, *Handler, *TokenManager) {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:     "test-super-secret-key-that-is-at-least-32-bytes-long",
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 7 * 24 * time.Hour,
	}
	tokenMgr, err := NewTokenManager(cfg.JWTSecret, cfg.JWTAccessTTL)
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}
	auditRec := audit.NewRecorder(pool)
	settingsSvc := settings.NewService(pool)
	rbacCache := rbac.NewCache(30 * time.Second)
	rbacSvc := rbac.NewService(pool, rbacCache)
	authSvc := NewService(pool, cfg, tokenMgr, settingsSvc, rbacSvc, auditRec)
	handler := NewHandler(authSvc, auditRec)
	return authSvc, handler, tokenMgr
}
