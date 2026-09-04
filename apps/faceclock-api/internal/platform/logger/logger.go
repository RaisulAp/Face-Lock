// Package logger provides the JSON structured logger used by every
// component of faceclock-api, plus the redaction rule that keeps biometric
// and secret data out of logs (Fase 0 § 2.6, master plan § 9).
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

type ctxKey struct{}

var requestIDKey = ctxKey{}

// New builds the process-wide slog.Logger. level is one of
// debug|info|warn|error (already validated by config.Load). w defaults to
// os.Stdout when nil.
func New(level string, w io.Writer) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	if w == nil {
		w = os.Stdout
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       lvl,
		ReplaceAttr: redactAttr,
	})
	return slog.New(handler)
}

// WithRequestID returns a context carrying the given request id, so that
// FromContext can attach it to every log line emitted while handling one
// request.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// FromContext returns a logger that automatically includes request_id, if
// one was attached to ctx via WithRequestID.
func FromContext(ctx context.Context, base *slog.Logger) *slog.Logger {
	if base == nil {
		base = slog.Default()
	}
	if id, ok := ctx.Value(requestIDKey).(string); ok && id != "" {
		return base.With(slog.String("request_id", id))
	}
	return base
}

// forbiddenKeys lists slog attribute keys that must never reach a log line,
// regardless of who set them or how deeply they are nested. This is the
// implementation of Fase 0 § 2.6: byte gambar, embedding, password/token,
// dan note absensi tidak boleh pernah masuk log.
//
// Matching is substring-based and case-insensitive on purpose — it is much
// safer to over-redact a key that merely contains "token" than to miss a
// variant spelling like "AccessToken" or "refresh_token".
var forbiddenKeys = []string{
	"password",
	"token",
	"authorization",
	"embedding",
	"image",
	"photo",
	"base64",
	"note",
}

const redactedPlaceholder = "[REDACTED]"

// redactAttr is the slog.HandlerOptions.ReplaceAttr hook. It runs on every
// attribute (top-level and inside slog.Group) before it is written out.
func redactAttr(groups []string, a slog.Attr) slog.Attr {
	if isForbiddenKey(a.Key) {
		return slog.String(a.Key, redactedPlaceholder)
	}
	return a
}

func isForbiddenKey(key string) bool {
	lower := strings.ToLower(key)
	for _, forbidden := range forbiddenKeys {
		if strings.Contains(lower, forbidden) {
			return true
		}
	}
	return false
}

// Redact returns value unless key looks like one of the forbidden fields, in
// which case it returns the redaction placeholder. Handlers that build log
// attributes from dynamic maps (e.g. HTTP headers) should route every value
// through this instead of relying solely on ReplaceAttr, so the rule holds
// even for code paths that never go through slog.Attr construction directly.
func Redact(key, value string) string {
	if isForbiddenKey(key) {
		return redactedPlaceholder
	}
	return value
}
