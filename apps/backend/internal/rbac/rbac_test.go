package rbac

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPrincipal_Permissions(t *testing.T) {
	adminID := uuid.New()
	pSuper := &Principal{
		UserID: adminID,
		Roles:  []string{"super_admin"},
	}

	if !pSuper.IsSuperAdmin() {
		t.Errorf("expected is super admin")
	}
	if !pSuper.HasPermission("any.arbitrary.permission") {
		t.Errorf("super admin should have any permission")
	}

	empID := uuid.New()
	pEmp := &Principal{
		UserID: empID,
		Roles:  []string{"employee"},
		Permissions: map[string]struct{}{
			PermEmployeeReadSelf:  {},
			PermAttendanceCheckin: {},
		},
	}

	if pEmp.IsSuperAdmin() {
		t.Errorf("employee should not be super admin")
	}
	if !pEmp.HasPermission(PermAttendanceCheckin) {
		t.Errorf("expected employee to have attendance.checkin")
	}
	if pEmp.HasPermission(PermEmployeeDelete) {
		t.Errorf("employee should not have employee.delete")
	}
	if !pEmp.HasAnyPermission(PermEmployeeDelete, PermAttendanceCheckin) {
		t.Errorf("employee should match has any permission")
	}
	if pEmp.HasAllPermissions(PermEmployeeDelete, PermAttendanceCheckin) {
		t.Errorf("employee does not have all permissions")
	}
}

func TestRBAC_Cache(t *testing.T) {
	cache := NewCache(50 * time.Millisecond)
	uID := uuid.New()

	roles := []string{"admin"}
	perms := map[string]struct{}{PermEmployeeRead: {}}

	cache.Set(uID, roles, perms)

	r, p, ok := cache.Get(uID)
	if !ok {
		t.Fatalf("expected cache hit")
	}
	if len(r) != 1 || r[0] != "admin" {
		t.Errorf("unexpected roles: %v", r)
	}
	if _, has := p[PermEmployeeRead]; !has {
		t.Errorf("expected permission employee.read in cache")
	}

	// Invalidate single user
	cache.InvalidateUser(uID)
	_, _, ok = cache.Get(uID)
	if ok {
		t.Errorf("expected cache miss after user invalidation")
	}

	// Test InvalidateAll
	cache.Set(uID, roles, perms)
	cache.InvalidateAll()
	_, _, ok = cache.Get(uID)
	if ok {
		t.Errorf("expected cache miss after invalidate all")
	}
}
