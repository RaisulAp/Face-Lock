package auth

import (
	"encoding/json"
	"net/http"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
)

// Handler handles HTTP requests for authentication.
type Handler struct {
	svc   *Service
	audit *audit.Recorder
}

// NewHandler creates a new auth HTTP handler.
func NewHandler(svc *Service, audit *audit.Recorder) *Handler {
	return &Handler{svc: svc, audit: audit}
}

// LoginRequest request payload for login.
type LoginRequest struct {
	Email      string  `json:"email" validate:"required,email"`
	Password   string  `json:"password" validate:"required"`
	DeviceInfo *string `json:"device_info,omitempty"`
}

// Login handles POST /api/v1/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.Fail(r.Context(), w, err)
		return
	}

	// httpx.ClientIP strips the port from RemoteAddr; without this the
	// value never parses and every refresh_tokens.ip_address is stored NULL.
	ip := httpx.ClientIP(r)

	res, err := h.svc.Login(r.Context(), LoginInput{
		Email:      req.Email,
		Password:   req.Password,
		DeviceInfo: req.DeviceInfo,
		IPAddress:  ip,
		UserAgent:  r.UserAgent(),
	})
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "login failed"))
		return
	}

	SetAuthCookies(w, res.AccessToken, res.RefreshToken, h.svc.IsDev())
	httpx.OK(w, res)
}

// RefreshRequest request payload for token refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Refresh handles POST /api/v1/auth/refresh
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var tokenStr string

	// 1. Primary: Extract from refresh_token cookie
	if cookie, err := r.Cookie(RefreshTokenCookieName); err == nil && cookie.Value != "" {
		tokenStr = cookie.Value
	}

	// 2. Fallback: Extract from request body if no cookie
	if tokenStr == "" {
		var req RefreshRequest
		if err := httpx.DecodeAndValidate(r, &req); err != nil {
			httpx.Fail(r.Context(), w, err)
			return
		}
		tokenStr = req.RefreshToken
	}

	ip := httpx.ClientIP(r)

	res, err := h.svc.Refresh(r.Context(), tokenStr, ip, r.UserAgent())
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "refresh failed"))
		return
	}

	SetAuthCookies(w, res.AccessToken, res.RefreshToken, h.svc.IsDev())
	httpx.OK(w, res)
}

// LogoutRequest request payload for logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"omitempty"`
}

// Logout handles POST /api/v1/auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var refreshToken string

	// 1. Extract from cookie
	if cookie, err := r.Cookie(RefreshTokenCookieName); err == nil && cookie.Value != "" {
		refreshToken = cookie.Value
	}

	// 2. Fallback: Extract from request body
	if refreshToken == "" && r.Body != nil {
		var req LogoutRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		refreshToken = req.RefreshToken
	}

	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		ClearAuthCookies(w, h.svc.IsDev())
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	_ = h.svc.Logout(r.Context(), p.UserID, refreshToken)
	ClearAuthCookies(w, h.svc.IsDev())

	resID := p.UserID.String()
	_ = h.audit.RecordFromRequest(r, "auth.logout", "user", &resID, nil)

	httpx.NoContent(w)
}

// ChangePasswordRequest request payload for password change.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=10"`
}

// ChangePassword handles POST /api/v1/auth/change-password
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req ChangePasswordRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.Fail(r.Context(), w, err)
		return
	}

	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	if err := h.svc.ChangePassword(r.Context(), p.UserID, req.CurrentPassword, req.NewPassword); err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to change password"))
		return
	}

	resID := p.UserID.String()
	_ = h.audit.RecordFromRequest(r, "auth.password_changed", "user", &resID, nil)

	httpx.OK(w, map[string]string{"message": "password updated successfully"})
}

// GetMe handles GET /api/v1/auth/me
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	profile, err := h.svc.GetMe(r.Context(), p.UserID)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to retrieve profile"))
		return
	}

	httpx.OK(w, profile)
}

// RevokeAll handles POST /api/v1/auth/revoke-all
func (h *Handler) RevokeAll(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		ClearAuthCookies(w, h.svc.IsDev())
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	if err := h.svc.RevokeAll(r.Context(), p.UserID); err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to revoke sessions"))
		return
	}

	ClearAuthCookies(w, h.svc.IsDev())

	resID := p.UserID.String()
	_ = h.audit.RecordFromRequest(r, "auth.revoke_all", "user", &resID, nil)

	httpx.NoContent(w)
}
