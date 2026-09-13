package enrollment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSessionNotFound = errors.New("enrollment: session not found")
	ErrPhotoNotFound   = errors.New("enrollment: photo not found")
)

type Session struct {
	ID             uuid.UUID
	EmployeeID     uuid.UUID
	CreatedBy      uuid.UUID
	Status         string
	Mode           string
	ModelVersion   string
	RequiredPhotos int
	MaxPhotos      int
	ExpiresAt      time.Time
	CommittedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Photo struct {
	ID            uuid.UUID
	SessionID     uuid.UUID
	Position      int
	Embedding     []float32
	QualityScore  float32
	DetScore      float32
	PhotoKey      string
	PhotoSHA256   string
	PhotoBytes    int
	PhotoMIME     string
	CaptureSource string
	CreatedAt     time.Time
}

type Repository interface {
	CreateSession(ctx context.Context, s *Session) error
	GetSessionByID(ctx context.Context, id uuid.UUID) (*Session, error)
	GetActiveDraftSession(ctx context.Context, employeeID uuid.UUID) (*Session, error)
	ExpireSession(ctx context.Context, id uuid.UUID) error
	CancelSession(ctx context.Context, id uuid.UUID) error
	CountActiveReferences(ctx context.Context, employeeID uuid.UUID) (int, error)
	IsReindexRunning(ctx context.Context) (bool, error)
	GetEmployeeAttendanceMode(ctx context.Context, employeeID uuid.UUID) (string, error)
	EmployeeExists(ctx context.Context, employeeID uuid.UUID) (bool, error)

	InsertPhoto(ctx context.Context, p *Photo) error
	ListPhotosBySession(ctx context.Context, sessionID uuid.UUID) ([]Photo, error)
	GetPhotoByID(ctx context.Context, sessionID, photoID uuid.UUID) (*Photo, error)
	DeletePhoto(ctx context.Context, sessionID, photoID uuid.UUID) error
	PhotoHashExists(ctx context.Context, sessionID uuid.UUID, hash string) (bool, error)
	NextPosition(ctx context.Context, sessionID uuid.UUID) (int, error)

	CheckDuplicateFaces(ctx context.Context, embedding []float32, modelVersion string, excludeEmployeeID uuid.UUID, threshold float32) ([]DuplicateMatch, error)
	CommitSession(ctx context.Context, session *Session, photos []Photo, callerID uuid.UUID) ([]uuid.UUID, []uuid.UUID, error)
}

type DuplicateMatch struct {
	EmployeeID uuid.UUID
	Similarity float32
}

type pgRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) CreateSession(ctx context.Context, s *Session) error {
	query := `
		INSERT INTO face_enrollment_sessions (
			id, employee_id, created_by, status, mode, model_version,
			required_photos, max_photos, expires_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW()
		)
	`
	_, err := r.db.Exec(ctx, query,
		s.ID, s.EmployeeID, s.CreatedBy, s.Status, s.Mode, s.ModelVersion,
		s.RequiredPhotos, s.MaxPhotos, s.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *pgRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (*Session, error) {
	query := `
		SELECT id, employee_id, created_by, status, mode, model_version,
		       required_photos, max_photos, expires_at, committed_at, created_at, updated_at
		FROM face_enrollment_sessions
		WHERE id = $1
	`
	var s Session
	err := r.db.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.EmployeeID, &s.CreatedBy, &s.Status, &s.Mode, &s.ModelVersion,
		&s.RequiredPhotos, &s.MaxPhotos, &s.ExpiresAt, &s.CommittedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	return &s, nil
}

func (r *pgRepository) GetActiveDraftSession(ctx context.Context, employeeID uuid.UUID) (*Session, error) {
	query := `
		SELECT id, employee_id, created_by, status, mode, model_version,
		       required_photos, max_photos, expires_at, committed_at, created_at, updated_at
		FROM face_enrollment_sessions
		WHERE employee_id = $1 AND status = 'draft' AND expires_at > NOW()
		LIMIT 1
	`
	var s Session
	err := r.db.QueryRow(ctx, query, employeeID).Scan(
		&s.ID, &s.EmployeeID, &s.CreatedBy, &s.Status, &s.Mode, &s.ModelVersion,
		&s.RequiredPhotos, &s.MaxPhotos, &s.ExpiresAt, &s.CommittedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get active draft session: %w", err)
	}
	return &s, nil
}

func (r *pgRepository) ExpireSession(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE face_enrollment_sessions
		SET status = 'expired', updated_at = NOW()
		WHERE id = $1 AND status = 'draft'
	`, id)
	return err
}

func (r *pgRepository) CancelSession(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE face_enrollment_sessions
		SET status = 'cancelled', updated_at = NOW()
		WHERE id = $1 AND status = 'draft'
	`, id)
	return err
}

func (r *pgRepository) CountActiveReferences(ctx context.Context, employeeID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM face_references
		WHERE employee_id = $1 AND is_active = true
	`, employeeID).Scan(&count)
	return count, err
}

func (r *pgRepository) IsReindexRunning(ctx context.Context) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM face_reindex_jobs WHERE status IN ('pending', 'running'))
	`).Scan(&exists)
	return exists, err
}

