package employee

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/face"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// FaceEnrollResult contains the confirmation data returned upon successful biometric enrollment.
type FaceEnrollResult struct {
	EmployeeID   uuid.UUID `json:"employee_id"`
	PhotoPath    string    `json:"photo_path"`
	ModelVersion string    `json:"model_version"`
	QualityScore float32   `json:"quality_score"`
	EnrolledAt   time.Time `json:"enrolled_at"`
}

// SetFaceBiometrics configures the biometric engine and storage provider for the employee service.
func (s *Service) SetFaceBiometrics(engine face.FaceEngine, store storage.Store) {
	s.faceEngine = engine
	s.store = store
}

// EnrollFace verifies photo quality, computes the biometric embedding,
// saves the reference image to object storage, and records the vector in the database.
func (s *Service) EnrollFace(ctx context.Context, empID uuid.UUID, photoBytes []byte, contentType string) (*FaceEnrollResult, error) {
	if s.faceEngine == nil {
		return nil, httpx.NewAppError(httpx.CodeFaceServiceNotConfigured, "face recognition engine is not configured")
	}
	if s.store == nil {
		return nil, httpx.NewAppError(httpx.CodeInternalError, "storage provider is not configured")
	}
	if len(photoBytes) == 0 {
		return nil, httpx.NewAppError(httpx.CodeBadRequest, "photo payload cannot be empty")
	}

	// 1. Verify that employee exists and is active
	var exists bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM employees WHERE id = $1 AND deleted_at IS NULL)", empID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("checking employee existence: %w", err)
	}
	if !exists {
		return nil, httpx.NewAppError(httpx.CodeNotFound, "employee not found")
	}

	// 2. Detect face and generate embedding
	detResult, err := s.faceEngine.DetectAndEmbed(ctx, photoBytes)
	if err != nil {
		if errors.Is(err, face.ErrNoFaceDetected) {
			return nil, httpx.NewAppError(httpx.CodeFaceNotUsable, "no face detected in uploaded photo")
		}
		if errors.Is(err, face.ErrMultipleFaces) {
			return nil, httpx.NewAppError(httpx.CodeFaceNotUsable, "multiple faces detected in photo; only one face allowed")
		}
		if errors.Is(err, face.ErrFaceNotUsable) {
			return nil, httpx.NewAppError(httpx.CodeFaceNotUsable, "face photo does not meet quality requirements")
		}
		return nil, httpx.NewAppError(httpx.CodeFaceNotUsable, fmt.Sprintf("face analysis failed: %v", err))
	}

	if !detResult.IsUsable || len(detResult.Embedding) == 0 {
		hintsStr := strings.Join(detResult.Hints, ", ")
		if hintsStr == "" {
			hintsStr = "unusable face quality"
		}
		return nil, httpx.NewAppError(httpx.CodeFaceNotUsable, "face photo rejected: "+hintsStr)
	}

	// 3. Save photo to object storage
	photoKey := fmt.Sprintf("faces/enrollments/%s/%d.jpg", empID.String(), time.Now().UnixNano())
	if contentType == "" {
		contentType = "image/jpeg"
	}
	storedKey, err := s.store.Put(ctx, photoKey, bytes.NewReader(photoBytes), contentType)
	if err != nil {
		return nil, fmt.Errorf("storing face photo: %w", err)
	}
	if storedKey != "" {
		photoKey = storedKey
	}

	pgVector := face.VectorToPGVector(detResult.Embedding)
	modVer := detResult.ModelVersion
	if modVer == "" {
		modVer = s.faceEngine.ModelVersion()
	}

	// 4. Update employee and insert into face_references within a transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const updateEmpQ = `
		UPDATE employees
		SET face_embedding = $1::vector,
		    face_photo_path = $2,
		    face_enrolled_at = NOW(),
		    face_model_version = $3,
		    updated_at = NOW()
		WHERE id = $4 AND deleted_at IS NULL
		RETURNING face_enrolled_at
	`
	var enrolledAt time.Time
	err = tx.QueryRow(ctx, updateEmpQ, pgVector, photoKey, modVer, empID).Scan(&enrolledAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "employee not found")
		}
		return nil, fmt.Errorf("updating employee face record: %w", err)
	}

	photoSHA256 := fmt.Sprintf("%x", sha256.Sum256(photoBytes))
	validMIMEs := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
	}
	if !validMIMEs[contentType] {
		contentType = "image/jpeg"
	}

	const insertRefQ = `
		INSERT INTO face_references (
			employee_id, photo_key, photo_sha256, photo_bytes, photo_mime, capture_source, embedding, model_version, is_active, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7::vector, $8, true, NOW()
		)
	`
	if _, err := tx.Exec(ctx, insertRefQ, empID, photoKey, photoSHA256, len(photoBytes), contentType, "admin_upload", pgVector, modVer); err != nil {
		return nil, fmt.Errorf("inserting face reference: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing face enrollment: %w", err)
	}

	return &FaceEnrollResult{
		EmployeeID:   empID,
		PhotoPath:    photoKey,
		ModelVersion: modVer,
		QualityScore: detResult.QualityScore,
		EnrolledAt:   enrolledAt,
	}, nil
}

// FaceEnroll handles POST /api/v1/employees/{id}/face-enroll
func (h *Handler) FaceEnroll(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid employee id format"))
		return
	}

	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	// Must have face.enroll_any, OR face.enroll_self if enrolling self
	if !p.HasPermission(rbac.PermFaceEnrollAny) {
		if !p.HasPermission(rbac.PermFaceEnrollSelf) || p.EmployeeID == nil || *p.EmployeeID != id {
			httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeForbidden, "insufficient permissions to enroll face"))
			return
		}
	}

	photoBytes, contentType, err := extractPhotoBytes(r)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, err.Error()))
		return
	}

	res, err := h.svc.EnrollFace(r.Context(), id, photoBytes, contentType)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to enroll face"))
		return
	}

	resID := id.String()
	_ = h.audit.RecordFromRequest(r, "employee.face_enroll", "employee", &resID, map[string]any{
		"photo_path":    res.PhotoPath,
		"model_version": res.ModelVersion,
		"quality_score": res.QualityScore,
	})

	httpx.Created(w, res)
}

// extractPhotoBytes reads the raw photo bytes from multipart/form-data, json, or octet-stream.
func extractPhotoBytes(r *http.Request) ([]byte, string, error) {
	ct := r.Header.Get("Content-Type")

	// 1. Multipart form upload
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			return nil, "", httpx.NewAppError(httpx.CodeBadRequest, "failed to parse multipart form")
		}
		for _, field := range []string{"photo", "image", "file"} {
			file, header, err := r.FormFile(field)
			if err == nil {
				defer file.Close()
				data, err := io.ReadAll(file)
				if err != nil {
					return nil, "", httpx.NewAppError(httpx.CodeBadRequest, "failed to read uploaded file")
				}
				cType := header.Header.Get("Content-Type")
				if cType == "" || cType == "application/octet-stream" {
					cType = http.DetectContentType(data)
				}
				if cType == "application/octet-stream" {
					cType = "image/jpeg"
				}
				return data, cType, nil
			}
		}
		return nil, "", httpx.NewAppError(httpx.CodeBadRequest, "missing photo/image file field in multipart form")
	}

	// 2. JSON with base64 encoded photo
	if strings.HasPrefix(ct, "application/json") {
		var payload struct {
			PhotoBase64 string `json:"photo_base64"`
			ImageBase64 string `json:"image_base64"`
			Photo       string `json:"photo"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			return nil, "", httpx.NewAppError(httpx.CodeBadRequest, "malformed json payload")
		}
		raw := payload.PhotoBase64
		if raw == "" {
			raw = payload.ImageBase64
		}
		if raw == "" {
			raw = payload.Photo
		}
		if raw == "" {
			return nil, "", httpx.NewAppError(httpx.CodeBadRequest, "missing photo_base64 or photo field in json")
		}
		if idx := strings.Index(raw, ";base64,"); idx != -1 {
			raw = raw[idx+8:]
		}
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, "", httpx.NewAppError(httpx.CodeBadRequest, "invalid base64 image data")
		}
		return decoded, "image/jpeg", nil
	}

	// 3. Direct binary stream
	if strings.HasPrefix(ct, "image/") || ct == "application/octet-stream" {
		buf := &bytes.Buffer{}
		if _, err := io.Copy(buf, r.Body); err != nil {
			return nil, "", httpx.NewAppError(httpx.CodeBadRequest, "failed to read image body")
		}
		return buf.Bytes(), ct, nil
	}

	return nil, "", httpx.NewAppError(httpx.CodeUnsupportedMedia, "unsupported content type: must be multipart/form-data, application/json, or image/*")
}
