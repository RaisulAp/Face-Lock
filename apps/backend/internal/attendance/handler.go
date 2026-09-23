package attendance

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx/middleware"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/go-chi/chi/v5"
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

// ClockIn handles POST /api/v1/attendance/check-in (#53)
func (h *Handler) ClockIn(w http.ResponseWriter, r *http.Request) {
	h.handleClock(w, r, ClockTypeCheckIn)
}

// ClockOut handles POST /api/v1/attendance/check-out (#54)
func (h *Handler) ClockOut(w http.ResponseWriter, r *http.Request) {
	h.handleClock(w, r, ClockTypeCheckOut)
}

func (h *Handler) handleClock(w http.ResponseWriter, r *http.Request, clockType ClockType) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	req, err := parseClockRequest(r)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, err.Error()))
		return
	}

	req.RequestID = middleware.RequestIDFromContext(r.Context())
	req.IP = extractClientIP(r)
	req.UserAgent = r.UserAgent()

	att, dto, err := h.svc.Clock(r.Context(), p, clockType, *req)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to record attendance: "+err.Error()))
		return
	}

	resID := att.ID.String()
	action := "attendance.clock_in"
	if clockType == ClockTypeCheckOut {
		action = "attendance.clock_out"
	}

	_ = h.audit.RecordFromRequest(r, action, "attendance", &resID, map[string]any{
		"employee_id": att.EmployeeID.String(),
		"type":        string(att.Type),
		"status":      string(att.Status),
		"method":      string(att.Method),
	})

	httpx.Created(w, dto)
}

// GetContext handles GET /api/v1/attendance/context (#55)
func (h *Handler) GetContext(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	resp, err := h.svc.GetContext(r.Context(), p)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, err.Error()))
		return
	}

	httpx.OK(w, resp)
}

// GetMyToday handles GET /api/v1/attendance/my-today (#56)
func (h *Handler) GetMyToday(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	todayIn, todayOut, err := h.svc.GetMyToday(r.Context(), p)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, err.Error()))
		return
	}

	httpx.OK(w, map[string]any{
		"check_in":  todayIn,
		"check_out": todayOut,
	})
}

// GetMeToday handles GET /api/v1/attendances/me/today (#57)
func (h *Handler) GetMeToday(w http.ResponseWriter, r *http.Request) {
	h.GetMyToday(w, r)
}

// GetMyHistory handles GET /api/v1/attendances/me (#56)
func (h *Handler) GetMyHistory(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	page, perPage := parsePagination(r)
	filter := AttendanceFilter{
		Page:      page,
		PerPage:   perPage,
		StartDate: r.URL.Query().Get("start_date"),
		EndDate:   r.URL.Query().Get("end_date"),
		Status:    r.URL.Query().Get("status"),
		Type:      r.URL.Query().Get("type"),
	}

	records, total, err := h.svc.GetMyHistory(r.Context(), p, filter)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, err.Error()))
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	httpx.Paginated(w, records, httpx.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetMe handles GET /api/v1/attendances/me (#56)
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	h.GetMyHistory(w, r)
}

// GetRecordByID handles GET /api/v1/attendances/{id} (#58).
// It returns dual DTOs based on caller permissions (REV-EP-09):
// Admin/Manager (attendance.read_all) -> AdminAttendanceDTO with full telemetry.
// Employee (attendance.read_self) -> EmployeeAttendanceDTO with anti-score leakage.
func (h *Handler) GetRecordByID(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid attendance id"))
		return
	}

	// Check if caller has read_all permission
	if p.HasPermission(rbac.PermAttendanceReadAll) {
		dto, err := h.svc.GetAdminRecordByID(r.Context(), id)
		if err != nil {
			if appErr, ok := err.(*httpx.AppError); ok {
				httpx.Fail(r.Context(), w, appErr)
				return
			}
			httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeNotFound, "data absensi tidak ditemukan"))
			return
		}
		httpx.OK(w, dto)
		return
	}

	// Employee self-view
	dto, err := h.svc.GetMyHistoryByID(r.Context(), p, id)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeNotFound, "data absensi tidak ditemukan"))
		return
	}

	httpx.OK(w, dto)
}

