package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recoverer must sit inside StructuredLogger (so a panic is still logged)
// and outside the handler. On panic it logs the stack trace — never sent to
// the client — and calls onPanic to write the actual HTTP response, so this
// package does not need to import internal/httpx (which would create an
// import cycle: httpx already imports httpx/middleware for the chain).
func Recoverer(base *slog.Logger, onPanic func(w http.ResponseWriter, r *http.Request)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					requestID := RequestIDFromContext(r.Context())
					base.Error("panic recovered",
						slog.String("request_id", requestID),
						slog.Any("panic", rec),
						slog.String("stack", string(debug.Stack())),
					)
					onPanic(w, r)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
