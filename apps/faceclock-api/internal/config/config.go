// Package config loads faceclock-api configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds every environment-derived setting the service needs at startup.
type Config struct {
	AppEnv   string // development | staging | production
	AppPort  string
	LogLevel string // debug | info | warn | error

	DatabaseURL      string
	DatabaseMaxConns int32

	CORSAllowedOrigins []string
	RequestTimeout     time.Duration
	MaxBodyBytes       int64

	StorageDriver        string // local | s3
	StorageLocalPath     string
	StoragePublicBaseURL string

	InferenceBaseURL string
	InferenceToken   string
	InferenceTimeout time.Duration

	// JWT* are read but only enforced as required starting Fase 1. Kept here
	// now so the fail-fast validation rule (E1) has one place to grow into.
	JWTSecret string

	Version   string
	Commit    string
	BuildTime string
}

// requiredVar names an environment variable that must be non-empty, together
// with the condition under which it becomes required.
type requiredVar struct {
	name      string
	value     string
	whenEmpty bool // true = missing value is itself the failure (always required)
}

// Load reads configuration from the process environment. It never returns a
// zero-value Config on error — on any validation failure the caller is
// expected to log the returned error and exit(1), per Fase 0 § 2.9: a
// service that starts with silently-defaulted secrets is a failure that is
// only discovered after it leaks.
func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:   getEnv("APP_ENV", "development"),
		AppPort:  getEnv("APP_PORT", "8080"),
		LogLevel: getEnv("APP_LOG_LEVEL", "info"),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		StorageDriver:        getEnv("STORAGE_DRIVER", "local"),
		StorageLocalPath:     getEnv("STORAGE_LOCAL_PATH", "/data/uploads"),
		StoragePublicBaseURL: os.Getenv("STORAGE_PUBLIC_BASE_URL"),

		InferenceBaseURL: os.Getenv("INFERENCE_BASE_URL"),
		InferenceToken:   os.Getenv("INFERENCE_TOKEN"),

		JWTSecret: os.Getenv("JWT_SECRET"),

		Version:   getEnv("VERSION", "0.1.0"),
		Commit:    getEnv("COMMIT", "unknown"),
		BuildTime: getEnv("BUILD_TIME", "unknown"),
	}

	var missing []string

	// Always-required variables, regardless of APP_ENV.
	always := []requiredVar{
		{"DATABASE_URL", cfg.DatabaseURL, true},
	}
	for _, v := range always {
		if v.whenEmpty && v.value == "" {
			missing = append(missing, v.name)
		}
	}

	maxConnsRaw := getEnv("DATABASE_MAX_CONNS", "10")
	maxConns, err := strconv.Atoi(maxConnsRaw)
	if err != nil || maxConns <= 0 {
		missing = append(missing, "DATABASE_MAX_CONNS (must be a positive integer)")
	} else {
		cfg.DatabaseMaxConns = int32(maxConns)
	}

	originsRaw := getEnv("CORS_ALLOWED_ORIGINS", "")
	if originsRaw == "" {
		missing = append(missing, "CORS_ALLOWED_ORIGINS")
	} else {
		for _, o := range strings.Split(originsRaw, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				cfg.CORSAllowedOrigins = append(cfg.CORSAllowedOrigins, o)
			}
		}
	}
	// E9: wildcard CORS is never acceptable outside development.
	if cfg.AppEnv != "development" {
		for _, o := range cfg.CORSAllowedOrigins {
			if o == "*" {
				missing = append(missing, "CORS_ALLOWED_ORIGINS (must not be \"*\" when APP_ENV != development)")
				break
			}
		}
	}

	timeoutSecRaw := getEnv("REQUEST_TIMEOUT_SECONDS", "30")
	timeoutSec, err := strconv.Atoi(timeoutSecRaw)
	if err != nil || timeoutSec <= 0 {
		missing = append(missing, "REQUEST_TIMEOUT_SECONDS (must be a positive integer)")
	} else {
		cfg.RequestTimeout = time.Duration(timeoutSec) * time.Second
	}

	maxBodyRaw := getEnv("MAX_BODY_BYTES", "10485760")
	maxBody, err := strconv.ParseInt(maxBodyRaw, 10, 64)
	if err != nil || maxBody <= 0 {
		missing = append(missing, "MAX_BODY_BYTES (must be a positive integer)")
	} else {
		cfg.MaxBodyBytes = maxBody
	}

	inferenceTimeoutRaw := getEnv("INFERENCE_TIMEOUT_MS", "6000")
	inferenceTimeoutMS, err := strconv.Atoi(inferenceTimeoutRaw)
	if err != nil || inferenceTimeoutMS <= 0 {
		missing = append(missing, "INFERENCE_TIMEOUT_MS (must be a positive integer)")
	} else {
		cfg.InferenceTimeout = time.Duration(inferenceTimeoutMS) * time.Millisecond
	}

	if cfg.InferenceBaseURL == "" {
		missing = append(missing, "INFERENCE_BASE_URL")
	}

	// Forward-looking fail-fast rules (Fase 0 § 2.9): these are not yet
	// exercised by any Fase 0 feature, but the RULE is locked now so Fase 1/2
	// cannot "forget" it later.
	if cfg.AppEnv != "development" {
		if cfg.JWTSecret == "" {
			missing = append(missing, "JWT_SECRET (required when APP_ENV != development, starting Fase 1)")
		}
		if cfg.InferenceToken == "" {
			missing = append(missing, "INFERENCE_TOKEN (required when APP_ENV != development, starting Fase 2)")
		}
	}

	switch cfg.AppEnv {
	case "development", "staging", "production":
	default:
		missing = append(missing, fmt.Sprintf("APP_ENV (got %q, must be development|staging|production)", cfg.AppEnv))
	}

	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		missing = append(missing, fmt.Sprintf("APP_LOG_LEVEL (got %q, must be debug|info|warn|error)", cfg.LogLevel))
	}

	switch cfg.StorageDriver {
	case "local", "s3":
	default:
		missing = append(missing, fmt.Sprintf("STORAGE_DRIVER (got %q, must be local|s3)", cfg.StorageDriver))
	}

	if len(missing) > 0 {
		// E1: report every missing/invalid variable at once, not one at a time.
		return nil, fmt.Errorf("config: invalid or missing environment variables: %s", strings.Join(missing, "; "))
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
