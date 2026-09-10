package attendance

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/google/uuid"
)

// Handler exposes HTTP handlers for attendance operations.
type Handler struct {
	svc   *Service
	audit *audit.Recorder
}

// NewHandler constructs an attendance Handler.
func NewHandler(svc *Service, audit *audit.Recorder) *Handler {
	return &Handler{
		svc:   svc,
		audit: audit,
	}
}

// ClockIn handles POST /api/v1/attendance/clock-in
func (h *Handler) ClockIn(w http.ResponseWriter, r *http.Request) {
	h.handleClock(w, r, ClockTypeIn)
}

// ClockOut handles POST /api/v1/attendance/clock-out
func (h *Handler) ClockOut(w http.ResponseWriter, r *http.Request) {
	h.handleClock(w, r, ClockTypeOut)
}

func (h *Handler) handleClock(w http.ResponseWriter, r *http.Request, clockType ClockType) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	photoBytes, contentType, notes, overrideEmpID, err := extractClockRequest(r)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, err.Error()))
		return
	}

	targetEmpID := p.EmployeeID
	if overrideEmpID != nil {
		// Allow overriding only if user is super_admin or has attendance management permissions
		if p.IsSuperAdmin() || p.HasPermission(rbac.PermAttendanceReadAll) || p.HasPermission(rbac.PermAttendanceApprove) {
			targetEmpID = overrideEmpID
		}
	}

	if targetEmpID == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "user is not associated with an employee profile"))
		return
	}

	params := ClockParams{
		EmployeeID:  targetEmpID,
		PhotoBytes:  photoBytes,
		ContentType: contentType,
		Notes:       notes,
	}

	rec, err := h.svc.Clock(r.Context(), clockType, params)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to record attendance"))
		return
	}

	resID := rec.ID.String()
	action := "attendance.clock_in"
	if clockType == ClockTypeOut {
		action = "attendance.clock_out"
	}

	_ = h.audit.RecordFromRequest(r, action, "attendance", &resID, map[string]any{
		"employee_id": rec.EmployeeID.String(),
		"type":        string(rec.Type),
		"similarity":  rec.SimilarityScore,
	})

	httpx.Created(w, rec)
}

// extractClockRequest extracts image bytes, content-type, notes, and optional target employee ID
// from multipart/form-data, json, or raw body streams.
func extractClockRequest(r *http.Request) ([]byte, string, *string, *uuid.UUID, error) {
	ct := r.Header.Get("Content-Type")

	// 1. Multipart form upload
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			return nil, "", nil, nil, httpx.NewAppError(httpx.CodeBadRequest, "failed to parse multipart form")
		}

		var notes *string
		if n := r.FormValue("notes"); n != "" {
			notes = &n
		}

		var empID *uuid.UUID
		if idStr := r.FormValue("employee_id"); idStr != "" {
			if parsed, err := uuid.Parse(idStr); err == nil {
				empID = &parsed
			}
		}

		for _, field := range []string{"photo", "image", "file"} {
			file, header, err := r.FormFile(field)
			if err == nil {
				defer file.Close()
				data, err := io.ReadAll(file)
				if err != nil {
					return nil, "", nil, nil, httpx.NewAppError(httpx.CodeBadRequest, "failed to read uploaded file")
				}
				cType := header.Header.Get("Content-Type")
				if cType == "" {
					cType = "image/jpeg"
				}
				return data, cType, notes, empID, nil
			}
		}
		return nil, "", nil, nil, httpx.NewAppError(httpx.CodeBadRequest, "missing photo/image file field in multipart form")
	}

	// 2. JSON payload
	if strings.HasPrefix(ct, "application/json") {
		var payload struct {
			EmployeeID  *string `json:"employee_id,omitempty"`
			PhotoBase64 string  `json:"photo_base64"`
			ImageBase64 string  `json:"image_base64"`
			Photo       string  `json:"photo"`
			Notes       *string `json:"notes,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			return nil, "", nil, nil, httpx.NewAppError(httpx.CodeBadRequest, "malformed json payload")
		}
		raw := payload.PhotoBase64
		if raw == "" {
			raw = payload.ImageBase64
		}
		if raw == "" {
			raw = payload.Photo
		}
		if raw == "" {
			return nil, "", nil, nil, httpx.NewAppError(httpx.CodeBadRequest, "missing photo_base64/image in payload")
		}
		if idx := strings.Index(raw, ";base64,"); idx != -1 {
			raw = raw[idx+8:]
		}
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, "", nil, nil, httpx.NewAppError(httpx.CodeBadRequest, "invalid base64 image data")
		}

		var empID *uuid.UUID
		if payload.EmployeeID != nil && *payload.EmployeeID != "" {
			if parsed, err := uuid.Parse(*payload.EmployeeID); err == nil {
				empID = &parsed
			}
		}

		return decoded, "image/jpeg", payload.Notes, empID, nil
	}

	// 3. Direct binary stream
	if strings.HasPrefix(ct, "image/") || ct == "application/octet-stream" {
		buf := &bytes.Buffer{}
		if _, err := io.Copy(buf, r.Body); err != nil {
			return nil, "", nil, nil, httpx.NewAppError(httpx.CodeBadRequest, "failed to read image body")
		}
		return buf.Bytes(), ct, nil, nil, nil
	}

	return nil, "", nil, nil, httpx.NewAppError(httpx.CodeUnsupportedMedia, "unsupported content type: must be multipart/form-data, application/json, or image/*")
}
