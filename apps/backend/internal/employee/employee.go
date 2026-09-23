package employee

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/face"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Employee represents an employee domain model according to Fase 1 § 3.1 & § 4.2 and Fase 5 REV-EP-03.
//
// EmployeeNumber became optional in migration 000025: an admin can now
// register an employee account with only name/email/phone/department/office,
// and the employee supplies their real NIP from the portal afterwards.
type Employee struct {
	ID               uuid.UUID  `json:"id"`
	EmployeeNumber   *string    `json:"employee_number"`
	FullName         string     `json:"full_name"`
	Department       *string    `json:"department"`
	Position         *string    `json:"position"`
	Phone            *string    `json:"phone"`
	Email            *string    `json:"email"`
	JoinDate         *string    `json:"join_date"` // YYYY-MM-DD
	EmploymentStatus string     `json:"employment_status"`
	HasUserAccount   bool       `json:"has_user_account"`
	AttendanceMode   string     `json:"attendance_mode"`
	OfficeLocationID *uuid.UUID `json:"office_location_id"`
	// OfficeLocationName is joined for display convenience.
	OfficeLocationName *string `json:"office_location_name"`
	// ProfileCompleted is true once the employee has supplied the fields HR
	// left to them (real NIP, position, join date).
	ProfileCompleted bool       `json:"profile_completed"`
	ConsentStatus    string     `json:"consent_status"`
	EnrollmentStatus string     `json:"enrollment_status"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// Filter carries query parameters for listing employees.
type Filter struct {
	Page             int
	PerPage          int
	Search           string
	Department       string
	EmploymentStatus string
	HasUser          *bool
	AttendanceMode   string
	ConsentStatus    string
	EnrollmentStatus string
	// ProfileCompleted filters on whether the employee finished their own
	// profile. nil means "no filter".
	ProfileCompleted *bool
}

// Service provides employee operations.
type Service struct {
	db         *pgxpool.Pool
	faceEngine face.FaceEngine
	store      storage.Store
}

const consentStatusExpr = `(CASE
	WHEN (SELECT bc.status FROM biometric_consents bc WHERE bc.employee_id = e.id ORDER BY bc.created_at DESC LIMIT 1) = 'withdrawn' THEN 'withdrawn'
	WHEN (SELECT bc.status FROM biometric_consents bc WHERE bc.employee_id = e.id ORDER BY bc.created_at DESC LIMIT 1) = 'granted' THEN
		CASE
			WHEN (SELECT bc.document_version FROM biometric_consents bc WHERE bc.employee_id = e.id ORDER BY bc.created_at DESC LIMIT 1) = COALESCE((SELECT cd.version FROM consent_documents cd WHERE cd.is_active = true LIMIT 1), '2026-09-v1') THEN 'granted'
			ELSE 'outdated'
		END
	ELSE 'none'
END)`

const enrollmentStatusExpr = `(CASE
	WHEN COALESCE((SELECT COUNT(*) FROM face_references fr WHERE fr.employee_id = e.id AND fr.is_active = true), 0) = 0 THEN 'none'
	WHEN (SELECT COUNT(*) FROM face_references fr WHERE fr.employee_id = e.id AND fr.is_active = true) < 3 THEN 'incomplete'
	ELSE 'complete'
END)`

// NewService creates a new employee service.
func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

// List returns a paginated list of employees.
func (s *Service) List(ctx context.Context, f Filter) ([]Employee, httpx.Meta, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PerPage <= 0 {
		f.PerPage = 20
	}
	if f.PerPage > 100 {
		f.PerPage = 100
	}

	whereClauses := []string{"e.deleted_at IS NULL"}
	var args []any
	argIdx := 1

	if f.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(e.employee_number ILIKE $%d OR e.full_name ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+f.Search+"%")
		argIdx++
	}
	if f.Department != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("e.department = $%d", argIdx))
		args = append(args, f.Department)
		argIdx++
	}
	if f.EmploymentStatus != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("e.employment_status = $%d", argIdx))
		args = append(args, f.EmploymentStatus)
		argIdx++
	}
	if f.HasUser != nil {
		if *f.HasUser {
			whereClauses = append(whereClauses, "EXISTS (SELECT 1 FROM users u WHERE u.employee_id = e.id AND u.deleted_at IS NULL)")
		} else {
			whereClauses = append(whereClauses, "NOT EXISTS (SELECT 1 FROM users u WHERE u.employee_id = e.id AND u.deleted_at IS NULL)")
		}
	}
	if f.AttendanceMode != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("COALESCE(e.attendance_mode, 'face') = $%d", argIdx))
		args = append(args, f.AttendanceMode)
		argIdx++
	}
	if f.ConsentStatus != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", consentStatusExpr, argIdx))
		args = append(args, f.ConsentStatus)
		argIdx++
	}
	if f.EnrollmentStatus != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", enrollmentStatusExpr, argIdx))
		args = append(args, f.EnrollmentStatus)
		argIdx++
	}
	if f.ProfileCompleted != nil {
		if *f.ProfileCompleted {
			whereClauses = append(whereClauses, "e.profile_completed_at IS NOT NULL")
		} else {
			whereClauses = append(whereClauses, "e.profile_completed_at IS NULL")
		}
	}

	whereSQL := "WHERE " + strings.Join(whereClauses, " AND ")

	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM employees e %s", whereSQL)
	var total int
	if err := s.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, httpx.Meta{}, fmt.Errorf("counting employees: %w", err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + f.PerPage - 1) / f.PerPage
	}

	offset := (f.Page - 1) * f.PerPage
	querySQL := fmt.Sprintf(`
		SELECT
			e.id,
			e.employee_number,
			e.full_name,
			e.department,
			e.position,
			e.phone,
			e.email,
			e.join_date::text,
			e.employment_status,
			EXISTS (SELECT 1 FROM users u WHERE u.employee_id = e.id AND u.deleted_at IS NULL) AS has_user_account,
			COALESCE(e.attendance_mode, 'face') AS attendance_mode,
			e.office_location_id,
			ol.name AS office_location_name,
			(e.profile_completed_at IS NOT NULL) AS profile_completed,
			`+consentStatusExpr+` AS consent_status,
			`+enrollmentStatusExpr+` AS enrollment_status,
			e.created_at,
			e.updated_at
		FROM employees e
		LEFT JOIN office_locations ol ON ol.id = e.office_location_id
		%s
		ORDER BY e.full_name ASC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, f.PerPage, offset)

	rows, err := s.db.Query(ctx, querySQL, args...)
	if err != nil {
		return nil, httpx.Meta{}, fmt.Errorf("querying employees: %w", err)
	}
	defer rows.Close()

	var employees []Employee
	for rows.Next() {
		var e Employee
		if err := rows.Scan(
			&e.ID,
			&e.EmployeeNumber,
			&e.FullName,
			&e.Department,
			&e.Position,
			&e.Phone,
			&e.Email,
			&e.JoinDate,
			&e.EmploymentStatus,
			&e.HasUserAccount,
			&e.AttendanceMode,
			&e.OfficeLocationID,
			&e.OfficeLocationName,
			&e.ProfileCompleted,
			&e.ConsentStatus,
			&e.EnrollmentStatus,
			&e.CreatedAt,
			&e.UpdatedAt,
		); err != nil {
			return nil, httpx.Meta{}, fmt.Errorf("scanning employee: %w", err)
		}
		employees = append(employees, e)
	}

	if employees == nil {
		employees = []Employee{}
	}

	meta := httpx.Meta{
		Page:       f.Page,
		PerPage:    f.PerPage,
		Total:      total,
		TotalPages: totalPages,
	}

	return employees, meta, nil
}

