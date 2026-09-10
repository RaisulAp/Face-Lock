package httpx

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx/middleware"
)

// Handlers contains all the HTTP handler functions for Fase 1.
type Handlers struct {
	// Auth
	AuthLogin          http.HandlerFunc
	AuthRefresh        http.HandlerFunc
	AuthLogout         http.HandlerFunc
	AuthLogoutAll      http.HandlerFunc
	AuthGetMe          http.HandlerFunc
	AuthChangePassword http.HandlerFunc

	// Employees
	EmployeeList    http.HandlerFunc
	EmployeeCreate  http.HandlerFunc
	EmployeeGetMe   http.HandlerFunc
	EmployeeGetByID http.HandlerFunc
	EmployeeUpdate  http.HandlerFunc
	EmployeeDelete  http.HandlerFunc

	// Users
	UserList          http.HandlerFunc
	UserCreate        http.HandlerFunc
	UserGetByID       http.HandlerFunc
	UserUpdate        http.HandlerFunc
	UserDelete        http.HandlerFunc
	UserUpdateStatus  http.HandlerFunc
	UserAssignRoles   http.HandlerFunc
	UserResetPassword http.HandlerFunc

	// Roles & Permissions
	RoleList              http.HandlerFunc
	RoleCreate            http.HandlerFunc
	RoleGetByID           http.HandlerFunc
	RoleUpdate            http.HandlerFunc
	RoleDelete            http.HandlerFunc
	RoleAssignPermissions http.HandlerFunc
	PermissionList        http.HandlerFunc

	// Settings
	SettingsList     http.HandlerFunc
	SettingsGetByKey http.HandlerFunc
	SettingsUpdate   http.HandlerFunc

	// Audit Logs
	AuditQuery http.HandlerFunc
}

