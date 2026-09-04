package middleware

import (
	"context"
	"net/http"
	"time"
)

// Timeout pushes a deadline onto the request context so that everything
// downstream — including, from Fase 1 onward, the Authenticate DB lookup —
// is bounded by it (Fase 0 § 2.6: "Authenticate harus di dalam Timeout supaya
// query DB untuk auth ikut terpotong deadline").
//
// Unlike http.TimeoutHandler this does not race a second goroutine against
// the handler; it only sets ctx and lets BodyLimit/ratelimit/handlers observe
// it via ctx.Err() and DB calls that accept ctx. That keeps behavior simple
// and avoids the well-known TimeoutHandler footgun of the handler goroutine
// continuing to run (and panicking on a hijacked ResponseWriter) after the
// timeout fires.
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
