package attendance_test

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/attendance"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/employee"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/face"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/settings"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://faceclock:devpassword123@localhost:5434/faceclock?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skipping db test, cannot connect to %s: %v", dsn, err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skipping db test, ping failed: %v", err)
	}
	return pool
}

func TestAttendanceFlow(t *testing.T) {
	pool := getTestDB(t)
	ctx := context.Background()

	mockEngine := face.NewMockEngine()
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("faceclock_att_store_%d", time.Now().UnixNano()))
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	localStore, err := storage.NewLocalStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create local store: %v", err)
	}

	empSvc := employee.NewService(pool)
	empSvc.SetFaceBiometrics(mockEngine, localStore)

	settingsSvc := settings.NewService(pool)
	attSvc := attendance.NewService(pool, mockEngine, localStore, settingsSvc)
	auditRec := audit.NewRecorder(pool)
	attHandler := attendance.NewHandler(attSvc, auditRec)

	// Create employee
	empNum := fmt.Sprintf("EMP-ATT-%d", time.Now().UnixNano()%1000000)
	emp, err := empSvc.Create(ctx, employee.CreateParams{
		EmployeeNumber: &empNum,
		FullName:       "Attendance Test Employee",
	})
	if err != nil {
		t.Fatalf("failed to create employee: %v", err)
	}

	facePhotoA := []byte("employee-original-face-photo-bytes-xyz-1")
	facePhotoA2 := []byte("employee-evening-face-photo-bytes-xyz-3")
	facePhotoB := []byte("different-person-face-photo-bytes-xyz-2")

	principal := &rbac.Principal{
		UserID:      uuid.New(),
		EmployeeID:  &emp.ID,
		Permissions: map[string]struct{}{rbac.PermAttendanceCheckin: {}},
	}

	t.Run("clock-in fails when face is not enrolled", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		fw, _ := w.CreateFormFile("photo", "face.jpg")
		fw.Write(facePhotoA)
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/attendance/clock-in", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req = req.WithContext(rbac.WithPrincipal(req.Context(), principal))

		rec := httptest.NewRecorder()
		attHandler.ClockIn(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 Unprocessable Entity (FACE_NOT_ENROLLED), got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("enroll face for employee", func(t *testing.T) {
		_, err := empSvc.EnrollFace(ctx, emp.ID, facePhotoA, "image/jpeg")
		if err != nil {
			t.Fatalf("failed to enroll face: %v", err)
		}
	})

	t.Run("clock-in fails with mismatched face", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		fw, _ := w.CreateFormFile("photo", "face.jpg")
		fw.Write(facePhotoB) // Different photo
		w.WriteField("latitude", "-6.2088")
		w.WriteField("longitude", "106.8456")
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/attendance/clock-in", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req = req.WithContext(rbac.WithPrincipal(req.Context(), principal))

		rec := httptest.NewRecorder()
		attHandler.ClockIn(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 Unprocessable Entity (FACE_MISMATCH), got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("clock-in succeeds with matching enrolled face", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		fw, _ := w.CreateFormFile("photo", "face.jpg")
		fw.Write(facePhotoA) // Matching photo
		w.WriteField("latitude", "-6.2088")
		w.WriteField("longitude", "106.8456")
		w.WriteField("notes", "Morning check-in on time")
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/attendance/clock-in", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req = req.WithContext(rbac.WithPrincipal(req.Context(), principal))

		rec := httptest.NewRecorder()
		attHandler.ClockIn(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("clock-out succeeds with matching enrolled face", func(t *testing.T) {
		// Enroll second face photo for employee so evening photo matches biometrics
		_, err := empSvc.EnrollFace(ctx, emp.ID, facePhotoA2, "image/jpeg")
		if err != nil {
			t.Fatalf("failed to enroll evening face: %v", err)
		}

		// Backdate check-in record so minimum interval (>5 mins) check passes
		_, _ = pool.Exec(ctx, "UPDATE attendances SET server_timestamp = NOW() - INTERVAL '10 minutes' WHERE employee_id = $1", emp.ID)

		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		fw, _ := w.CreateFormFile("photo", "face.jpg")
		fw.Write(facePhotoA2) // Matching distinct photo (Anti-Replay safe)
		w.WriteField("latitude", "-6.2088")
		w.WriteField("longitude", "106.8456")
		w.WriteField("notes", "Evening check-out")
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/attendance/clock-out", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req = req.WithContext(rbac.WithPrincipal(req.Context(), principal))

		rec := httptest.NewRecorder()
		attHandler.ClockOut(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}
