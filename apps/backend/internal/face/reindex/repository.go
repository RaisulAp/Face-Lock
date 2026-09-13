package reindex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const FaceReindexLockID = 20260903

var (
	ErrJobNotFound          = errors.New("face_reindex: job not found")
	ErrJobCannotCancel      = errors.New("face_reindex: job is not in pending or running state")
	ErrReindexAlreadyActive = errors.New("face_reindex: another reindex job is already pending or running")
)

type JobModel struct {
	ID                       uuid.UUID
	FromModelVersion         string
	ToModelVersion           string
	Status                   string
	TotalCount               int
	ProcessedCount           int
	SucceededCount           int
	FailedCount              int
	EmployeesReadyCount      int
	EmployeesIncompleteCount int
	Error                    *string
	CreatedBy                *uuid.UUID
	StartedAt                *time.Time
	FinishedAt               *time.Time
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type ItemCandidate struct {
	JobID           uuid.UUID
	FaceReferenceID uuid.UUID
	EmployeeID      uuid.UUID
	PhotoKey        string
	PhotoSHA256     string
	PhotoBytes      int
	PhotoMIME       string
	Position        int
	CaptureSource   string
	EnrolledBy      *uuid.UUID
}

type ReferenceInsert struct {
	EmployeeID    uuid.UUID
	Position      int
	Embedding     []float32
	QualityScore  float32
	DetScore      float32
	ModelVersion  string
	PhotoKey      string
	PhotoSHA256   string
	PhotoBytes    int
	PhotoMIME     string
	CaptureSource string
	EnrolledBy    *uuid.UUID
}

type Repository interface {
	HasRunningJob(ctx context.Context) (bool, error)
	CountCandidates(ctx context.Context, fromModel string) (totalCount int, affectedEmployees int, err error)
	CreateJob(ctx context.Context, fromModel, toModel string, createdBy *uuid.UUID) (*JobModel, int, error)
	GetJob(ctx context.Context, id uuid.UUID, minRequired int) (*JobModel, []IncompleteEmployee, error)
	ListJobs(ctx context.Context) ([]JobModel, error)
	CancelJob(ctx context.Context, id uuid.UUID) (*JobModel, error)

	// Worker methods
	TryAcquireLock(ctx context.Context) (bool, error)
	ReleaseLock(ctx context.Context) error
	GetNextPendingOrRunningJob(ctx context.Context) (*JobModel, error)
	MarkJobRunning(ctx context.Context, id uuid.UUID) error
	FetchPendingItems(ctx context.Context, jobID uuid.UUID, limit int) ([]ItemCandidate, error)
	SaveItemSuccess(ctx context.Context, jobID, refID uuid.UUID, newRef ReferenceInsert) (uuid.UUID, error)
	SaveItemFailure(ctx context.Context, jobID, refID uuid.UUID, reason string, hints []string) error
	IncrementJobCounters(ctx context.Context, jobID uuid.UUID, succeededDelta, failedDelta int) error
	FinalizeEmployeeSwaps(ctx context.Context, jobID uuid.UUID, minRequired int) (int, int, error)
	MarkJobFailed(ctx context.Context, jobID uuid.UUID, errMsg string) error
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

var _ Repository = (*PostgresRepository)(nil)

func (r *PostgresRepository) HasRunningJob(ctx context.Context) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM face_reindex_jobs WHERE status IN ('pending', 'running'))`
	var exists bool
	if err := r.pool.QueryRow(ctx, query).Scan(&exists); err != nil {
		return false, fmt.Errorf("check running job: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) CountCandidates(ctx context.Context, fromModel string) (int, int, error) {
	const query = `
		SELECT COUNT(*), COUNT(DISTINCT employee_id)
		FROM face_references
		WHERE model_version = $1 AND is_active = true
	`
	var total, affected int
	if err := r.pool.QueryRow(ctx, query, fromModel).Scan(&total, &affected); err != nil {
		return 0, 0, fmt.Errorf("count candidates: %w", err)
	}
	return total, affected, nil
}

func (r *PostgresRepository) CreateJob(ctx context.Context, fromModel, toModel string, createdBy *uuid.UUID) (*JobModel, int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check if already running
	var running bool
	const checkQuery = `SELECT EXISTS(SELECT 1 FROM face_reindex_jobs WHERE status IN ('pending', 'running'))`
	if err := tx.QueryRow(ctx, checkQuery).Scan(&running); err != nil {
		return nil, 0, fmt.Errorf("check running: %w", err)
	}
	if running {
		return nil, 0, ErrReindexAlreadyActive
	}

	// Count candidates
	var total, affected int
	const countQuery = `
		SELECT COUNT(*), COUNT(DISTINCT employee_id)
		FROM face_references
		WHERE model_version = $1 AND is_active = true
	`
	if err := tx.QueryRow(ctx, countQuery, fromModel).Scan(&total, &affected); err != nil {
		return nil, 0, fmt.Errorf("count candidates: %w", err)
	}

	// Insert job
	const insertJob = `
		INSERT INTO face_reindex_jobs (
			from_model_version,
			to_model_version,
			status,
			total_count,
			created_by
		) VALUES ($1, $2, 'pending', $3, $4)
		RETURNING
			id,
			from_model_version,
			to_model_version,
			status,
			total_count,
			processed_count,
			succeeded_count,
			failed_count,
			employees_ready_count,
			employees_incomplete_count,
			error,
			created_by,
			started_at,
			finished_at,
			created_at,
			updated_at
	`
	var j JobModel
	if err := tx.QueryRow(ctx, insertJob, fromModel, toModel, total, createdBy).Scan(
		&j.ID,
		&j.FromModelVersion,
		&j.ToModelVersion,
		&j.Status,
		&j.TotalCount,
		&j.ProcessedCount,
		&j.SucceededCount,
		&j.FailedCount,
		&j.EmployeesReadyCount,
		&j.EmployeesIncompleteCount,
		&j.Error,
		&j.CreatedBy,
		&j.StartedAt,
		&j.FinishedAt,
		&j.CreatedAt,
		&j.UpdatedAt,
	); err != nil {
		return nil, 0, fmt.Errorf("insert job: %w", err)
	}

	// Populate items
	const populateItems = `
		INSERT INTO face_reindex_items (job_id, face_reference_id, status)
		SELECT $1, id, 'pending'
		FROM face_references
		WHERE model_version = $2 AND is_active = true
	`
	if _, err := tx.Exec(ctx, populateItems, j.ID, fromModel); err != nil {
		return nil, 0, fmt.Errorf("populate items: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("commit tx: %w", err)
	}

	return &j, affected, nil
}

func (r *PostgresRepository) GetJob(ctx context.Context, id uuid.UUID, minRequired int) (*JobModel, []IncompleteEmployee, error) {
	const query = `
		SELECT
			id,
			from_model_version,
			to_model_version,
			status,
			total_count,
			processed_count,
			succeeded_count,
			failed_count,
			employees_ready_count,
			employees_incomplete_count,
			error,
			created_by,
			started_at,
			finished_at,
			created_at,
			updated_at
		FROM face_reindex_jobs
		WHERE id = $1
	`
	var j JobModel
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&j.ID,
		&j.FromModelVersion,
		&j.ToModelVersion,
		&j.Status,
		&j.TotalCount,
		&j.ProcessedCount,
		&j.SucceededCount,
		&j.FailedCount,
		&j.EmployeesReadyCount,
		&j.EmployeesIncompleteCount,
		&j.Error,
		&j.CreatedBy,
		&j.StartedAt,
		&j.FinishedAt,
		&j.CreatedAt,
		&j.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrJobNotFound
		}
		return nil, nil, fmt.Errorf("get job: %w", err)
	}

	// Incomplete employees:
	var incomplete []IncompleteEmployee
	if j.Status == "completed" || j.Status == "running" {
		const incQuery = `
			SELECT
				e.id,
				e.employee_number,
				COUNT(fri.new_reference_id) FILTER (WHERE fri.status = 'ok') AS succeeded
			FROM face_reindex_items fri
			JOIN face_references fr ON fr.id = fri.face_reference_id
			JOIN employees e ON e.id = fr.employee_id
			WHERE fri.job_id = $1
			GROUP BY e.id, e.employee_number
			HAVING COUNT(fri.new_reference_id) FILTER (WHERE fri.status = 'ok') < $2
			ORDER BY e.employee_number ASC
		`
		rows, err := r.pool.Query(ctx, incQuery, id, minRequired)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var ie IncompleteEmployee
				ie.Required = minRequired
				if err := rows.Scan(&ie.EmployeeID, &ie.EmployeeNumber, &ie.Succeeded); err == nil {
					incomplete = append(incomplete, ie)
				}
			}
		}
	}

	return &j, incomplete, nil
}

func (r *PostgresRepository) ListJobs(ctx context.Context) ([]JobModel, error) {
	const query = `
		SELECT
			id,
			from_model_version,
			to_model_version,
			status,
			total_count,
			processed_count,
			succeeded_count,
			failed_count,
			employees_ready_count,
			employees_incomplete_count,
			error,
			created_by,
			started_at,
			finished_at,
			created_at,
			updated_at
		FROM face_reindex_jobs
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []JobModel
	for rows.Next() {
		var j JobModel
		if err := rows.Scan(
			&j.ID,
			&j.FromModelVersion,
			&j.ToModelVersion,
			&j.Status,
			&j.TotalCount,
			&j.ProcessedCount,
			&j.SucceededCount,
			&j.FailedCount,
			&j.EmployeesReadyCount,
			&j.EmployeesIncompleteCount,
			&j.Error,
			&j.CreatedBy,
			&j.StartedAt,
			&j.FinishedAt,
			&j.CreatedAt,
			&j.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		jobs = append(jobs, j)
	}

	return jobs, nil
}

func (r *PostgresRepository) CancelJob(ctx context.Context, id uuid.UUID) (*JobModel, error) {
	const query = `
		UPDATE face_reindex_jobs
		SET status = 'cancelled', updated_at = NOW()
		WHERE id = $1 AND status IN ('pending', 'running')
		RETURNING
			id,
			from_model_version,
			to_model_version,
			status,
			total_count,
			processed_count,
			succeeded_count,
			failed_count,
			employees_ready_count,
			employees_incomplete_count,
			error,
			created_by,
			started_at,
			finished_at,
			created_at,
			updated_at
	`
	var j JobModel
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&j.ID,
		&j.FromModelVersion,
		&j.ToModelVersion,
		&j.Status,
		&j.TotalCount,
		&j.ProcessedCount,
		&j.SucceededCount,
		&j.FailedCount,
		&j.EmployeesReadyCount,
		&j.EmployeesIncompleteCount,
		&j.Error,
		&j.CreatedBy,
		&j.StartedAt,
		&j.FinishedAt,
		&j.CreatedAt,
		&j.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrJobCannotCancel
		}
		return nil, fmt.Errorf("cancel job: %w", err)
	}
	return &j, nil
}

