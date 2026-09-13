package consent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDocumentNotFound       = errors.New("consent: document not found")
	ErrConsentNotFound        = errors.New("consent: active consent not found")
	ErrConsentAlreadyGranted  = errors.New("consent: already granted")
	ErrConsentVersionOutdated = errors.New("consent: document version is outdated")
)

type ConsentDocument struct {
	Version     string
	Title       string
	Body        string
	ContentHash string
	IsActive    bool
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type BiometricConsent struct {
	ID              uuid.UUID
	EmployeeID      uuid.UUID
	DocumentVersion string
	ContentHash     string
	Status          string
	Method          string
	GrantedAt       time.Time
	WithdrawnAt     *time.Time
	WithdrawnReason *string
	RecordedBy      *uuid.UUID
	IP              *string
	UserAgent       *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Repository interface {
	GetActiveDocument(ctx context.Context) (*ConsentDocument, error)
	GetDocumentByVersion(ctx context.Context, version string) (*ConsentDocument, error)
	GetActiveConsentByEmployeeID(ctx context.Context, employeeID uuid.UUID) (*BiometricConsent, error)
	GetLatestConsentByEmployeeID(ctx context.Context, employeeID uuid.UUID) (*BiometricConsent, error)
	GrantConsent(ctx context.Context, c *BiometricConsent) error
	WithdrawConsent(ctx context.Context, employeeID uuid.UUID, reason string) (int, error)
	HasActiveConsent(ctx context.Context, employeeID uuid.UUID) (bool, error)
}

type pgRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

func (r *pgRepository) GetActiveDocument(ctx context.Context) (*ConsentDocument, error) {
	query := `
		SELECT version, title, body, content_hash, is_active, published_at, created_at, updated_at
		FROM consent_documents
		WHERE is_active = true
		LIMIT 1;
	`
	doc := &ConsentDocument{}
	err := r.pool.QueryRow(ctx, query).Scan(
		&doc.Version, &doc.Title, &doc.Body, &doc.ContentHash, &doc.IsActive,
		&doc.PublishedAt, &doc.CreatedAt, &doc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("consent repo: get active document: %w", err)
	}
	return doc, nil
}

func (r *pgRepository) GetDocumentByVersion(ctx context.Context, version string) (*ConsentDocument, error) {
	query := `
		SELECT version, title, body, content_hash, is_active, published_at, created_at, updated_at
		FROM consent_documents
		WHERE version = $1;
	`
	doc := &ConsentDocument{}
	err := r.pool.QueryRow(ctx, query, version).Scan(
		&doc.Version, &doc.Title, &doc.Body, &doc.ContentHash, &doc.IsActive,
		&doc.PublishedAt, &doc.CreatedAt, &doc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("consent repo: get document by version: %w", err)
	}
	return doc, nil
}

func (r *pgRepository) GetActiveConsentByEmployeeID(ctx context.Context, employeeID uuid.UUID) (*BiometricConsent, error) {
	query := `
		SELECT id, employee_id, document_version, content_hash, status, method,
		       granted_at, withdrawn_at, withdrawn_reason, recorded_by, ip::text, user_agent,
		       created_at, updated_at
		FROM biometric_consents
		WHERE employee_id = $1 AND status = 'granted'
		LIMIT 1;
	`
	c := &BiometricConsent{}
	err := r.pool.QueryRow(ctx, query, employeeID).Scan(
		&c.ID, &c.EmployeeID, &c.DocumentVersion, &c.ContentHash, &c.Status, &c.Method,
		&c.GrantedAt, &c.WithdrawnAt, &c.WithdrawnReason, &c.RecordedBy, &c.IP, &c.UserAgent,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrConsentNotFound
		}
		return nil, fmt.Errorf("consent repo: get active consent: %w", err)
	}
	return c, nil
}

func (r *pgRepository) GetLatestConsentByEmployeeID(ctx context.Context, employeeID uuid.UUID) (*BiometricConsent, error) {
	query := `
		SELECT id, employee_id, document_version, content_hash, status, method,
		       granted_at, withdrawn_at, withdrawn_reason, recorded_by, ip::text, user_agent,
		       created_at, updated_at
		FROM biometric_consents
		WHERE employee_id = $1
		ORDER BY granted_at DESC
		LIMIT 1;
	`
	c := &BiometricConsent{}
	err := r.pool.QueryRow(ctx, query, employeeID).Scan(
		&c.ID, &c.EmployeeID, &c.DocumentVersion, &c.ContentHash, &c.Status, &c.Method,
		&c.GrantedAt, &c.WithdrawnAt, &c.WithdrawnReason, &c.RecordedBy, &c.IP, &c.UserAgent,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrConsentNotFound
		}
		return nil, fmt.Errorf("consent repo: get latest consent: %w", err)
	}
	return c, nil
}

func (r *pgRepository) GrantConsent(ctx context.Context, c *BiometricConsent) error {
	query := `
		INSERT INTO biometric_consents (
			id, employee_id, document_version, content_hash, status, method,
			granted_at, recorded_by, ip, user_agent
		) VALUES (
			$1, $2, $3, $4, 'granted', $5,
			now(), $6, $7::inet, $8
		)
		RETURNING granted_at;
	`
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	err := r.pool.QueryRow(
		ctx, query,
		c.ID, c.EmployeeID, c.DocumentVersion, c.ContentHash, c.Method,
		c.RecordedBy, c.IP, c.UserAgent,
	).Scan(&c.GrantedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique violation
			return ErrConsentAlreadyGranted
		}
		return fmt.Errorf("consent repo: grant consent: %w", err)
	}
	c.Status = "granted"
	return nil
}

func (r *pgRepository) WithdrawConsent(ctx context.Context, employeeID uuid.UUID, reason string) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("consent repo: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	updateConsentQuery := `
		UPDATE biometric_consents
		SET status = 'withdrawn',
		    withdrawn_at = now(),
		    withdrawn_reason = $2
		WHERE employee_id = $1 AND status = 'granted'
		RETURNING id;
	`
	var consentID uuid.UUID
	err = tx.QueryRow(ctx, updateConsentQuery, employeeID, reason).Scan(&consentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrConsentNotFound
		}
		return 0, fmt.Errorf("consent repo: update consent to withdrawn: %w", err)
	}

	updateRefQuery := `
		UPDATE face_references
		SET is_active = false,
		    deactivated_at = now(),
		    deactivated_reason = 'consent_withdrawn'
		WHERE employee_id = $1 AND is_active = true
		RETURNING id;
	`
	rows, err := tx.Query(ctx, updateRefQuery, employeeID)
	if err != nil {
		return 0, fmt.Errorf("consent repo: deactivate face references: %w", err)
	}
	defer rows.Close()

	deactivatedCount := 0
	for rows.Next() {
		deactivatedCount++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("consent repo: iterate deactivated face references: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("consent repo: commit tx: %w", err)
	}

	return deactivatedCount, nil
}

func (r *pgRepository) HasActiveConsent(ctx context.Context, employeeID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM biometric_consents WHERE employee_id = $1 AND status = 'granted');`
	var exists bool
	err := r.pool.QueryRow(ctx, query, employeeID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("consent repo: has active consent: %w", err)
	}
	return exists, nil
}
