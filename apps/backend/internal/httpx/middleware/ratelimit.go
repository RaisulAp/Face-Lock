package middleware

import (
	"net/http"
	"sync"
	"time"
)

// RateLimit is a deliberately simple fixed-window limiter, keyed by remote
// IP (set by chi's RealIP earlier in the chain). Fase 0 only needs "a"
// baseline limiter to exist in the chain at the right position; per-route
// tuning (e.g. login attempts in Fase 1, failed face match attempts in
// Fase 4) is layered on top later, not replaced.
func RateLimit(limit int, window time.Duration, onLimited func(w http.ResponseWriter, r *http.Request)) func(http.Handler) http.Handler {
	type bucket struct {
		count      int
		windowFrom time.Time
	}

	var mu sync.Mutex
	buckets := make(map[string]*bucket)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr
			now := time.Now()

			mu.Lock()
			b, ok := buckets[key]
			if !ok || now.Sub(b.windowFrom) > window {
				b = &bucket{count: 0, windowFrom: now}
				buckets[key] = b
			}
			b.count++
			exceeded := b.count > limit
			mu.Unlock()

			if exceeded {
				onLimited(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
