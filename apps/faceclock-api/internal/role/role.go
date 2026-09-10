package role

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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

// Permission represents a system permission item.
type Permission struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Role represents a user role.
type Role struct {
	ID              uuid.UUID    `json:"id"`
	Name            string       `json:"name"`
	DisplayName     string       `json:"display_name"`
	Description     *string      `json:"description,omitempty"`
	IsSystem        bool         `json:"is_system"`
	UserCount       int          `json:"user_count,omitempty"`
	PermissionCount int          `json:"permission_count,omitempty"`
	Permissions     []Permission `json:"permissions,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

// Service handles role and permission operations.
type Service struct {
	db      *pgxpool.Pool
	rbacSvc *rbac.Service
}

// NewService creates a new role service.
func NewService(db *pgxpool.Pool, rbacSvc *rbac.Service) *Service {
	return &Service{db: db, rbacSvc: rbacSvc}
}

// ListRoles returns all active roles with counts.
func (s *Service) ListRoles(ctx context.Context) ([]Role, error) {
	const q = `
		SELECT r.id, r.name, r.display_name, r.description, r.is_system, r.created_at, r.updated_at,
		       COUNT(DISTINCT ur.user_id) AS user_count,
		       COUNT(DISTINCT rp.permission_id) AS permission_count
		FROM roles r
		LEFT JOIN user_roles ur ON r.id = ur.role_id
		LEFT JOIN users u ON ur.user_id = u.id AND u.deleted_at IS NULL
		LEFT JOIN role_permissions rp ON r.id = rp.role_id
		WHERE r.deleted_at IS NULL
		GROUP BY r.id, r.name, r.display_name, r.description, r.is_system, r.created_at, r.updated_at
		ORDER BY r.is_system DESC, r.name ASC
	`
	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("querying roles: %w", err)
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var r Role
		if err := rows.Scan(
			&r.ID,
			&r.Name,
			&r.DisplayName,
			&r.Description,
			&r.IsSystem,
			&r.CreatedAt,
			&r.UpdatedAt,
			&r.UserCount,
			&r.PermissionCount,
		); err != nil {
			return nil, fmt.Errorf("scanning role: %w", err)
		}
		roles = append(roles, r)
	}

	if roles == nil {
		roles = []Role{}
	}

	return roles, nil
}

// GetRoleByID fetches a role and its assigned permissions.
func (s *Service) GetRoleByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	const q = `
		SELECT id, name, display_name, description, is_system, created_at, updated_at
		FROM roles
		WHERE id = $1 AND deleted_at IS NULL
	`
	var r Role
	err := s.db.QueryRow(ctx, q, id).Scan(
		&r.ID,
		&r.Name,
		&r.DisplayName,
		&r.Description,
		&r.IsSystem,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "role not found")
		}
		return nil, fmt.Errorf("querying role: %w", err)
	}

	const permQ = `
		SELECT p.id, p.name, p.resource, p.action, p.description, p.created_at
		FROM role_permissions rp
		JOIN permissions p ON rp.permission_id = p.id
		WHERE rp.role_id = $1
		ORDER BY p.name ASC
	`
	pRows, err := s.db.Query(ctx, permQ, id)
	if err != nil {
		return nil, fmt.Errorf("querying role permissions: %w", err)
	}
	defer pRows.Close()

	r.Permissions = []Permission{}
	for pRows.Next() {
		var p Permission
		if err := pRows.Scan(&p.ID, &p.Name, &p.Resource, &p.Action, &p.Description, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning permission: %w", err)
		}
		r.Permissions = append(r.Permissions, p)
	}

	return &r, nil
}

// CreateRoleParams input for creating a role.
type CreateRoleParams struct {
	Name          string      `json:"name" validate:"required,min=2,max=50"`
	DisplayName   string      `json:"display_name" validate:"required,min=2,max=100"`
	Description   *string     `json:"description" validate:"omitempty,max=255"`
	PermissionIDs []uuid.UUID `json:"permission_ids" validate:"omitempty"`
}

// CreateRole creates a new custom role.
func (s *Service) CreateRole(ctx context.Context, actor *rbac.Principal, p CreateRoleParams) (*Role, error) {
	// Check privilege escalation on permission IDs
	if len(p.PermissionIDs) > 0 && !actor.IsSuperAdmin() {
		names, err := s.getPermissionNamesByIDs(ctx, p.PermissionIDs)
		if err != nil {
			return nil, err
		}
		if err := s.rbacSvc.CheckPrivilegeEscalation(actor, names); err != nil {
			return nil, err
		}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertQ = `
		INSERT INTO roles (name, display_name, description, is_system)
		VALUES ($1, $2, $3, false)
		RETURNING id, name, display_name, description, is_system, created_at, updated_at
	`
	var role Role
	err = tx.QueryRow(ctx, insertQ, strings.TrimSpace(p.Name), strings.TrimSpace(p.DisplayName), p.Description).Scan(
		&role.ID,
		&role.Name,
		&role.DisplayName,
		&role.Description,
		&role.IsSystem,
		&role.CreatedAt,
		&role.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, httpx.NewAppError(httpx.CodeRoleNameTaken, "role name already exists")
		}
		return nil, fmt.Errorf("inserting role: %w", err)
	}

	if len(p.PermissionIDs) > 0 {
		for _, permID := range p.PermissionIDs {
			const insPermQ = `
				INSERT INTO role_permissions (role_id, permission_id)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING
			`
			if _, err := tx.Exec(ctx, insPermQ, role.ID, permID); err != nil {
				return nil, fmt.Errorf("assigning permission %s: %w", permID, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing role: %w", err)
	}

	s.rbacSvc.InvalidateAll()
	return s.GetRoleByID(ctx, role.ID)
}

// UpdateRoleParams input for updating a role.
type UpdateRoleParams struct {
	DisplayName *string `json:"display_name" validate:"omitempty,min=2,max=100"`
	Description *string `json:"description" validate:"omitempty,max=255"`
}

// UpdateRole updates role name or description.
func (s *Service) UpdateRole(ctx context.Context, id uuid.UUID, p UpdateRoleParams) (*Role, error) {
	existing, err := s.GetRoleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	displayName := existing.DisplayName
	if p.DisplayName != nil && *p.DisplayName != "" {
		displayName = strings.TrimSpace(*p.DisplayName)
	}
	desc := existing.Description
	if p.Description != nil {
		desc = p.Description
	}

	const q = `
		UPDATE roles
		SET display_name = $1, description = $2, updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING id, name, display_name, description, is_system, created_at, updated_at
	`
	var updated Role
	err = s.db.QueryRow(ctx, q, displayName, desc, id).Scan(
		&updated.ID,
		&updated.Name,
		&updated.DisplayName,
		&updated.Description,
		&updated.IsSystem,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("updating role: %w", err)
	}

	s.rbacSvc.InvalidateAll()
	return s.GetRoleByID(ctx, updated.ID)
}

// DeleteRole soft-deletes a role.
func (s *Service) DeleteRole(ctx context.Context, id uuid.UUID) error {
	existing, err := s.GetRoleByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.IsSystem {
		return httpx.NewAppError(httpx.CodeSystemRoleImmutable, "cannot delete a system role")
	}

	// Check if role in use
	const checkInUseQ = `
		SELECT COUNT(*)
		FROM user_roles ur
		JOIN users u ON ur.user_id = u.id
		WHERE ur.role_id = $1 AND u.deleted_at IS NULL
	`
	var inUseCount int
	if err := s.db.QueryRow(ctx, checkInUseQ, id).Scan(&inUseCount); err != nil {
		return fmt.Errorf("checking role usage: %w", err)
	}
	if inUseCount > 0 {
		return httpx.NewAppError(httpx.CodeRoleInUse, fmt.Sprintf("role is currently assigned to %d active users", inUseCount))
	}

	const deleteQ = `
		UPDATE roles
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	tag, err := s.db.Exec(ctx, deleteQ, id)
	if err != nil {
		return fmt.Errorf("deleting role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return httpx.NewAppError(httpx.CodeNotFound, "role not found")
	}

	s.rbacSvc.InvalidateAll()
	return nil
}

// AssignPermissions assigns a list of permissions to a role.
func (s *Service) AssignPermissions(ctx context.Context, actor *rbac.Principal, roleID uuid.UUID, permIDs []uuid.UUID) error {
	role, err := s.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}

	// If modifying super_admin role permissions, only super_admin can do it
	if role.Name == "super_admin" && !actor.IsSuperAdmin() {
		return httpx.NewAppError(httpx.CodeSystemRoleImmutable, "cannot modify permissions of the super_admin role")
	}

	// Check privilege escalation
	if len(permIDs) > 0 && !actor.IsSuperAdmin() {
		names, err := s.getPermissionNamesByIDs(ctx, permIDs)
		if err != nil {
			return err
		}
		if err := s.rbacSvc.CheckPrivilegeEscalation(actor, names); err != nil {
			return err
		}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Remove existing permissions
	const delQ = `DELETE FROM role_permissions WHERE role_id = $1`
	if _, err := tx.Exec(ctx, delQ, roleID); err != nil {
		return fmt.Errorf("clearing role permissions: %w", err)
	}

	// Insert new permissions
	for _, pid := range permIDs {
		const insQ = `
			INSERT INTO role_permissions (role_id, permission_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`
		if _, err := tx.Exec(ctx, insQ, roleID, pid); err != nil {
			return fmt.Errorf("assigning permission %s: %w", pid, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing role permissions: %w", err)
	}

	s.rbacSvc.InvalidateAll()
	return nil
}

// ListPermissions returns all 34 catalog permissions.
func (s *Service) ListPermissions(ctx context.Context) ([]Permission, error) {
	const q = `
		SELECT id, name, resource, action, description, created_at
		FROM permissions
		ORDER BY resource ASC, name ASC
	`
	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("querying permissions: %w", err)
	}
	defer rows.Close()

	var perms []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Resource, &p.Action, &p.Description, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning permission: %w", err)
		}
		perms = append(perms, p)
	}

	if perms == nil {
		perms = []Permission{}
	}

	return perms, nil
}

