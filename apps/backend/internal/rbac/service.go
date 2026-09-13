package rbac

import (
	"context"
	"fmt"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service provides RBAC queries, caching, and guardrail validations.
type Service struct {
	db    *pgxpool.Pool
	cache *Cache
}

// NewService creates a new RBAC service.
func NewService(db *pgxpool.Pool, cache *Cache) *Service {
	return &Service{
		db:    db,
		cache: cache,
	}
}

// GetUserRolesAndPermissions fetches the roles and deduplicated permissions for a user,
// utilizing the in-memory cache when available.
func (s *Service) GetUserRolesAndPermissions(ctx context.Context, userID uuid.UUID) ([]string, map[string]struct{}, error) {
	if s.cache != nil {
		if roles, perms, ok := s.cache.Get(userID); ok {
			return roles, perms, nil
		}
	}

	// Fetch roles
	const roleQuery = `
		SELECT r.name
		FROM user_roles ur
		JOIN roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1 AND r.deleted_at IS NULL
		ORDER BY r.name ASC
	`
	rows, err := s.db.Query(ctx, roleQuery, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("querying user roles: %w", err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var roleName string
		if err := rows.Scan(&roleName); err != nil {
			return nil, nil, fmt.Errorf("scanning user role: %w", err)
		}
		roles = append(roles, roleName)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterating user roles: %w", err)
	}

	// Fetch permissions across all user roles
	const permQuery = `
		SELECT DISTINCT p.name
		FROM user_roles ur
		JOIN roles r ON ur.role_id = r.id
		JOIN role_permissions rp ON r.id = rp.role_id
		JOIN permissions p ON rp.permission_id = p.id
		WHERE ur.user_id = $1 AND r.deleted_at IS NULL
	`
	pRows, err := s.db.Query(ctx, permQuery, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("querying user permissions: %w", err)
	}
	defer pRows.Close()

	permissions := make(map[string]struct{})
	for pRows.Next() {
		var permName string
		if err := pRows.Scan(&permName); err != nil {
			return nil, nil, fmt.Errorf("scanning permission: %w", err)
		}
		permissions[permName] = struct{}{}
	}
	if err := pRows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterating permissions: %w", err)
	}

	if s.cache != nil {
		s.cache.Set(userID, roles, permissions)
	}

	return roles, permissions, nil
}

// GetPrincipal returns the Principal struct for a user with cached roles and permissions.
func (s *Service) GetPrincipal(ctx context.Context, userID uuid.UUID) (*Principal, error) {
	roles, perms, err := s.GetUserRolesAndPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &Principal{
		UserID:      userID,
		Roles:       roles,
		Permissions: perms,
	}, nil
}

// CheckPrivilegeEscalation verifies that the actor has all permissions they are attempting
// to grant. If the actor is super_admin, all grants are permitted.
func (s *Service) CheckPrivilegeEscalation(actor *Principal, targetPermissions []string) error {
	if actor.IsSuperAdmin() {
		return nil
	}

	for _, perm := range targetPermissions {
		if !actor.HasPermission(perm) {
			return httpx.NewAppError(
				httpx.CodeRoleEscalationDenied,
				fmt.Sprintf("cannot assign permission %q which you do not possess", perm),
			)
		}
	}

	return nil
}

// CheckPrivilegeEscalationRoleIDs resolves all permissions of targetRoleIDs and checks
// whether actor possesses every one of them.
func (s *Service) CheckPrivilegeEscalationRoleIDs(ctx context.Context, actor *Principal, targetRoleIDs []uuid.UUID) error {
	if actor.IsSuperAdmin() {
		return nil
	}

	if len(targetRoleIDs) == 0 {
		return nil
	}

	const q = `
		SELECT DISTINCT p.name
		FROM role_permissions rp
		JOIN permissions p ON rp.permission_id = p.id
		WHERE rp.role_id = ANY($1)
	`
	rows, err := s.db.Query(ctx, q, targetRoleIDs)
	if err != nil {
		return fmt.Errorf("querying role permissions for escalation check: %w", err)
	}
	defer rows.Close()

	var perms []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return fmt.Errorf("scanning role permission: %w", err)
		}
		perms = append(perms, p)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating role permissions: %w", err)
	}

	return s.CheckPrivilegeEscalation(actor, perms)
}

// CheckLastSuperAdmin ensures that deleting, deactivating, or demoting targetUserID
// does not eliminate the last remaining active super_admin in the system (§ 5.5 B).
func (s *Service) CheckLastSuperAdmin(ctx context.Context, targetUserID uuid.UUID) error {
	// Check if target user is currently an active super_admin
	const targetCheck = `
		SELECT EXISTS (
			SELECT 1
			FROM user_roles ur
			JOIN users u ON ur.user_id = u.id
			JOIN roles r ON ur.role_id = r.id
			WHERE ur.user_id = $1
			  AND r.name = 'super_admin'
			  AND u.is_active = true
			  AND u.deleted_at IS NULL
			  AND r.deleted_at IS NULL
		)
	`
	var isTargetActiveSuperAdmin bool
	if err := s.db.QueryRow(ctx, targetCheck, targetUserID).Scan(&isTargetActiveSuperAdmin); err != nil {
		return fmt.Errorf("checking target super admin status: %w", err)
	}

	if !isTargetActiveSuperAdmin {
		return nil
	}

	// Count total active super admins in system
	const countSuperAdmins = `
		SELECT COUNT(DISTINCT u.id)
		FROM user_roles ur
		JOIN users u ON ur.user_id = u.id
		JOIN roles r ON ur.role_id = r.id
		WHERE r.name = 'super_admin'
		  AND u.is_active = true
		  AND u.deleted_at IS NULL
		  AND r.deleted_at IS NULL
	`
	var totalSuperAdmins int
	if err := s.db.QueryRow(ctx, countSuperAdmins).Scan(&totalSuperAdmins); err != nil {
		return fmt.Errorf("counting active super admins: %w", err)
	}

	if totalSuperAdmins <= 1 {
		return httpx.NewAppError(
			httpx.CodeLastSuperAdmin,
			"cannot modify or deactivate the last active super admin",
		)
	}

	return nil
}

// InvalidateUser invalidates cache for a user.
func (s *Service) InvalidateUser(userID uuid.UUID) {
	if s.cache != nil {
		s.cache.InvalidateUser(userID)
	}
}

// InvalidateAll clears all cached roles and permissions.
func (s *Service) InvalidateAll() {
	if s.cache != nil {
		s.cache.InvalidateAll()
	}
}
