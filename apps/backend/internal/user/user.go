package user

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/auth"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User represents a user account domain model.
type User struct {
	ID                 uuid.UUID  `json:"id"`
	Email              string     `json:"email"`
	EmployeeID         *uuid.UUID `json:"employee_id,omitempty"`
	EmployeeName       *string    `json:"employee_name,omitempty"`
	EmployeeNumber     *string    `json:"employee_number,omitempty"`
	IsActive           bool       `json:"is_active"`
	MustChangePassword bool       `json:"must_change_password"`
	FailedLoginCount   int        `json:"failed_login_count"`
	LockedUntil        *time.Time `json:"locked_until,omitempty"`
	TokenVersion       int        `json:"token_version"`
	Roles              []string   `json:"roles,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// Filter holds parameters for listing users.
type Filter struct {
	Page     int
	PerPage  int
	Search   string
	Role     string
	IsActive *bool
}

// Service provides user account operations.
type Service struct {
	db      *pgxpool.Pool
	rbacSvc *rbac.Service
}

// NewService creates a new user service.
func NewService(db *pgxpool.Pool, rbacSvc *rbac.Service) *Service {
	return &Service{db: db, rbacSvc: rbacSvc}
}

// List returns a paginated list of users.
func (s *Service) List(ctx context.Context, f Filter) ([]User, httpx.Meta, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PerPage <= 0 {
		f.PerPage = 20
	}
	if f.PerPage > 100 {
		f.PerPage = 100
	}

	whereClauses := []string{"u.deleted_at IS NULL"}
	var args []any
	argIdx := 1

	if f.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(u.email ILIKE $%d OR e.full_name ILIKE $%d OR e.employee_number ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+f.Search+"%")
		argIdx++
	}
	if f.Role != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("EXISTS (SELECT 1 FROM user_roles ur2 JOIN roles r2 ON ur2.role_id = r2.id WHERE ur2.user_id = u.id AND r2.name = $%d AND r2.deleted_at IS NULL)", argIdx))
		args = append(args, f.Role)
		argIdx++
	}
	if f.IsActive != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("u.is_active = $%d", argIdx))
		args = append(args, *f.IsActive)
		argIdx++
	}

	whereSQL := "WHERE " + strings.Join(whereClauses, " AND ")

	countSQL := fmt.Sprintf(`
		SELECT COUNT(DISTINCT u.id)
		FROM users u
		LEFT JOIN employees e ON u.employee_id = e.id
		%s
	`, whereSQL)

	var total int
	if err := s.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, httpx.Meta{}, fmt.Errorf("counting users: %w", err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + f.PerPage - 1) / f.PerPage
	}

	offset := (f.Page - 1) * f.PerPage
	querySQL := fmt.Sprintf(`
		SELECT u.id, u.email, u.employee_id, e.full_name, e.employee_number,
		       u.is_active, u.must_change_password, u.failed_login_count, u.locked_until,
		       u.token_version, u.created_at, u.updated_at,
		       COALESCE(ARRAY_AGG(r.name) FILTER (WHERE r.name IS NOT NULL), '{}') AS roles
		FROM users u
		LEFT JOIN employees e ON u.employee_id = e.id
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles r ON ur.role_id = r.id AND r.deleted_at IS NULL
		%s
		GROUP BY u.id, u.email, u.employee_id, e.full_name, e.employee_number,
		         u.is_active, u.must_change_password, u.failed_login_count, u.locked_until,
		         u.token_version, u.created_at, u.updated_at
		ORDER BY u.email ASC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, f.PerPage, offset)

	rows, err := s.db.Query(ctx, querySQL, args...)
	if err != nil {
		return nil, httpx.Meta{}, fmt.Errorf("querying users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var (
			u     User
			roles []string
		)
		if err := rows.Scan(
			&u.ID,
			&u.Email,
			&u.EmployeeID,
			&u.EmployeeName,
			&u.EmployeeNumber,
			&u.IsActive,
			&u.MustChangePassword,
			&u.FailedLoginCount,
			&u.LockedUntil,
			&u.TokenVersion,
			&u.CreatedAt,
			&u.UpdatedAt,
			&roles,
		); err != nil {
			return nil, httpx.Meta{}, fmt.Errorf("scanning user: %w", err)
		}
		u.Roles = roles
		users = append(users, u)
	}

	if users == nil {
		users = []User{}
	}

	meta := httpx.Meta{
		Page:       f.Page,
		PerPage:    f.PerPage,
		Total:      total,
		TotalPages: totalPages,
	}

	return users, meta, nil
}