// RouterDeps is every dependency the router needs to wire the middleware
// chain and mount all endpoints.
type RouterDeps struct {
	Logger             *slog.Logger
	CORSAllowedOrigins []string
	RequestTimeout     time.Duration
	MaxBodyBytes       int64

	HealthzHandler http.HandlerFunc
	ReadyzHandler  http.HandlerFunc
	VersionHandler http.HandlerFunc

	AuthMiddleware    func(http.Handler) http.Handler
	RBACMiddleware    func(permission string) func(http.Handler) http.Handler
	RBACAnyMiddleware func(permissions ...string) func(http.Handler) http.Handler

	Handlers Handlers
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
	r.Use(middleware.CSRFProtection(deps.CORSAllowedOrigins, writeCSRFFail))
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

// mountAPIv1 mounts all versioned API routes under /api/v1.
func mountAPIv1(r chi.Router, deps RouterDeps) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/version", deps.VersionHandler)

		// Public Auth
		if deps.Handlers.AuthLogin != nil {
			r.Post("/auth/login", deps.Handlers.AuthLogin)
		}
		if deps.Handlers.AuthRefresh != nil {
			r.Post("/auth/refresh", deps.Handlers.AuthRefresh)
		}

		// Authenticated Routes Group
		r.Group(func(r chi.Router) {
			if deps.AuthMiddleware != nil {
				r.Use(deps.AuthMiddleware)
			}

			// Auth (Authenticated)
			if deps.Handlers.AuthLogout != nil {
				r.Post("/auth/logout", deps.Handlers.AuthLogout)
			}
			if deps.Handlers.AuthLogoutAll != nil {
				r.Post("/auth/logout-all", deps.Handlers.AuthLogoutAll)
			}
			if deps.Handlers.AuthGetMe != nil {
				r.Get("/auth/me", deps.Handlers.AuthGetMe)
			}
			if deps.Handlers.AuthChangePassword != nil {
				r.Post("/auth/change-password", deps.Handlers.AuthChangePassword)
			}

			// Employees
			if deps.Handlers.EmployeeList != nil {
				r.With(rbacGuard(deps, "employee.read")).Get("/employees", deps.Handlers.EmployeeList)
			}
			if deps.Handlers.EmployeeCreate != nil {
				r.With(rbacGuard(deps, "employee.create")).Post("/employees", deps.Handlers.EmployeeCreate)
			}
			if deps.Handlers.EmployeeGetMe != nil {
				r.With(rbacGuard(deps, "employee.read_self")).Get("/employees/me", deps.Handlers.EmployeeGetMe)
			}
			if deps.Handlers.EmployeeGetByID != nil {
				r.With(rbacAnyGuard(deps, "employee.read", "employee.read_self")).Get("/employees/{id}", deps.Handlers.EmployeeGetByID)
			}
			if deps.Handlers.EmployeeUpdate != nil {
				r.With(rbacGuard(deps, "employee.update")).Patch("/employees/{id}", deps.Handlers.EmployeeUpdate)
			}
			if deps.Handlers.EmployeeDelete != nil {
				r.With(rbacGuard(deps, "employee.delete")).Delete("/employees/{id}", deps.Handlers.EmployeeDelete)
			}

			// Users
			if deps.Handlers.UserList != nil {
				r.With(rbacGuard(deps, "user.read")).Get("/users", deps.Handlers.UserList)
			}
			if deps.Handlers.UserCreate != nil {
				r.With(rbacGuard(deps, "user.create")).Post("/users", deps.Handlers.UserCreate)
			}
			if deps.Handlers.UserGetByID != nil {
				r.With(rbacGuard(deps, "user.read")).Get("/users/{id}", deps.Handlers.UserGetByID)
			}
			if deps.Handlers.UserUpdate != nil {
				r.With(rbacGuard(deps, "user.update")).Patch("/users/{id}", deps.Handlers.UserUpdate)
			}
			if deps.Handlers.UserDelete != nil {
				r.With(rbacGuard(deps, "user.delete")).Delete("/users/{id}", deps.Handlers.UserDelete)
			}
			if deps.Handlers.UserUpdateStatus != nil {
				r.With(rbacGuard(deps, "user.update")).Patch("/users/{id}/status", deps.Handlers.UserUpdateStatus)
			}
			if deps.Handlers.UserAssignRoles != nil {
				r.With(rbacGuard(deps, "user.assign_role")).Put("/users/{id}/roles", deps.Handlers.UserAssignRoles)
			}
			if deps.Handlers.UserResetPassword != nil {
				r.With(rbacGuard(deps, "user.reset_password")).Post("/users/{id}/reset-password", deps.Handlers.UserResetPassword)
			}

			// Roles & Permissions
			if deps.Handlers.RoleList != nil {
				r.With(rbacGuard(deps, "role.read")).Get("/roles", deps.Handlers.RoleList)
			}
			if deps.Handlers.RoleCreate != nil {
				r.With(rbacGuard(deps, "role.create")).Post("/roles", deps.Handlers.RoleCreate)
			}
			if deps.Handlers.RoleGetByID != nil {
				r.With(rbacGuard(deps, "role.read")).Get("/roles/{id}", deps.Handlers.RoleGetByID)
			}
			if deps.Handlers.RoleUpdate != nil {
				r.With(rbacGuard(deps, "role.update")).Patch("/roles/{id}", deps.Handlers.RoleUpdate)
			}
			if deps.Handlers.RoleDelete != nil {
				r.With(rbacGuard(deps, "role.delete")).Delete("/roles/{id}", deps.Handlers.RoleDelete)
			}
			if deps.Handlers.RoleAssignPermissions != nil {
				r.With(rbacGuard(deps, "role.assign_permission")).Put("/roles/{id}/permissions", deps.Handlers.RoleAssignPermissions)
			}
			if deps.Handlers.PermissionList != nil {
				r.With(rbacGuard(deps, "permission.read")).Get("/permissions", deps.Handlers.PermissionList)
			}

			// Settings
			if deps.Handlers.SettingsList != nil {
				r.With(rbacGuard(deps, "settings.read")).Get("/settings", deps.Handlers.SettingsList)
			}
			if deps.Handlers.SettingsGetByKey != nil {
				r.With(rbacGuard(deps, "settings.read")).Get("/settings/{key}", deps.Handlers.SettingsGetByKey)
			}
			if deps.Handlers.SettingsUpdate != nil {
				r.With(rbacGuard(deps, "settings.update")).Put("/settings/{key}", deps.Handlers.SettingsUpdate)
			}

			// Audit Logs
			if deps.Handlers.AuditQuery != nil {
				r.With(rbacGuard(deps, "audit.read")).Get("/audit-logs", deps.Handlers.AuditQuery)
			}
		})
	})
}

func rbacGuard(deps RouterDeps, perm string) func(http.Handler) http.Handler {
	if deps.RBACMiddleware != nil {
		return deps.RBACMiddleware(perm)
	}
	return func(next http.Handler) http.Handler { return next }
}

func rbacAnyGuard(deps RouterDeps, perms ...string) func(http.Handler) http.Handler {
	if deps.RBACAnyMiddleware != nil {
		return deps.RBACAnyMiddleware(perms...)
	}
	return func(next http.Handler) http.Handler { return next }
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

func writeCSRFFail(w http.ResponseWriter, r *http.Request, code string, msg string) {
	Fail(r.Context(), w, NewAppError(ErrorCode(code), msg))
}

func writeRateLimitResponse(w http.ResponseWriter, r *http.Request) {
	Fail(r.Context(), w, NewAppError(CodeRateLimited, "Terlalu banyak request, coba lagi nanti"))
}
