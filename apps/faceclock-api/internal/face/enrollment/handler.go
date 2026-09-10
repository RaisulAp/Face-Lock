package enrollment

import (
	"encoding/json"
	"errors"
	"io"
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

// POST /face/enrollments
func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "Autentikasi diperlukan"))
		return
	}

	var req CreateSessionRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "Format JSON tidak valid"))
			return
		}
	}

	var callerEmpID uuid.UUID
	if p.EmployeeID != nil {
		callerEmpID = *p.EmployeeID
	}
	canEnrollAny := p.HasPermission("face.enroll_any")

	resp, err := h.svc.CreateSession(r.Context(), req, p.UserID, callerEmpID, canEnrollAny)
	if err != nil {
		handleError(w, r, err)
		return
	}

	httpx.Created(w, resp)
}

// GET /face/enrollments/{id}
func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "Autentikasi diperlukan"))
		return
	}

	idStr := chi.URLParam(r, "id")
	sessID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "ID sesi tidak valid"))
		return
	}

	var callerEmpID uuid.UUID
	if p.EmployeeID != nil {
		callerEmpID = *p.EmployeeID
	}
	canReadAny := p.HasPermission("face.read_any")

	resp, err := h.svc.GetSession(r.Context(), sessID, p.UserID, callerEmpID, canReadAny)
	if err != nil {
		handleError(w, r, err)
		return
	}

	httpx.OK(w, resp)
}

// POST /face/enrollments/{id}/photos
func (h *Handler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "Autentikasi diperlukan"))
		return
	}

	idStr := chi.URLParam(r, "id")
	sessID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "ID sesi tidak valid"))
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "Gagal memproses form data multipart"))
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeValidationError, "Field 'image' wajib disertakan"))
		return
	}
	defer file.Close()

	captureSource := r.FormValue("capture_source")
	if captureSource == "" {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeValidationError, "Field 'capture_source' wajib diisi"))
		return
	}

	imageBytes, err := io.ReadAll(file)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "Gagal membaca file gambar"))
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(imageBytes)
	}

	var callerEmpID uuid.UUID
	if p.EmployeeID != nil {
		callerEmpID = *p.EmployeeID
	}
	canEnrollAny := p.HasPermission("face.enroll_any")

	resp, err := h.svc.UploadPhoto(r.Context(), sessID, imageBytes, mimeType, captureSource, p.UserID, callerEmpID, canEnrollAny)
	if err != nil {
		handleError(w, r, err)
		return
	}

	httpx.Created(w, resp)
}

// DELETE /face/enrollments/{id}/photos/{pid}
func (h *Handler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "Autentikasi diperlukan"))
		return
	}

	idStr := chi.URLParam(r, "id")
	sessID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "ID sesi tidak valid"))
		return
	}

	pidStr := chi.URLParam(r, "pid")
	photoID, err := uuid.Parse(pidStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "ID foto tidak valid"))
		return
	}

	var callerEmpID uuid.UUID
	if p.EmployeeID != nil {
		callerEmpID = *p.EmployeeID
	}
	canEnrollAny := p.HasPermission("face.enroll_any")

	if err := h.svc.DeletePhoto(r.Context(), sessID, photoID, p.UserID, callerEmpID, canEnrollAny); err != nil {
		handleError(w, r, err)
		return
	}

	httpx.NoContent(w)
}

// POST /face/enrollments/{id}/commit
func (h *Handler) CommitSession(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "Autentikasi diperlukan"))
		return
	}

	idStr := chi.URLParam(r, "id")
	sessID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "ID sesi tidak valid"))
		return
	}

	var req CommitSessionRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "Format JSON tidak valid"))
			return
		}
	}

	var callerEmpID uuid.UUID
	if p.EmployeeID != nil {
		callerEmpID = *p.EmployeeID
	}
	canEnrollAny := p.HasPermission("face.enroll_any")

	resp, err := h.svc.CommitSession(r.Context(), sessID, req, p.UserID, callerEmpID, canEnrollAny)
	if err != nil {
		handleError(w, r, err)
		return
	}

	httpx.OK(w, resp)
}

// DELETE /face/enrollments/{id}
func (h *Handler) CancelSession(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "Autentikasi diperlukan"))
		return
	}

	idStr := chi.URLParam(r, "id")
	sessID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "ID sesi tidak valid"))
		return
	}

	var callerEmpID uuid.UUID
	if p.EmployeeID != nil {
		callerEmpID = *p.EmployeeID
	}
	canEnrollAny := p.HasPermission("face.enroll_any")

	if err := h.svc.CancelSession(r.Context(), sessID, p.UserID, callerEmpID, canEnrollAny); err != nil {
		handleError(w, r, err)
		return
	}

	httpx.NoContent(w)
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *httpx.AppError
	if errors.As(err, &appErr) {
		httpx.Fail(r.Context(), w, appErr)
		return
	}
	httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "Terjadi kesalahan internal server"))
}