// GetMyHistoryByID handles legacy GET /api/v1/attendance/my-history/{id}
func (h *Handler) GetMyHistoryByID(w http.ResponseWriter, r *http.Request) {
	h.GetRecordByID(w, r)
}

// GetPhoto streams the attendance photo, checking permissions and data retention (#59).
func (h *Handler) GetPhoto(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid attendance id"))
		return
	}

	rdr, mime, err := h.svc.GetAttendancePhoto(r.Context(), p, id)
	if err != nil {
		if errors.Is(err, ErrPhotoPurged) {
			httpx.FailWithStatus(r.Context(), w, http.StatusGone, httpx.NewAppError(httpx.CodeNotFound, "Foto absensi telah dihapus sesuai masa retensi data"))
			return
		}
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeNotFound, "foto absensi tidak ditemukan"))
		return
	}
	defer rdr.Close()

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, no-transform")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rdr)
}

// GetAdminRecords handles GET /api/v1/attendances (#60)
func (h *Handler) GetAdminRecords(w http.ResponseWriter, r *http.Request) {
	page, perPage := parsePagination(r)
	filter := AttendanceFilter{
		Page:         page,
		PerPage:      perPage,
		Department:   r.URL.Query().Get("department"),
		StartDate:    r.URL.Query().Get("start_date"),
		EndDate:      r.URL.Query().Get("end_date"),
		WorkDate:     r.URL.Query().Get("work_date"),
		Status:       r.URL.Query().Get("status"),
		Type:         r.URL.Query().Get("type"),
		Method:       r.URL.Query().Get("method"),
		IsPending:    r.URL.Query().Get("is_pending") == "true",
		OnlyFallback: r.URL.Query().Get("only_fallback") == "true",
		Search:       r.URL.Query().Get("search"),
	}

	if empStr := r.URL.Query().Get("employee_id"); empStr != "" {
		if empID, err := uuid.Parse(empStr); err == nil {
			filter.EmployeeID = &empID
		}
	}

	records, total, err := h.svc.GetAdminRecords(r.Context(), filter)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, err.Error()))
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	httpx.Paginated(w, records, httpx.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetPendingRecords handles GET /api/v1/attendances/pending (#61)
func (h *Handler) GetPendingRecords(w http.ResponseWriter, r *http.Request) {
	page, perPage := parsePagination(r)
	filter := AttendanceFilter{
		Page:       page,
		PerPage:    perPage,
		Department: r.URL.Query().Get("department"),
		StartDate:  r.URL.Query().Get("start_date"),
		EndDate:    r.URL.Query().Get("end_date"),
		IsPending:  true,
		Search:     r.URL.Query().Get("search"),
	}

	records, total, err := h.svc.GetAdminRecords(r.Context(), filter)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, err.Error()))
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	httpx.Paginated(w, records, httpx.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetSummary handles GET /api/v1/attendances/summary (#71)
func (h *Handler) GetSummary(w http.ResponseWriter, r *http.Request) {
	page, perPage := parsePagination(r)
	filter := AttendanceSummaryFilter{
		Page:       page,
		PerPage:    perPage,
		Department: r.URL.Query().Get("department"),
		StartDate:  r.URL.Query().Get("start_date"),
		EndDate:    r.URL.Query().Get("end_date"),
	}
	if empStr := r.URL.Query().Get("employee_id"); empStr != "" {
		if empID, err := uuid.Parse(empStr); err == nil {
			filter.EmployeeID = &empID
		}
	}

	summary, err := h.svc.GetSummary(r.Context(), filter)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, err.Error()))
		return
	}

	totalPages := int(math.Ceil(float64(summary.Meta.TotalEmployees) / float64(perPage)))
	httpx.Paginated(w, summary.Data, httpx.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      summary.Meta.TotalEmployees,
		TotalPages: totalPages,
	})
}

