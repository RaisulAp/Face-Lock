package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/consent"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/face/enrollment"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/face/reference"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/face/reindex"
	"github.com/google/uuid"
)

// Helper to generate a valid JPEG in memory
func generateTestJPEG(seed byte) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{R: 210, G: 180, B: seed, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

// Helper to login and get session cookies
func loginWithCookies(t *testing.T, app *TestApp, email, password string) []*http.Cookie {
	t.Helper()
	loginPayload, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginPayload))
	req.Header.Set("Content-Type", "application/json")
	rec, body := app.ExecuteRequest(req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login failed for %s: code %d, body: %s", email, rec.Code, string(body))
	}

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("no cookies returned from login for %s", email)
	}
	return cookies
}

// Helper to make an authenticated request using cookies
func execCookieRequest(app *TestApp, method, path string, cookies []*http.Cookie, body io.Reader, contentType string) (*http.Response, []byte) {
	req, _ := http.NewRequest(method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec, respBody := app.ExecuteRequest(req)
	res := rec.Result()
	return res, respBody
}

func TestBiometricConsent_FullLifecycle(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Pool.Close()

	// 1. Create employee user
	empUser := app.EnsureUserWithRole(t, "employee")
	adminUser := app.EnsureUserWithRole(t, "admin")

	empCookies := loginWithCookies(t, app, empUser.Email, "TestPass123!")
	adminCookies := loginWithCookies(t, app, adminUser.Email, "TestPass123!")

	// 2. GET /api/v1/consents/document -> Returns active document
	res, body := execCookieRequest(app, http.MethodGet, "/api/v1/consents/document", empCookies, nil, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get document failed: %d, body: %s", res.StatusCode, string(body))
	}
	var docResp struct {
		Data consent.ConsentDocumentResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &docResp); err != nil {
		t.Fatalf("unmarshal document response failed: %v", err)
	}
	if docResp.Data.Version != "2026-09-v1" {
		t.Fatalf("expected document version '2026-09-v1', got %s", docResp.Data.Version)
	}
	if docResp.Data.ContentHash == "" {
		t.Fatalf("expected non-empty content_hash")
	}

	// 3. GET /api/v1/consents/me -> Initially returns not_submitted
	res, body = execCookieRequest(app, http.MethodGet, "/api/v1/consents/me", empCookies, nil, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get my consent failed: %d, body: %s", res.StatusCode, string(body))
	}
	var meResp struct {
		Data consent.ConsentMeResponse `json:"data"`
	}
	_ = json.Unmarshal(body, &meResp)
	if meResp.Data.Status != "none" {
		t.Fatalf("expected status 'none', got %s", meResp.Data.Status)
	}

	// 4. Negative test: POST /api/v1/consents with invalid document version -> 409 Conflict
	badPayload, _ := json.Marshal(consent.GrantConsentRequest{
		DocumentVersion: "wrong-version-v99",
		Agreed:          true,
	})
	res, _ = execCookieRequest(app, http.MethodPost, "/api/v1/consents", empCookies, bytes.NewReader(badPayload), "application/json")
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for wrong version, got %d", res.StatusCode)
	}

	// 5. Validation test: POST /api/v1/consents with agreed=false -> 422 Unprocessable Entity
	refusePayload, _ := json.Marshal(consent.GrantConsentRequest{
		DocumentVersion: docResp.Data.Version,
		Agreed:          false,
	})
	res, body = execCookieRequest(app, http.MethodPost, "/api/v1/consents", empCookies, bytes.NewReader(refusePayload), "application/json")
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for agreed=false, got %d: %s", res.StatusCode, string(body))
	}

	// 6. Grant test: POST /api/v1/consents with agreed=true -> Consent active
	grantPayload, _ := json.Marshal(consent.GrantConsentRequest{
		DocumentVersion: docResp.Data.Version,
		Agreed:          true,
	})
	res, body = execCookieRequest(app, http.MethodPost, "/api/v1/consents", empCookies, bytes.NewReader(grantPayload), "application/json")
	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		t.Fatalf("grant consent failed: %d, body: %s", res.StatusCode, string(body))
	}

	// 7. GET /api/v1/consents/me -> Now returns status active
	res, body = execCookieRequest(app, http.MethodGet, "/api/v1/consents/me", empCookies, nil, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get my consent after grant failed: %d, body: %s", res.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &meResp)
	if meResp.Data.Status != "granted" {
		t.Fatalf("expected status 'granted', got %s", meResp.Data.Status)
	}
	if !meResp.Data.IsCurrentVersion {
		t.Fatalf("expected IsCurrentVersion to be true")
	}

	// 8. Admin inspects employee's consent: GET /api/v1/employees/{id}/consent
	adminInspectURL := fmt.Sprintf("/api/v1/employees/%s/consent", empUser.EmployeeID.String())
	res, body = execCookieRequest(app, http.MethodGet, adminInspectURL, adminCookies, nil, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("admin inspect consent failed: %d, body: %s", res.StatusCode, string(body))
	}
	var adminInspectResp struct {
		Data consent.EmployeeConsentResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &adminInspectResp); err != nil {
		t.Fatalf("unmarshal admin inspect response failed: %v", err)
	}
	if adminInspectResp.Data.Status != "granted" {
		t.Fatalf("expected admin to see 'granted', got %s", adminInspectResp.Data.Status)
	}

	// 9. Admin records physical paper consent on behalf of another employee
	otherEmpUser := app.EnsureUserWithRole(t, "employee")
	adminRecordURL := fmt.Sprintf("/api/v1/employees/%s/consent", otherEmpUser.EmployeeID.String())
	adminRecordPayload, _ := json.Marshal(consent.RecordAdminConsentRequest{
		DocumentVersion: docResp.Data.Version,
		Note:            "Ditandatangani secara fisik di HRD",
	})
	res, body = execCookieRequest(app, http.MethodPost, adminRecordURL, adminCookies, bytes.NewReader(adminRecordPayload), "application/json")
	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		t.Fatalf("admin record consent failed: %d, body: %s", res.StatusCode, string(body))
	}

	// 10. Withdrawal test: POST /api/v1/consents/withdraw -> Withdrawn
	withdrawPayload, _ := json.Marshal(consent.WithdrawConsentRequest{
		Reason: "Menolak pemrosesan data wajah sesuai UU PDP",
	})
	res, body = execCookieRequest(app, http.MethodPost, "/api/v1/consents/withdraw", empCookies, bytes.NewReader(withdrawPayload), "application/json")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("withdraw consent failed: %d, body: %s", res.StatusCode, string(body))
	}
	var withdrawResp struct {
		Data consent.WithdrawConsentResponse `json:"data"`
	}
	_ = json.Unmarshal(body, &withdrawResp)
	if withdrawResp.Data.Status != "withdrawn" {
		t.Fatalf("expected withdraw status 'withdrawn', got %s", withdrawResp.Data.Status)
	}
	if withdrawResp.Data.AttendanceModeHint == "" {
		t.Fatalf("expected non-empty attendance_mode_hint")
	}

	// Check me status after withdrawal
	res, body = execCookieRequest(app, http.MethodGet, "/api/v1/consents/me", empCookies, nil, "")
	_ = json.Unmarshal(body, &meResp)
	if meResp.Data.Status != "withdrawn" {
		t.Fatalf("expected status 'withdrawn', got %s", meResp.Data.Status)
	}
}