func (r *pgRepository) GetEmployeeAttendanceMode(ctx context.Context, employeeID uuid.UUID) (string, error) {
	var mode string
	err := r.db.QueryRow(ctx, `
		SELECT attendance_mode FROM employees
		WHERE id = $1 AND deleted_at IS NULL AND employment_status != 'resigned'
	`, employeeID).Scan(&mode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("employee not found or inactive")
		}
		return "", err
	}
	return mode, nil
}

func (r *pgRepository) EmployeeExists(ctx context.Context, employeeID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM employees
			WHERE id = $1 AND deleted_at IS NULL AND employment_status != 'resigned'
		)
	`, employeeID).Scan(&exists)
	return exists, err
}

func (r *pgRepository) InsertPhoto(ctx context.Context, p *Photo) error {
	// vector format in pgvector: "[0.1,0.2,...]"
	vecStr := formatVector(p.Embedding)
	query := `
		INSERT INTO face_enrollment_photos (
			id, session_id, position, embedding, quality_score, det_score,
			photo_key, photo_sha256, photo_bytes, photo_mime, capture_source, created_at
		) VALUES (
			$1, $2, $3, $4::vector, $5, $6, $7, $8, $9, $10, $11, NOW()
		)
	`
	_, err := r.db.Exec(ctx, query,
		p.ID, p.SessionID, p.Position, vecStr, p.QualityScore, p.DetScore,
		p.PhotoKey, p.PhotoSHA256, p.PhotoBytes, p.PhotoMIME, p.CaptureSource,
	)
	if err != nil {
		return fmt.Errorf("insert staging photo: %w", err)
	}
	return nil
}

func (r *pgRepository) ListPhotosBySession(ctx context.Context, sessionID uuid.UUID) ([]Photo, error) {
	query := `
		SELECT id, session_id, position, embedding::text, quality_score, det_score,
		       photo_key, photo_sha256, photo_bytes, photo_mime, capture_source, created_at
		FROM face_enrollment_photos
		WHERE session_id = $1
		ORDER BY position ASC
	`
	rows, err := r.db.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list staging photos: %w", err)
	}
	defer rows.Close()

	var photos []Photo
	for rows.Next() {
		var p Photo
		var vecStr string
		if err := rows.Scan(
			&p.ID, &p.SessionID, &p.Position, &vecStr, &p.QualityScore, &p.DetScore,
			&p.PhotoKey, &p.PhotoSHA256, &p.PhotoBytes, &p.PhotoMIME, &p.CaptureSource, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan staging photo: %w", err)
		}
		p.Embedding = parseVector(vecStr)
		photos = append(photos, p)
	}
	return photos, nil
}

func (r *pgRepository) GetPhotoByID(ctx context.Context, sessionID, photoID uuid.UUID) (*Photo, error) {
	query := `
		SELECT id, session_id, position, embedding::text, quality_score, det_score,
		       photo_key, photo_sha256, photo_bytes, photo_mime, capture_source, created_at
		FROM face_enrollment_photos
		WHERE session_id = $1 AND id = $2
	`
	var p Photo
	var vecStr string
	err := r.db.QueryRow(ctx, query, sessionID, photoID).Scan(
		&p.ID, &p.SessionID, &p.Position, &vecStr, &p.QualityScore, &p.DetScore,
		&p.PhotoKey, &p.PhotoSHA256, &p.PhotoBytes, &p.PhotoMIME, &p.CaptureSource, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPhotoNotFound
		}
		return nil, fmt.Errorf("get photo: %w", err)
	}
	p.Embedding = parseVector(vecStr)
	return &p, nil
}

func (r *pgRepository) DeletePhoto(ctx context.Context, sessionID, photoID uuid.UUID) error {
	res, err := r.db.Exec(ctx, `
		DELETE FROM face_enrollment_photos
		WHERE session_id = $1 AND id = $2
	`, sessionID, photoID)
	if err != nil {
		return fmt.Errorf("delete photo: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrPhotoNotFound
	}
	return nil
}

func (r *pgRepository) PhotoHashExists(ctx context.Context, sessionID uuid.UUID, hash string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM face_enrollment_photos
			WHERE session_id = $1 AND photo_sha256 = $2
		)
	`, sessionID, hash).Scan(&exists)
	return exists, err
}