// Export handles GET /api/v1/attendances/export (#72)
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	filter := ExportFilter{
		Format: r.URL.Query().Get("format"),
		Scope:  r.URL.Query().Get("scope"),
		AttendanceFilter: AttendanceFilter{
			Department:   r.URL.Query().Get("department"),
			StartDate:    r.URL.Query().Get("start_date"),
			EndDate:      r.URL.Query().Get("end_date"),
			WorkDate:     r.URL.Query().Get("work_date"),
			Status:       r.URL.Query().Get("status"),
			Type:         r.URL.Query().Get("type"),
			Method:       r.URL.Query().Get("method"),
			IsPending:    r.URL.Query().Get("is_pending") == "true",
			OnlyFallback: r.URL.Query().Get("only_fallback") == "true",
			Search:       r.URL.Query().Get("search"),
		},
	}
	if empStr := r.URL.Query().Get("employee_id"); empStr != "" {
		if empID, err := uuid.Parse(empStr); err == nil {
			filter.EmployeeID = &empID
		}
	}
	if filter.Format == "" {
		filter.Format = "csv"
	}
	if filter.Scope == "" {
		filter.Scope = "detail"
	}

	// 1. Validate max rows before streaming headers
	_, err := h.svc.ValidateExport(r.Context(), filter)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, err.Error()))
		return
	}

	// 2. Set headers
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	fromStr := filter.StartDate
	if fromStr == "" {
		fromStr = "all"
	}
	toStr := filter.EndDate
	if toStr == "" {
		toStr = time.Now().Format("2006-01-02")
	}
	filename := fmt.Sprintf("faceclock-absensi-%s_%s.csv", fromStr, toStr)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	// 3. Stream CSV
	var flusher http.Flusher
	if f, ok := w.(http.Flusher); ok {
		flusher = f
	}
	rowCount, err := h.svc.ExportCSV(r.Context(), w, flusher, filter)
	if err != nil {
		return
	}

	// 4. Record audit log attendance.exported
	if h.audit != nil {
		filterMap := map[string]any{
			"department": filter.Department,
			"status":     filter.Status,
			"type":       filter.Type,
			"method":     filter.Method,
			"scope":      filter.Scope,
		}
		_ = h.audit.RecordFromRequest(r, "attendance.exported", "attendance", nil, map[string]any{
			"from":      filter.StartDate,
			"to":        filter.EndDate,
			"filters":   filterMap,
			"row_count": rowCount,
		})
	}
}

// ApproveRecord handles POST /api/v1/attendances/{id}/approve (#62)
func (h *Handler) ApproveRecord(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid attendance id"))
		return
	}

	var req ReviewRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	req.Action = "approve"

	dto, err := h.svc.ReviewRecord(r.Context(), p, id, req)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, err.Error()))
		return
	}

	reviewerName := ""
	if dto.ReviewedByName != nil {
		reviewerName = *dto.ReviewedByName
	}
	resID := id.String()
	_ = h.audit.RecordFromRequest(r, "attendance.approve", "attendance", &resID, map[string]any{
		"action":        "approve",
		"attendance_id": resID,
		"reviewed_by":   p.UserID.String(),
	})

	reviewedAt := time.Now()
	if dto.ReviewedAt != nil {
		reviewedAt = *dto.ReviewedAt
	}

	httpx.OK(w, ApproveResponse{
		ID:     dto.ID,
		Status: dto.Status,
		ReviewedBy: ReviewerInfo{
			ID:    p.UserID,
			Email: p.Email,
			Name:  reviewerName,
		},
		ReviewedAt: reviewedAt,
	})
}