// GetByID finds an employee by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Employee, error) {
	querySQL := `
		SELECT
			e.id,
			e.employee_number,
			e.full_name,
			e.department,
			e.position,
			e.phone,
			e.email,
			e.join_date::text,
			e.employment_status,
			EXISTS (SELECT 1 FROM users u WHERE u.employee_id = e.id AND u.deleted_at IS NULL) AS has_user_account,
			COALESCE(e.attendance_mode, 'face') AS attendance_mode,
			e.office_location_id,
			ol.name AS office_location_name,
			(e.profile_completed_at IS NOT NULL) AS profile_completed,
			` + consentStatusExpr + ` AS consent_status,
			` + enrollmentStatusExpr + ` AS enrollment_status,
			e.created_at,
			e.updated_at
		FROM employees e
		LEFT JOIN office_locations ol ON ol.id = e.office_location_id
		WHERE e.id = $1 AND e.deleted_at IS NULL
	`
	var e Employee
	err := s.db.QueryRow(ctx, querySQL, id).Scan(
		&e.ID,
		&e.EmployeeNumber,
		&e.FullName,
		&e.Department,
		&e.Position,
		&e.Phone,
		&e.Email,
		&e.JoinDate,
		&e.EmploymentStatus,
		&e.HasUserAccount,
		&e.AttendanceMode,
		&e.OfficeLocationID,
		&e.OfficeLocationName,
		&e.ProfileCompleted,
		&e.ConsentStatus,
		&e.EnrollmentStatus,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "employee not found")
		}
		return nil, fmt.Errorf("querying employee: %w", err)
	}
	return &e, nil
}

