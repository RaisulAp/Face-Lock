package rbac

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://faceclock:devpassword123@localhost:5434/faceclock?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skipping db test, cannot connect to %s: %v", dsn, err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skipping db test, ping failed: %v", err)
	}
	return pool
}

func TestIsValidPermission(t *testing.T) {
	if !IsValidPermission(PermEmployeeRead) {
		t.Errorf("expected %s to be valid", PermEmployeeRead)
	}
	if !IsValidPermission(PermRoleAssignPermission) {
		t.Errorf("expected %s to be valid", PermRoleAssignPermission)
	}
	if IsValidPermission("invalid.permission.name") {
		t.Errorf("expected invalid.permission.name to be invalid")
	}
}

func TestRBACService_GetUserRolesAndPermissions(t *testing.T) {
	pool := getTestDB(t)
	cache := NewCache(30 * time.Second)
	svc := NewService(pool, cache)

	ctx := context.Background()

	// Find the seeded super_admin user
	var superAdminID uuid.UUID
	err := pool.QueryRow(ctx, "SELECT id FROM users WHERE email = 'admin@faceclock.local' AND deleted_at IS NULL").Scan(&superAdminID)
	if err != nil {
		t.Fatalf("failed to query superadmin user: %v", err)
	}

	// 1. Get roles and permissions (cache miss, loads from DB)
	roles, perms, err := svc.GetUserRolesAndPermissions(ctx, superAdminID)
	if err != nil {
		t.Fatalf("failed to get roles and perms: %v", err)
	}
	if len(roles) == 0 {
		t.Fatalf("expected roles for super admin")
	}
	if roles[0] != "super_admin" {
		t.Errorf("expected super_admin role, got %s", roles[0])
	}
	if len(perms) == 0 {
		t.Errorf("expected permissions for super admin")
	}

	// 2. Second call should hit cache
	rolesCached, permsCached, err := svc.GetUserRolesAndPermissions(ctx, superAdminID)
	if err != nil {
		t.Fatalf("failed to get cached roles and perms: %v", err)
	}
	if len(rolesCached) != len(roles) || len(permsCached) != len(perms) {
		t.Errorf("cached roles/perms do not match original")
	}

	// 3. GetPrincipal
	principal, err := svc.GetPrincipal(ctx, superAdminID)
	if err != nil {
		t.Fatalf("failed to get principal: %v", err)
	}
	if principal.UserID != superAdminID {
		t.Errorf("expected user ID %s, got %s", superAdminID, principal.UserID)
	}
	if !principal.IsSuperAdmin() {
		t.Errorf("expected principal to be super admin")
	}
}

func TestRBACService_CheckPrivilegeEscalationRoleIDs(t *testing.T) {
	pool := getTestDB(t)
	cache := NewCache(30 * time.Second)
	svc := NewService(pool, cache)
	ctx := context.Background()

	// 1. Super admin actor -> always nil
	superAdminPrincipal := &Principal{
		UserID: uuid.New(),
		Roles:  []string{"super_admin"},
	}
	var superAdminRoleID uuid.UUID
	err := pool.QueryRow(ctx, "SELECT id FROM roles WHERE name = 'super_admin'").Scan(&superAdminRoleID)
	if err != nil {
		t.Fatalf("failed to query super_admin role ID: %v", err)
	}

	if err := svc.CheckPrivilegeEscalationRoleIDs(ctx, superAdminPrincipal, []uuid.UUID{superAdminRoleID}); err != nil {
		t.Errorf("super admin actor should never fail escalation check: %v", err)
	}

	// 2. Empty role IDs -> returns nil
	if err := svc.CheckPrivilegeEscalationRoleIDs(ctx, &Principal{UserID: uuid.New()}, nil); err != nil {
		t.Errorf("empty role IDs should return nil: %v", err)
	}

	// 3. Non-super admin actor with only employee permissions trying to grant super_admin role
	employeeActor := &Principal{
		UserID: uuid.New(),
		Roles:  []string{"employee"},
		Permissions: map[string]struct{}{
			PermEmployeeReadSelf:  {},
			PermAttendanceCheckin: {},
		},
	}
	err = svc.CheckPrivilegeEscalationRoleIDs(ctx, employeeActor, []uuid.UUID{superAdminRoleID})
	if err == nil {
		t.Fatalf("expected escalation error when employee tries to grant super_admin role")
	}
	appErr, ok := err.(*httpx.AppError)
	if !ok || appErr.Code != httpx.CodeRoleEscalationDenied {
		t.Errorf("expected CodeRoleEscalationDenied, got: %v", err)
	}
}

