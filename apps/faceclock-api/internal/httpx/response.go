package httpx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx/middleware"
)

// envelope mirrors the two shapes locked in Fase 0 § 2.6: {"data": ...} for
// success, {"error": {...}} for failure. Both are always top-level objects.
type successEnvelope struct {
	Data any   `json:"data"`
	Meta *Meta `json:"meta,omitempty"`
}

// Meta is the pagination block attached to list endpoints from Fase 1 on.
type Meta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      ErrorCode      `json:"code"`
	Message   string         `json:"message"`
	Details   []FieldError   `json:"details,omitempty"`
	RequestID string         `json:"request_id"`
	Extra     map[string]any `json:"-"` // merged flat into the JSON object by MarshalJSON below
}

// MarshalJSON flattens Extra into the same JSON object as the other fields,
// so error-specific fields like can_fallback sit next to code/message
// instead of nested under a generic "extra" key.
func (b errorBody) MarshalJSON() ([]byte, error) {
	out := map[string]any{
		"code":       b.Code,
		"message":    b.Message,
		"request_id": b.RequestID,
	}
	if len(b.Details) > 0 {
		out["details"] = b.Details
	}
	for k, v := range b.Extra {
		out[k] = v
	}
	return json.Marshal(out)
}

// OK writes a 200 success envelope.
func OK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, successEnvelope{Data: data})
}

// Created writes a 201 success envelope.
func Created(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusCreated, successEnvelope{Data: data})
}

// NoContent writes a bare 204, per Fase 1's logout convention. No envelope
// body is possible on 204 by definition.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Paginated writes a 200 success envelope with a meta block.
func Paginated(w http.ResponseWriter, data any, meta Meta) {
	writeJSON(w, http.StatusOK, successEnvelope{Data: data, Meta: &meta})
}

// Status writes a success envelope at a caller-chosen status — used by
// /readyz, which must return 503 while still shaping the body like a normal
// {"data": ...} response.
func Status(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, successEnvelope{Data: data})
}

// Fail writes an {"error": ...} envelope for the given AppError, resolving
// request_id from ctx (set by middleware.RequestID). This is the ONLY
// function in the codebase that should turn an AppError into bytes on the
// wire.
func Fail(ctx context.Context, w http.ResponseWriter, err *AppError) {
	FailWithStatus(ctx, w, StatusFor(err.Code), err)
}

// FailWithStatus writes the same {"error": ...} shape as Fail, but at a
// caller-chosen HTTP status instead of the one StatusFor(err.Code) would
// derive. This exists for the one case in the catalog that has a status
// without a matching code — 405 Method Not Allowed, which chi's router
// produces directly and which Fase 0 § 11.5 requires to still come back as
// an envelope, not chi's default plain-text body.
func FailWithStatus(ctx context.Context, w http.ResponseWriter, status int, err *AppError) {
	writeJSON(w, status, errorEnvelope{Error: errorBody{
		Code:      err.Code,
		Message:   err.Message,
		Details:   err.Details,
		RequestID: middleware.RequestIDFromContext(ctx),
		Extra:     err.Extra,
	}})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	// Encoding errors here mean the connection is already broken (client
	// gone) — nothing useful to do with the error at this point, and it must
	// not be written back into the same broken response.
	_ = json.NewEncoder(w).Encode(body)
}
