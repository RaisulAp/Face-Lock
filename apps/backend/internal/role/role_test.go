package role

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestPool(t *testing.T) *pgxpool.Pool {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://faceclock:devpassword123@localhost:5434/faceclock?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Skipf("skipping role tests: database not reachable: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skipping role tests: ping failed: %v", err)
	}
	return pool
}

func TestRoleServiceAndHandler(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	ctx := context.Background()
	rbacCache := rbac.NewCache(30 * time.Second)
	rbacSvc := rbac.NewService(pool, rbacCache)
	roleSvc := NewService(pool, rbacSvc)
	auditRec := audit.NewRecorder(pool)
	handler := NewHandler(roleSvc, auditRec)

	superActor := &rbac.Principal{
		UserID:      uuid.New(),
		Roles:       []string{"super_admin"},
		Permissions: map[string]struct{}{"*": {}},
	}

	adminActor := &rbac.Principal{
		UserID: uuid.New(),
		Roles:  []string{"admin"},
		Permissions: map[string]struct{}{
			"role.read":   {},
			"role.create": {},
			"role.update": {},
			"user.read":   {},
		},
	}

	// 1. ListPermissions
	perms, err := roleSvc.ListPermissions(ctx)
	if err != nil {
		t.Fatalf("ListPermissions failed: %v", err)
	}
	if len(perms) < 34 {
		t.Errorf("expected at least 34 permissions, got %d", len(perms))
	}
	var permIDs []uuid.UUID
	for i := 0; i < 3 && i < len(perms); i++ {
		permIDs = append(permIDs, perms[i].ID)
	}

	// 2. ListRoles
	roles, err := roleSvc.ListRoles(ctx)
	if err != nil {
		t.Fatalf("ListRoles failed: %v", err)
	}
	if len(roles) < 3 {
		t.Fatalf("expected at least 3 seeded roles, got %d", len(roles))
	}

	var superRole, employeeRole, adminRole *Role
	for i := range roles {
		switch roles[i].Name {
		case "super_admin":
			superRole = &roles[i]
		case "employee":
			employeeRole = &roles[i]
		case "admin":
			adminRole = &roles[i]
		}
	}
	if superRole == nil || employeeRole == nil || adminRole == nil {
		t.Fatalf("missing seeded system roles")
	}

	// 3. GetRoleByID
	gotRole, err := roleSvc.GetRoleByID(ctx, superRole.ID)
	if err != nil {
		t.Fatalf("GetRoleByID failed: %v", err)
	}
	if gotRole.Name != "super_admin" {
		t.Errorf("expected super_admin, got %s", gotRole.Name)
	}
	if len(gotRole.Permissions) == 0 {
		t.Errorf("expected super_admin to have permissions")
	}

	// Non-existent role Get
	_, err = roleSvc.GetRoleByID(ctx, uuid.New())
	if err == nil {
		t.Errorf("expected error for non-existent role")
	}

	// 4. ListModules
	modules, err := roleSvc.ListModules(ctx)
	if err != nil {
		t.Fatalf("ListModules failed: %v", err)
	}
	if len(modules) != 8 {
		t.Errorf("expected 8 modules, got %d", len(modules))
	}
	for _, m := range modules {
		if len(m.Permissions) == 0 {
			t.Errorf("module %s has no permissions", m.Code)
		}
	}

	// 5. Verify dynamic active modules on system roles
	if len(superRole.Modules) != 8 {
		t.Errorf("expected super_admin to have 8 active modules, got %d", len(superRole.Modules))
	}
	if len(adminRole.Modules) != 8 {
		t.Errorf("expected admin to have 8 active modules, got %d", len(adminRole.Modules))
	}
	if len(employeeRole.Modules) != 5 {
		t.Errorf("expected employee to have 5 active modules, got %d", len(employeeRole.Modules))
	}

	// 6. Role Immutability: Custom role operations must be forbidden
	roleName := fmt.Sprintf("custom_role_%d", time.Now().UnixNano())
	desc := "Custom role attempt"
	_, err = roleSvc.CreateRole(ctx, superActor, CreateRoleParams{
		Name:          roleName,
		DisplayName:   "Custom Role",
		Description:   &desc,
		PermissionIDs: permIDs,
	})
	if err == nil {
		t.Errorf("expected CreateRole to be forbidden")
	}
	if appErr, ok := err.(*httpx.AppError); !ok || appErr.Code != httpx.CodeForbidden {
		t.Errorf("expected CodeForbidden, got %v", err)
	}

	newDN := "Updated Role"
	_, err = roleSvc.UpdateRole(ctx, superRole.ID, UpdateRoleParams{
		DisplayName: &newDN,
	})
	if err == nil {
		t.Errorf("expected UpdateRole to be forbidden")
	}
	if appErr, ok := err.(*httpx.AppError); !ok || appErr.Code != httpx.CodeForbidden {
		t.Errorf("expected CodeForbidden, got %v", err)
	}

	err = roleSvc.DeleteRole(ctx, superRole.ID)
	if err == nil {
		t.Errorf("expected DeleteRole to be forbidden")
	}
	if appErr, ok := err.(*httpx.AppError); !ok || appErr.Code != httpx.CodeForbidden {
		t.Errorf("expected CodeForbidden, got %v", err)
	}

	err = roleSvc.AssignPermissions(ctx, superActor, superRole.ID, permIDs)
	if err == nil {
		t.Errorf("expected AssignPermissions to be forbidden")
	}
	if appErr, ok := err.(*httpx.AppError); !ok || appErr.Code != httpx.CodeForbidden {
		t.Errorf("expected CodeForbidden, got %v", err)
	}

	// Also admin cannot perform these
	err = roleSvc.AssignPermissions(ctx, adminActor, adminRole.ID, permIDs)
	if err == nil {
		t.Errorf("expected AssignPermissions for admin to be forbidden")
	}

	// 7. Handler Endpoints
	// Handler ListRoles
	req := httptest.NewRequest(http.MethodGet, "/api/v1/roles", nil)
	w := httptest.NewRecorder()
	handler.ListRoles(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for ListRoles, got %d", w.Code)
	}

	// Handler GetRoleByID
	rCtx := chi.NewRouteContext()
	rCtx.URLParams.Add("id", superRole.ID.String())
	req = httptest.NewRequest(http.MethodGet, "/api/v1/roles/"+superRole.ID.String(), nil).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.GetRoleByID(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for GetRoleByID, got %d", w.Code)
	}

	// Handler ListModules
	req = httptest.NewRequest(http.MethodGet, "/api/v1/modules", nil)
	w = httptest.NewRecorder()
	handler.ListModules(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for ListModules, got %d", w.Code)
	}

	// Handler CreateRole (forbidden)
	createBody, _ := json.Marshal(CreateRoleParams{
		Name:        roleName,
		DisplayName: "Handler Role",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/roles", bytes.NewReader(createBody))
	req = req.WithContext(rbac.WithPrincipal(req.Context(), superActor))
	w = httptest.NewRecorder()
	handler.CreateRole(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for handler CreateRole, got %d", w.Code)
	}

	// Handler UpdateRole (forbidden)
	updateBody, _ := json.Marshal(UpdateRoleParams{
		DisplayName: &newDN,
	})
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/roles/"+superRole.ID.String(), bytes.NewReader(updateBody)).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.UpdateRole(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for handler UpdateRole, got %d", w.Code)
	}

	// Handler AssignPermissions (forbidden)
	assignBody, _ := json.Marshal(AssignPermissionsRequest{
		PermissionIDs: permIDs,
	})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/roles/"+superRole.ID.String()+"/permissions", bytes.NewReader(assignBody)).
		WithContext(context.WithValue(rbac.WithPrincipal(context.Background(), superActor), chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.AssignPermissions(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for handler AssignPermissions, got %d", w.Code)
	}

	// Handler DeleteRole (forbidden)
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/roles/"+superRole.ID.String(), nil).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.DeleteRole(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for handler DeleteRole, got %d", w.Code)
	}

	// Handler ListPermissions (flat)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/permissions", nil)
	w = httptest.NewRecorder()
	handler.ListPermissions(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for ListPermissions, got %d", w.Code)
	}

	// Handler ListPermissions (grouped)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/permissions?group_by=resource", nil)
	w = httptest.NewRecorder()
	handler.ListPermissions(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for ListPermissions grouped, got %d", w.Code)
	}

	// Handler Error Cases (400, 401, 404)
	badRCtx := chi.NewRouteContext()
	badRCtx.URLParams.Add("id", "invalid-uuid")

	// GetRoleByID 400
	req = httptest.NewRequest(http.MethodGet, "/api/v1/roles/bad", nil).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, badRCtx))
	w = httptest.NewRecorder()
	handler.GetRoleByID(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad id in GetRoleByID, got %d", w.Code)
	}

	// UpdateRole 400
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/roles/bad", bytes.NewReader([]byte(`{}`))).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, badRCtx))
	w = httptest.NewRecorder()
	handler.UpdateRole(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad id in UpdateRole, got %d", w.Code)
	}

	// DeleteRole 400
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/roles/bad", nil).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, badRCtx))
	w = httptest.NewRecorder()
	handler.DeleteRole(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad id in DeleteRole, got %d", w.Code)
	}

	// AssignPermissions 400
	req = httptest.NewRequest(http.MethodPut, "/api/v1/roles/bad/permissions", bytes.NewReader([]byte(`{}`))).
		WithContext(context.WithValue(rbac.WithPrincipal(context.Background(), superActor), chi.RouteCtxKey, badRCtx))
	w = httptest.NewRecorder()
	handler.AssignPermissions(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad id in AssignPermissions, got %d", w.Code)
	}

	// CreateRole 401 unauthenticated
	req = httptest.NewRequest(http.MethodPost, "/api/v1/roles", bytes.NewReader(createBody))
	w = httptest.NewRecorder()
	handler.CreateRole(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated CreateRole, got %d", w.Code)
	}

	// AssignPermissions 401 unauthenticated
	req = httptest.NewRequest(http.MethodPut, "/api/v1/roles/"+superRole.ID.String()+"/permissions", bytes.NewReader(assignBody)).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.AssignPermissions(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated AssignPermissions, got %d", w.Code)
	}
}