func TestMultiPhotoEnrollment_CompleteFlow(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Pool.Close()

	empUser := app.EnsureUserWithRole(t, "employee")
	empCookies := loginWithCookies(t, app, empUser.Email, "TestPass123!")

	// 1. Without consent: POST /api/v1/face/enrollments -> 403 Forbidden
	res, body := execCookieRequest(app, http.MethodPost, "/api/v1/face/enrollments", empCookies, bytes.NewReader([]byte("{}")), "application/json")
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 when creating session without consent, got %d: %s", res.StatusCode, string(body))
	}

	// 2. Grant consent
	grantPayload, _ := json.Marshal(consent.GrantConsentRequest{
		DocumentVersion: "2026-09-v1",
		Agreed:          true,
	})
	res, _ = execCookieRequest(app, http.MethodPost, "/api/v1/consents", empCookies, bytes.NewReader(grantPayload), "application/json")
	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		t.Fatalf("grant consent failed: %d", res.StatusCode)
	}

	// 3. Create enrollment session: POST /api/v1/face/enrollments -> 201 Created
	createSessPayload, _ := json.Marshal(enrollment.CreateSessionRequest{
		Mode: "replace",
	})
	res, body = execCookieRequest(app, http.MethodPost, "/api/v1/face/enrollments", empCookies, bytes.NewReader(createSessPayload), "application/json")
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create enrollment session failed: %d: %s", res.StatusCode, string(body))
	}
	var sessResp struct {
		Data enrollment.SessionResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &sessResp); err != nil {
		t.Fatalf("unmarshal session response failed: %v", err)
	}
	sessID := sessResp.Data.ID
	if sessID == uuid.Nil {
		t.Fatalf("expected valid session ID")
	}
	if sessResp.Data.Status != "draft" {
		t.Fatalf("expected session status 'draft', got %s", sessResp.Data.Status)
	}
	if sessResp.Data.RequiredPhotos != 3 {
		t.Fatalf("expected required_photos=3, got %d", sessResp.Data.RequiredPhotos)
	}

	// 4. Upload 3 photos sequentially
	uploadURL := fmt.Sprintf("/api/v1/face/enrollments/%s/photos", sessID.String())

	for i := 1; i <= 3; i++ {
		jpegBytes := generateTestJPEG(byte(i * 40))
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		part, err := w.CreateFormFile("image", fmt.Sprintf("photo_%d.jpg", i))
		if err != nil {
			t.Fatalf("create form file failed: %v", err)
		}
		if _, err := part.Write(jpegBytes); err != nil {
			t.Fatalf("write photo bytes failed: %v", err)
		}
		_ = w.WriteField("capture_source", "web_camera")
		_ = w.Close()

		res, body = execCookieRequest(app, http.MethodPost, uploadURL, empCookies, &b, w.FormDataContentType())
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("upload photo %d failed: %d: %s", i, res.StatusCode, string(body))
		}
		var photoResp struct {
			Data enrollment.UploadPhotoResponse `json:"data"`
		}
		if err := json.Unmarshal(body, &photoResp); err != nil {
			t.Fatalf("unmarshal photo response failed: %v", err)
		}
		if photoResp.Data.Position != i {
			t.Fatalf("expected photo position %d, got %d", i, photoResp.Data.Position)
		}
	}

	// 5. Check session: GET /api/v1/face/enrollments/{id} -> contains 3 photos
	getSessURL := fmt.Sprintf("/api/v1/face/enrollments/%s", sessID.String())
	res, body = execCookieRequest(app, http.MethodGet, getSessURL, empCookies, nil, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get session failed: %d: %s", res.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &sessResp)
	if len(sessResp.Data.Photos) != 3 {
		t.Fatalf("expected 3 photos in session, got %d", len(sessResp.Data.Photos))
	}

	// 6. Commit session: POST /api/v1/face/enrollments/{id}/commit -> 200 OK
	commitURL := fmt.Sprintf("/api/v1/face/enrollments/%s/commit", sessID.String())
	res, body = execCookieRequest(app, http.MethodPost, commitURL, empCookies, nil, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("commit session failed: %d: %s", res.StatusCode, string(body))
	}

	// 7. Verify face references created in database
	var refCount int
	err := app.Pool.QueryRow(t.Context(), `
		SELECT COUNT(*) FROM face_references 
		WHERE employee_id = $1 AND is_active = true
	`, empUser.EmployeeID).Scan(&refCount)
	if err != nil {
		t.Fatalf("count references query failed: %v", err)
	}
	if refCount != 3 {
		t.Fatalf("expected 3 active references in DB, got %d", refCount)
	}

	// 8. Verify employee attendance_mode updated to face_biometric
	var attMode string
	err = app.Pool.QueryRow(t.Context(), "SELECT attendance_mode FROM employees WHERE id = $1", empUser.EmployeeID).Scan(&attMode)
	if err != nil {
		t.Fatalf("query attendance mode failed: %v", err)
	}
	if attMode != "face" {
		t.Fatalf("expected attendance_mode 'face', got %s", attMode)
	}

	// 9. Verify enrollment status: GET /api/v1/face/enrollment-status/me
	res, body = execCookieRequest(app, http.MethodGet, "/api/v1/face/enrollment-status/me", empCookies, nil, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get enrollment status me failed: %d: %s", res.StatusCode, string(body))
	}
	var statusResp struct {
		Data reference.EnrollmentStatusResponse `json:"data"`
	}
	_ = json.Unmarshal(body, &statusResp)
	if !statusResp.Data.IsEnrolled {
		t.Fatalf("expected IsEnrolled to be true")
	}
	if statusResp.Data.ActiveReferenceCount != 3 {
		t.Fatalf("expected 3 active references, got %d", statusResp.Data.ActiveReferenceCount)
	}

	// 10. Verify reference list: GET /api/v1/employees/{id}/face-references
	refsURL := fmt.Sprintf("/api/v1/employees/%s/face-references", empUser.EmployeeID.String())
	res, body = execCookieRequest(app, http.MethodGet, refsURL, empCookies, nil, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list references failed: %d: %s", res.StatusCode, string(body))
	}
	var refsListResp struct {
		Data []reference.ReferenceItem `json:"data"`
	}
	_ = json.Unmarshal(body, &refsListResp)
	if len(refsListResp.Data) != 3 {
		t.Fatalf("expected 3 references in response, got %d", len(refsListResp.Data))
	}

	// 11. Verify photo download: GET /api/v1/face/references/{id}/photo
	photoRefID := refsListResp.Data[0].ID
	photoURL := fmt.Sprintf("/api/v1/face/references/%s/photo", photoRefID.String())
	res, body = execCookieRequest(app, http.MethodGet, photoURL, empCookies, nil, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get photo failed: %d: %s", res.StatusCode, string(body))
	}
	if len(body) == 0 {
		t.Fatalf("expected non-empty photo bytes")
	}
}