// GetByID returns user details by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	const q = `
		SELECT u.id, u.email, u.employee_id, e.full_name, e.employee_number,
		       u.is_active, u.must_change_password, u.failed_login_count, u.locked_until,
		       u.token_version, u.created_at, u.updated_at,
		       COALESCE(ARRAY_AGG(r.name) FILTER (WHERE r.name IS NOT NULL), '{}') AS roles
		FROM users u
		LEFT JOIN employees e ON u.employee_id = e.id
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles r ON ur.role_id = r.id AND r.deleted_at IS NULL
		WHERE u.id = $1 AND u.deleted_at IS NULL
		GROUP BY u.id, u.email, u.employee_id, e.full_name, e.employee_number,
		         u.is_active, u.must_change_password, u.failed_login_count, u.locked_until,
		         u.token_version, u.created_at, u.updated_at
	`
	var (
		u     User
		roles []string
	)
	err := s.db.QueryRow(ctx, q, id).Scan(
		&u.ID,
		&u.Email,
		&u.EmployeeID,
		&u.EmployeeName,
		&u.EmployeeNumber,
		&u.IsActive,
		&u.MustChangePassword,
		&u.FailedLoginCount,
		&u.LockedUntil,
		&u.TokenVersion,
		&u.CreatedAt,
		&u.UpdatedAt,
		&roles,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "user not found")
		}
		return nil, fmt.Errorf("querying user: %w", err)
	}
	u.Roles = roles
	return &u, nil
}

// CreateParams input for creating a user account.
type CreateParams struct {
	Email              string      `json:"email" validate:"required,email"`
	EmployeeID         *uuid.UUID  `json:"employee_id" validate:"omitempty"`
	RoleIDs            []uuid.UUID `json:"role_ids" validate:"omitempty"`
	Password           *string     `json:"password" validate:"omitempty"`
	MustChangePassword *bool       `json:"must_change_password" validate:"omitempty"`
}

// CreateResult carries the created User and optional temporary password.
type CreateResult struct {
	User              *User  `json:"user"`
	TemporaryPassword string `json:"temporary_password,omitempty"`
}

// Create creates a new user.
func (s *Service) Create(ctx context.Context, actor *rbac.Principal, p CreateParams) (*CreateResult, error) {
	// If no role specified, default to employee role
	if len(p.RoleIDs) == 0 {
		var empRoleID uuid.UUID
		if err := s.db.QueryRow(ctx, "SELECT id FROM roles WHERE name = 'employee' AND deleted_at IS NULL").Scan(&empRoleID); err == nil {
			p.RoleIDs = []uuid.UUID{empRoleID}
		}
	}

	// Strictly enforce exactly 1 role per user
	if len(p.RoleIDs) != 1 {
		return nil, httpx.NewAppError(httpx.CodeValidationError, "setiap user wajib memiliki tepat 1 role akses")
	}

	// Verify the role exists and is an active system role
	var isSystem bool
	if err := s.db.QueryRow(ctx, "SELECT is_system FROM roles WHERE id = $1 AND deleted_at IS NULL", p.RoleIDs[0]).Scan(&isSystem); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "role tidak ditemukan")
		}
		return nil, fmt.Errorf("checking role: %w", err)
	}

	// Privilege escalation check on role IDs
	if err := s.rbacSvc.CheckPrivilegeEscalationRoleIDs(ctx, actor, p.RoleIDs); err != nil {
		return nil, err
	}

	email := strings.ToLower(strings.TrimSpace(p.Email))

	// Password handling
	var (
		plainPassword string
		mustChange    bool
	)
	if p.Password != nil && *p.Password != "" {
		plainPassword = *p.Password
		if err := auth.ValidatePassword(plainPassword, 10); err != nil {
			return nil, httpx.NewAppError(httpx.CodeValidationError, err.Error())
		}
		if p.MustChangePassword != nil {
			mustChange = *p.MustChangePassword
		}
	} else {
		var err error
		plainPassword, err = auth.GenerateTemporaryPassword()
		if err != nil {
			return nil, fmt.Errorf("generating temporary password: %w", err)
		}
		mustChange = true
	}

	hash, err := auth.HashPassword(plainPassword)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertQ = `
		INSERT INTO users (
			email, employee_id, password_hash, is_active, must_change_password, token_version
		) VALUES (
			$1, $2, $3, true, $4, 1
		)
		RETURNING id
	`
	var newUserID uuid.UUID
	err = tx.QueryRow(ctx, insertQ, email, p.EmployeeID, hash, mustChange).Scan(&newUserID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				if strings.Contains(pgErr.ConstraintName, "email") {
					return nil, httpx.NewAppError(httpx.CodeEmailTaken, "email is already registered")
				}
				if strings.Contains(pgErr.ConstraintName, "employee_id") {
					return nil, httpx.NewAppError(httpx.CodeEmployeeAlreadyHasUser, "employee already has a user account")
				}
			}
		}
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	// Assign role (strictly 1 role)
	const insRoleQ = `
		INSERT INTO user_roles (user_id, role_id)
		VALUES ($1, $2)
	`
	if _, err := tx.Exec(ctx, insRoleQ, newUserID, p.RoleIDs[0]); err != nil {
		return nil, fmt.Errorf("assigning role: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing user: %w", err)
	}

	user, err := s.GetByID(ctx, newUserID)
	if err != nil {
		return nil, err
	}

	res := &CreateResult{User: user}
	if p.Password == nil || *p.Password == "" {
		res.TemporaryPassword = plainPassword
	}

	return res, nil
}

// UpdateParams input for updating user profile.
type UpdateParams struct {
	Email      *string    `json:"email" validate:"omitempty,email"`
	EmployeeID *uuid.UUID `json:"employee_id" validate:"omitempty"`
	IsActive   *bool      `json:"is_active" validate:"omitempty"`
}

// Update updates user fields and enforces last super admin check on deactivation.
func (s *Service) Update(ctx context.Context, id uuid.UUID, p UpdateParams) (*User, error) {
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	email := existing.Email
	if p.Email != nil && *p.Email != "" {
		email = strings.ToLower(strings.TrimSpace(*p.Email))
	}
	empID := existing.EmployeeID
	if p.EmployeeID != nil {
		empID = p.EmployeeID
	}
	isActive := existing.IsActive
	if p.IsActive != nil {
		isActive = *p.IsActive
	}

	// If deactivating user, ensure not the last active super admin
	if existing.IsActive && !isActive {
		if err := s.rbacSvc.CheckLastSuperAdmin(ctx, id); err != nil {
			return nil, err
		}
	}

	const q = `
		UPDATE users
		SET email = $1, employee_id = $2, is_active = $3,
		    token_version = CASE WHEN is_active != $3 AND $3 = false THEN token_version + 1 ELSE token_version END,
		    updated_at = NOW()
		WHERE id = $4 AND deleted_at IS NULL
	`
	tag, err := s.db.Exec(ctx, q, email, empID, isActive, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "email") {
				return nil, httpx.NewAppError(httpx.CodeEmailTaken, "email is already registered")
			}
			if strings.Contains(pgErr.ConstraintName, "employee_id") {
				return nil, httpx.NewAppError(httpx.CodeEmployeeAlreadyHasUser, "employee already has a user account")
			}
		}
		return nil, fmt.Errorf("updating user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, httpx.NewAppError(httpx.CodeNotFound, "user not found")
	}

	// If deactivated, revoke all refresh tokens and clear cache
	if existing.IsActive && !isActive {
		const revokeTokensQ = `
			UPDATE refresh_tokens
			SET is_revoked = true, revoked_reason = 'user_deactivated'
			WHERE user_id = $1 AND is_revoked = false
		`
		_, _ = s.db.Exec(ctx, revokeTokensQ, id)
		s.rbacSvc.InvalidateUser(id)
	}

	return s.GetByID(ctx, id)
}

// Delete soft-deletes a user and checks last super admin guard.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.rbacSvc.CheckLastSuperAdmin(ctx, id); err != nil {
		return err
	}

	const q = `
		UPDATE users
		SET deleted_at = NOW(), is_active = false, token_version = token_version + 1, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	tag, err := s.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return httpx.NewAppError(httpx.CodeNotFound, "user not found")
	}

	const revokeTokensQ = `
		UPDATE refresh_tokens
		SET is_revoked = true, revoked_reason = 'user_deleted'
		WHERE user_id = $1 AND is_revoked = false
	`
	_, _ = s.db.Exec(ctx, revokeTokensQ, id)
	s.rbacSvc.InvalidateUser(id)

	return nil
}