// RejectRecord handles POST /api/v1/attendances/{id}/reject (#63)
func (h *Handler) RejectRecord(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid attendance id"))
		return
	}

	var req ReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid json payload: "+err.Error()))
		return
	}
	req.Action = "reject"

	note := strings.TrimSpace(req.ReviewNote)
	if note == "" {
		note = strings.TrimSpace(req.ReviewNotes)
	}
	if len(note) < 3 || len(note) > 500 {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeValidationError, "review_note is required when rejecting (3-500 characters)"))
		return
	}

	dto, err := h.svc.ReviewRecord(r.Context(), p, id, req)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, err.Error()))
		return
	}

	reviewerName := ""
	if dto.ReviewedByName != nil {
		reviewerName = *dto.ReviewedByName
	}
	resID := id.String()
	_ = h.audit.RecordFromRequest(r, "attendance.reject", "attendance", &resID, map[string]any{
		"action":        "reject",
		"attendance_id": resID,
		"reviewed_by":   p.UserID.String(),
	})

	reviewedAt := time.Now()
	if dto.ReviewedAt != nil {
		reviewedAt = *dto.ReviewedAt
	}

	httpx.OK(w, RejectResponse{
		ID:               dto.ID,
		Status:           dto.Status,
		EmployeeCanRetry: true,
		ReviewedBy: ReviewerInfo{
			ID:    p.UserID,
			Email: p.Email,
			Name:  reviewerName,
		},
		ReviewedAt: reviewedAt,
	})
}

// BulkReviewRecords handles POST /api/v1/attendances/reviews (#64)
func (h *Handler) BulkReviewRecords(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	var req BulkReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid json payload: "+err.Error()))
		return
	}

	resp, err := h.svc.BulkReviewRecords(r.Context(), p, req)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, err.Error()))
		return
	}

	_ = h.audit.RecordFromRequest(r, "attendance.bulk_review", "attendance", nil, map[string]any{
		"processed": resp.Processed,
		"succeeded": resp.Succeeded,
		"failed":    resp.Failed,
	})

	httpx.OK(w, resp)
}

// GetAdminRecordByID handles GET /api/v1/attendance/admin/records/{id} (legacy alias)
func (h *Handler) GetAdminRecordByID(w http.ResponseWriter, r *http.Request) {
	h.GetRecordByID(w, r)
}

// ReviewRecord handles legacy single review POST /api/v1/attendance/admin/records/{id}/review
func (h *Handler) ReviewRecord(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid attendance id"))
		return
	}

	var req ReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid json payload: "+err.Error()))
		return
	}

	if req.Action == "reject" {
		h.RejectRecord(w, r)
		return
	}

	dto, err := h.svc.ReviewRecord(r.Context(), p, id, req)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, err.Error()))
		return
	}

	httpx.OK(w, dto)
}

// GetTeamRecords handles GET /api/v1/attendance/admin/team (#63)
func (h *Handler) GetTeamRecords(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	page, perPage := parsePagination(r)
	filter := AttendanceFilter{
		Page:      page,
		PerPage:   perPage,
		StartDate: r.URL.Query().Get("start_date"),
		EndDate:   r.URL.Query().Get("end_date"),
		Status:    r.URL.Query().Get("status"),
		Type:      r.URL.Query().Get("type"),
		Search:    r.URL.Query().Get("search"),
	}

	records, total, err := h.svc.GetTeamRecords(r.Context(), p, filter)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, err.Error()))
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	httpx.Paginated(w, records, httpx.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetStats handles GET /api/v1/attendance/admin/stats (#64)
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	stats, err := h.svc.GetStats(r.Context(), startDate, endDate)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, err.Error()))
		return
	}

	httpx.OK(w, stats)
}

// GetAttempts handles GET /api/v1/attendance/admin/attempts (#65)
func (h *Handler) GetAttempts(w http.ResponseWriter, r *http.Request) {
	page, perPage := parsePagination(r)
	filter := AttemptFilter{
		Page:      page,
		PerPage:   perPage,
		Outcome:   r.URL.Query().Get("outcome"),
		Type:      r.URL.Query().Get("type"),
		StartDate: r.URL.Query().Get("start_date"),
		EndDate:   r.URL.Query().Get("end_date"),
	}

	if empStr := r.URL.Query().Get("employee_id"); empStr != "" {
		if empID, err := uuid.Parse(empStr); err == nil {
			filter.EmployeeID = &empID
		}
	}

	attempts, total, err := h.svc.GetAttempts(r.Context(), filter)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, err.Error()))
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	httpx.Paginated(w, attempts, httpx.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	})
}