func (s *Service) getPermissionNamesByIDs(ctx context.Context, ids []uuid.UUID) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	const q = `SELECT name FROM permissions WHERE id = ANY($1)`
	rows, err := s.db.Query(ctx, q, ids)
	if err != nil {
		return nil, fmt.Errorf("querying permission names: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, fmt.Errorf("scanning permission name: %w", err)
		}
		names = append(names, n)
	}

	return names, nil
}

// Handler handles HTTP requests for roles and permissions.
type Handler struct {
	svc   *Service
	audit *audit.Recorder
}

// NewHandler creates a new role handler.
func NewHandler(svc *Service, audit *audit.Recorder) *Handler {
	return &Handler{svc: svc, audit: audit}
}

// ListRoles handles GET /api/v1/roles
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.ListRoles(r.Context())
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to list roles"))
		return
	}
	httpx.OK(w, roles)
}

// GetRoleByID handles GET /api/v1/roles/{id}
func (h *Handler) GetRoleByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid role id format"))
		return
	}

	role, err := h.svc.GetRoleByID(r.Context(), id)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to get role"))
		return
	}

	httpx.OK(w, role)
}

// CreateRole handles POST /api/v1/roles
func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var params CreateRoleParams
	if err := httpx.DecodeAndValidate(r, &params); err != nil {
		httpx.Fail(r.Context(), w, err)
		return
	}

	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	role, err := h.svc.CreateRole(r.Context(), p, params)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to create role"))
		return
	}

	resID := role.ID.String()
	_ = h.audit.RecordFromRequest(r, "role.create", "role", &resID, map[string]any{
		"name": role.Name,
	})

	httpx.Created(w, role)
}

