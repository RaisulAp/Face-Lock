package rbac

import (
	"testing"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/google/uuid"
)

func TestCheckPrivilegeEscalation(t *testing.T) {
	svc := &Service{}

	superAdmin := &Principal{
		UserID: uuid.New(),
		Roles:  []string{"super_admin"},
	}

	// Super admin can assign any permission
	err := svc.CheckPrivilegeEscalation(superAdmin, []string{"anything.create", "nuclear.launch"})
	if err != nil {
		t.Fatalf("super admin should have no escalation restrictions: %v", err)
	}

	admin := &Principal{
		UserID: uuid.New(),
		Roles:  []string{"admin"},
		Permissions: map[string]struct{}{
			PermEmployeeRead:   {},
			PermEmployeeCreate: {},
			PermEmployeeUpdate: {},
		},
	}

	// Admin assigning subset they possess
	err = svc.CheckPrivilegeEscalation(admin, []string{PermEmployeeRead, PermEmployeeCreate})
	if err != nil {
		t.Fatalf("admin assigning subset should succeed: %v", err)
	}

	// Admin assigning permission they do NOT possess
	err = svc.CheckPrivilegeEscalation(admin, []string{PermEmployeeRead, PermRoleAssignPermission})
	if err == nil {
		t.Fatalf("expected error when assigning permission admin does not possess")
	}

	appErr, ok := err.(*httpx.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != httpx.CodeRoleEscalationDenied {
		t.Errorf("expected code %s, got %s", httpx.CodeRoleEscalationDenied, appErr.Code)
	}
}