// CreateParams input for creating an employee.
//
// EmployeeNumber is optional since migration 000025. When omitted, the service
// auto-generates a placeholder NIP (e.g. EMP-2026-0001) so the employee always
// has a non-blank identifier, and marks the profile as not yet completed.
type CreateParams struct {
	EmployeeNumber   *string    `json:"employee_number" validate:"omitempty,max=50"`
	FullName         string     `json:"full_name" validate:"required,min=2,max=120"`
	Department       *string    `json:"department" validate:"omitempty,max=100"`
	Position         *string    `json:"position" validate:"omitempty,max=100"`
	Phone            *string    `json:"phone" validate:"omitempty,max=20"`
	Email            *string    `json:"email" validate:"omitempty,email"`
	JoinDate         *string    `json:"join_date" validate:"omitempty,datetime=2006-01-02"`
	EmploymentStatus string     `json:"employment_status" validate:"omitempty,oneof=active inactive resigned"`
	AttendanceMode   *string    `json:"attendance_mode" validate:"omitempty,oneof=face pin remote hybrid"`
	OfficeLocationID *uuid.UUID `json:"office_location_id" validate:"omitempty"`
	// ProfileCompleted should be set by callers that collect the full HR
	// profile up-front (the classic employee form). Single-step registration
	// leaves it false so the employee completes it via the portal.
	ProfileCompleted bool `json:"-"`
}

// generateEmployeeNumber produces a placeholder NIP in the form
// EMP-<year>-<4-digit sequence>, unique across non-deleted employees.
func (s *Service) generateEmployeeNumber(ctx context.Context) (string, error) {
	const q = `
		SELECT COALESCE(MAX(
			CASE
				WHEN employee_number ~ ('^EMP-' || $1 || '-[0-9]+$')
				THEN NULLIF(regexp_replace(employee_number::text, '^.*-', ''), '')::int
			END
		), 0) + 1
		FROM employees
		WHERE deleted_at IS NULL
	`
	year := time.Now().Year()
	var next int
	if err := s.db.QueryRow(ctx, q, strconv.Itoa(year)).Scan(&next); err != nil {
		return "", fmt.Errorf("computing next employee number: %w", err)
	}
	return fmt.Sprintf("EMP-%d-%04d", year, next), nil
}