func TestEnrollmentSession_CancelFlow(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Pool.Close()

	empUser := app.EnsureUserWithRole(t, "employee")
	empCookies := loginWithCookies(t, app, empUser.Email, "TestPass123!")

	// Grant consent
	grantPayload, _ := json.Marshal(consent.GrantConsentRequest{
		DocumentVersion: "2026-09-v1",
		Agreed:          true,
	})
	_, _ = execCookieRequest(app, http.MethodPost, "/api/v1/consents", empCookies, bytes.NewReader(grantPayload), "application/json")

	// Create session
	res, body := execCookieRequest(app, http.MethodPost, "/api/v1/face/enrollments", empCookies, bytes.NewReader([]byte("{}")), "application/json")
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create session failed: %d", res.StatusCode)
	}
	var sessResp struct {
		Data enrollment.SessionResponse `json:"data"`
	}
	_ = json.Unmarshal(body, &sessResp)
	sessID := sessResp.Data.ID

	// Cancel session: DELETE /api/v1/face/enrollments/{id}
	cancelURL := fmt.Sprintf("/api/v1/face/enrollments/%s", sessID.String())
	res, body = execCookieRequest(app, http.MethodDelete, cancelURL, empCookies, nil, "")
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("cancel session failed: %d: %s", res.StatusCode, string(body))
	}

	// Verify session is cancelled
	getSessURL := fmt.Sprintf("/api/v1/face/enrollments/%s", sessID.String())
	res, body = execCookieRequest(app, http.MethodGet, getSessURL, empCookies, nil, "")
	_ = json.Unmarshal(body, &sessResp)
	if sessResp.Data.Status != "cancelled" {
		t.Fatalf("expected status 'cancelled', got %s", sessResp.Data.Status)
	}
}

