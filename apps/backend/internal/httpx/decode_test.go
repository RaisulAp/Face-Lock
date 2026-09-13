package httpx_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx/middleware"
)

type samplePayload struct {
	Email string `json:"email" validate:"required,email"`
}

// TestBodyLimit_TriggersPayloadTooLarge is the E6 test that Fase 0 § 11.5
// wants: an oversized body must produce 413 PAYLOAD_TOO_LARGE. It is
// written here directly against BodyLimit + DecodeAndValidate rather than
// against GET /api/v1/version, because that endpoint (§ 4) is intentionally
// GET-only and never reads a body — a POST to it is correctly rejected with
// 405 by the router before BodyLimit's wrapped reader is ever exercised.
// This test exercises the actual code path a Fase 1+ POST handler will use.
func TestBodyLimit_TriggersPayloadTooLarge(t *testing.T) {
	const limit = 1024 // 1 KiB, small enough to keep the test fast

	handler := middleware.BodyLimit(limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload samplePayload
		if appErr := httpx.DecodeAndValidate(r, &payload); appErr != nil {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.OK(w, payload)
	}))

	oversized := bytes.Repeat([]byte("a"), limit*2)
	body := `{"email":"` + string(oversized) + `@example.com"}`

	req := httptest.NewRequest(http.MethodPost, "/whatever", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d: %s", rec.Code, rec.Body.String())
	}

	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if envelope.Error.Code != string(httpx.CodePayloadTooLarge) {
		t.Errorf("expected code PAYLOAD_TOO_LARGE, got %q", envelope.Error.Code)
	}
}

// TestDecodeAndValidate_MalformedJSONNeverEchoesBody is E7: a broken body
// must produce a generic 400, and the response must never contain the raw
// body content that failed to parse.
func TestDecodeAndValidate_MalformedJSONNeverEchoesBody(t *testing.T) {
	const secretLookingGarbage = `{"email": this-is-not-json-and-mentions-a-fake-token-xyz987`

	req := httptest.NewRequest(http.MethodPost, "/whatever", strings.NewReader(secretLookingGarbage))
	var payload samplePayload
	appErr := httpx.DecodeAndValidate(req, &payload)

	if appErr == nil {
		t.Fatal("expected an AppError for malformed JSON, got nil")
	}
	if appErr.Code != httpx.CodeBadRequest {
		t.Errorf("expected BAD_REQUEST, got %s", appErr.Code)
	}
	if strings.Contains(appErr.Message, "xyz987") {
		t.Errorf("error message leaked raw body content: %q", appErr.Message)
	}
}

// TestDecodeAndValidate_ValidationErrorHasFieldDetails is the 422 path.
func TestDecodeAndValidate_ValidationErrorHasFieldDetails(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/whatever", strings.NewReader(`{"email":"not-an-email"}`))
	var payload samplePayload
	appErr := httpx.DecodeAndValidate(req, &payload)

	if appErr == nil {
		t.Fatal("expected a validation AppError, got nil")
	}
	if appErr.Code != httpx.CodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %s", appErr.Code)
	}
	if len(appErr.Details) != 1 || appErr.Details[0].Field != "email" {
		t.Errorf("expected one field error on \"email\", got %+v", appErr.Details)
	}
}
