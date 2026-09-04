package config

import (
	"os"
	"strings"
	"testing"
)

// clearEnv wipes every variable Load() reads, so each test starts from a
// known-empty environment instead of inheriting whatever the host has set.
func clearEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"APP_ENV", "APP_PORT", "APP_LOG_LEVEL",
		"DATABASE_URL", "DATABASE_MAX_CONNS",
		"CORS_ALLOWED_ORIGINS", "REQUEST_TIMEOUT_SECONDS", "MAX_BODY_BYTES",
		"STORAGE_DRIVER", "STORAGE_LOCAL_PATH", "STORAGE_PUBLIC_BASE_URL",
		"INFERENCE_BASE_URL", "INFERENCE_TOKEN", "INFERENCE_TIMEOUT_MS",
		"JWT_SECRET", "VERSION", "COMMIT", "BUILD_TIME",
	}
	for _, k := range keys {
		t.Setenv(k, "")
		os.Unsetenv(k) //nolint:errcheck // best-effort cleanup for empty-string edge case
	}
}

func validEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/faceclock?sslmode=disable")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")
	t.Setenv("INFERENCE_BASE_URL", "http://faceclock-inference:8000")
}

func TestLoad_Success(t *testing.T) {
	clearEnv(t)
	validEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.AppEnv != "development" {
		t.Errorf("expected default APP_ENV=development, got %q", cfg.AppEnv)
	}
	if cfg.DatabaseMaxConns != 10 {
		t.Errorf("expected default DATABASE_MAX_CONNS=10, got %d", cfg.DatabaseMaxConns)
	}
	if len(cfg.CORSAllowedOrigins) != 1 || cfg.CORSAllowedOrigins[0] != "http://localhost:5173" {
		t.Errorf("unexpected CORS origins: %v", cfg.CORSAllowedOrigins)
	}
}

// TestLoad_ReportsAllMissingAtOnce is the direct test for E1: a run with
// several missing/invalid variables must report all of them in one error,
// not fail on the first and hide the rest.
func TestLoad_ReportsAllMissingAtOnce(t *testing.T) {
	clearEnv(t)
	// Deliberately leave DATABASE_URL, CORS_ALLOWED_ORIGINS and
	// INFERENCE_BASE_URL unset, and give MAX_BODY_BYTES an invalid value.
	t.Setenv("MAX_BODY_BYTES", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	for _, want := range []string{"DATABASE_URL", "CORS_ALLOWED_ORIGINS", "INFERENCE_BASE_URL", "MAX_BODY_BYTES"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected error to mention %q, got: %v", want, err)
		}
	}
}

func TestLoad_ProductionRequiresJWTSecretAndInferenceToken(t *testing.T) {
	clearEnv(t)
	validEnv(t)
	t.Setenv("APP_ENV", "production")

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error in production without JWT_SECRET/INFERENCE_TOKEN, got nil")
	}
	for _, want := range []string{"JWT_SECRET", "INFERENCE_TOKEN"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected error to mention %q, got: %v", want, err)
		}
	}
}

func TestLoad_ProductionRejectsWildcardCORS(t *testing.T) {
	clearEnv(t)
	validEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("INFERENCE_TOKEN", "token")
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error for wildcard CORS in production, got nil")
	}
	if !strings.Contains(err.Error(), "CORS_ALLOWED_ORIGINS") {
		t.Errorf("expected error to mention CORS_ALLOWED_ORIGINS, got: %v", err)
	}
}

func TestLoad_InvalidAppEnv(t *testing.T) {
	clearEnv(t)
	validEnv(t)
	t.Setenv("APP_ENV", "bogus")

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error for invalid APP_ENV, got nil")
	}
}
