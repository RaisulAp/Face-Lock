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

	// 4. CreateRole
	roleName := fmt.Sprintf("custom_role_%d", time.Now().UnixNano())
	desc := "Custom role for testing"
	created, err := roleSvc.CreateRole(ctx, superActor, CreateRoleParams{
		Name:          roleName,
		DisplayName:   "Custom Role",
		Description:   &desc,
		PermissionIDs: permIDs,
	})
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}
	if created.Name != roleName || created.IsSystem {
		t.Errorf("unexpected created role: %+v", created)
	}
	if len(created.Permissions) != len(permIDs) {
		t.Errorf("expected %d permissions, got %d", len(permIDs), len(created.Permissions))
	}

	// Duplicate role name -> CodeRoleNameTaken
	_, err = roleSvc.CreateRole(ctx, superActor, CreateRoleParams{
		Name:        roleName,
		DisplayName: "Duplicate Role",
	})
	if err == nil {
		t.Errorf("expected duplicate role name to fail")
	}
	if appErr, ok := err.(*httpx.AppError); !ok || appErr.Code != httpx.CodeRoleNameTaken {
		t.Errorf("expected CodeRoleNameTaken, got %v", err)
	}

	// Privilege escalation on CreateRole (admin actor assigning permissions they don't have)
	// Find a permission adminActor doesn't have, e.g. system.settings.write or user.delete
	var unheldPerm uuid.UUID
	for _, p := range perms {
		if _, ok := adminActor.Permissions[p.Name]; !ok {
			unheldPerm = p.ID
			break
		}
	}
	if unheldPerm != uuid.Nil {
		_, err = roleSvc.CreateRole(ctx, adminActor, CreateRoleParams{
			Name:          fmt.Sprintf("esc_role_%d", time.Now().UnixNano()),
			DisplayName:   "Escalation Role",
			PermissionIDs: []uuid.UUID{unheldPerm},
		})
		if err == nil {
			t.Errorf("expected privilege escalation on CreateRole to fail")
		}
	}

	// 5. UpdateRole
	newDN := "Updated Custom Role"
	newDesc := "Updated description"
	updated, err := roleSvc.UpdateRole(ctx, created.ID, UpdateRoleParams{
		DisplayName: &newDN,
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("UpdateRole failed: %v", err)
	}
	if updated.DisplayName != newDN || *updated.Description != newDesc {
		t.Errorf("update role mismatch: %+v", updated)
	}

	// Update non-existent
	_, err = roleSvc.UpdateRole(ctx, uuid.New(), UpdateRoleParams{DisplayName: &newDN})
	if err == nil {
		t.Errorf("expected error updating non-existent role")
	}

	// 6. AssignPermissions
	var singlePerm []uuid.UUID
	if len(permIDs) > 0 {
		singlePerm = []uuid.UUID{permIDs[0]}
	}
	err = roleSvc.AssignPermissions(ctx, superActor, created.ID, singlePerm)
	if err != nil {
		t.Fatalf("AssignPermissions failed: %v", err)
	}
	refetched, _ := roleSvc.GetRoleByID(ctx, created.ID)
	if len(refetched.Permissions) != len(singlePerm) {
		t.Errorf("expected %d permission, got %d", len(singlePerm), len(refetched.Permissions))
	}

	// Privilege escalation on AssignPermissions
	if unheldPerm != uuid.Nil {
		err = roleSvc.AssignPermissions(ctx, adminActor, created.ID, []uuid.UUID{unheldPerm})
		if err == nil {
			t.Errorf("expected privilege escalation on AssignPermissions to fail")
		}
	}

	// Non-super_admin modifying super_admin permissions
	err = roleSvc.AssignPermissions(ctx, adminActor, superRole.ID, singlePerm)
	if err == nil {
		t.Errorf("expected non-super_admin editing super_admin role permissions to fail")
	}

	// 7. DeleteRole - System role immutability
	err = roleSvc.DeleteRole(ctx, superRole.ID)
	if err == nil {
		t.Errorf("expected system role deletion to fail")
	}
	if appErr, ok := err.(*httpx.AppError); !ok || appErr.Code != httpx.CodeSystemRoleImmutable {
		t.Errorf("expected CodeSystemRoleImmutable, got %v", err)
	}

	// Role in use guard: assign role to an active user, then attempt to delete
	var activeUserID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT id FROM users WHERE is_active = true AND deleted_at IS NULL LIMIT 1`).Scan(&activeUserID)
	if err == nil {
		_, _ = pool.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, activeUserID, created.ID)
		err = roleSvc.DeleteRole(ctx, created.ID)
		if err == nil {
			t.Errorf("expected role in use deletion to fail")
		}
		if appErr, ok := err.(*httpx.AppError); !ok || appErr.Code != httpx.CodeRoleInUse {
			t.Errorf("expected CodeRoleInUse, got %v", err)
		}
		// Unassign role from user
		_, _ = pool.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`, activeUserID, created.ID)
	}

	// Delete custom role succeeds
	err = roleSvc.DeleteRole(ctx, created.ID)
	if err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}

	// Delete already deleted role -> 404
	err = roleSvc.DeleteRole(ctx, created.ID)
	if err == nil {
		t.Errorf("expected error deleting already deleted role")
	}

	// 8. Handler Endpoints
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

	// Handler CreateRole
	createBody, _ := json.Marshal(CreateRoleParams{
		Name:        fmt.Sprintf("h_role_%d", time.Now().UnixNano()),
		DisplayName: "Handler Role",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/roles", bytes.NewReader(createBody))
	req = req.WithContext(rbac.WithPrincipal(req.Context(), superActor))
	w = httptest.NewRecorder()
	handler.CreateRole(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201 for handler CreateRole, got %d", w.Code)
	}
	var resp struct {
		Data Role `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	hRoleID := resp.Data.ID

	// Handler UpdateRole
	updateBody, _ := json.Marshal(UpdateRoleParams{
		DisplayName: &newDN,
	})
	rCtx = chi.NewRouteContext()
	rCtx.URLParams.Add("id", hRoleID.String())
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/roles/"+hRoleID.String(), bytes.NewReader(updateBody)).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.UpdateRole(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for handler UpdateRole, got %d", w.Code)
	}

	// Handler AssignPermissions
	assignBody, _ := json.Marshal(AssignPermissionsRequest{
		PermissionIDs: permIDs,
	})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/roles/"+hRoleID.String()+"/permissions", bytes.NewReader(assignBody)).
		WithContext(context.WithValue(rbac.WithPrincipal(context.Background(), superActor), chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.AssignPermissions(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for handler AssignPermissions, got %d", w.Code)
	}

	// Handler DeleteRole
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/roles/"+hRoleID.String(), nil).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.DeleteRole(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for handler DeleteRole, got %d", w.Code)
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
