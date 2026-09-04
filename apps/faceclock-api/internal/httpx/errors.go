// Package httpx implements the response envelope, error catalog, and router
// wiring shared by every faceclock-api endpoint from Fase 0 onward (locked
// in Plan/01-Fase0.md § 2.6). No handler should ever call json.Encode
// directly or write a bare error string — everything goes through OK/Fail
// so the envelope shape can never drift per-endpoint.
package httpx

import "net/http"

// ErrorCode is the closed vocabulary of `error.code` values. Fase 0 owns the
// base 13. Later fases only ADD to this list via a reported contract
// revision (Plan/09-Revisions-Log.md) — never repurpose an existing code for
// a new meaning.
type ErrorCode string

// Base catalog, Fase 0 § 2.6.
const (
	CodeBadRequest         ErrorCode = "BAD_REQUEST"
	CodeUnauthenticated    ErrorCode = "UNAUTHENTICATED"
	CodeForbidden          ErrorCode = "FORBIDDEN"
	CodeNotFound           ErrorCode = "NOT_FOUND"
	CodeConflict           ErrorCode = "CONFLICT"
	CodePayloadTooLarge    ErrorCode = "PAYLOAD_TOO_LARGE"
	CodeUnsupportedMedia   ErrorCode = "UNSUPPORTED_MEDIA_TYPE"
	CodeValidationError    ErrorCode = "VALIDATION_ERROR"
	CodeRateLimited        ErrorCode = "RATE_LIMITED"
	CodeInternalError      ErrorCode = "INTERNAL_ERROR"
	CodeUpstreamError      ErrorCode = "UPSTREAM_ERROR"
	CodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	CodeUpstreamTimeout    ErrorCode = "UPSTREAM_TIMEOUT"
)

// codeStatus maps every known code to its HTTP status. Kept as a single
// source of truth so a code can never be thrown with a status that
// contradicts the catalog.
var codeStatus = map[ErrorCode]int{
	CodeBadRequest:         http.StatusBadRequest,
	CodeUnauthenticated:    http.StatusUnauthorized,
	CodeForbidden:          http.StatusForbidden,
	CodeNotFound:           http.StatusNotFound,
	CodeConflict:           http.StatusConflict,
	CodePayloadTooLarge:    http.StatusRequestEntityTooLarge,
	CodeUnsupportedMedia:   http.StatusUnsupportedMediaType,
	CodeValidationError:    http.StatusUnprocessableEntity,
	CodeRateLimited:        http.StatusTooManyRequests,
	CodeInternalError:      http.StatusInternalServerError,
	CodeUpstreamError:      http.StatusBadGateway,
	CodeServiceUnavailable: http.StatusServiceUnavailable,
	CodeUpstreamTimeout:    http.StatusGatewayTimeout,
}

// StatusFor returns the HTTP status registered for code. Panics if code was
// never registered via codeStatus — this is a programmer error (a new code
// used without updating the catalog), and Fase 0 § 2.6 explicitly wants that
// caught immediately rather than silently defaulting to 500.
func StatusFor(code ErrorCode) int {
	status, ok := codeStatus[code]
	if !ok {
		panic("httpx: unregistered error code used: " + string(code))
	}
	return status
}

// FieldError is one entry of the `error.details` array — always the shape
// used for validation errors.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// AppError is the typed error every handler should return for a non-2xx
// response. Wrap lower-level errors with NewAppError so message never leaks
// internal detail to the client (E7: never echo raw body/internal errors).
type AppError struct {
	Code    ErrorCode
	Message string
	Details []FieldError
	// Extra carries fields specific to one error (e.g. can_fallback,
	// existing_session_id in later fases) without widening the envelope
	// for every other error.
	Extra map[string]any
}

func (e *AppError) Error() string {
	return string(e.Code) + ": " + e.Message
}

// NewAppError builds a bare AppError with no details.
func NewAppError(code ErrorCode, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// NewValidationError builds a 422 VALIDATION_ERROR with field details.
func NewValidationError(message string, details []FieldError) *AppError {
	return &AppError{Code: CodeValidationError, Message: message, Details: details}
}