func TestBiometricWithdrawal_DeactivatesReferences(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Pool.Close()

	empUser := app.EnsureUserWithRole(t, "employee")
	empCookies := loginWithCookies(t, app, empUser.Email, "TestPass123!")

	// Grant consent
	grantPayload, _ := json.Marshal(consent.GrantConsentRequest{
		DocumentVersion: "2026-09-v1",
		Agreed:          true,
	})
	_, _ = execCookieRequest(app, http.MethodPost, "/api/v1/consents", empCookies, bytes.NewReader(grantPayload), "application/json")

	// Create session & upload 3 photos
	res, body := execCookieRequest(app, http.MethodPost, "/api/v1/face/enrollments", empCookies, bytes.NewReader([]byte("{}")), "application/json")
	var sessResp struct {
		Data enrollment.SessionResponse `json:"data"`
	}
	_ = json.Unmarshal(body, &sessResp)
	sessID := sessResp.Data.ID

	uploadURL := fmt.Sprintf("/api/v1/face/enrollments/%s/photos", sessID.String())

	for i := 1; i <= 3; i++ {
		jpegBytes := generateTestJPEG(byte(i * 40))
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		part, _ := w.CreateFormFile("image", fmt.Sprintf("photo_%d.jpg", i))
		_, _ = part.Write(jpegBytes)
		_ = w.WriteField("capture_source", "web_camera")
		_ = w.Close()
		_, _ = execCookieRequest(app, http.MethodPost, uploadURL, empCookies, &b, w.FormDataContentType())
	}

	// Commit session
	commitURL := fmt.Sprintf("/api/v1/face/enrollments/%s/commit", sessID.String())
	_, _ = execCookieRequest(app, http.MethodPost, commitURL, empCookies, nil, "")

	// Verify 3 active references
	var activeCount int
	_ = app.Pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM face_references WHERE employee_id = $1 AND is_active = true", empUser.EmployeeID).Scan(&activeCount)
	if activeCount != 3 {
		t.Fatalf("expected 3 active references, got %d", activeCount)
	}

	// Withdraw consent
	withdrawPayload, _ := json.Marshal(consent.WithdrawConsentRequest{
		Reason: "Tarik persetujuan PDP",
	})
	res, body = execCookieRequest(app, http.MethodPost, "/api/v1/consents/withdraw", empCookies, bytes.NewReader(withdrawPayload), "application/json")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("withdraw failed: %d: %s", res.StatusCode, string(body))
	}
	var withdrawResp struct {
		Data consent.WithdrawConsentResponse `json:"data"`
	}
	_ = json.Unmarshal(body, &withdrawResp)
	if withdrawResp.Data.AttendanceModeHint != "manual" {
		t.Fatalf("expected attendance_mode_hint 'manual', got %s", withdrawResp.Data.AttendanceModeHint)
	}
	if withdrawResp.Data.DeactivatedReferenceCount != 3 {
		t.Fatalf("expected 3 deactivated references, got %d", withdrawResp.Data.DeactivatedReferenceCount)
	}

	// Verify references are deactivated with deactivated_reason='consent_withdrawn'
	_ = app.Pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM face_references WHERE employee_id = $1 AND is_active = true", empUser.EmployeeID).Scan(&activeCount)
	if activeCount != 0 {
		t.Fatalf("expected 0 active references after withdrawal, got %d", activeCount)
	}

	var deactReason string
	err := app.Pool.QueryRow(t.Context(), "SELECT deactivated_reason FROM face_references WHERE employee_id = $1 LIMIT 1", empUser.EmployeeID).Scan(&deactReason)
	if err != nil {
		t.Fatalf("query deactivated_reason failed: %v", err)
	}
	if deactReason != "consent_withdrawn" {
		t.Fatalf("expected deactivated_reason 'consent_withdrawn', got %s", deactReason)
	}

	// Verify employee attendance_mode was NOT automatically altered (PDP § 4.3 non-automatic invariant)
	var attMode string
	_ = app.Pool.QueryRow(t.Context(), "SELECT attendance_mode FROM employees WHERE id = $1", empUser.EmployeeID).Scan(&attMode)
	if attMode == "manual" {
		t.Fatalf("attendance_mode should NOT be automatically set to manual on withdrawal")
	}
}