func parsePagination(r *http.Request) (page, perPage int) {
	page = 1
	perPage = 20
	if pStr := r.URL.Query().Get("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}
	if ppStr := r.URL.Query().Get("per_page"); ppStr != "" {
		if pp, err := strconv.Atoi(ppStr); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}
	return page, perPage
}

// extractClientIP returns the caller's bare IP address, preferring X-Real-IP
// over X-Forwarded-For since this deployment sets the former at the edge.
//
// The parsing (port stripping, IPv6 brackets, validation) lives in
// httpx.ClientIP so that attendance, audit and token issuance all agree on
// what a "client IP" is. Previously this returned r.RemoteAddr verbatim, which
// includes a port, and each consumer had to remember to strip it — one of them
// did (attempts.go), the others silently recorded nothing.
func extractClientIP(r *http.Request) string {
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if ip := httpx.ClientIP(&http.Request{
			Header: http.Header{"X-Forwarded-For": []string{xri}},
		}); ip != "" {
			return ip
		}
	}
	return httpx.ClientIP(r)
}

func parseClockRequest(r *http.Request) (*ClockRequest, error) {
	ct := r.Header.Get("Content-Type")

	var req ClockRequest

	// Extract Idempotency Key from header if present
	if idem := r.Header.Get("X-Idempotency-Key"); idem != "" {
		req.IdempotencyKey = &idem
	} else if idem := r.Header.Get("Idempotency-Key"); idem != "" {
		req.IdempotencyKey = &idem
	}

	if strings.HasPrefix(ct, "multipart/form-data") {
		const maxMem = 10 << 20 // 10MB
		if err := r.ParseMultipartForm(maxMem); err != nil {
			return nil, httpx.NewAppError(httpx.CodeBadRequest, "failed to parse multipart form: "+err.Error())
		}
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}

		file, header, err := r.FormFile("photo")
		if err != nil {
			// Try fallback names
			for _, fn := range []string{"image", "file"} {
				var e error
				file, header, e = r.FormFile(fn)
				if e == nil {
					err = nil
					break
				}
			}
			if err != nil {
				return nil, httpx.NewAppError(httpx.CodeBadRequest, "missing required 'photo' file in form")
			}
		}
		defer file.Close()

		photoBytes, err := io.ReadAll(file)
		if err != nil {
			return nil, httpx.NewAppError(httpx.CodeBadRequest, "failed to read photo file: "+err.Error())
		}
		req.PhotoBytes = photoBytes
		req.PhotoMime = header.Header.Get("Content-Type")
		if req.PhotoMime == "" {
			req.PhotoMime = "image/jpeg"
		}

		if latStr := r.FormValue("latitude"); latStr != "" {
			if lat, err := strconv.ParseFloat(latStr, 64); err == nil {
				req.Latitude = &lat
			}
		}
		if lonStr := r.FormValue("longitude"); lonStr != "" {
			if lon, err := strconv.ParseFloat(lonStr, 64); err == nil {
				req.Longitude = &lon
			}
		}
		if accStr := r.FormValue("accuracy"); accStr != "" {
			if acc, err := strconv.ParseFloat(accStr, 64); err == nil {
				req.Accuracy = &acc
			}
		}
		if mockStr := r.FormValue("location_is_mocked"); mockStr != "" {
			req.LocationIsMocked = mockStr == "true" || mockStr == "1"
		}
		if cTimeStr := r.FormValue("client_reported_at"); cTimeStr != "" {
			if parsed, err := time.Parse(time.RFC3339, cTimeStr); err == nil {
				req.ClientReportedAt = parsed
			}
		}
		if idem := r.FormValue("idempotency_key"); idem != "" {
			req.IdempotencyKey = &idem
		}
		if fb := r.FormValue("allow_fallback"); fb != "" {
			req.AllowFallback = fb == "true" || fb == "1"
		}
		if fbr := r.FormValue("fallback_reason"); fbr != "" {
			req.FallbackReason = &fbr
		}
		if fbn := r.FormValue("fallback_note"); fbn != "" {
			req.FallbackNote = &fbn
		}

		return &req, nil
	}

	// JSON request
	if strings.HasPrefix(ct, "application/json") {
		var jsonBody struct {
			PhotoBase64      string   `json:"photo_base64"`
			ImageBase64      string   `json:"image_base64"`
			Photo            string   `json:"photo"`
			Latitude         *float64 `json:"latitude"`
			Longitude        *float64 `json:"longitude"`
			Accuracy         *float64 `json:"accuracy"`
			LocationIsMocked bool     `json:"location_is_mocked"`
			ClientReportedAt *string  `json:"client_reported_at"`
			IdempotencyKey   *string  `json:"idempotency_key"`
			AllowFallback    bool     `json:"allow_fallback"`
			FallbackReason   *string  `json:"fallback_reason"`
			FallbackNote     *string  `json:"fallback_note"`
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, httpx.NewAppError(httpx.CodeBadRequest, "failed to read request body: "+err.Error())
		}
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		if err := json.Unmarshal(bodyBytes, &jsonBody); err != nil {
			return nil, httpx.NewAppError(httpx.CodeBadRequest, "invalid json payload: "+err.Error())
		}

		cleanBase64 := jsonBody.PhotoBase64
		if cleanBase64 == "" {
			cleanBase64 = jsonBody.ImageBase64
		}
		if cleanBase64 == "" {
			cleanBase64 = jsonBody.Photo
		}
		if cleanBase64 == "" {
			return nil, httpx.NewAppError(httpx.CodeBadRequest, "photo_base64 is required")
		}

		mime := "image/jpeg"
		if idx := strings.Index(cleanBase64, ","); idx != -1 {
			prefix := cleanBase64[:idx]
			if strings.Contains(prefix, "image/png") {
				mime = "image/png"
			} else if strings.Contains(prefix, "image/webp") {
				mime = "image/webp"
			}
			cleanBase64 = cleanBase64[idx+1:]
		}

		photoBytes, err := base64.StdEncoding.DecodeString(cleanBase64)
		if err != nil {
			return nil, httpx.NewAppError(httpx.CodeBadRequest, "invalid base64 photo encoding")
		}

		req.PhotoBytes = photoBytes
		req.PhotoMime = mime
		req.Latitude = jsonBody.Latitude
		req.Longitude = jsonBody.Longitude
		req.Accuracy = jsonBody.Accuracy
		req.LocationIsMocked = jsonBody.LocationIsMocked
		req.AllowFallback = jsonBody.AllowFallback
		req.FallbackReason = jsonBody.FallbackReason
		req.FallbackNote = jsonBody.FallbackNote

		if jsonBody.IdempotencyKey != nil && *jsonBody.IdempotencyKey != "" {
			req.IdempotencyKey = jsonBody.IdempotencyKey
		}

		if jsonBody.ClientReportedAt != nil && *jsonBody.ClientReportedAt != "" {
			if parsed, err := time.Parse(time.RFC3339, *jsonBody.ClientReportedAt); err == nil {
				req.ClientReportedAt = parsed
			}
		}

		return &req, nil
	}

	// Direct binary stream
	if strings.HasPrefix(ct, "image/") || ct == "application/octet-stream" {
		buf := &bytes.Buffer{}
		if _, err := io.Copy(buf, r.Body); err != nil {
			return nil, httpx.NewAppError(httpx.CodeBadRequest, "failed to read image body")
		}
		req.PhotoBytes = buf.Bytes()
		req.PhotoMime = ct
		return &req, nil
	}

	return nil, httpx.NewAppError(httpx.CodeUnsupportedMedia, "unsupported content type: must be multipart/form-data or application/json")
}
