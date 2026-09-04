package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// DecodeAndValidate reads r.Body into dst (a pointer to a struct with
// `validate:"..."` tags), and returns a ready-to-send *AppError on any
// failure — never a bare error the caller has to re-classify.
//
// Distinguishes:
//   - body too large (E6, from the BodyLimit-wrapped reader) → 413 PAYLOAD_TOO_LARGE
//   - malformed JSON (E7) → 400 BAD_REQUEST, generic message, body never echoed back
//   - well-formed JSON that fails validation → 422 VALIDATION_ERROR with field details
func DecodeAndValidate(r *http.Request, dst any) *AppError {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		if isMaxBytesError(err) {
			return NewAppError(CodePayloadTooLarge, "Ukuran request melebihi batas maksimum")
		}
		if errors.Is(err, io.EOF) {
			return NewAppError(CodeBadRequest, "Request body kosong")
		}
		// E7: never reflect the parser's message (which can contain a
		// snippet of the client's raw body) back into the response.
		return NewAppError(CodeBadRequest, "Request tidak valid")
	}

	if err := validate.Struct(dst); err != nil {
		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) {
			details := make([]FieldError, 0, len(verrs))
			for _, fe := range verrs {
				details = append(details, FieldError{
					Field:   toSnakeCase(fe.Field()),
					Message: humanizeValidation(fe),
				})
			}
			return NewValidationError("Request tidak valid", details)
		}
		return NewAppError(CodeBadRequest, "Request tidak valid")
	}

	return nil
}

// isMaxBytesError detects http.MaxBytesReader's sentinel without depending
// on stdlib version-specific error types (the exported http.MaxBytesError
// was only added in Go 1.19+; this string check is the compatible fallback
// used until every deployment target is confirmed >= 1.19).
func isMaxBytesError(err error) bool {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return true
	}
	return strings.Contains(err.Error(), "http: request body too large")
}

func humanizeValidation(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "email":
		return "format email tidak valid"
	case "min":
		return "nilai terlalu kecil"
	case "max":
		return "nilai terlalu besar"
	default:
		return "nilai tidak valid"
	}
}

// toSnakeCase converts a Go struct field name (PascalCase) to the
// snake_case name used on the wire (D4), so validation error field names
// match what the client actually sent.
func toSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}