func TestRightToBeForgotten_AdminDeleteFaceData(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Pool.Close()

	empUser := app.EnsureUserWithRole(t, "employee")
	adminUser := app.EnsureUserWithRole(t, "admin")

	empCookies := loginWithCookies(t, app, empUser.Email, "TestPass123!")
	adminCookies := loginWithCookies(t, app, adminUser.Email, "TestPass123!")

	// Grant consent & enroll
	grantPayload, _ := json.Marshal(consent.GrantConsentRequest{
		DocumentVersion: "2026-09-v1",
		Agreed:          true,
	})
	_, _ = execCookieRequest(app, http.MethodPost, "/api/v1/consents", empCookies, bytes.NewReader(grantPayload), "application/json")

	res, body := execCookieRequest(app, http.MethodPost, "/api/v1/face/enrollments", empCookies, bytes.NewReader([]byte("{}")), "application/json")
	var sessResp struct {
		Data enrollment.SessionResponse `json:"data"`
	}
	_ = json.Unmarshal(body, &sessResp)
	sessID := sessResp.Data.ID

	uploadURL := fmt.Sprintf("/api/v1/face/enrollments/%s/photos", sessID.String())

	for i := 1; i <= 3; i++ {
		jpegBytes := generateTestJPEG(byte(i * 40))
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		part, _ := w.CreateFormFile("image", fmt.Sprintf("photo_%d.jpg", i))
		_, _ = part.Write(jpegBytes)
		_ = w.WriteField("capture_source", "web_camera")
		_ = w.Close()
		_, _ = execCookieRequest(app, http.MethodPost, uploadURL, empCookies, &b, w.FormDataContentType())
	}

	commitURL := fmt.Sprintf("/api/v1/face/enrollments/%s/commit", sessID.String())
	_, _ = execCookieRequest(app, http.MethodPost, commitURL, empCookies, nil, "")

	// Admin executes DELETE /api/v1/employees/{id}/face-data
	deleteURL := fmt.Sprintf("/api/v1/employees/%s/face-data", empUser.EmployeeID.String())
	deletePayload, _ := json.Marshal(reference.DeleteFaceDataRequest{
		Reason: "Permintaan penghapusan data biometrik UU PDP Pasal 43",
	})
	res, body = execCookieRequest(app, http.MethodDelete, deleteURL, adminCookies, bytes.NewReader(deletePayload), "application/json")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("delete face data failed: %d: %s", res.StatusCode, string(body))
	}

	// Verify all references deleted from DB
	var remainingRefs int
	_ = app.Pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM face_references WHERE employee_id = $1", empUser.EmployeeID).Scan(&remainingRefs)
	if remainingRefs != 0 {
		t.Fatalf("expected 0 face references in DB after permanent erasure, got %d", remainingRefs)
	}

	// Verify employee attendance_mode is manual
	var attMode string
	_ = app.Pool.QueryRow(t.Context(), "SELECT attendance_mode FROM employees WHERE id = $1", empUser.EmployeeID).Scan(&attMode)
	if attMode != "manual" {
		t.Fatalf("expected attendance_mode 'manual', got %s", attMode)
	}
}

