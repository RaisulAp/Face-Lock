package reference

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
	ErrReferenceNotFound = errors.New("face_reference: not found")
	ErrEmployeeNotFound  = errors.New("face_reference: employee not found")
)

type ReferenceModel struct {
	ID                uuid.UUID
	EmployeeID        uuid.UUID
	Position          int
	QualityScore      float32
	DetScore          float32
	ModelVersion      string
	PhotoKey          string
	PhotoMIME         string
	CaptureSource     string
	IsActive          bool
	EnrolledBy        *uuid.UUID
	EnrolledByEmail   *string
	PhotoPurgedAt     *time.Time
	DeactivatedAt     *time.Time
	DeactivatedReason *string
	CreatedAt         time.Time
}

type EmployeeFaceSummary struct {
	EmployeeID      uuid.UUID
	AttendanceMode  string
	ConsentStatus   string
	ConsentDocVer   string
	IsCurrentDocVer bool
	ActiveRefCount  int
	DraftSessionID  *uuid.UUID
	LatestModelVer  string
}

type Repository interface {
	ListByEmployee(ctx context.Context, employeeID uuid.UUID, includeInactive bool) ([]ReferenceModel, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ReferenceModel, error)
	CountActive(ctx context.Context, employeeID uuid.UUID) (int, error)
	Deactivate(ctx context.Context, id uuid.UUID, reason string) (*ReferenceModel, error)
	DeleteAllForEmployee(ctx context.Context, employeeID uuid.UUID) (deletedCount int, photoKeys []string, err error)
	GetEmployeeFaceSummary(ctx context.Context, employeeID uuid.UUID, currentDocVersion string) (*EmployeeFaceSummary, error)
	EmployeeExists(ctx context.Context, employeeID uuid.UUID) (bool, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

var _ Repository = (*PostgresRepository)(nil)

func (r *PostgresRepository) EmployeeExists(ctx context.Context, employeeID uuid.UUID) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM employees WHERE id = $1 AND deleted_at IS NULL)`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, employeeID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check employee exists: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) ListByEmployee(ctx context.Context, employeeID uuid.UUID, includeInactive bool) ([]ReferenceModel, error) {
	query := `
		SELECT
			fr.id,
			fr.employee_id,
			fr.position,
			fr.quality_score,
			fr.det_score,
			fr.model_version,
			fr.photo_key,
			fr.photo_mime,
			fr.capture_source,
			fr.is_active,
			fr.enrolled_by,
			u.email AS enrolled_by_email,
			fr.photo_purged_at,
			fr.deactivated_at,
			fr.deactivated_reason,
			fr.created_at
		FROM face_references fr
		LEFT JOIN users u ON u.id = fr.enrolled_by
		WHERE fr.employee_id = $1
	`
	if !includeInactive {
		query += ` AND fr.is_active = true`
	}
	query += ` ORDER BY fr.position ASC, fr.created_at ASC`

	rows, err := r.pool.Query(ctx, query, employeeID)
	if err != nil {
		return nil, fmt.Errorf("list face references: %w", err)
	}
	defer rows.Close()

	var refs []ReferenceModel
	for rows.Next() {
		var rm ReferenceModel
		if err := rows.Scan(
			&rm.ID,
			&rm.EmployeeID,
			&rm.Position,
			&rm.QualityScore,
			&rm.DetScore,
			&rm.ModelVersion,
			&rm.PhotoKey,
			&rm.PhotoMIME,
			&rm.CaptureSource,
			&rm.IsActive,
			&rm.EnrolledBy,
			&rm.EnrolledByEmail,
			&rm.PhotoPurgedAt,
			&rm.DeactivatedAt,
			&rm.DeactivatedReason,
			&rm.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan face reference: %w", err)
		}
		refs = append(refs, rm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iterate face references: %w", err)
	}

	return refs, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*ReferenceModel, error) {
	const query = `
		SELECT
			fr.id,
			fr.employee_id,
			fr.position,
			fr.quality_score,
			fr.det_score,
			fr.model_version,
			fr.photo_key,
			fr.photo_mime,
			fr.capture_source,
			fr.is_active,
			fr.enrolled_by,
			u.email AS enrolled_by_email,
			fr.photo_purged_at,
			fr.deactivated_at,
			fr.deactivated_reason,
			fr.created_at
		FROM face_references fr
		LEFT JOIN users u ON u.id = fr.enrolled_by
		WHERE fr.id = $1
	`
	var rm ReferenceModel
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&rm.ID,
		&rm.EmployeeID,
		&rm.Position,
		&rm.QualityScore,
		&rm.DetScore,
		&rm.ModelVersion,
		&rm.PhotoKey,
		&rm.PhotoMIME,
		&rm.CaptureSource,
		&rm.IsActive,
		&rm.EnrolledBy,
		&rm.EnrolledByEmail,
		&rm.PhotoPurgedAt,
		&rm.DeactivatedAt,
		&rm.DeactivatedReason,
		&rm.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReferenceNotFound
		}
		return nil, fmt.Errorf("get face reference by id: %w", err)
	}

	return &rm, nil
}

func (r *PostgresRepository) CountActive(ctx context.Context, employeeID uuid.UUID) (int, error) {
	const query = `SELECT COUNT(*) FROM face_references WHERE employee_id = $1 AND is_active = true`
	var count int
	if err := r.pool.QueryRow(ctx, query, employeeID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count active references: %w", err)
	}
	return count, nil
}

func (r *PostgresRepository) Deactivate(ctx context.Context, id uuid.UUID, reason string) (*ReferenceModel, error) {
	const query = `
		UPDATE face_references
		SET is_active = false,
		    deactivated_at = NOW(),
		    deactivated_reason = $2,
		    updated_at = NOW()
		WHERE id = $1 AND is_active = true
		RETURNING
			id,
			employee_id,
			position,
			quality_score,
			det_score,
			model_version,
			photo_key,
			photo_mime,
			capture_source,
			is_active,
			enrolled_by,
			photo_purged_at,
			deactivated_at,
			deactivated_reason,
			created_at
	`
	var rm ReferenceModel
	err := r.pool.QueryRow(ctx, query, id, reason).Scan(
		&rm.ID,
		&rm.EmployeeID,
		&rm.Position,
		&rm.QualityScore,
		&rm.DetScore,
		&rm.ModelVersion,
		&rm.PhotoKey,
		&rm.PhotoMIME,
		&rm.CaptureSource,
		&rm.IsActive,
		&rm.EnrolledBy,
		&rm.PhotoPurgedAt,
		&rm.DeactivatedAt,
		&rm.DeactivatedReason,
		&rm.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReferenceNotFound
		}
		return nil, fmt.Errorf("deactivate face reference: %w", err)
	}
	return &rm, nil
}

func (r *PostgresRepository) DeleteAllForEmployee(ctx context.Context, employeeID uuid.UUID) (int, []string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Collect photo keys before deleting
	const selectKeys = `SELECT photo_key FROM face_references WHERE employee_id = $1`
	rows, err := tx.Query(ctx, selectKeys, employeeID)
	if err != nil {
		return 0, nil, fmt.Errorf("query photo keys: %w", err)
	}
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err == nil {
			keys = append(keys, k)
		}
	}
	rows.Close()

	// Hard delete references
	const deleteRefs = `DELETE FROM face_references WHERE employee_id = $1`
	cmdTag, err := tx.Exec(ctx, deleteRefs, employeeID)
	if err != nil {
		return 0, nil, fmt.Errorf("delete face references: %w", err)
	}
	deletedCount := int(cmdTag.RowsAffected())

	// Cancel any active draft sessions
	const cancelSessions = `
		UPDATE face_enrollment_sessions
		SET status = 'cancelled', updated_at = NOW()
		WHERE employee_id = $1 AND status = 'draft'
	`
	if _, err := tx.Exec(ctx, cancelSessions, employeeID); err != nil {
		return 0, nil, fmt.Errorf("cancel active draft sessions: %w", err)
	}

	// Update attendance_mode to 'manual'
	const setManual = `UPDATE employees SET attendance_mode = 'manual', updated_at = NOW() WHERE id = $1`
	if _, err := tx.Exec(ctx, setManual, employeeID); err != nil {
		return 0, nil, fmt.Errorf("set attendance_mode to manual: %w", err)
	}

	// Update biometric consent to withdrawn
	const withdrawConsent = `
		UPDATE biometric_consents
		SET status = 'withdrawn', withdrawn_at = NOW(), updated_at = NOW()
		WHERE employee_id = $1 AND status = 'granted'
	`
	if _, err := tx.Exec(ctx, withdrawConsent, employeeID); err != nil {
		return 0, nil, fmt.Errorf("withdraw biometric consents: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, nil, fmt.Errorf("commit tx: %w", err)
	}

	return deletedCount, keys, nil
}

func (r *PostgresRepository) GetEmployeeFaceSummary(ctx context.Context, employeeID uuid.UUID, currentDocVersion string) (*EmployeeFaceSummary, error) {
	summary := &EmployeeFaceSummary{
		EmployeeID: employeeID,
	}

	// 1. Employee attendance_mode
	const empQuery = `SELECT attendance_mode FROM employees WHERE id = $1 AND deleted_at IS NULL`
	if err := r.pool.QueryRow(ctx, empQuery, employeeID).Scan(&summary.AttendanceMode); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEmployeeNotFound
		}
		return nil, fmt.Errorf("get attendance mode: %w", err)
	}

	// 2. Biometric consent
	const consentQuery = `
		SELECT status, document_version
		FROM biometric_consents
		WHERE employee_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	var status, docVer string
	err := r.pool.QueryRow(ctx, consentQuery, employeeID).Scan(&status, &docVer)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			summary.ConsentStatus = "none"
		} else {
			return nil, fmt.Errorf("get consent: %w", err)
		}
	} else {
		summary.ConsentStatus = status
		summary.ConsentDocVer = docVer
		summary.IsCurrentDocVer = (status == "granted" && docVer == currentDocVersion)
	}

	// 3. Active face references count & latest model_version
	const refQuery = `
		SELECT COUNT(*), COALESCE(MAX(model_version), '')
		FROM face_references
		WHERE employee_id = $1 AND is_active = true
	`
	if err := r.pool.QueryRow(ctx, refQuery, employeeID).Scan(&summary.ActiveRefCount, &summary.LatestModelVer); err != nil {
		return nil, fmt.Errorf("count active refs: %w", err)
	}

	// 4. Active draft session
	const draftQuery = `
		SELECT id
		FROM face_enrollment_sessions
		WHERE employee_id = $1 AND status = 'draft' AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`
	var draftID uuid.UUID
	err = r.pool.QueryRow(ctx, draftQuery, employeeID).Scan(&draftID)
	if err == nil {
		summary.DraftSessionID = &draftID
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get active draft session: %w", err)
	}

	return summary, nil
}
