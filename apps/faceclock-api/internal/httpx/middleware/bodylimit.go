package middleware

import "net/http"

// BodyLimit wraps the request body with http.MaxBytesReader so an oversized
// body fails fast (E6) instead of being read fully into memory first. The
// actual 413 PAYLOAD_TOO_LARGE response is produced where the body is
// decoded (internal/httpx/decode.go), because that is the first point a
// MaxBytesReader error surfaces.
func BodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