func TestFaceReindex_JobLifecycle(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Pool.Close()

	adminUser := app.EnsureUserWithRole(t, "admin")
	adminCookies := loginWithCookies(t, app, adminUser.Email, "TestPass123!")

	// Update inference engine to report target model version (buffalo_s@v1)
	app.MockInfEngine.SetModelVersion("buffalo_s@v1")

	// 1. Create reindex job: POST /api/v1/face/reindex-jobs
	createJobPayload, _ := json.Marshal(reindex.CreateJobRequest{
		ToModelVersion: "buffalo_s@v1",
		DryRun:         false,
	})
	res, body := execCookieRequest(app, http.MethodPost, "/api/v1/face/reindex-jobs", adminCookies, bytes.NewReader(createJobPayload), "application/json")
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("create reindex job failed: %d: %s", res.StatusCode, string(body))
	}
	var jobResp struct {
		Data reindex.CreateJobResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &jobResp); err != nil {
		t.Fatalf("unmarshal create job response failed: %v", err)
	}
	if jobResp.Data.ID == nil {
		t.Fatalf("expected non-nil job ID")
	}
	jobID := *jobResp.Data.ID

	// 2. List jobs: GET /api/v1/face/reindex-jobs
	res, body = execCookieRequest(app, http.MethodGet, "/api/v1/face/reindex-jobs", adminCookies, nil, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list jobs failed: %d: %s", res.StatusCode, string(body))
	}

	// 3. Get job details: GET /api/v1/face/reindex-jobs/{id}
	getJobURL := fmt.Sprintf("/api/v1/face/reindex-jobs/%s", jobID.String())
	res, body = execCookieRequest(app, http.MethodGet, getJobURL, adminCookies, nil, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get job failed: %d: %s", res.StatusCode, string(body))
	}

	// 4. Cancel job: POST /api/v1/face/reindex-jobs/{id}/cancel
	cancelJobURL := fmt.Sprintf("/api/v1/face/reindex-jobs/%s/cancel", jobID.String())
	res, body = execCookieRequest(app, http.MethodPost, cancelJobURL, adminCookies, nil, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("cancel job failed: %d: %s", res.StatusCode, string(body))
	}

	// Verify job is cancelled
	res, body = execCookieRequest(app, http.MethodGet, getJobURL, adminCookies, nil, "")
	var getJobResp struct {
		Data reindex.JobDetailResponse `json:"data"`
	}
	_ = json.Unmarshal(body, &getJobResp)
	if getJobResp.Data.Status != "cancelled" {
		t.Fatalf("expected job status 'cancelled', got %s", getJobResp.Data.Status)
	}
}

