package role

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Module represents a system module entity.
type Module struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RoleModuleSummary provides aggregated module coverage for a role.
type RoleModuleSummary struct {
	ID                uuid.UUID `json:"id"`
	Code              string    `json:"code"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	Icon              string    `json:"icon"`
	SortOrder         int       `json:"sort_order"`
	TotalPermissions  int       `json:"total_permissions"`
	ActivePermissions int       `json:"active_permissions"`
}

// ModuleWithPermissions represents a module and all its permissions in the system catalog.
type ModuleWithPermissions struct {
	ID          uuid.UUID    `json:"id"`
	Code        string       `json:"code"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Icon        string       `json:"icon"`
	SortOrder   int          `json:"sort_order"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Permissions []Permission `json:"permissions"`
}

// Permission represents a system permission item.
type Permission struct {
	ID          uuid.UUID  `json:"id"`
	ModuleID    *uuid.UUID `json:"module_id,omitempty"`
	ModuleCode  string     `json:"module_code,omitempty"`
	ModuleName  string     `json:"module_name,omitempty"`
	Name        string     `json:"name"`
	Resource    string     `json:"resource"`
	Action      string     `json:"action"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Role represents a user role.
type Role struct {
	ID              uuid.UUID           `json:"id"`
	Name            string              `json:"name"`
	DisplayName     string              `json:"display_name"`
	Description     *string             `json:"description,omitempty"`
	IsSystem        bool                `json:"is_system"`
	UserCount       int                 `json:"user_count,omitempty"`
	PermissionCount int                 `json:"permission_count,omitempty"`
	ModuleCount     int                 `json:"module_count,omitempty"`
	TotalModules    int                 `json:"total_modules,omitempty"`
	Modules         []RoleModuleSummary `json:"modules,omitempty"`
	Permissions     []Permission        `json:"permissions,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
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

// ListRoles returns all active roles with counts, dynamic active module summaries, and permissions.
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
		ORDER BY CASE r.name
			WHEN 'super_admin' THEN 1
			WHEN 'admin' THEN 2
			WHEN 'employee' THEN 3
			ELSE 4
		END
	`
	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("querying roles: %w", err)
	}
	defer rows.Close()

	var roles []Role
	roleIndexMap := make(map[uuid.UUID]int)
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
		r.Modules = []RoleModuleSummary{}
		r.Permissions = []Permission{}
		roleIndexMap[r.ID] = len(roles)
		roles = append(roles, r)
	}

	if len(roles) == 0 {
		return []Role{}, nil
	}

	// Query module summaries for each role
	const moduleQ = `
		SELECT 
			r.id AS role_id,
			m.id AS module_id,
			m.code AS module_code,
			m.name AS module_name,
			m.description AS module_desc,
			m.icon AS module_icon,
			m.sort_order AS module_sort_order,
			COUNT(p.id) AS total_permissions,
			COUNT(rp.permission_id) AS active_permissions
		FROM roles r
		CROSS JOIN modules m
		JOIN permissions p ON p.module_id = m.id
		LEFT JOIN role_permissions rp ON rp.role_id = r.id AND rp.permission_id = p.id
		WHERE r.deleted_at IS NULL
		GROUP BY r.id, m.id, m.code, m.name, m.description, m.icon, m.sort_order
		ORDER BY r.id, m.sort_order ASC
	`
	mRows, err := s.db.Query(ctx, moduleQ)
	if err != nil {
		return nil, fmt.Errorf("querying role modules: %w", err)
	}
	defer mRows.Close()

	for mRows.Next() {
		var roleID uuid.UUID
		var mod RoleModuleSummary
		if err := mRows.Scan(
			&roleID,
			&mod.ID,
			&mod.Code,
			&mod.Name,
			&mod.Description,
			&mod.Icon,
			&mod.SortOrder,
			&mod.TotalPermissions,
			&mod.ActivePermissions,
		); err != nil {
			return nil, fmt.Errorf("scanning role module: %w", err)
		}
		if idx, ok := roleIndexMap[roleID]; ok {
			roles[idx].TotalModules++
			if mod.ActivePermissions > 0 {
				roles[idx].Modules = append(roles[idx].Modules, mod)
				roles[idx].ModuleCount++
			}
		}
	}

	// Query active permissions for each role
	const permQ = `
		SELECT 
			rp.role_id,
			p.id,
			p.module_id,
			m.code AS module_code,
			m.name AS module_name,
			p.name,
			p.resource,
			p.action,
			p.description,
			p.created_at
		FROM role_permissions rp
		JOIN permissions p ON rp.permission_id = p.id
		JOIN modules m ON p.module_id = m.id
		ORDER BY rp.role_id, m.sort_order ASC, p.name ASC
	`
	pRows, err := s.db.Query(ctx, permQ)
	if err != nil {
		return nil, fmt.Errorf("querying role permissions: %w", err)
	}
	defer pRows.Close()

	for pRows.Next() {
		var roleID uuid.UUID
		var p Permission
		if err := pRows.Scan(
			&roleID,
			&p.ID,
			&p.ModuleID,
			&p.ModuleCode,
			&p.ModuleName,
			&p.Name,
			&p.Resource,
			&p.Action,
			&p.Description,
			&p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning role permission: %w", err)
		}
		if idx, ok := roleIndexMap[roleID]; ok {
			roles[idx].Permissions = append(roles[idx].Permissions, p)
		}
	}

	return roles, nil
}

