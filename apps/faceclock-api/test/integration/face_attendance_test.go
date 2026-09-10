package integration

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/auth"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/face"
)

// executeWithCookie sends a request attaching the access_token HttpOnly cookie
func executeWithCookie(app *TestApp, req *http.Request, token string, csrfHeader bool) (*http.Response, []byte) {
	if token != "" {
		req.AddCookie(&http.Cookie{
			Name:     auth.AccessTokenCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			MaxAge:   auth.AccessTokenTTL,
			SameSite: http.SameSiteLaxMode,
		})
	}
	if !csrfHeader {
		req.Header.Set("X-Skip-CSRF-Header", "true")
	}

	rec, body := app.ExecuteRequest(req)
	return rec.Result(), body
}

func TestFaceEnrollmentAndAttendance_Integration(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Pool.Close()

	ctx := context.Background()

	// 1. Setup Admin & Employee accounts
	adminUser := app.EnsureUserWithRole(t, "admin")
	empUser1 := app.EnsureUserWithRole(t, "employee")
	empUser2 := app.EnsureUserWithRole(t, "employee")

	empID1 := *empUser1.EmployeeID
	empID2 := *empUser2.EmployeeID

	// Sample mock photo payloads
	facePhotoA := []byte("face_photo_of_alice_in_wonderland_high_res_sample_data_123")
	facePhotoB := []byte("face_photo_of_bob_the_builder_completely_different_face_789")

	// -------------------------------------------------------------
	// Test Anti-CSRF Shield on Face Enroll & Attendance
	// -------------------------------------------------------------
	t.Run("Anti-CSRF shield blocks mutating endpoints without header", func(t *testing.T) {
		enrollURL := fmt.Sprintf("/api/v1/employees/%s/face-enroll", empID1.String())
		req, _ := http.NewRequest(http.MethodPost, enrollURL, bytes.NewReader(facePhotoA))
		req.Header.Set("Content-Type", "image/jpeg")

		res, body := executeWithCookie(app, req, empUser1.Token, false)
		if res.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for missing CSRF header, got %d: %s", res.StatusCode, string(body))
		}

		clockReq, _ := http.NewRequest(http.MethodPost, "/api/v1/attendance/clock-in", bytes.NewReader([]byte("{}")))
		clockReq.Header.Set("Content-Type", "application/json")
		resClock, bodyClock := executeWithCookie(app, clockReq, empUser1.Token, false)
		if resClock.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for missing CSRF header on clock-in, got %d: %s", resClock.StatusCode, string(bodyClock))
		}
	})

	// -------------------------------------------------------------
	// Test RBAC on Face Enrollment
	// -------------------------------------------------------------
	t.Run("RBAC: Employee cannot enroll another employee's face", func(t *testing.T) {
		// empUser1 attempts to enroll empID2
		enrollURL := fmt.Sprintf("/api/v1/employees/%s/face-enroll", empID2.String())
		req, _ := http.NewRequest(http.MethodPost, enrollURL, bytes.NewReader(facePhotoA))
		req.Header.Set("Content-Type", "image/jpeg")

		res, body := executeWithCookie(app, req, empUser1.Token, true)
		if res.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden when employee enrolls another employee, got %d: %s", res.StatusCode, string(body))
		}
	})

	// -------------------------------------------------------------
	// Test Face Enrollment via Multipart Form
	// -------------------------------------------------------------
	t.Run("Employee can enroll self face via multipart form", func(t *testing.T) {
		bodyBuf := &bytes.Buffer{}
		writer := multipart.NewWriter(bodyBuf)
		part, err := writer.CreateFormFile("photo", "face.jpg")
		if err != nil {
			t.Fatalf("create form file failed: %v", err)
		}
		_, _ = part.Write(facePhotoA)
		_ = writer.Close()

		enrollURL := fmt.Sprintf("/api/v1/employees/%s/face-enroll", empID1.String())
		req, _ := http.NewRequest(http.MethodPost, enrollURL, bodyBuf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		res, body := executeWithCookie(app, req, empUser1.Token, true)
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201 Created on face enrollment, got %d: %s", res.StatusCode, string(body))
		}

		var respData struct {
			Data struct {
				EmployeeID string  `json:"employee_id"`
				PhotoPath  string  `json:"photo_path"`
				Quality    float64 `json:"quality_score"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &respData); err != nil {
			t.Fatalf("failed decoding json response: %v", err)
		}

		if respData.Data.PhotoPath == "" {
			t.Errorf("expected photo_path to be non-empty")
		}

		// Verify database state: face_enrolled_at NOT NULL, face_embedding NOT NULL
		var isReg bool
		var embStr *string
		err = app.Pool.QueryRow(ctx, "SELECT face_enrolled_at IS NOT NULL, face_embedding::text FROM employees WHERE id = $1", empID1).Scan(&isReg, &embStr)
		if err != nil {
			t.Fatalf("failed querying employee face record: %v", err)
		}
		if !isReg || embStr == nil || *embStr == "" {
			t.Fatalf("database not updated: face_enrolled_at IS NOT NULL=%v, embedding=%v", isReg, embStr)
		}

		// Verify audit log
		var auditCount int
		err = app.Pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM audit_logs
			WHERE action = 'employee.face_enroll' AND resource_id = $1
		`, empID1.String()).Scan(&auditCount)
		if err != nil {
			t.Fatalf("querying audit log failed: %v", err)
		}
		if auditCount < 1 {
			t.Errorf("expected audit log for employee.face_enroll, got %d records", auditCount)
		}
	})

	// -------------------------------------------------------------
	// Test Admin enrolling another employee via JSON base64
	// -------------------------------------------------------------
	t.Run("Admin can enroll any employee via JSON base64 payload", func(t *testing.T) {
		jsonPayload, _ := json.Marshal(map[string]string{
			"photo_base64": base64.StdEncoding.EncodeToString(facePhotoB),
		})

		enrollURL := fmt.Sprintf("/api/v1/employees/%s/face-enroll", empID2.String())
		req, _ := http.NewRequest(http.MethodPost, enrollURL, bytes.NewReader(jsonPayload))
		req.Header.Set("Content-Type", "application/json")

		res, body := executeWithCookie(app, req, adminUser.Token, true)
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("admin enroll failed: expected 201, got %d: %s", res.StatusCode, string(body))
		}

		var isReg bool
		_ = app.Pool.QueryRow(ctx, "SELECT face_enrolled_at IS NOT NULL FROM employees WHERE id = $1", empID2).Scan(&isReg)
		if !isReg {
			t.Fatalf("expected empID2 face_enrolled_at to be non-null")
		}
	})

	// -------------------------------------------------------------
	// Test Clock-In before face enrolled for an un-enrolled employee
	// -------------------------------------------------------------
	t.Run("Clock-in fails if employee face is not registered", func(t *testing.T) {
		empNoFaceUser := app.EnsureUserWithRole(t, "employee")

		clockInPayload, _ := json.Marshal(map[string]any{
			"photo_base64": base64.StdEncoding.EncodeToString(facePhotoA),
			"latitude":     -6.2088,
			"longitude":    106.8456,
		})

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/attendance/clock-in", bytes.NewReader(clockInPayload))
		req.Header.Set("Content-Type", "application/json")

		res, body := executeWithCookie(app, req, empNoFaceUser.Token, true)
		if res.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 for un-enrolled employee, got %d: %s", res.StatusCode, string(body))
		}
	})

	// -------------------------------------------------------------
	// Test Clock-In with Mismatched Face (422 FACE_MISMATCH)
	// -------------------------------------------------------------
	t.Run("Clock-in with mismatched face returns 422 FACE_MISMATCH", func(t *testing.T) {
		// empID1 is enrolled with facePhotoA. Let's submit facePhotoB.
		clockInPayload, _ := json.Marshal(map[string]any{
			"photo_base64": base64.StdEncoding.EncodeToString(facePhotoB),
			"latitude":     -6.2088,
			"longitude":    106.8456,
			"device_info":  "Pixel 8 / Chrome 120",
		})

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/attendance/clock-in", bytes.NewReader(clockInPayload))
		req.Header.Set("Content-Type", "application/json")

		res, body := executeWithCookie(app, req, empUser1.Token, true)
		if res.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 Unprocessable Entity on face mismatch, got %d: %s", res.StatusCode, string(body))
		}

		// Verify that a failed record was recorded in attendances
		var failedCount int
		_ = app.Pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM attendances
			WHERE employee_id = $1 AND status = 'failed'
		`, empID1).Scan(&failedCount)
		if failedCount < 1 {
			t.Errorf("expected failed attendance record to be logged, got %d", failedCount)
		}
	})

	// -------------------------------------------------------------
	// Test Clock-In with Matching Face (Success 201)
	// -------------------------------------------------------------
	t.Run("Clock-in with matching face succeeds and records similarity", func(t *testing.T) {
		// empID1 submits facePhotoA (identical to enrolled face)
		clockInPayload, _ := json.Marshal(map[string]any{
			"photo_base64": base64.StdEncoding.EncodeToString(facePhotoA),
			"latitude":     -6.2088,
			"longitude":    106.8456,
			"accuracy":     12.5,
			"device_info":  "iPhone 15 / Safari 17",
		})

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/attendance/clock-in", bytes.NewReader(clockInPayload))
		req.Header.Set("Content-Type", "application/json")

		res, body := executeWithCookie(app, req, empUser1.Token, true)
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201 Created on matching clock-in, got %d: %s", res.StatusCode, string(body))
		}

		var respData struct {
			Data struct {
				ID              string  `json:"id"`
				Type            string  `json:"type"`
				Status          string  `json:"status"`
				SimilarityScore float32 `json:"similarity_score"`
				Distance        float32 `json:"distance"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &respData); err != nil {
			t.Fatalf("failed decoding json response: %v", err)
		}

		if respData.Data.Status != "success" {
			t.Errorf("expected status 'success', got '%s'", respData.Data.Status)
		}
		if respData.Data.Type != "in" {
			t.Errorf("expected clock type 'in', got '%s'", respData.Data.Type)
		}
		if respData.Data.SimilarityScore < 0.75 {
			t.Errorf("expected face similarity >= 0.75, got %f", respData.Data.SimilarityScore)
		}

		// Verify audit log for attendance.clock_in
		var auditCount int
		qErr := app.Pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM audit_logs
			WHERE action = 'attendance.clock_in' AND resource_id = $1
		`, respData.Data.ID).Scan(&auditCount)
		if qErr != nil {
			t.Fatalf("querying attendance audit log failed: %v", qErr)
		}
		if auditCount < 1 {
			t.Errorf("expected audit log for attendance.clock_in, got %d", auditCount)
		}
	})

	// -------------------------------------------------------------
	// Test Clock-Out with Matching Face (Success 201)
	// -------------------------------------------------------------
	t.Run("Clock-out with matching face succeeds", func(t *testing.T) {
		clockOutPayload, _ := json.Marshal(map[string]any{
			"photo_base64": base64.StdEncoding.EncodeToString(facePhotoA),
			"latitude":     -6.2088,
			"longitude":    106.8456,
			"accuracy":     10.0,
			"device_info":  "iPhone 15 / Safari 17",
		})

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/attendance/clock-out", bytes.NewReader(clockOutPayload))
		req.Header.Set("Content-Type", "application/json")

		res, body := executeWithCookie(app, req, empUser1.Token, true)
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201 Created on clock-out, got %d: %s", res.StatusCode, string(body))
		}

		var respData struct {
			Data struct {
				ID     string `json:"id"`
				Type   string `json:"type"`
				Status string `json:"status"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &respData); err != nil {
			t.Fatalf("failed decoding json response: %v", err)
		}

		if respData.Data.Status != "success" {
			t.Errorf("expected status 'success', got '%s'", respData.Data.Status)
		}
		if respData.Data.Type != "out" {
			t.Errorf("expected clock type 'out', got '%s'", respData.Data.Type)
		}
	})

	// -------------------------------------------------------------
	// Test Quality Rejection (No Face Detected)
	// -------------------------------------------------------------
	t.Run("Enrollment fails with 422 if no face is detected in image", func(t *testing.T) {
		app.FaceEngine.SetForceError(face.ErrNoFaceDetected)
		defer app.FaceEngine.SetForceError(nil)

		enrollURL := fmt.Sprintf("/api/v1/employees/%s/face-enroll", empID1.String())
		req, _ := http.NewRequest(http.MethodPost, enrollURL, bytes.NewReader(facePhotoA))
		req.Header.Set("Content-Type", "image/jpeg")

		res, body := executeWithCookie(app, req, empUser1.Token, true)
		if res.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 Unprocessable Entity when no face detected, got %d: %s", res.StatusCode, string(body))
		}
	})
}