func TestFaceReindex_WorkerExecution(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Pool.Close()

	adminUser := app.EnsureUserWithRole(t, "admin")
	adminCookies := loginWithCookies(t, app, adminUser.Email, "TestPass123!")

	empUser := app.EnsureUserWithRole(t, "employee")
	empCookies := loginWithCookies(t, app, empUser.Email, "TestPass123!")

	// 1. Employee grants consent
	consentPayload, _ := json.Marshal(consent.GrantConsentRequest{
		DocumentVersion: "2026-09-v1",
		Agreed:          true,
	})
	res, body := execCookieRequest(app, http.MethodPost, "/api/v1/consents", empCookies, bytes.NewReader(consentPayload), "application/json")
	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		t.Fatalf("grant consent failed: %d: %s", res.StatusCode, string(body))
	}

	// 2. Start session & upload 3 photos
	createSessPayload, _ := json.Marshal(enrollment.CreateSessionRequest{
		Mode: "replace",
	})
	res, body = execCookieRequest(app, http.MethodPost, "/api/v1/face/enrollments", empCookies, bytes.NewReader(createSessPayload), "application/json")
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create session failed: %d: %s", res.StatusCode, string(body))
	}
	var sessionResp struct {
		Data enrollment.SessionResponse `json:"data"`
	}
	_ = json.Unmarshal(body, &sessionResp)
	sessionID := sessionResp.Data.ID

	uploadURL := fmt.Sprintf("/api/v1/face/enrollments/%s/photos", sessionID.String())
	for i := 1; i <= 3; i++ {
		jpegBytes := generateTestJPEG(byte(i * 30))
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		part, err := w.CreateFormFile("image", fmt.Sprintf("photo_%d.jpg", i))
		if err != nil {
			t.Fatalf("create form file failed: %v", err)
		}
		if _, err := part.Write(jpegBytes); err != nil {
			t.Fatalf("write photo bytes failed: %v", err)
		}
		_ = w.WriteField("capture_source", "web_camera")
		_ = w.Close()

		res, body = execCookieRequest(app, http.MethodPost, uploadURL, empCookies, &b, w.FormDataContentType())
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("photo upload %d failed: %d: %s", i, res.StatusCode, string(body))
		}
	}

	// 3. Commit session
	commitURL := fmt.Sprintf("/api/v1/face/enrollments/%s/commit", sessionID.String())
	res, body = execCookieRequest(app, http.MethodPost, commitURL, empCookies, bytes.NewReader([]byte(`{}`)), "application/json")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("commit session failed: %d: %s", res.StatusCode, string(body))
	}

	// 4. Verify 3 active references with buffalo_l@v1
	var initialCount int
	_ = app.Pool.QueryRow(t.Context(), "SELECT count(*) FROM face_references WHERE employee_id = $1 AND is_active = true AND model_version = 'buffalo_l@v1'", empUser.EmployeeID).Scan(&initialCount)
	if initialCount != 3 {
		t.Fatalf("expected 3 active references with buffalo_l@v1, got %d", initialCount)
	}

	// 5. Update mock inference engine to new model
	app.MockInfEngine.SetModelVersion("buffalo_s@v1")

	// 6. Admin creates reindex job to buffalo_s@v1
	createJobPayload, _ := json.Marshal(reindex.CreateJobRequest{
		ToModelVersion: "buffalo_s@v1",
		DryRun:         false,
	})
	res, body = execCookieRequest(app, http.MethodPost, "/api/v1/face/reindex-jobs", adminCookies, bytes.NewReader(createJobPayload), "application/json")
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("create reindex job failed: %d: %s", res.StatusCode, string(body))
	}

	var jobResp struct {
		Data reindex.CreateJobResponse `json:"data"`
	}
	_ = json.Unmarshal(body, &jobResp)
	if jobResp.Data.ID == nil {
		t.Fatalf("job ID must not be nil")
	}
	jobID := *jobResp.Data.ID

	// 7. Run reindex worker pass
	worker := reindex.NewWorker(app.ReindexRepo, app.Store, app.MockInfEngine, app.SettingsSvc, app.AuditRec, slog.Default(), 0)
	if err := worker.RunOnce(t.Context()); err != nil {
		t.Fatalf("worker RunOnce failed: %v", err)
	}

	// 8. Verify job status is completed
	getJobURL := fmt.Sprintf("/api/v1/face/reindex-jobs/%s", jobID.String())
	res, body = execCookieRequest(app, http.MethodGet, getJobURL, adminCookies, nil, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get job failed: %d: %s", res.StatusCode, string(body))
	}
	var getJobResp struct {
		Data reindex.JobDetailResponse `json:"data"`
	}
	_ = json.Unmarshal(body, &getJobResp)
	if getJobResp.Data.Status != "completed" {
		t.Fatalf("expected job status 'completed', got %s", getJobResp.Data.Status)
	}
	if getJobResp.Data.ProcessedCount != 3 {
		t.Fatalf("expected 3 processed items, got %d", getJobResp.Data.ProcessedCount)
	}

	// 9. Verify face references updated: 3 active for buffalo_s@v1, 3 inactive with replaced_by_reindex
	var newActiveCount int
	_ = app.Pool.QueryRow(t.Context(), "SELECT count(*) FROM face_references WHERE employee_id = $1 AND is_active = true AND model_version = 'buffalo_s@v1'", empUser.EmployeeID).Scan(&newActiveCount)
	if newActiveCount != 3 {
		t.Fatalf("expected 3 active references with buffalo_s@v1, got %d", newActiveCount)
	}

	var oldDeactivatedCount int
	_ = app.Pool.QueryRow(t.Context(), "SELECT count(*) FROM face_references WHERE employee_id = $1 AND is_active = false AND deactivated_reason = 'model_reindex'", empUser.EmployeeID).Scan(&oldDeactivatedCount)
	if oldDeactivatedCount != 3 {
		t.Fatalf("expected 3 deactivated references with model_reindex, got %d", oldDeactivatedCount)
	}

	// 10. Verify system settings was updated to buffalo_s@v1
	currentModel := app.SettingsSvc.GetString(t.Context(), "face.model_version", "")
	if currentModel != "buffalo_s@v1" {
		t.Fatalf("expected system setting face.model_version 'buffalo_s@v1', got %s", currentModel)
	}
}