// GetRoleByID fetches a role and its assigned permissions along with module summaries.
func (s *Service) GetRoleByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	const q = `
		SELECT r.id, r.name, r.display_name, r.description, r.is_system, r.created_at, r.updated_at,
		       COUNT(DISTINCT ur.user_id) AS user_count,
		       COUNT(DISTINCT rp.permission_id) AS permission_count
		FROM roles r
		LEFT JOIN user_roles ur ON r.id = ur.role_id
		LEFT JOIN users u ON ur.user_id = u.id AND u.deleted_at IS NULL
		LEFT JOIN role_permissions rp ON r.id = rp.role_id
		WHERE r.id = $1 AND r.deleted_at IS NULL
		GROUP BY r.id, r.name, r.display_name, r.description, r.is_system, r.created_at, r.updated_at
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
		&r.UserCount,
		&r.PermissionCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "role not found")
		}
		return nil, fmt.Errorf("querying role: %w", err)
	}

	r.Modules = []RoleModuleSummary{}
	r.Permissions = []Permission{}

	// Query module summaries for this role
	const moduleQ = `
		SELECT 
			m.id AS module_id,
			m.code AS module_code,
			m.name AS module_name,
			m.description AS module_desc,
			m.icon AS module_icon,
			m.sort_order AS module_sort_order,
			COUNT(p.id) AS total_permissions,
			COUNT(rp.permission_id) AS active_permissions
		FROM modules m
		JOIN permissions p ON p.module_id = m.id
		LEFT JOIN role_permissions rp ON rp.role_id = $1 AND rp.permission_id = p.id
		GROUP BY m.id, m.code, m.name, m.description, m.icon, m.sort_order
		ORDER BY m.sort_order ASC
	`
	mRows, err := s.db.Query(ctx, moduleQ, id)
	if err != nil {
		return nil, fmt.Errorf("querying role modules: %w", err)
	}
	defer mRows.Close()

	for mRows.Next() {
		var mod RoleModuleSummary
		if err := mRows.Scan(
			&mod.ID,
			&mod.Code,
			&mod.Name,
			&mod.Description,
			&mod.Icon,
			&mod.SortOrder,
			&mod.TotalPermissions,
			&mod.ActivePermissions,
		); err != nil {
			return nil, fmt.Errorf("scanning role module: %w", err)
		}
		r.TotalModules++
		if mod.ActivePermissions > 0 {
			r.ModuleCount++
		}
		r.Modules = append(r.Modules, mod)
	}

	const permQ = `
		SELECT p.id, p.module_id, m.code AS module_code, m.name AS module_name,
		       p.name, p.resource, p.action, p.description, p.created_at
		FROM role_permissions rp
		JOIN permissions p ON rp.permission_id = p.id
		JOIN modules m ON p.module_id = m.id
		WHERE rp.role_id = $1
		ORDER BY m.sort_order ASC, p.name ASC
	`
	pRows, err := s.db.Query(ctx, permQ, id)
	if err != nil {
		return nil, fmt.Errorf("querying role permissions: %w", err)
	}
	defer pRows.Close()

	for pRows.Next() {
		var p Permission
		if err := pRows.Scan(
			&p.ID,
			&p.ModuleID,
			&p.ModuleCode,
			&p.ModuleName,
			&p.Name,
			&p.Resource,
			&p.Action,
			&p.Description,
			&p.CreatedAt,
		); err != nil {
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

// CreateRole creates a new custom role. Custom roles are not permitted.
func (s *Service) CreateRole(ctx context.Context, actor *rbac.Principal, p CreateRoleParams) (*Role, error) {
	return nil, httpx.NewAppError(httpx.CodeForbidden, "pembuatan role kustom tidak diizinkan; hanya role sistem yang tersedia")
}

// UpdateRoleParams input for updating a role.
type UpdateRoleParams struct {
	DisplayName *string `json:"display_name" validate:"omitempty,min=2,max=100"`
	Description *string `json:"description" validate:"omitempty,max=255"`
}

// UpdateRole updates role name or description. System roles cannot be modified.
func (s *Service) UpdateRole(ctx context.Context, id uuid.UUID, p UpdateRoleParams) (*Role, error) {
	return nil, httpx.NewAppError(httpx.CodeForbidden, "role sistem bersifat permanen dan tidak dapat diubah")
}

// DeleteRole soft-deletes a role. System roles cannot be deleted.
func (s *Service) DeleteRole(ctx context.Context, id uuid.UUID) error {
	return httpx.NewAppError(httpx.CodeForbidden, "role sistem tidak dapat dihapus")
}

// AssignPermissions assigns a list of permissions to a role. System role permissions are fixed.
func (s *Service) AssignPermissions(ctx context.Context, actor *rbac.Principal, roleID uuid.UUID, permIDs []uuid.UUID) error {
	return httpx.NewAppError(httpx.CodeForbidden, "hak akses role sistem bersifat permanen dan tidak dapat diubah")
}

// ListPermissions returns all catalog permissions linked with module information.
func (s *Service) ListPermissions(ctx context.Context) ([]Permission, error) {
	const q = `
		SELECT p.id, p.module_id, m.code AS module_code, m.name AS module_name,
		       p.name, p.resource, p.action, p.description, p.created_at
		FROM permissions p
		JOIN modules m ON p.module_id = m.id
		ORDER BY m.sort_order ASC, p.name ASC
	`
	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("querying permissions: %w", err)
	}
	defer rows.Close()

	var perms []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(
			&p.ID,
			&p.ModuleID,
			&p.ModuleCode,
			&p.ModuleName,
			&p.Name,
			&p.Resource,
			&p.Action,
			&p.Description,
			&p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning permission: %w", err)
		}
		perms = append(perms, p)
	}

	if perms == nil {
		perms = []Permission{}
	}

	return perms, nil
}

// ListModules returns all modules with their associated permissions.
func (s *Service) ListModules(ctx context.Context) ([]ModuleWithPermissions, error) {
	const modQ = `
		SELECT id, code, name, description, icon, sort_order, created_at, updated_at
		FROM modules
		ORDER BY sort_order ASC
	`
	mRows, err := s.db.Query(ctx, modQ)
	if err != nil {
		return nil, fmt.Errorf("querying modules: %w", err)
	}
	defer mRows.Close()

	var modules []ModuleWithPermissions
	modIndexMap := make(map[uuid.UUID]int)
	for mRows.Next() {
		var m ModuleWithPermissions
		if err := mRows.Scan(
			&m.ID,
			&m.Code,
			&m.Name,
			&m.Description,
			&m.Icon,
			&m.SortOrder,
			&m.CreatedAt,
			&m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning module: %w", err)
		}
		m.Permissions = []Permission{}
		modIndexMap[m.ID] = len(modules)
		modules = append(modules, m)
	}

	if len(modules) == 0 {
		return []ModuleWithPermissions{}, nil
	}

	const permQ = `
		SELECT p.id, p.module_id, m.code AS module_code, m.name AS module_name,
		       p.name, p.resource, p.action, p.description, p.created_at
		FROM permissions p
		JOIN modules m ON p.module_id = m.id
		ORDER BY m.sort_order ASC, p.name ASC
	`
	pRows, err := s.db.Query(ctx, permQ)
	if err != nil {
		return nil, fmt.Errorf("querying permissions for modules: %w", err)
	}
	defer pRows.Close()

	for pRows.Next() {
		var p Permission
		if err := pRows.Scan(
			&p.ID,
			&p.ModuleID,
			&p.ModuleCode,
			&p.ModuleName,
			&p.Name,
			&p.Resource,
			&p.Action,
			&p.Description,
			&p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning module permission: %w", err)
		}
		if p.ModuleID != nil {
			if idx, ok := modIndexMap[*p.ModuleID]; ok {
				modules[idx].Permissions = append(modules[idx].Permissions, p)
			}
		}
	}

	return modules, nil
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
	if r.URL.Query().Get("group_by") == "module" {
		modules, err := h.svc.ListModules(r.Context())
		if err != nil {
			httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to list modules"))
			return
		}
		httpx.OK(w, modules)
		return
	}

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

// ListModules handles GET /api/v1/modules
func (h *Handler) ListModules(w http.ResponseWriter, r *http.Request) {
	modules, err := h.svc.ListModules(r.Context())
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to list modules"))
		return
	}
	httpx.OK(w, modules)
}
