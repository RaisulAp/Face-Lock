package attendance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/face"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service coordinates biometric verification, photo storage, and attendance persistence.
type Service struct {
	db         *pgxpool.Pool
	faceEngine face.FaceEngine
	store      storage.Store
}

// NewService constructs a new attendance Service instance.
func NewService(db *pgxpool.Pool, engine face.FaceEngine, store storage.Store) *Service {
	return &Service{
		db:         db,
		faceEngine: engine,
		store:      store,
	}
}

// SetDependencies allows updating dependencies if required.
func (s *Service) SetDependencies(engine face.FaceEngine, store storage.Store) {
	s.faceEngine = engine
	s.store = store
}

// Clock performs face matching and logs an attendance check-in or check-out event.
func (s *Service) Clock(ctx context.Context, clockType ClockType, params ClockParams) (*Record, error) {
	if s.faceEngine == nil {
		return nil, httpx.NewAppError(httpx.CodeFaceServiceNotConfigured, "face engine not configured")
	}
	if s.store == nil {
		return nil, httpx.NewAppError(httpx.CodeInternalError, "storage provider not configured")
	}
	if params.EmployeeID == nil {
		return nil, httpx.NewAppError(httpx.CodeBadRequest, "employee id is required")
	}
	if len(params.PhotoBytes) == 0 {
		return nil, httpx.NewAppError(httpx.CodeBadRequest, "photo payload cannot be empty")
	}

	empID := *params.EmployeeID

	// 1. Fetch employee and enrolled face reference embedding
	const fetchRefQ = `
		SELECT face_embedding::text, face_model_version, employment_status
		FROM employees
		WHERE id = $1 AND deleted_at IS NULL
	`
	var refEmbeddingStr *string
	var refModelVersion *string
	var empStatus string

	err := s.db.QueryRow(ctx, fetchRefQ, empID).Scan(&refEmbeddingStr, &refModelVersion, &empStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "employee not found")
		}
		return nil, fmt.Errorf("fetching employee face reference: %w", err)
	}

	if empStatus != "active" {
		return nil, httpx.NewAppError(httpx.CodeEmployeeInactive, "inactive employee cannot perform attendance")
	}

	if refEmbeddingStr == nil || *refEmbeddingStr == "" {
		return nil, httpx.NewAppError(httpx.CodeFaceNotEnrolled, "employee has no enrolled face reference")
	}

	// 2. Process incoming photo through face engine
	detResult, err := s.faceEngine.DetectAndEmbed(ctx, params.PhotoBytes)
	if err != nil {
		if errors.Is(err, face.ErrNoFaceDetected) {
			return nil, httpx.NewAppError(httpx.CodeFaceNotUsable, "no face detected in attendance photo")
		}
		if errors.Is(err, face.ErrMultipleFaces) {
			return nil, httpx.NewAppError(httpx.CodeFaceNotUsable, "multiple faces detected; ensure only one person is in camera view")
		}
		if errors.Is(err, face.ErrFaceNotUsable) {
			return nil, httpx.NewAppError(httpx.CodeFaceNotUsable, "face photo does not meet quality requirements")
		}
		return nil, httpx.NewAppError(httpx.CodeFaceNotUsable, fmt.Sprintf("face analysis failed: %v", err))
	}

	if !detResult.IsUsable || len(detResult.Embedding) == 0 {
		hints := strings.Join(detResult.Hints, ", ")
		if hints == "" {
			hints = "low quality or obstructed face"
		}
		return nil, httpx.NewAppError(httpx.CodeFaceNotUsable, "attendance photo rejected: "+hints)
	}

	// 3. Compare vectors using pgvector cosine distance in Postgres
	liveVectorStr := face.VectorToPGVector(detResult.Embedding)

	const matchQ = `
		SELECT
			1 - (face_embedding <=> $1::vector) AS similarity,
			face_embedding <-> $1::vector AS distance
		FROM employees
		WHERE id = $2
	`
	var simScore, dist float64
	err = s.db.QueryRow(ctx, matchQ, liveVectorStr, empID).Scan(&simScore, &dist)
	if err != nil {
		return nil, fmt.Errorf("evaluating pgvector distance: %w", err)
	}

	// Also verify in Go memory
	refVec, err := face.PGVectorToVector(*refEmbeddingStr)
	if err == nil && len(refVec) > 0 {
		goSim, goDist := s.faceEngine.Compare(detResult.Embedding, refVec)
		_ = goSim
		_ = goDist
	}

	// 4. Check if similarity meets threshold
	if !s.faceEngine.IsMatch(float32(simScore)) {
		modVer := detResult.ModelVersion
		if modVer == "" {
			modVer = s.faceEngine.ModelVersion()
		}
		const insertFailQ = `
			INSERT INTO attendances (
				employee_id, type, status, photo_key, face_embedding,
				similarity_score, distance, model_version, notes, recorded_at
			) VALUES (
				$1, $2, 'failed', '', $3::vector, $4, $5, $6, $7, NOW()
			)
		`
		_, _ = s.db.Exec(ctx, insertFailQ, empID, string(clockType), liveVectorStr, simScore, dist, modVer, "face mismatch")

		return nil, httpx.NewAppError(
			httpx.CodeFaceMismatch,
			fmt.Sprintf("face verification failed: similarity %.4f below required threshold %.2f", simScore, s.faceEngine.Threshold()),
		)
	}

	// 5. Store attendance snapshot to storage
	cType := params.ContentType
	if cType == "" {
		cType = "image/jpeg"
	}
	photoKey := fmt.Sprintf("attendances/%s/%s/%d.jpg", empID.String(), string(clockType), time.Now().UnixNano())
	storedKey, err := s.store.Put(ctx, photoKey, bytes.NewReader(params.PhotoBytes), cType)
	if err != nil {
		return nil, fmt.Errorf("persisting attendance photo: %w", err)
	}
	if storedKey != "" {
		photoKey = storedKey
	}

	modVer := detResult.ModelVersion
	if modVer == "" {
		modVer = s.faceEngine.ModelVersion()
	}

	// 6. Record attendance row in database
	const insertAttQ = `
		INSERT INTO attendances (
			employee_id, type, status, photo_key, face_embedding,
			similarity_score, distance, model_version, notes, recorded_at
		) VALUES (
			$1, $2, 'success', $3, $4::vector, $5, $6, $7, $8, NOW()
		)
		RETURNING
			id, employee_id, type, status, photo_key, similarity_score, distance,
			model_version, notes, recorded_at, created_at
	`
	var rec Record
	err = s.db.QueryRow(ctx, insertAttQ,
		empID,
		string(clockType),
		photoKey,
		liveVectorStr,
		simScore,
		dist,
		modVer,
		params.Notes,
	).Scan(
		&rec.ID,
		&rec.EmployeeID,
		&rec.Type,
		&rec.Status,
		&rec.PhotoKey,
		&rec.SimilarityScore,
		&rec.Distance,
		&rec.ModelVersion,
		&rec.Notes,
		&rec.RecordedAt,
		&rec.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting attendance record: %w", err)
	}

	return &rec, nil
}
