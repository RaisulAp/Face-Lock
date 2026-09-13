package rbac

import (
	"net/http"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
)

// RequirePermission enforces that the authenticated principal possesses the given permission.
func RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := GetPrincipal(r.Context())
			if !ok || p == nil {
				httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
				return
			}

			if !p.HasPermission(permission) {
				httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeForbidden, "insufficient permissions"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission enforces that the principal has at least one of the specified permissions.
func RequireAnyPermission(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := GetPrincipal(r.Context())
			if !ok || p == nil {
				httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
				return
			}

			if !p.HasAnyPermission(permissions...) {
				httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeForbidden, "insufficient permissions"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireSuperAdmin enforces that the principal has the super_admin role.
func RequireSuperAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := GetPrincipal(r.Context())
			if !ok || p == nil {
				httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
				return
			}

			if !p.IsSuperAdmin() {
				httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeForbidden, "super admin privileges required"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