// Create inserts a new employee.
func (s *Service) Create(ctx context.Context, p CreateParams) (*Employee, error) {
	status := p.EmploymentStatus
	if status == "" {
		status = "active"
	}
	mode := "face"
	if p.AttendanceMode != nil && *p.AttendanceMode != "" {
		mode = *p.AttendanceMode
	}

	empNumber := p.EmployeeNumber
	if empNumber != nil {
		trimmed := strings.TrimSpace(*empNumber)
		// Treat whitespace-only input as "not provided".
		if trimmed == "" {
			empNumber = nil
		} else {
			empNumber = &trimmed
		}
	}

	// Auto-generate a placeholder NIP when the admin did not supply one. The
	// employee can correct it later from the portal.
	if empNumber == nil {
		generated, err := s.generateEmployeeNumber(ctx)
		if err != nil {
			return nil, err
		}
		empNumber = &generated
	}

	// A profile is complete only when it was created through the full HR form,
	// which always supplies a NIP plus the self-service fields.
	profileCompleted := p.ProfileCompleted

	const q = `
		INSERT INTO employees (
			employee_number, full_name, department, position, phone, email, join_date,
			employment_status, attendance_mode, office_location_id, profile_completed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7::date, $8, $9, $10,
			CASE WHEN $11 THEN NOW() ELSE NULL END
		)
		RETURNING id
	`
	var newID uuid.UUID
	err := s.db.QueryRow(ctx, q,
		*empNumber,
		strings.TrimSpace(p.FullName),
		p.Department,
		p.Position,
		p.Phone,
		p.Email,
		p.JoinDate,
		status,
		mode,
		p.OfficeLocationID,
		profileCompleted,
	).Scan(&newID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, httpx.NewAppError(httpx.CodeEmployeeNumberTaken, "employee number is already registered")
		}
		return nil, fmt.Errorf("inserting employee: %w", err)
	}

	return s.GetByID(ctx, newID)
}

// CreateTx inserts an employee inside an existing transaction. Used by the
// single-step registration flow so the employee and its user account are
// created atomically.
func (s *Service) CreateTx(ctx context.Context, tx pgx.Tx, p CreateParams) (uuid.UUID, error) {
	status := p.EmploymentStatus
	if status == "" {
		status = "active"
	}
	mode := "face"
	if p.AttendanceMode != nil && *p.AttendanceMode != "" {
		mode = *p.AttendanceMode
	}

	empNumber := p.EmployeeNumber
	if empNumber != nil {
		trimmed := strings.TrimSpace(*empNumber)
		if trimmed == "" {
			empNumber = nil
		} else {
			empNumber = &trimmed
		}
	}

	if empNumber == nil {
		// Generating outside the transaction would race under concurrency, so
		// compute it from inside the same transaction on the same connection.
		const genQ = `
			SELECT COALESCE(MAX(
				CASE
					WHEN employee_number ~ ('^EMP-' || $1 || '-[0-9]+$')
					THEN NULLIF(regexp_replace(employee_number::text, '^.*-', ''), '')::int
				END
			), 0) + 1
			FROM employees
			WHERE deleted_at IS NULL
		`
		year := time.Now().Year()
		var next int
		if err := tx.QueryRow(ctx, genQ, strconv.Itoa(year)).Scan(&next); err != nil {
			return uuid.Nil, fmt.Errorf("computing next employee number: %w", err)
		}
		generated := fmt.Sprintf("EMP-%d-%04d", year, next)
		empNumber = &generated
	}

	const q = `
		INSERT INTO employees (
			employee_number, full_name, department, position, phone, email, join_date,
			employment_status, attendance_mode, office_location_id, profile_completed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7::date, $8, $9, $10,
			CASE WHEN $11 THEN NOW() ELSE NULL END
		)
		RETURNING id
	`
	var newID uuid.UUID
	err := tx.QueryRow(ctx, q,
		*empNumber,
		strings.TrimSpace(p.FullName),
		p.Department,
		p.Position,
		p.Phone,
		p.Email,
		p.JoinDate,
		status,
		mode,
		p.OfficeLocationID,
		p.ProfileCompleted,
	).Scan(&newID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return uuid.Nil, httpx.NewAppError(httpx.CodeEmployeeNumberTaken, "employee number is already registered")
		}
		return uuid.Nil, fmt.Errorf("inserting employee: %w", err)
	}

	return newID, nil
}

