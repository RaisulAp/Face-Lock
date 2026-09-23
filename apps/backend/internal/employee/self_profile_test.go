package employee

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
)

// TestSelfProfileParamsDecoding locks down the API contract for the
// self-service profile endpoint. These are the three properties that matter:
//
//  1. The four whitelisted fields decode.
//  2. The admin-only fields (employment_status, department, email) are REJECTED
//     rather than silently ignored, because DecodeAndValidate sets
//     DisallowUnknownFields. A client trying to escalate privileges must get a
//     400, not a 200 with the field quietly dropped.
//  3. Field-level validation still produces 422 with the offending field named.
func TestSelfProfileParamsDecoding(t *testing.T) {
	post := func(t *testing.T, body string) *httpx.AppError {
		t.Helper()
		req := httptest.NewRequest("PATCH", "/api/v1/employees/me/profile", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		var params SelfProfileParams
		return httpx.DecodeAndValidate(req, &params)
	}

	t.Run("accepts the whitelisted fields", func(t *testing.T) {
		if err := post(t, `{
			"employee_number": "EMP-00042",
			"position": "Staf Keuangan",
			"join_date": "2026-02-01",
			"phone": "081234567890"
		}`); err != nil {
			t.Fatalf("expected payload to be accepted, got %v (%s)", err, err.Code)
		}
	})

	t.Run("accepts a partial payload", func(t *testing.T) {
		// Employees fill the form in stages; omitting fields must be allowed.
		if err := post(t, `{"phone": "081234567890"}`); err != nil {
			t.Fatalf("expected partial payload to be accepted, got %v (%s)", err, err.Code)
		}
	})

	t.Run("rejects admin-only fields", func(t *testing.T) {
		// Each of these is a field an employee must not be able to set.
		for _, body := range []string{
			`{"employment_status": "inactive"}`,
			`{"department": "Finance"}`,
			`{"email": "someone@example.com"}`,
			`{"office_location_id": "00000000-0000-0000-0000-000000000000"}`,
			`{"profile_completed": true}`,
		} {
			err := post(t, body)
			if err == nil {
				t.Fatalf("expected %s to be rejected, but it was accepted", body)
			}
			if err.Code != httpx.CodeBadRequest {
				t.Errorf("expected %s to yield %s, got %s", body, httpx.CodeBadRequest, err.Code)
			}
		}
	})

	t.Run("maps bad values to 422 with the field name", func(t *testing.T) {
		cases := []struct {
			body      string
			wantField string
		}{
			{`{"join_date": "01-02-2026"}`, "join_date"},
			{`{"employee_number": "x"}`, "employee_number"},
			{`{"phone": "1234567890123456789012345"}`, "phone"},
		}
		for _, tc := range cases {
			err := post(t, tc.body)
			if err == nil {
				t.Fatalf("expected %s to fail validation", tc.body)
			}
			if err.Code != httpx.CodeValidationError {
				t.Errorf("expected %s to yield %s, got %s", tc.body, httpx.CodeValidationError, err.Code)
				continue
			}
			if len(err.Details) == 0 {
				t.Errorf("expected field details for %s, got none", tc.body)
				continue
			}
			if err.Details[0].Field != tc.wantField {
				t.Errorf("for %s expected field %q, got %q", tc.body, tc.wantField, err.Details[0].Field)
			}
		}
	})
}
