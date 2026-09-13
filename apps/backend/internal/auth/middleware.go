package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Authenticate returns an HTTP middleware that extracts and verifies JWT access tokens,
// checks user status in PostgreSQL, fetches RBAC permissions, and attaches the Principal to context.
// Token extraction follows the priority:
// 1. Primary: HttpOnly "access_token" cookie
// 2. Fallback: "Authorization: Bearer <token>" header
func Authenticate(tokenMgr *TokenManager, rbacSvc *rbac.Service, db *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tokenStr string

			// 1. Primary: Extract from access_token cookie
			if cookie, err := r.Cookie(AccessTokenCookieName); err == nil && cookie.Value != "" {
				tokenStr = strings.TrimSpace(cookie.Value)
			}

			// 2. Fallback: Extract from Authorization Bearer header
			if tokenStr == "" {
				authHeader := r.Header.Get("Authorization")
				if authHeader != "" {
					parts := strings.SplitN(authHeader, " ", 2)
					if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
						tokenStr = strings.TrimSpace(parts[1])
					}
				}
			}

			if tokenStr == "" {
				httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "missing authentication token"))
				return
			}

			claims, err := tokenMgr.ParseAccessToken(tokenStr)
			if err != nil {
				httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "invalid or expired token"))
				return
			}

			// Validate user against database
			const q = `
				SELECT email, is_active, locked_until, token_version
				FROM users
				WHERE id = $1 AND deleted_at IS NULL
			`
			var (
				email        string
				isActive     bool
				lockedUntil  *time.Time
				tokenVersion int
			)
			err = db.QueryRow(r.Context(), q, claims.UserID).Scan(&email, &isActive, &lockedUntil, &tokenVersion)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "user not found or deleted"))
					return
				}
				httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to verify user status"))
				return
			}

			if !isActive {
				httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "account is deactivated"))
				return
			}

			if lockedUntil != nil && time.Now().UTC().Before(*lockedUntil) {
				httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "account is temporarily locked"))
				return
			}

			if tokenVersion != claims.TokenVersion {
				httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "token has been revoked"))
				return
			}

			// Fetch roles and permissions (cached)
			roles, perms, err := rbacSvc.GetUserRolesAndPermissions(r.Context(), claims.UserID)
			if err != nil {
				httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to load user permissions"))
				return
			}

			principal := &rbac.Principal{
				UserID:       claims.UserID,
				EmployeeID:   claims.EmployeeID,
				Email:        email,
				TokenVersion: claims.TokenVersion,
				Roles:        roles,
				Permissions:  perms,
			}

			ctx := rbac.WithPrincipal(r.Context(), principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