// UpdateParams input for updating an employee.
type UpdateParams struct {
	EmployeeNumber   *string    `json:"employee_number" validate:"omitempty,max=50"`
	FullName         *string    `json:"full_name" validate:"omitempty,min=2,max=120"`
	Department       *string    `json:"department" validate:"omitempty,max=100"`
	Position         *string    `json:"position" validate:"omitempty,max=100"`
	Phone            *string    `json:"phone" validate:"omitempty,max=20"`
	Email            *string    `json:"email" validate:"omitempty,email"`
	JoinDate         *string    `json:"join_date" validate:"omitempty,datetime=2006-01-02"`
	EmploymentStatus *string    `json:"employment_status" validate:"omitempty,oneof=active inactive resigned"`
	AttendanceMode   *string    `json:"attendance_mode" validate:"omitempty,oneof=face pin remote hybrid"`
	OfficeLocationID *uuid.UUID `json:"office_location_id" validate:"omitempty"`
}

// Update updates an existing employee.
func (s *Service) Update(ctx context.Context, id uuid.UUID, p UpdateParams) (*Employee, error) {
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	empNum := existing.EmployeeNumber
	if p.EmployeeNumber != nil && strings.TrimSpace(*p.EmployeeNumber) != "" {
		trimmed := strings.TrimSpace(*p.EmployeeNumber)
		empNum = &trimmed
	}
	fullName := existing.FullName
	if p.FullName != nil && *p.FullName != "" {
		fullName = strings.TrimSpace(*p.FullName)
	}
	department := existing.Department
	if p.Department != nil {
		department = p.Department
	}
	pos := existing.Position
	if p.Position != nil {
		pos = p.Position
	}
	phone := existing.Phone
	if p.Phone != nil {
		phone = p.Phone
	}
	email := existing.Email
	if p.Email != nil {
		email = p.Email
	}
	joinDate := existing.JoinDate
	if p.JoinDate != nil {
		joinDate = p.JoinDate
	}
	status := existing.EmploymentStatus
	if p.EmploymentStatus != nil && *p.EmploymentStatus != "" {
		status = *p.EmploymentStatus
	}
	attendanceMode := existing.AttendanceMode
	if p.AttendanceMode != nil && *p.AttendanceMode != "" {
		attendanceMode = *p.AttendanceMode
	}
	officeLocationID := existing.OfficeLocationID
	if p.OfficeLocationID != nil {
		officeLocationID = p.OfficeLocationID
	}

	// Mark the profile complete once the self-service fields are all present.
	// This lets the portal "lengkapi profil" flow clear the incomplete banner.
	profileCompleted := existing.ProfileCompleted
	if !profileCompleted && empNum != nil && pos != nil && joinDate != nil {
		profileCompleted = true
	}

	const q = `
		UPDATE employees
		SET employee_number = $1, full_name = $2, department = $3, position = $4, phone = $5, email = $6, join_date = $7::date,
		    employment_status = $8, attendance_mode = $9, office_location_id = $10,
		    profile_completed_at = CASE
		        WHEN $11 AND profile_completed_at IS NULL THEN NOW()
		        ELSE profile_completed_at
		    END,
		    updated_at = NOW()
		WHERE id = $12 AND deleted_at IS NULL
	`
	res, err := s.db.Exec(ctx, q, empNum, fullName, department, pos, phone, email, joinDate, status, attendanceMode, officeLocationID, profileCompleted, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, httpx.NewAppError(httpx.CodeEmployeeNumberTaken, "employee number is already registered")
		}
		return nil, fmt.Errorf("updating employee: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, httpx.NewAppError(httpx.CodeNotFound, "employee not found")
	}

	return s.GetByID(ctx, id)
}

// SelfProfileParams is the narrow slice of an employee record that the employee
// themselves is allowed to edit from the portal.
//
// Design note: this is deliberately a SEPARATE type from UpdateParams rather
// than reusing it. UpdateParams is admin-facing and includes fields that must
// never be self-service (employment_status, department, email, office
// location), because letting an employee change those would be a privilege
// escalation. Keeping the two types apart means a new admin field cannot
// silently become employee-editable just because it was added to UpdateParams.
type SelfProfileParams struct {
	EmployeeNumber *string `json:"employee_number" validate:"omitempty,min=2,max=50"`
	Position       *string `json:"position" validate:"omitempty,min=2,max=100"`
	JoinDate       *string `json:"join_date" validate:"omitempty,datetime=2006-01-02"`
	Phone          *string `json:"phone" validate:"omitempty,max=20"`
}