// ResetPassword resets a user's password to a temporary password or provided password.
func (s *Service) ResetPassword(ctx context.Context, id uuid.UUID, customPassword *string) (string, error) {
	_, err := s.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	var plainPassword string
	if customPassword != nil && *customPassword != "" {
		plainPassword = *customPassword
		if err := auth.ValidatePassword(plainPassword, 10); err != nil {
			return "", httpx.NewAppError(httpx.CodeValidationError, err.Error())
		}
	} else {
		var genErr error
		plainPassword, genErr = auth.GenerateTemporaryPassword()
		if genErr != nil {
			return "", fmt.Errorf("generating password: %w", genErr)
		}
	}

	hash, err := auth.HashPassword(plainPassword)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}

	const q = `
		UPDATE users
		SET password_hash = $1, must_change_password = true, failed_login_count = 0,
		    locked_until = NULL, token_version = token_version + 1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`
	tag, err := s.db.Exec(ctx, q, hash, id)
	if err != nil {
		return "", fmt.Errorf("updating user password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return "", httpx.NewAppError(httpx.CodeNotFound, "user not found")
	}

	const revokeTokensQ = `
		UPDATE refresh_tokens
		SET is_revoked = true, revoked_reason = 'password_reset'
		WHERE user_id = $1 AND is_revoked = false
	`
	_, _ = s.db.Exec(ctx, revokeTokensQ, id)
	s.rbacSvc.InvalidateUser(id)

	return plainPassword, nil
}

