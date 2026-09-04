package httpx

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx/middleware"
)

// RouterDeps is every dependency the router needs to wire the middleware
// chain and mount Fase 0's three endpoints.
type RouterDeps struct {
	Logger             *slog.Logger
	CORSAllowedOrigins []string
	RequestTimeout     time.Duration
	MaxBodyBytes       int64

	HealthzHandler http.HandlerFunc
	ReadyzHandler  http.HandlerFunc
	VersionHandler http.HandlerFunc
}

// NewRouter builds the chi router with the middleware chain locked in
// Fase 0 § 2.6:
//
//	RequestID → RealIP → StructuredLogger → Recoverer → CORS
//	  → Timeout(30s) → BodyLimit(10MB) → RateLimit
//	    → [Fase 1] Authenticate → [Fase 1] RequirePermission(...) → [Fase 3] RequireConsent(...)
//	      → handler
//
// The bracketed steps do not exist yet — Fase 0 intentionally stops at
// RateLimit. Route groups that will carry Authenticate/RequirePermission
// starting Fase 1 are prepared below (see mountAPIv1) so that fase only has
// to add middleware to an existing group, not invent the mount point.
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.StructuredLogger(deps.Logger))
	r.Use(middleware.Recoverer(deps.Logger, writePanicResponse))
	r.Use(middleware.CORS(deps.CORSAllowedOrigins))
	r.Use(middleware.Timeout(deps.RequestTimeout))
	r.Use(middleware.BodyLimit(deps.MaxBodyBytes))
	r.Use(middleware.RateLimit(defaultRateLimit, defaultRateLimitWindow, writeRateLimitResponse))

	r.NotFound(notFoundHandler)
	r.MethodNotAllowed(methodNotAllowedHandler)

	// Liveness/readiness are intentionally outside /api/v1 and outside any
	// future auth group — an orchestrator must be able to reach them with
	// zero credentials (Fase 0 § 4).
	r.Get("/healthz", deps.HealthzHandler)
	r.Get("/readyz", deps.ReadyzHandler)

	mountAPIv1(r, deps)

	return r
}

// defaultRateLimit is deliberately generous for Fase 0 — it exists to prove
// the middleware slot works, not to tune production traffic shaping (that
// happens per-route starting Fase 1, e.g. login attempts).
const (
	defaultRateLimit       = 300
	defaultRateLimitWindow = time.Minute
)

// mountAPIv1 is the one place that grows in every later fase. Fase 0 only
// mounts /version; the r.Group below is where Fase 1 attaches
// Authenticate/RequirePermission without touching anything above it.
func mountAPIv1(r chi.Router, deps RouterDeps) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/version", deps.VersionHandler)

		// [Fase 1] r.Group(func(r chi.Router) {
		//     r.Use(authn.Authenticate)
		//     ... every authenticated route from Fase 1 onward mounts here ...
		// })
	})
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	Fail(r.Context(), w, NewAppError(CodeNotFound, "Endpoint tidak ditemukan"))
}

func methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	// 405 has no dedicated catalog code (Fase 0 § 2.6 lists 13 codes, none of
	// them 405) — Fase 0 § 11.5 only requires the ENVELOPE shape and the
	// real HTTP status, so this bypasses StatusFor via FailWithStatus rather
	// than stretching an existing code's registered status.
	FailWithStatus(r.Context(), w, http.StatusMethodNotAllowed,
		NewAppError(CodeBadRequest, "Method tidak diizinkan untuk endpoint ini"))
}

func writePanicResponse(w http.ResponseWriter, r *http.Request) {
	Fail(r.Context(), w, NewAppError(CodeInternalError, "Terjadi kesalahan pada server"))
}

func writeRateLimitResponse(w http.ResponseWriter, r *http.Request) {
	Fail(r.Context(), w, NewAppError(CodeRateLimited, "Terlalu banyak request, coba lagi nanti"))
}
