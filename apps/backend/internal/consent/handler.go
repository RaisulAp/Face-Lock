package consent

import (
	"errors"
	"net/http"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetActiveDocument handles GET /consents/document (Endpoint #32)
func (h *Handler) GetActiveDocument(w http.ResponseWriter, r *http.Request) {
	doc, err := h.svc.GetActiveDocument(r.Context())
	if err != nil {
		handleError(w, r, err)
		return
	}
	httpx.OK(w, doc)
}

// GetMyConsent handles GET /consents/me (Endpoint #33)
func (h *Handler) GetMyConsent(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "unauthenticated"))
		return
	}
	if p.EmployeeID == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeForbidden, "authenticated user is not linked to an employee"))
		return
	}

	res, err := h.svc.GetMyConsent(r.Context(), *p.EmployeeID)
	if err != nil {
		handleError(w, r, err)
		return
	}
	httpx.OK(w, res)
}

// GrantConsent handles POST /consents (Endpoint #34)
func (h *Handler) GrantConsent(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "unauthenticated"))
		return
	}
	if p.EmployeeID == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeForbidden, "authenticated user is not linked to an employee"))
		return
	}

	var req GrantConsentRequest
	if appErr := httpx.DecodeAndValidate(r, &req); appErr != nil {
		httpx.Fail(r.Context(), w, appErr)
		return
	}

	ip := clientIP(r)
	res, err := h.svc.GrantConsent(r.Context(), *p.EmployeeID, req, ip, r.UserAgent(), &p.UserID)
	if err != nil {
		handleError(w, r, err)
		return
	}
	httpx.Created(w, res)
}

// WithdrawConsent handles POST /consents/withdraw (Endpoint #35)
func (h *Handler) WithdrawConsent(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "unauthenticated"))
		return
	}
	if p.EmployeeID == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeForbidden, "authenticated user is not linked to an employee"))
		return
	}

	var req WithdrawConsentRequest
	if appErr := httpx.DecodeAndValidate(r, &req); appErr != nil {
		httpx.Fail(r.Context(), w, appErr)
		return
	}

	ip := clientIP(r)
	res, err := h.svc.WithdrawConsent(r.Context(), *p.EmployeeID, req, ip, r.UserAgent(), &p.UserID)
	if err != nil {
		handleError(w, r, err)
		return
	}
	httpx.OK(w, res)
}

// GetEmployeeConsent handles GET /employees/{id}/consent (Endpoint #36)
func (h *Handler) GetEmployeeConsent(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	empID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid employee ID"))
		return
	}

	res, err := h.svc.GetEmployeeConsent(r.Context(), empID)
	if err != nil {
		handleError(w, r, err)
		return
	}
	httpx.OK(w, res)
}

// AdminRecordConsent handles POST /employees/{id}/consent (Endpoint #37)
func (h *Handler) AdminRecordConsent(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "unauthenticated"))
		return
	}

	idStr := chi.URLParam(r, "id")
	empID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid employee ID"))
		return
	}

	var req RecordAdminConsentRequest
	if appErr := httpx.DecodeAndValidate(r, &req); appErr != nil {
		httpx.Fail(r.Context(), w, appErr)
		return
	}

	ip := clientIP(r)
	res, err := h.svc.AdminRecordConsent(r.Context(), empID, req, ip, r.UserAgent(), p.UserID)
	if err != nil {
		handleError(w, r, err)
		return
	}
	httpx.Created(w, res)
}

// clientIP delegates to httpx.ClientIP so consent records the same bare,
// validated client address as every other audit-bearing path. The previous
// inline version returned r.RemoteAddr unstripped in the fallback case, which
// carries a port and is rejected by the `inet` column it is written to.
func clientIP(r *http.Request) string {
	return httpx.ClientIP(r)
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *httpx.AppError
	if errors.As(err, &appErr) {
		httpx.Fail(r.Context(), w, appErr)
		return
	}
	httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "internal error"))
}