// Worker methods

func (r *PostgresRepository) TryAcquireLock(ctx context.Context) (bool, error) {
	var locked bool
	const q = `SELECT pg_try_advisory_lock($1)`
	if err := r.pool.QueryRow(ctx, q, FaceReindexLockID).Scan(&locked); err != nil {
		return false, fmt.Errorf("advisory lock: %w", err)
	}
	return locked, nil
}

func (r *PostgresRepository) ReleaseLock(ctx context.Context) error {
	var unlocked bool
	const q = `SELECT pg_advisory_unlock($1)`
	if err := r.pool.QueryRow(ctx, q, FaceReindexLockID).Scan(&unlocked); err != nil {
		return fmt.Errorf("advisory unlock: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetNextPendingOrRunningJob(ctx context.Context) (*JobModel, error) {
	const query = `
		SELECT
			id,
			from_model_version,
			to_model_version,
			status,
			total_count,
			processed_count,
			succeeded_count,
			failed_count,
			employees_ready_count,
			employees_incomplete_count,
			error,
			created_by,
			started_at,
			finished_at,
			created_at,
			updated_at
		FROM face_reindex_jobs
		WHERE status IN ('pending', 'running')
		ORDER BY created_at ASC
		LIMIT 1
	`
	var j JobModel
	err := r.pool.QueryRow(ctx, query).Scan(
		&j.ID,
		&j.FromModelVersion,
		&j.ToModelVersion,
		&j.Status,
		&j.TotalCount,
		&j.ProcessedCount,
		&j.SucceededCount,
		&j.FailedCount,
		&j.EmployeesReadyCount,
		&j.EmployeesIncompleteCount,
		&j.Error,
		&j.CreatedBy,
		&j.StartedAt,
		&j.FinishedAt,
		&j.CreatedAt,
		&j.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get pending job: %w", err)
	}
	return &j, nil
}

func (r *PostgresRepository) MarkJobRunning(ctx context.Context, id uuid.UUID) error {
	const query = `
		UPDATE face_reindex_jobs
		SET status = 'running', started_at = COALESCE(started_at, NOW()), updated_at = NOW()
		WHERE id = $1 AND status = 'pending'
	`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *PostgresRepository) FetchPendingItems(ctx context.Context, jobID uuid.UUID, limit int) ([]ItemCandidate, error) {
	const query = `
		SELECT
			fri.job_id,
			fri.face_reference_id,
			fr.employee_id,
			fr.photo_key,
			fr.photo_sha256,
			fr.photo_bytes,
			fr.photo_mime,
			fr.position,
			fr.capture_source,
			fr.enrolled_by
		FROM face_reindex_items fri
		JOIN face_references fr ON fr.id = fri.face_reference_id
		WHERE fri.job_id = $1 AND fri.status = 'pending'
		ORDER BY fr.employee_id ASC, fr.position ASC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, query, jobID, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch pending items: %w", err)
	}
	defer rows.Close()

	var items []ItemCandidate
	for rows.Next() {
		var it ItemCandidate
		if err := rows.Scan(
			&it.JobID,
			&it.FaceReferenceID,
			&it.EmployeeID,
			&it.PhotoKey,
			&it.PhotoSHA256,
			&it.PhotoBytes,
			&it.PhotoMIME,
			&it.Position,
			&it.CaptureSource,
			&it.EnrolledBy,
		); err != nil {
			return nil, fmt.Errorf("scan item candidate: %w", err)
		}
		items = append(items, it)
	}

	return items, nil
}

func (r *PostgresRepository) SaveItemSuccess(ctx context.Context, jobID, refID uuid.UUID, newRef ReferenceInsert) (uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert new reference as inactive
	newID := uuid.New()
	const insertRef = `
		INSERT INTO face_references (
			id,
			employee_id,
			position,
			embedding,
			quality_score,
			det_score,
			model_version,
			photo_key,
			photo_sha256,
			photo_bytes,
			photo_mime,
			capture_source,
			is_active,
			deactivated_at,
			deactivated_reason,
			enrolled_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, false, NOW(), 'model_reindex', $13)
	`
	vecStr := formatVector(newRef.Embedding)
	if _, err := tx.Exec(ctx, insertRef,
		newID,
		newRef.EmployeeID,
		newRef.Position,
		vecStr,
		newRef.QualityScore,
		newRef.DetScore,
		newRef.ModelVersion,
		newRef.PhotoKey,
		newRef.PhotoSHA256,
		newRef.PhotoBytes,
		newRef.PhotoMIME,
		newRef.CaptureSource,
		newRef.EnrolledBy,
	); err != nil {
		return uuid.Nil, fmt.Errorf("insert new reference: %w", err)
	}

	// Update superseded_by on old reference
	const updateOld = `UPDATE face_references SET superseded_by = $1, updated_at = NOW() WHERE id = $2`
	if _, err := tx.Exec(ctx, updateOld, newID, refID); err != nil {
		return uuid.Nil, fmt.Errorf("update superseded_by: %w", err)
	}

	// Update item
	const updateItem = `
		UPDATE face_reindex_items
		SET status = 'ok', new_reference_id = $1, processed_at = NOW()
		WHERE job_id = $2 AND face_reference_id = $3
	`
	if _, err := tx.Exec(ctx, updateItem, newID, jobID, refID); err != nil {
		return uuid.Nil, fmt.Errorf("update item: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit tx: %w", err)
	}

	return newID, nil
}

func (r *PostgresRepository) SaveItemFailure(ctx context.Context, jobID, refID uuid.UUID, reason string, hints []string) error {
	hintsJSON, _ := json.Marshal(hints)
	const query = `
		UPDATE face_reindex_items
		SET status = 'failed', reason = $1, hints = $2, processed_at = NOW()
		WHERE job_id = $3 AND face_reference_id = $4
	`
	_, err := r.pool.Exec(ctx, query, reason, hintsJSON, jobID, refID)
	return err
}

func (r *PostgresRepository) IncrementJobCounters(ctx context.Context, jobID uuid.UUID, succeededDelta, failedDelta int) error {
	processedDelta := succeededDelta + failedDelta
	const query = `
		UPDATE face_reindex_jobs
		SET processed_count = processed_count + $1,
		    succeeded_count = succeeded_count + $2,
		    failed_count = failed_count + $3,
		    updated_at = NOW()
		WHERE id = $4
	`
	_, err := r.pool.Exec(ctx, query, processedDelta, succeededDelta, failedDelta, jobID)
	return err
}

func (r *PostgresRepository) FinalizeEmployeeSwaps(ctx context.Context, jobID uuid.UUID, minRequired int) (int, int, error) {
	// Find all affected employees
	const empQuery = `
		SELECT DISTINCT fr.employee_id
		FROM face_reindex_items fri
		JOIN face_references fr ON fr.id = fri.face_reference_id
		WHERE fri.job_id = $1
	`
	rows, err := r.pool.Query(ctx, empQuery, jobID)
	if err != nil {
		return 0, 0, fmt.Errorf("query affected employees: %w", err)
	}
	defer rows.Close()

	var empIDs []uuid.UUID
	for rows.Next() {
		var eid uuid.UUID
		if err := rows.Scan(&eid); err == nil {
			empIDs = append(empIDs, eid)
		}
	}
	rows.Close()

	readyCount := 0
	incompleteCount := 0

	for _, empID := range empIDs {
		// Single transaction per employee
		tx, err := r.pool.Begin(ctx)
		if err != nil {
			return readyCount, incompleteCount, fmt.Errorf("begin tx for emp %s: %w", empID, err)
		}

		// Count succeeded new references for this employee from this job
		const countSucc = `
			SELECT COUNT(*)
			FROM face_reindex_items fri
			JOIN face_references fr ON fr.id = fri.face_reference_id
			WHERE fri.job_id = $1 AND fr.employee_id = $2 AND fri.status = 'ok'
		`
		var succCount int
		if err := tx.QueryRow(ctx, countSucc, jobID, empID).Scan(&succCount); err != nil {
			tx.Rollback(ctx)
			return readyCount, incompleteCount, fmt.Errorf("count succeeded for emp: %w", err)
		}

		if succCount >= minRequired {
			// Swap!
			// 1. Deactivate old references
			const deactOld = `
				UPDATE face_references
				SET is_active = false, deactivated_reason = 'model_reindex', deactivated_at = NOW(), updated_at = NOW()
				WHERE id IN (
					SELECT face_reference_id FROM face_reindex_items WHERE job_id = $1
				) AND employee_id = $2 AND is_active = true
			`
			if _, err := tx.Exec(ctx, deactOld, jobID, empID); err != nil {
				tx.Rollback(ctx)
				return readyCount, incompleteCount, fmt.Errorf("deactivate old: %w", err)
			}

			// 2. Activate new references
			const actNew = `
				UPDATE face_references
				SET is_active = true, deactivated_at = NULL, deactivated_reason = NULL, updated_at = NOW()
				WHERE id IN (
					SELECT new_reference_id FROM face_reindex_items WHERE job_id = $1 AND new_reference_id IS NOT NULL
				) AND employee_id = $2
			`
			if _, err := tx.Exec(ctx, actNew, jobID, empID); err != nil {
				tx.Rollback(ctx)
				return readyCount, incompleteCount, fmt.Errorf("activate new: %w", err)
			}

			if err := tx.Commit(ctx); err != nil {
				return readyCount, incompleteCount, fmt.Errorf("commit swap: %w", err)
			}
			readyCount++
		} else {
			// Do not swap! Keep old references.
			tx.Rollback(ctx)
			incompleteCount++
		}
	}

	// Update job record as completed
	const finalizeJob = `
		UPDATE face_reindex_jobs
		SET status = 'completed',
		    employees_ready_count = $1,
		    employees_incomplete_count = $2,
		    finished_at = NOW(),
		    updated_at = NOW()
		WHERE id = $3
	`
	if _, err := r.pool.Exec(ctx, finalizeJob, readyCount, incompleteCount, jobID); err != nil {
		return readyCount, incompleteCount, fmt.Errorf("finalize job: %w", err)
	}

	return readyCount, incompleteCount, nil
}

func (r *PostgresRepository) MarkJobFailed(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	const query = `
		UPDATE face_reindex_jobs
		SET status = 'failed', error = $1, finished_at = NOW(), updated_at = NOW()
		WHERE id = $2
	`
	_, err := r.pool.Exec(ctx, query, errMsg, jobID)
	return err
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
