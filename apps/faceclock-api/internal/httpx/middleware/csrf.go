package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
)

// CSRFFailFunc is called when a request fails the Anti-CSRF shield checks.
type CSRFFailFunc func(w http.ResponseWriter, r *http.Request, code string, msg string)

// defaultCSRFFail writes a standard JSON error response if no custom fail func is provided.
func defaultCSRFFail(w http.ResponseWriter, r *http.Request, code string, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": msg,
		},
	})
}

// CSRFProtection enforces Anti-CSRF defense for cookie-based authentication.
//
// 1. Safe methods (GET, HEAD, OPTIONS, TRACE) bypass this check.
// 2. State-changing / mutating methods (POST, PUT, PATCH, DELETE) require:
//   - Custom Header: `X-Requested-With: XMLHttpRequest` OR non-empty `X-CSRF-Token`.
//     Standard HTML forms / simple cross-origin requests cannot set custom headers without
//     triggering a preflight CORS check.
//   - Origin verification: If `Origin` header is present, it must match one of the allowedOrigins
//     (or allowedOrigins contains "*").
func CSRFProtection(allowedOrigins []string, onFail ...CSRFFailFunc) func(http.Handler) http.Handler {
	var fail CSRFFailFunc
	if len(onFail) > 0 && onFail[0] != nil {
		fail = onFail[0]
	} else {
		fail = defaultCSRFFail
	}

	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Bypass safe, idempotent read-only methods
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
				next.ServeHTTP(w, r)
				return
			}

			// Mutating methods (POST, PUT, PATCH, DELETE):
			// 1. Header validation: X-Requested-With == "XMLHttpRequest" OR non-empty X-CSRF-Token
			reqWith := r.Header.Get("X-Requested-With")
			csrfToken := r.Header.Get("X-CSRF-Token")

			if !strings.EqualFold(reqWith, "XMLHttpRequest") && csrfToken == "" {
				fail(w, r, "CSRF_HEADER_MISSING", "Permintaan mutasi memerlukan header X-Requested-With atau X-CSRF-Token")
				return
			}

			// 2. Origin validation if Origin header is present
			origin := r.Header.Get("Origin")
			if origin != "" && !allowed["*"] && !allowed[origin] {
				fail(w, r, "CSRF_UNTRUSTED_ORIGIN", "Origin permintaan tidak diizinkan")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
