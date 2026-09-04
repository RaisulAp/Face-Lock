package middleware

import (
	"context"
	"crypto/rand"
	"net/http"
	"time"

	"github.com/oklog/ulid/v2"
)

type requestIDCtxKey struct{}

// RequestID must be the outermost middleware in the chain (Fase 0 § 2.6):
// every later middleware, and the handler itself, needs the id already in
// context so every log line for this request can be correlated.
func RequestID(next http.Handler) http.Handler {
	entropy := ulid.Monotonic(rand.Reader, 0)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
		ctx := context.WithValue(r.Context(), requestIDCtxKey{}, id)
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// FromContext returns the request id stored by RequestID, or "" if none is
// present (e.g. in a unit test that calls a handler directly).
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDCtxKey{}).(string)
	return id
}
