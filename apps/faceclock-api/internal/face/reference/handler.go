package reference

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

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

// ListReferences handles GET /api/v1/employees/{id}/face-references (Endpoint #44)
func (h *Handler) ListReferences(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "Autentikasi diperlukan"))
		return
	}

	idStr := chi.URLParam(r, "id")
	targetEmpID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeValidationError, "ID karyawan tidak valid"))
		return
	}

	includeInactive := false
	if val := r.URL.Query().Get("include_inactive"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			includeInactive = b
		}
	}

	callerEmpID := uuid.Nil
	if p.EmployeeID != nil {
		callerEmpID = *p.EmployeeID
	}
	canReadAny := p.HasPermission(rbac.PermFaceReadAny)

	resp, err := h.svc.ListReferences(r.Context(), targetEmpID, includeInactive, p.UserID, callerEmpID, canReadAny)
	if err != nil {
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "Gagal mengambil daftar referensi wajah"))
		return
	}

	httpx.WithCustomMeta(w, resp.Data, resp.Meta)
}

// GetPhoto handles GET /api/v1/face/references/{id}/photo (Endpoint #45)
func (h *Handler) GetPhoto(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "Autentikasi diperlukan"))
		return
	}

	idStr := chi.URLParam(r, "id")
	refID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeValidationError, "ID referensi tidak valid"))
		return
	}

	callerEmpID := uuid.Nil
	if p.EmployeeID != nil {
		callerEmpID = *p.EmployeeID
	}
	canReadAny := p.HasPermission(rbac.PermFaceReadAny)

	reader, mimeType, err := h.svc.GetPhoto(r.Context(), refID, p.UserID, callerEmpID, canReadAny)
	if err != nil {
		if errors.Is(err, ErrPhotoPurged) {
			httpx.FailWithStatus(r.Context(), w, http.StatusGone, httpx.NewAppError(httpx.CodeNotFound, "Foto referensi sudah dihapus sesuai kebijakan retensi"))
			return
		}
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "Gagal memuat foto referensi"))
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Cache-Control", "private, max-age=60")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}

// DeactivateReference handles PATCH /api/v1/face/references/{id} (Endpoint #46)
func (h *Handler) DeactivateReference(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "Autentikasi diperlukan"))
		return
	}

	idStr := chi.URLParam(r, "id")
	refID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeValidationError, "ID referensi tidak valid"))
		return
	}

	var req DeactivateReferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeValidationError, "Body permintaan tidak valid"))
		return
	}

	resp, err := h.svc.DeactivateReference(r.Context(), refID, req, p.UserID)
	if err != nil {
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "Gagal menonaktifkan referensi wajah"))
		return
	}

	httpx.OK(w, resp)
}

// DeleteFaceData handles DELETE /api/v1/employees/{id}/face-data (Endpoint #47)
func (h *Handler) DeleteFaceData(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "Autentikasi diperlukan"))
		return
	}

	idStr := chi.URLParam(r, "id")
	targetEmpID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeValidationError, "ID karyawan tidak valid"))
		return
	}

	var req DeleteFaceDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeValidationError, "Body permintaan tidak valid. 'reason' wajib diisi"))
		return
	}

	resp, err := h.svc.DeleteFaceData(r.Context(), targetEmpID, req.Reason, p.UserID)
	if err != nil {
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "Gagal menghapus data biometrik karyawan"))
		return
	}

	httpx.OK(w, resp)
}

// GetEnrollmentStatusMe handles GET /api/v1/face/enrollment-status/me (Endpoint #48)
func (h *Handler) GetEnrollmentStatusMe(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "Autentikasi diperlukan"))
		return
	}

	if p.EmployeeID == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeValidationError, "Akun tidak terhubung dengan data karyawan"))
		return
	}

	resp, err := h.svc.GetEnrollmentStatus(r.Context(), *p.EmployeeID)
	if err != nil {
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "Gagal mengambil status enrollment"))
		return
	}

	httpx.OK(w, resp)
}

// GetEnrollmentStatusByEmployee handles GET /api/v1/employees/{id}/enrollment-status
func (h *Handler) GetEnrollmentStatusByEmployee(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "Autentikasi diperlukan"))
		return
	}

	idStr := chi.URLParam(r, "id")
	targetEmpID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeValidationError, "ID karyawan tidak valid"))
		return
	}

	callerEmpID := uuid.Nil
	if p.EmployeeID != nil {
		callerEmpID = *p.EmployeeID
	}
	if !p.HasPermission(rbac.PermFaceReadAny) && targetEmpID != callerEmpID {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeNotFound, "Karyawan tidak ditemukan"))
		return
	}

	resp, err := h.svc.GetEnrollmentStatus(r.Context(), targetEmpID)
	if err != nil {
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "Gagal mengambil status enrollment"))
		return
	}

	httpx.OK(w, resp)
}