// CompleteOwnProfile lets an employee fill in the HR fields that were left
// blank at registration (real NIP, position, join date) plus their own phone.
//
// It only ever touches the four fields above. Everything else on the row is
// left exactly as the admin set it. The profile is flagged complete as soon as
// NIP, position and join date are all present, mirroring the same rule the
// admin Update path uses so both routes agree on what "complete" means.
func (s *Service) CompleteOwnProfile(ctx context.Context, id uuid.UUID, p SelfProfileParams) (*Employee, error) {
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	empNum := existing.EmployeeNumber
	if p.EmployeeNumber != nil && strings.TrimSpace(*p.EmployeeNumber) != "" {
		trimmed := strings.TrimSpace(*p.EmployeeNumber)
		empNum = &trimmed
	}
	pos := existing.Position
	if p.Position != nil && strings.TrimSpace(*p.Position) != "" {
		trimmed := strings.TrimSpace(*p.Position)
		pos = &trimmed
	}
	joinDate := existing.JoinDate
	if p.JoinDate != nil && strings.TrimSpace(*p.JoinDate) != "" {
		joinDate = p.JoinDate
	}
	phone := existing.Phone
	if p.Phone != nil && strings.TrimSpace(*p.Phone) != "" {
		trimmed := strings.TrimSpace(*p.Phone)
		phone = &trimmed
	}

	completed := empNum != nil && pos != nil && joinDate != nil

	const q = `
		UPDATE employees
		SET employee_number = $1, position = $2, join_date = $3::date, phone = $4,
		    profile_completed_at = CASE
		        WHEN $5 AND profile_completed_at IS NULL THEN NOW()
		        ELSE profile_completed_at
		    END,
		    updated_at = NOW()
		WHERE id = $6 AND deleted_at IS NULL
	`
	res, err := s.db.Exec(ctx, q, empNum, pos, joinDate, phone, completed, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, httpx.NewAppError(httpx.CodeEmployeeNumberTaken, "employee number is already registered")
		}
		return nil, fmt.Errorf("completing own profile: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, httpx.NewAppError(httpx.CodeNotFound, "employee not found")
	}

	return s.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	// Check if employee has an active user account
	const checkUserQ = `
		SELECT EXISTS (
			SELECT 1 FROM users
			WHERE employee_id = $1 AND is_active = true AND deleted_at IS NULL
		)
	`
	var hasActiveUser bool
	if err := s.db.QueryRow(ctx, checkUserQ, id).Scan(&hasActiveUser); err != nil {
		return fmt.Errorf("checking employee user status: %w", err)
	}

	if hasActiveUser {
		return httpx.NewAppError(httpx.CodeEmployeeHasActiveUser, "cannot delete employee with an active user account")
	}

	const deleteQ = `
		UPDATE employees
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	tag, err := s.db.Exec(ctx, deleteQ, id)
	if err != nil {
		return fmt.Errorf("soft-deleting employee: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return httpx.NewAppError(httpx.CodeNotFound, "employee not found")
	}

	return nil
}

// Handler provides HTTP endpoints for Employee operations.
type Handler struct {
	svc   *Service
	audit *audit.Recorder
}

// NewHandler creates a new Employee handler.
func NewHandler(svc *Service, audit *audit.Recorder) *Handler {
	return &Handler{svc: svc, audit: audit}
}

// List handles GET /api/v1/employees
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := Filter{
		Page:             1,
		PerPage:          20,
		Search:           q.Get("q"),
		Department:       q.Get("department"),
		EmploymentStatus: q.Get("employment_status"),
	}
	if filter.Search == "" {
		filter.Search = q.Get("search")
	}
	if hasUserStr := q.Get("has_user"); hasUserStr != "" {
		hasUser := hasUserStr == "true" || hasUserStr == "1"
		filter.HasUser = &hasUser
	}
	if p, err := strconv.Atoi(q.Get("page")); err == nil && p > 0 {
		filter.Page = p
	}
	if pp, err := strconv.Atoi(q.Get("per_page")); err == nil && pp > 0 {
		filter.PerPage = pp
	}
	if mode := q.Get("attendance_mode"); mode != "" {
		filter.AttendanceMode = mode
	}
	if cs := q.Get("consent_status"); cs != "" {
		filter.ConsentStatus = cs
	}
	if es := q.Get("enrollment_status"); es != "" {
		filter.EnrollmentStatus = es
	}
	if pcStr := q.Get("profile_completed"); pcStr != "" {
		pc := pcStr == "true" || pcStr == "1"
		filter.ProfileCompleted = &pc
	}

	employees, meta, err := h.svc.List(r.Context(), filter)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to list employees"))
		return
	}

	httpx.Paginated(w, employees, meta)
}

// Create handles POST /api/v1/employees
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var params CreateParams
	if err := httpx.DecodeAndValidate(r, &params); err != nil {
		httpx.Fail(r.Context(), w, err)
		return
	}

	emp, err := h.svc.Create(r.Context(), params)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to create employee"))
		return
	}

	resID := emp.ID.String()
	auditData := map[string]any{
		"full_name":  emp.FullName,
		"department": emp.Department,
	}
	if emp.EmployeeNumber != nil {
		auditData["employee_number"] = *emp.EmployeeNumber
	}
	_ = h.audit.RecordFromRequest(r, "employee.create", "employee", &resID, auditData)

	httpx.Created(w, emp)
}

// GetByID handles GET /api/v1/employees/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid employee id format"))
		return
	}

	// Verify permission: if user does not have employee.read, but has employee.read_self, verify self
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	if !p.HasPermission(rbac.PermEmployeeRead) {
		if !p.HasPermission(rbac.PermEmployeeReadSelf) || p.EmployeeID == nil || *p.EmployeeID != id {
			httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeNotFound, "employee not found"))
			return
		}
	}

	emp, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to get employee"))
		return
	}

	httpx.OK(w, emp)
}

// GetMe handles GET /api/v1/employees/me
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	if p.EmployeeID == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeNotFound, "no employee record linked to current user"))
		return
	}

	emp, err := h.svc.GetByID(r.Context(), *p.EmployeeID)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to get employee"))
		return
	}

	httpx.OK(w, emp)
}

// CompleteOwnProfile handles PATCH /api/v1/employees/me/profile
//
// Unlike Update, this resolves the target employee from the authenticated
// principal rather than from a URL parameter, so an employee can never address
// another employee's record no matter what they put in the request.
func (h *Handler) CompleteOwnProfile(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	if p.EmployeeID == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeNotFound, "no employee record linked to current user"))
		return
	}

	var params SelfProfileParams
	if err := httpx.DecodeAndValidate(r, &params); err != nil {
		httpx.Fail(r.Context(), w, err)
		return
	}

	updated, err := h.svc.CompleteOwnProfile(r.Context(), *p.EmployeeID, params)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to complete profile"))
		return
	}

	resID := updated.ID.String()
	_ = h.audit.RecordFromRequest(r, "employee.complete_own_profile", "employee", &resID, map[string]any{
		"profile_completed": updated.ProfileCompleted,
	})

	httpx.OK(w, updated)
}

// Update handles PATCH /api/v1/employees/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid employee id format"))
		return
	}

	var params UpdateParams
	if err := httpx.DecodeAndValidate(r, &params); err != nil {
		httpx.Fail(r.Context(), w, err)
		return
	}

	updated, err := h.svc.Update(r.Context(), id, params)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to update employee"))
		return
	}

	resID := updated.ID.String()
	_ = h.audit.RecordFromRequest(r, "employee.update", "employee", &resID, map[string]any{
		"full_name": updated.FullName,
	})

	httpx.OK(w, updated)
}

// Delete handles DELETE /api/v1/employees/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid employee id format"))
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to delete employee"))
		return
	}

	resID := id.String()
	_ = h.audit.RecordFromRequest(r, "employee.delete", "employee", &resID, nil)

	httpx.NoContent(w)
}
