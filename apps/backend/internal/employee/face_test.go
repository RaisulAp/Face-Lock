package employee_test

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

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/employee"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/face"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/storage"
	"github.com/go-chi/chi/v5"
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

func TestEmployeeFaceEnrollment(t *testing.T) {
	pool := getTestDB(t)
	ctx := context.Background()

	mockEngine := face.NewMockEngine()
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("faceclock_test_store_%d", time.Now().UnixNano()))
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	localStore, err := storage.NewLocalStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create local store: %v", err)
	}

	empSvc := employee.NewService(pool)
	empSvc.SetFaceBiometrics(mockEngine, localStore)
	auditRec := audit.NewRecorder(pool)
	handler := employee.NewHandler(empSvc, auditRec)

	// Create test employee
	empNum := fmt.Sprintf("EMP-FE-%d", time.Now().UnixNano()%1000000)
	emp, err := empSvc.Create(ctx, employee.CreateParams{
		EmployeeNumber: empNum,
		FullName:       "Face Enrollment Test Employee",
	})
	if err != nil {
		t.Fatalf("failed to create employee: %v", err)
	}

	t.Run("enroll face successfully via multipart/form-data", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		fw, err := w.CreateFormFile("photo", "face.jpg")
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		if _, err := fw.Write([]byte("valid-test-face-image-bytes-12345")); err != nil {
			t.Fatalf("failed to write form file: %v", err)
		}
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/employees/"+emp.ID.String()+"/face-enroll", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())

		// Add principal with face.enroll_any
		principal := &rbac.Principal{
			UserID:      uuid.New(),
			Permissions: map[string]struct{}{rbac.PermFaceEnrollAny: {}},
		}
		req = req.WithContext(rbac.WithPrincipal(req.Context(), principal))

		// Router context for chi param
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", emp.ID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rec := httptest.NewRecorder()
		handler.FaceEnroll(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}

		// Verify that face_embedding and face_photo_path in DB are updated
		var facePhotoPath *string
		err = pool.QueryRow(ctx, "SELECT face_photo_path FROM employees WHERE id = $1", emp.ID).Scan(&facePhotoPath)
		if err != nil {
			t.Fatalf("querying employee face photo path: %v", err)
		}
		if facePhotoPath == nil || *facePhotoPath == "" {
			t.Fatalf("expected face_photo_path to be populated, got nil")
		}

		// Verify face_references entry exists
		var refCount int
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM face_references WHERE employee_id = $1 AND is_active = true", emp.ID).Scan(&refCount)
		if err != nil {
			t.Fatalf("querying face_references count: %v", err)
		}
		if refCount != 1 {
			t.Fatalf("expected 1 active face_reference, got %d", refCount)
		}
	})

	t.Run("enroll face rejected when no face detected", func(t *testing.T) {
		mockEngine.SetForceError(face.ErrNoFaceDetected)
		defer mockEngine.SetForceError(nil)

		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		fw, _ := w.CreateFormFile("photo", "noface.jpg")
		fw.Write([]byte("no-face-bytes"))
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/employees/"+emp.ID.String()+"/face-enroll", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())

		principal := &rbac.Principal{
			UserID:      uuid.New(),
			Permissions: map[string]struct{}{rbac.PermFaceEnrollAny: {}},
		}
		req = req.WithContext(rbac.WithPrincipal(req.Context(), principal))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", emp.ID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rec := httptest.NewRecorder()
		handler.FaceEnroll(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 Unprocessable Entity, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("enroll face forbidden without permission", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		fw, _ := w.CreateFormFile("photo", "face.jpg")
		fw.Write([]byte("some-face"))
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/employees/"+emp.ID.String()+"/face-enroll", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())

		// Principal with no face permissions
		principal := &rbac.Principal{
			UserID:      uuid.New(),
			Permissions: map[string]struct{}{rbac.PermEmployeeRead: {}},
		}
		req = req.WithContext(rbac.WithPrincipal(req.Context(), principal))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", emp.ID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rec := httptest.NewRecorder()
		handler.FaceEnroll(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}
