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
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Employee represents an employee domain model according to Fase 1 § 3.1 & § 4.2.
type Employee struct {
	ID               uuid.UUID  `json:"id"`
	EmployeeNumber   string     `json:"employee_number"`
	FullName         string     `json:"full_name"`
	Department       *string    `json:"department"`
	Position         *string    `json:"position"`
	Phone            *string    `json:"phone"`
	Email            *string    `json:"email"`
	JoinDate         *string    `json:"join_date"` // YYYY-MM-DD
	EmploymentStatus string     `json:"employment_status"`
	HasUserAccount   bool       `json:"has_user_account"`
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
}

// Service provides employee operations.
type Service struct {
	db *pgxpool.Pool
}

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
			e.created_at,
			e.updated_at
		FROM employees e
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
	const q = `
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
			e.created_at,
			e.updated_at
		FROM employees e
		WHERE e.id = $1 AND e.deleted_at IS NULL
	`
	var e Employee
	err := s.db.QueryRow(ctx, q, id).Scan(
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
type CreateParams struct {
	EmployeeNumber   string  `json:"employee_number" validate:"required,max=50"`
	FullName         string  `json:"full_name" validate:"required,min=2,max=120"`
	Department       *string `json:"department" validate:"omitempty,max=100"`
	Position         *string `json:"position" validate:"omitempty,max=100"`
	Phone            *string `json:"phone" validate:"omitempty,max=20"`
	Email            *string `json:"email" validate:"omitempty,email"`
	JoinDate         *string `json:"join_date" validate:"omitempty,datetime=2006-01-02"`
	EmploymentStatus string  `json:"employment_status" validate:"omitempty,oneof=active inactive resigned"`
}

// Create inserts a new employee.
func (s *Service) Create(ctx context.Context, p CreateParams) (*Employee, error) {
	status := p.EmploymentStatus
	if status == "" {
		status = "active"
	}

	const q = `
		INSERT INTO employees (
			employee_number, full_name, department, position, phone, email, join_date, employment_status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7::date, $8
		)
		RETURNING
			id, employee_number, full_name, department, position, phone, email, join_date::text, employment_status,
			false AS has_user_account, created_at, updated_at
	`
	var e Employee
	err := s.db.QueryRow(ctx, q,
		strings.TrimSpace(p.EmployeeNumber),
		strings.TrimSpace(p.FullName),
		p.Department,
		p.Position,
		p.Phone,
		p.Email,
		p.JoinDate,
		status,
	).Scan(
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
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, httpx.NewAppError(httpx.CodeEmployeeNumberTaken, "employee number is already registered")
		}
		return nil, fmt.Errorf("inserting employee: %w", err)
	}

	return &e, nil
}

// UpdateParams input for updating an employee.
type UpdateParams struct {
	EmployeeNumber   *string `json:"employee_number" validate:"omitempty,max=50"`
	FullName         *string `json:"full_name" validate:"omitempty,min=2,max=120"`
	Department       *string `json:"department" validate:"omitempty,max=100"`
	Position         *string `json:"position" validate:"omitempty,max=100"`
	Phone            *string `json:"phone" validate:"omitempty,max=20"`
	Email            *string `json:"email" validate:"omitempty,email"`
	JoinDate         *string `json:"join_date" validate:"omitempty,datetime=2006-01-02"`
	EmploymentStatus *string `json:"employment_status" validate:"omitempty,oneof=active inactive resigned"`
}

// Update updates an existing employee.
func (s *Service) Update(ctx context.Context, id uuid.UUID, p UpdateParams) (*Employee, error) {
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	empNum := existing.EmployeeNumber
	if p.EmployeeNumber != nil && *p.EmployeeNumber != "" {
		empNum = strings.TrimSpace(*p.EmployeeNumber)
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

	const q = `
		UPDATE employees
		SET employee_number = $1, full_name = $2, department = $3, position = $4, phone = $5, email = $6, join_date = $7::date,
		    employment_status = $8, updated_at = NOW()
		WHERE id = $9 AND deleted_at IS NULL
		RETURNING
			id, employee_number, full_name, department, position, phone, email, join_date::text, employment_status,
			EXISTS (SELECT 1 FROM users u WHERE u.employee_id = $9 AND u.deleted_at IS NULL) AS has_user_account,
			created_at, updated_at
	`
	var updated Employee
	err = s.db.QueryRow(ctx, q, empNum, fullName, department, pos, phone, email, joinDate, status, id).Scan(
		&updated.ID,
		&updated.EmployeeNumber,
		&updated.FullName,
		&updated.Department,
		&updated.Position,
		&updated.Phone,
		&updated.Email,
		&updated.JoinDate,
		&updated.EmploymentStatus,
		&updated.HasUserAccount,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, httpx.NewAppError(httpx.CodeEmployeeNumberTaken, "employee number is already registered")
		}
		return nil, fmt.Errorf("updating employee: %w", err)
	}

	return &updated, nil
}

// Delete soft-deletes an employee if they do not have an active user account.
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
	_ = h.audit.RecordFromRequest(r, "employee.create", "employee", &resID, map[string]any{
		"employee_number": emp.EmployeeNumber,
		"full_name":       emp.FullName,
		"department":      emp.Department,
	})

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