// UpdateStatus changes user active status, invalidating tokens and checking last super admin if deactivated.
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, isActive bool) (*User, error) {
	if !isActive {
		if err := s.rbacSvc.CheckLastSuperAdmin(ctx, id); err != nil {
			return nil, err
		}
		const q = `
			UPDATE users
			SET is_active = false, token_version = token_version + 1, updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
		`
		tag, err := s.db.Exec(ctx, q, id)
		if err != nil {
			return nil, fmt.Errorf("deactivating user: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "user not found")
		}

		const revokeTokensQ = `
			UPDATE refresh_tokens
			SET is_revoked = true, revoked_reason = 'user_deactivated'
			WHERE user_id = $1 AND is_revoked = false
		`
		_, _ = s.db.Exec(ctx, revokeTokensQ, id)
		s.rbacSvc.InvalidateUser(id)
	} else {
		const q = `
			UPDATE users
			SET is_active = true, failed_login_count = 0, locked_until = NULL, updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
		`
		tag, err := s.db.Exec(ctx, q, id)
		if err != nil {
			return nil, fmt.Errorf("activating user: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "user not found")
		}
		s.rbacSvc.InvalidateUser(id)
	}

	return s.GetByID(ctx, id)
}

// AssignRoles assigns new role to user with escalation check and last super admin guard.
func (s *Service) AssignRoles(ctx context.Context, actor *rbac.Principal, targetUserID uuid.UUID, roleIDs []uuid.UUID) error {
	if len(roleIDs) != 1 {
		return httpx.NewAppError(httpx.CodeValidationError, "setiap user wajib memiliki tepat 1 role akses")
	}

	// Verify the role exists and is an active system role
	var isSystem bool
	if err := s.db.QueryRow(ctx, "SELECT is_system FROM roles WHERE id = $1 AND deleted_at IS NULL", roleIDs[0]).Scan(&isSystem); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.NewAppError(httpx.CodeNotFound, "role tidak ditemukan")
		}
		return fmt.Errorf("checking role: %w", err)
	}

	targetUser, err := s.GetByID(ctx, targetUserID)
	if err != nil {
		return err
	}

	// Privilege escalation check
	if err := s.rbacSvc.CheckPrivilegeEscalationRoleIDs(ctx, actor, roleIDs); err != nil {
		return err
	}

	// Check if target is currently super_admin and new roles remove it
	hasSuperAdminNow := false
	for _, r := range targetUser.Roles {
		if r == "super_admin" {
			hasSuperAdminNow = true
			break
		}
	}

	if hasSuperAdminNow {
		var willHaveSuperAdmin bool
		if err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM roles WHERE id = $1 AND name = 'super_admin')`, roleIDs[0]).Scan(&willHaveSuperAdmin); err != nil {
			return fmt.Errorf("checking new roles: %w", err)
		}
		if !willHaveSuperAdmin {
			if err := s.rbacSvc.CheckLastSuperAdmin(ctx, targetUserID); err != nil {
				return err
			}
		}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const delQ = `DELETE FROM user_roles WHERE user_id = $1`
	if _, err := tx.Exec(ctx, delQ, targetUserID); err != nil {
		return fmt.Errorf("clearing user roles: %w", err)
	}

	const insQ = `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`
	if _, err := tx.Exec(ctx, insQ, targetUserID, roleIDs[0]); err != nil {
		return fmt.Errorf("inserting user role: %w", err)
	}

	const bumpQ = `UPDATE users SET token_version = token_version + 1, updated_at = NOW() WHERE id = $1`
	if _, err := tx.Exec(ctx, bumpQ, targetUserID); err != nil {
		return fmt.Errorf("bumping user token version: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing roles: %w", err)
	}

	s.rbacSvc.InvalidateUser(targetUserID)
	return nil
}

// Handler handles HTTP requests for User accounts.
type Handler struct {
	svc   *Service
	audit *audit.Recorder
}

// NewHandler creates a new user handler.
func NewHandler(svc *Service, audit *audit.Recorder) *Handler {
	return &Handler{svc: svc, audit: audit}
}

// List handles GET /api/v1/users
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := Filter{
		Page:    1,
		PerPage: 20,
		Search:  q.Get("search"),
		Role:    q.Get("role"),
	}
	if p, err := strconv.Atoi(q.Get("page")); err == nil && p > 0 {
		filter.Page = p
	}
	if pp, err := strconv.Atoi(q.Get("per_page")); err == nil && pp > 0 {
		filter.PerPage = pp
	}
	if activeStr := q.Get("is_active"); activeStr != "" {
		activeBool := strings.EqualFold(activeStr, "true") || activeStr == "1"
		filter.IsActive = &activeBool
	}

	users, meta, err := h.svc.List(r.Context(), filter)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to list users"))
		return
	}

	httpx.Paginated(w, users, meta)
}

// Create handles POST /api/v1/users
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var params CreateParams
	if err := httpx.DecodeAndValidate(r, &params); err != nil {
		httpx.Fail(r.Context(), w, err)
		return
	}

	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	res, err := h.svc.Create(r.Context(), p, params)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to create user"))
		return
	}

	resID := res.User.ID.String()
	_ = h.audit.RecordFromRequest(r, "user.create", "user", &resID, map[string]any{
		"email": res.User.Email,
		"roles": res.User.Roles,
	})

	httpx.Created(w, res)
}

// GetByID handles GET /api/v1/users/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid user id format"))
		return
	}

	user, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to get user"))
		return
	}

	httpx.OK(w, user)
}

// Update handles PATCH /api/v1/users/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid user id format"))
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
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to update user"))
		return
	}

	resID := updated.ID.String()
	_ = h.audit.RecordFromRequest(r, "user.update", "user", &resID, map[string]any{
		"email":     updated.Email,
		"is_active": updated.IsActive,
	})

	httpx.OK(w, updated)
}

// Delete handles DELETE /api/v1/users/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid user id format"))
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to delete user"))
		return
	}

	resID := id.String()
	_ = h.audit.RecordFromRequest(r, "user.delete", "user", &resID, nil)

	httpx.NoContent(w)
}

// ResetPasswordRequest payload for password reset.
type ResetPasswordRequest struct {
	Password *string `json:"password" validate:"omitempty"`
}

// ResetPassword handles POST /api/v1/users/{id}/reset-password
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid user id format"))
		return
	}

	var req ResetPasswordRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.Fail(r.Context(), w, err)
		return
	}

	newPwd, err := h.svc.ResetPassword(r.Context(), id, req.Password)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to reset user password"))
		return
	}

	resID := id.String()
	_ = h.audit.RecordFromRequest(r, "user.reset_password", "user", &resID, nil)

	httpx.OK(w, map[string]string{
		"temporary_password": newPwd,
	})
}

// UpdateStatusRequest payload for updating user active status.
type UpdateStatusRequest struct {
	IsActive bool `json:"is_active"`
}

// UpdateStatus handles PATCH /api/v1/users/{id}/status
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid user id format"))
		return
	}

	var req UpdateStatusRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.Fail(r.Context(), w, err)
		return
	}

	u, err := h.svc.UpdateStatus(r.Context(), id, req.IsActive)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to update user status"))
		return
	}

	resID := id.String()
	_ = h.audit.RecordFromRequest(r, "user.status.changed", "user", &resID, map[string]interface{}{
		"is_active": req.IsActive,
	})

	httpx.OK(w, u)
}

// AssignRolesRequest payload for assigning roles.
type AssignRolesRequest struct {
	RoleIDs []uuid.UUID `json:"role_ids" validate:"required,min=1,max=1"`
}

// AssignRoles handles PUT /api/v1/users/{id}/roles
func (h *Handler) AssignRoles(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid user id format"))
		return
	}

	var req AssignRolesRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.Fail(r.Context(), w, err)
		return
	}

	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	if err := h.svc.AssignRoles(r.Context(), p, id, req.RoleIDs); err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to assign roles"))
		return
	}

	resID := id.String()
	_ = h.audit.RecordFromRequest(r, "user.assign_roles", "user", &resID, map[string]any{
		"role_ids": req.RoleIDs,
	})

	user, _ := s_getByID(h.svc, r.Context(), id)
	httpx.OK(w, user)
}

func s_getByID(svc *Service, ctx context.Context, id uuid.UUID) (*User, error) {
	return svc.GetByID(ctx, id)
}
