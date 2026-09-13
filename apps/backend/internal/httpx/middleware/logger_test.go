package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/platform/logger"
)

// TestStructuredLogger_NeverLeaksSecretsFromRealRequest is the test required
// by Fase 0 § 11.6: a request carrying a bearer token and a password in its
// JSON body must never have either value appear anywhere in the log output,
// even though StructuredLogger logs every request that passes through it.
func TestStructuredLogger_NeverLeaksSecretsFromRealRequest(t *testing.T) {
	var buf bytes.Buffer
	base := logger.New("debug", &buf)

	handler := RequestID(StructuredLogger(base)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A well-behaved handler never logs the body or the Authorization
		// header at all — this exercises that the middleware alone is
		// already safe with a handler that does nothing special.
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader(`{"email":"a@b.com","password":"rahasia456"}`))
	req.Header.Set("Authorization", "Bearer rahasia123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	out := buf.String()
	if strings.Contains(out, "rahasia123") {
		t.Errorf("log output leaked the bearer token: %s", out)
	}
	if strings.Contains(out, "rahasia456") {
		t.Errorf("log output leaked the password: %s", out)
	}
	if !strings.Contains(out, "request started") || !strings.Contains(out, "request completed") {
		t.Errorf("expected both start and completion log lines, got: %s", out)
	}
}

// TestRedact_CatchesForbiddenKeysEvenWhenHandlerMisbehaves is the defense-in-
// depth half of the same rule: even a handler that WOULD have logged the
// secret directly (a mistake some future Fase 1/3/4 handler could make) must
// still be caught by the redaction hook in internal/platform/logger, because
// the attribute key matches the forbidden list.
func TestRedact_CatchesForbiddenKeysEvenWhenHandlerMisbehaves(t *testing.T) {
	var buf bytes.Buffer
	base := logger.New("debug", &buf)

	handler := RequestID(StructuredLogger(base)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context(), base)
		// Deliberately misbehaving: logging the raw header and body values
		// under keys that the forbidden-key filter must catch.
		log.Info("simulated buggy handler",
			slog.String("authorization", r.Header.Get("Authorization")),
			slog.String("password", "rahasia456"),
			slog.String("access_token", "rahasia123"),
			slog.String("embedding", "[0.1,0.2,0.3]"),
			slog.String("note", "catatan pribadi karyawan"),
		)
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", nil)
	req.Header.Set("Authorization", "Bearer rahasia123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	out := buf.String()
	for _, forbidden := range []string{"rahasia123", "rahasia456", "0.1,0.2,0.3", "catatan pribadi"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("redaction failed to catch forbidden value %q in output: %s", forbidden, out)
		}
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Errorf("expected redaction placeholder in output, got: %s", out)
	}
}

func TestRedact_AllowsSafeFields(t *testing.T) {
	if got := logger.Redact("similarity", "0.87"); got != "0.87" {
		t.Errorf("expected safe field to pass through unchanged, got %q", got)
	}
	if got := logger.Redact("model_version", "buffalo_l@v1"); got != "buffalo_l@v1" {
		t.Errorf("expected safe field to pass through unchanged, got %q", got)
	}
	if got := logger.Redact("refresh_token", "abc123"); got == "abc123" {
		t.Errorf("expected token field to be redacted, got unredacted value")
	}
}