// UpdateRole handles PATCH /api/v1/roles/{id}
func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid role id format"))
		return
	}

	var params UpdateRoleParams
	if err := httpx.DecodeAndValidate(r, &params); err != nil {
		httpx.Fail(r.Context(), w, err)
		return
	}

	role, err := h.svc.UpdateRole(r.Context(), id, params)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to update role"))
		return
	}

	resID := role.ID.String()
	_ = h.audit.RecordFromRequest(r, "role.update", "role", &resID, map[string]any{
		"name": role.Name,
	})

	httpx.OK(w, role)
}

// DeleteRole handles DELETE /api/v1/roles/{id}
func (h *Handler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid role id format"))
		return
	}

	if err := h.svc.DeleteRole(r.Context(), id); err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to delete role"))
		return
	}

	resID := id.String()
	_ = h.audit.RecordFromRequest(r, "role.delete", "role", &resID, nil)

	httpx.NoContent(w)
}

// AssignPermissionsRequest payload for assigning permissions.
type AssignPermissionsRequest struct {
	PermissionIDs []uuid.UUID `json:"permission_ids" validate:"required"`
}

// AssignPermissions handles PUT /api/v1/roles/{id}/permissions
func (h *Handler) AssignPermissions(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid role id format"))
		return
	}

	var req AssignPermissionsRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.Fail(r.Context(), w, err)
		return
	}

	p, ok := rbac.GetPrincipal(r.Context())
	if !ok || p == nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required"))
		return
	}

	if err := h.svc.AssignPermissions(r.Context(), p, id, req.PermissionIDs); err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to assign permissions"))
		return
	}

	resID := id.String()
	_ = h.audit.RecordFromRequest(r, "role.assign_permissions", "role", &resID, map[string]any{
		"permission_ids": req.PermissionIDs,
	})

	role, _ := h.svc.GetRoleByID(r.Context(), id)
	httpx.OK(w, role)
}

// ListPermissions handles GET /api/v1/permissions
func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := h.svc.ListPermissions(r.Context())
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to list permissions"))
		return
	}

	if r.URL.Query().Get("group_by") == "resource" {
		grouped := make(map[string][]Permission)
		for _, p := range perms {
			grouped[p.Resource] = append(grouped[p.Resource], p)
		}
		httpx.OK(w, grouped)
		return
	}

	httpx.OK(w, perms)
}