func TestRBACService_CheckLastSuperAdmin(t *testing.T) {
	pool := getTestDB(t)
	cache := NewCache(30 * time.Second)
	svc := NewService(pool, cache)
	ctx := context.Background()

	// 1. Check for a non-superadmin user (should succeed without error)
	nonSuperAdminID := uuid.New()
	if err := svc.CheckLastSuperAdmin(ctx, nonSuperAdminID); err != nil {
		t.Errorf("non-superadmin check should succeed: %v", err)
	}

	// 2. Find the primary super admin
	var superAdminID uuid.UUID
	err := pool.QueryRow(ctx, "SELECT id FROM users WHERE email = 'admin@faceclock.local' AND deleted_at IS NULL").Scan(&superAdminID)
	if err != nil {
		t.Fatalf("failed to query super admin: %v", err)
	}

	// If there's only 1 super admin in the database, checking them should fail with CodeLastSuperAdmin
	var count int
	_ = pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT u.id)
		FROM user_roles ur
		JOIN users u ON ur.user_id = u.id
		JOIN roles r ON ur.role_id = r.id
		WHERE r.name = 'super_admin' AND u.is_active = true AND u.deleted_at IS NULL AND r.deleted_at IS NULL
	`).Scan(&count)

	if count == 1 {
		err = svc.CheckLastSuperAdmin(ctx, superAdminID)
		if err == nil {
			t.Fatalf("expected CodeLastSuperAdmin error when only 1 active super admin exists")
		}
		appErr, ok := err.(*httpx.AppError)
		if !ok || appErr.Code != httpx.CodeLastSuperAdmin {
			t.Errorf("expected CodeLastSuperAdmin, got %v", err)
		}
	}

	// 3. Create a second super admin and verify CheckLastSuperAdmin now succeeds
	tempSuperAdminID := uuid.New()
	tempEmail := fmt.Sprintf("temp_super_%d@faceclock.local", time.Now().UnixNano())
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, 'dummy_hash', true, NOW(), NOW())`,
		tempSuperAdminID, tempEmail)
	if err != nil {
		t.Fatalf("failed to create temp super admin user: %v", err)
	}
	defer pool.Exec(ctx, "DELETE FROM users WHERE id = $1", tempSuperAdminID)

	var superRoleID uuid.UUID
	_ = pool.QueryRow(ctx, "SELECT id FROM roles WHERE name = 'super_admin'").Scan(&superRoleID)
	_, err = pool.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`,
		tempSuperAdminID, superRoleID)
	if err != nil {
		t.Fatalf("failed to assign super_admin role: %v", err)
	}
	defer pool.Exec(ctx, "DELETE FROM user_roles WHERE user_id = $1", tempSuperAdminID)

	// Now with 2 super admins, CheckLastSuperAdmin on the first should succeed
	err = svc.CheckLastSuperAdmin(ctx, superAdminID)
	if err != nil {
		t.Errorf("expected CheckLastSuperAdmin to succeed with 2 active super admins, got: %v", err)
	}
}

func TestRBACService_Invalidation(t *testing.T) {
	pool := getTestDB(t)
	cache := NewCache(30 * time.Second)
	svc := NewService(pool, cache)

	uID := uuid.New()
	cache.Set(uID, []string{"employee"}, map[string]struct{}{})

	svc.InvalidateUser(uID)
	if _, _, ok := cache.Get(uID); ok {
		t.Errorf("expected cache miss after InvalidateUser")
	}

	cache.Set(uID, []string{"employee"}, map[string]struct{}{})
	svc.InvalidateAll()
	if _, _, ok := cache.Get(uID); ok {
		t.Errorf("expected cache miss after InvalidateAll")
	}
}
