package reindex

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// CreateJob handles POST /api/v1/face/reindex-jobs (#49)
func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "Autentikasi diperlukan"))
		return
	}

	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeValidationError, "Format body JSON tidak valid"))
		return
	}

	resp, err := h.service.CreateJob(r.Context(), req, p.UserID)
	if err != nil {
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "Gagal membuat pekerjaan reindex"))
		return
	}

	if resp.DryRun {
		httpx.OK(w, resp)
	} else {
		httpx.Accepted(w, resp)
	}
}

// ListJobs handles GET /api/v1/face/reindex-jobs (#50)
func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.service.ListJobs(r.Context())
	if err != nil {
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "Gagal mengambil daftar pekerjaan reindex"))
		return
	}

	httpx.OK(w, map[string]any{
		"jobs": jobs,
	})
}

// GetJob handles GET /api/v1/face/reindex-jobs/{id} (#51)
func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeValidationError, "ID pekerjaan tidak valid"))
		return
	}

	job, err := h.service.GetJob(r.Context(), id)
	if err != nil {
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "Gagal mengambil detail pekerjaan reindex"))
		return
	}

	httpx.OK(w, job)
}

// CancelJob handles POST /api/v1/face/reindex-jobs/{id}/cancel (#52)
func (h *Handler) CancelJob(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "Autentikasi diperlukan"))
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeValidationError, "ID pekerjaan tidak valid"))
		return
	}

	job, err := h.service.CancelJob(r.Context(), id, p.UserID)
	if err != nil {
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "Gagal membatalkan pekerjaan reindex"))
		return
	}

	httpx.OK(w, job)
}