func (r *pgRepository) NextPosition(ctx context.Context, sessionID uuid.UUID) (int, error) {
	var next int
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(MAX(position), 0) + 1
		FROM face_enrollment_photos
		WHERE session_id = $1
	`, sessionID).Scan(&next)
	return next, err
}

func (r *pgRepository) CheckDuplicateFaces(ctx context.Context, embedding []float32, modelVersion string, excludeEmployeeID uuid.UUID, threshold float32) ([]DuplicateMatch, error) {
	vecStr := formatVector(embedding)
	query := `
		SELECT fr.employee_id, max(1 - (fr.embedding <=> $1::vector)) AS similarity
		FROM face_references fr
		WHERE fr.is_active = true
		  AND fr.model_version = $2
		  AND fr.employee_id <> $3
		GROUP BY fr.employee_id
		HAVING max(1 - (fr.embedding <=> $1::vector)) >= $4
		ORDER BY similarity DESC
		LIMIT 5
	`
	rows, err := r.db.Query(ctx, query, vecStr, modelVersion, excludeEmployeeID, threshold)
	if err != nil {
		return nil, fmt.Errorf("check duplicate faces: %w", err)
	}
	defer rows.Close()

	var matches []DuplicateMatch
	for rows.Next() {
		var m DuplicateMatch
		if err := rows.Scan(&m.EmployeeID, &m.Similarity); err != nil {
			return nil, fmt.Errorf("scan duplicate match: %w", err)
		}
		matches = append(matches, m)
	}
	return matches, nil
}

func (r *pgRepository) CommitSession(ctx context.Context, session *Session, photos []Photo, callerID uuid.UUID) ([]uuid.UUID, []uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Select session for update to guard against concurrent commits
	var currentStatus string
	err = tx.QueryRow(ctx, `
		SELECT status FROM face_enrollment_sessions
		WHERE id = $1 FOR UPDATE
	`, session.ID).Scan(&currentStatus)
	if err != nil {
		return nil, nil, fmt.Errorf("lock session: %w", err)
	}
	if currentStatus != "draft" {
		return nil, nil, fmt.Errorf("session status is %q, expected 'draft'", currentStatus)
	}

	var deactivatedIDs []uuid.UUID
	if session.Mode == "replace" {
		// Deactivate all current active references
		rows, err := tx.Query(ctx, `
			UPDATE face_references
			SET is_active = false, deactivated_at = NOW(), deactivated_reason = 're_enroll', updated_at = NOW()
			WHERE employee_id = $1 AND is_active = true
			RETURNING id
		`, session.EmployeeID)
		if err != nil {
			return nil, nil, fmt.Errorf("deactivate existing references: %w", err)
		}
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err == nil {
				deactivatedIDs = append(deactivatedIDs, id)
			}
		}
		rows.Close()
	}

	var createdIDs []uuid.UUID
	insertRefQuery := `
		INSERT INTO face_references (
			id, employee_id, embedding, quality_score, det_score, model_version,
			photo_key, photo_sha256, photo_bytes, photo_mime, capture_source,
			position, enrollment_session_id, is_active, enrolled_by, created_at, updated_at
		) VALUES (
			$1, $2, $3::vector, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, true, $14, NOW(), NOW()
		)
	`
	for _, p := range photos {
		refID := p.ID // or generate a new one
		vecStr := formatVector(p.Embedding)
		_, err := tx.Exec(ctx, insertRefQuery,
			refID, session.EmployeeID, vecStr, p.QualityScore, p.DetScore, session.ModelVersion,
			p.PhotoKey, p.PhotoSHA256, p.PhotoBytes, p.PhotoMIME, p.CaptureSource,
			p.Position, session.ID, callerID,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("insert reference: %w", err)
		}
		createdIDs = append(createdIDs, refID)
	}

	// Update session status to committed
	_, err = tx.Exec(ctx, `
		UPDATE face_enrollment_sessions
		SET status = 'committed', committed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, session.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("update session committed: %w", err)
	}

	// Delete staging photos from DB
	_, err = tx.Exec(ctx, `
		DELETE FROM face_enrollment_photos
		WHERE session_id = $1
	`, session.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("delete staging photos: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit tx: %w", err)
	}

	return createdIDs, deactivatedIDs, nil
}

func formatVector(v []float32) string {
	if len(v) == 0 {
		return "[]"
	}
	s := "["
	for i, f := range v {
		if i > 0 {
			s += ","
		}
		s += fmt.Sprintf("%f", f)
	}
	s += "]"
	return s
}

func parseVector(s string) []float32 {
	if len(s) < 2 {
		return nil
	}
	// Trim '[' and ']'
	inner := s[1 : len(s)-1]
	if len(inner) == 0 {
		return nil
	}
	var res []float32
	var current float32
	var sign float32 = 1
	var inFraction bool
	var fractionDiv float32 = 1
	// Fast parse floats separated by commas
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		switch {
		case c == '-':
			sign = -1
		case c == '.':
			inFraction = true
		case c >= '0' && c <= '9':
			if !inFraction {
				current = current*10 + float32(c-'0')
			} else {
				fractionDiv *= 10
				current += float32(c-'0') / fractionDiv
			}
		case c == ',':
			res = append(res, sign*current)
			current = 0
			sign = 1
			inFraction = false
			fractionDiv = 1
		}
	}
	res = append(res, sign*current)
	return res
}
