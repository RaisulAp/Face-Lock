package audit

import (
	"net/http"
	"strconv"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/google/uuid"
)

// Handler provides HTTP endpoints for audit log inspection.
type Handler struct {
	recorder *Recorder
}

// NewHandler creates a new audit log handler.
func NewHandler(recorder *Recorder) *Handler {
	return &Handler{recorder: recorder}
}

// List handles GET /api/v1/audit-logs
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := QueryFilter{
		Page:         1,
		PerPage:      20,
		Action:       q.Get("action"),
		ResourceType: q.Get("resource_type"),
	}

	if p, err := strconv.Atoi(q.Get("page")); err == nil && p > 0 {
		filter.Page = p
	}
	if pp, err := strconv.Atoi(q.Get("per_page")); err == nil && pp > 0 {
		filter.PerPage = pp
	}

	if actorRaw := q.Get("actor_user_id"); actorRaw != "" {
		if uid, err := uuid.Parse(actorRaw); err == nil {
			filter.ActorUserID = &uid
		} else {
			httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeValidationError, "invalid actor_user_id format"))
			return
		}
	}

	if fromRaw := q.Get("date_from"); fromRaw != "" {
		if t, err := parseDate(fromRaw); err == nil {
			filter.DateFrom = &t
		} else {
			httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeValidationError, "invalid date_from format"))
			return
		}
	}

	if toRaw := q.Get("date_to"); toRaw != "" {
		if t, err := parseDate(toRaw); err == nil {
			filter.DateTo = &t
		} else {
			httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeValidationError, "invalid date_to format"))
			return
		}
	}

	logs, meta, err := h.recorder.Query(r.Context(), filter)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to retrieve audit logs"))
		return
	}

	httpx.Paginated(w, logs, meta)
}

func parseDate(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}
