package user

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

func TestUserServiceAndHandler(t *testing.T) {
	pool := getTestDB(t)
	rbacCache := rbac.NewCache(30 * time.Second)
	rbacSvc := rbac.NewService(pool, rbacCache)
	auditRec := audit.NewRecorder(pool)
	userSvc := NewService(pool, rbacSvc)
	handler := NewHandler(userSvc, auditRec)

	ctx := context.Background()

	// Find super_admin role and admin role
	var (
		superAdminRoleID uuid.UUID
		adminRoleID      uuid.UUID
		employeeRoleID   uuid.UUID
	)
	_ = pool.QueryRow(ctx, "SELECT id FROM roles WHERE name = 'super_admin' AND deleted_at IS NULL").Scan(&superAdminRoleID)
	_ = pool.QueryRow(ctx, "SELECT id FROM roles WHERE name = 'admin' AND deleted_at IS NULL").Scan(&adminRoleID)
	_ = pool.QueryRow(ctx, "SELECT id FROM roles WHERE name = 'employee' AND deleted_at IS NULL").Scan(&employeeRoleID)

	// Create test actor (super_admin)
	superActor := &rbac.Principal{
		UserID:      uuid.New(),
		Roles:       []string{"super_admin"},
		Permissions: map[string]struct{}{"user.create": {}, "user.read": {}, "user.update": {}, "user.delete": {}},
	}

	// 1. Create user with auto-generated temp password
	uniqueEmail := fmt.Sprintf("test_user_%d@faceclock.local", time.Now().UnixNano())
	createRes, err := userSvc.Create(ctx, superActor, CreateParams{
		Email:   uniqueEmail,
		RoleIDs: []uuid.UUID{employeeRoleID},
	})
	if err != nil {
		t.Fatalf("userSvc.Create failed: %v", err)
	}
	if createRes.User.Email != uniqueEmail {
		t.Errorf("expected email %s, got %s", uniqueEmail, createRes.User.Email)
	}
	if createRes.TemporaryPassword == "" {
		t.Errorf("expected generated temporary password to be non-empty")
	}
	targetUserID := createRes.User.ID

	// 2. Duplicate email should fail with CodeEmailTaken
	_, err = userSvc.Create(ctx, superActor, CreateParams{
		Email: uniqueEmail,
	})
	if err == nil {
		t.Fatalf("expected duplicate email error")
	}
	appErr, ok := err.(*httpx.AppError)
	if !ok || appErr.Code != httpx.CodeEmailTaken {
		t.Errorf("expected CodeEmailTaken, got %v", err)
	}

	// 3. Create with invalid custom password (< 10 chars)
	badPwd := "short"
	_, err = userSvc.Create(ctx, superActor, CreateParams{
		Email:    fmt.Sprintf("badpwd_%d@faceclock.local", time.Now().UnixNano()),
		Password: &badPwd,
	})
	if err == nil {
		t.Fatalf("expected validation error for short password")
	}

	// 4. Create with custom valid password
	goodPwd := "SecurePass1234!"
	cRes2, err := userSvc.Create(ctx, superActor, CreateParams{
		Email:    fmt.Sprintf("goodpwd_%d@faceclock.local", time.Now().UnixNano()),
		Password: &goodPwd,
	})
	if err != nil {
		t.Fatalf("userSvc.Create with custom password failed: %v", err)
	}
	if cRes2.TemporaryPassword != "" {
		t.Errorf("expected empty temporary password when custom provided")
	}

	// 5. GetByID
	fetched, err := userSvc.GetByID(ctx, targetUserID)
	if err != nil {
		t.Fatalf("userSvc.GetByID failed: %v", err)
	}
	if fetched.Email != uniqueEmail {
		t.Errorf("expected email %s, got %s", uniqueEmail, fetched.Email)
	}

	// GetByID not found
	_, err = userSvc.GetByID(ctx, uuid.New())
	if err == nil {
		t.Fatalf("expected not found error")
	}

	// 6. List users
	activeTrue := true
	list, meta, err := userSvc.List(ctx, Filter{
		Page:     1,
		PerPage:  10,
		Search:   uniqueEmail,
		IsActive: &activeTrue,
	})
	if err != nil {
		t.Fatalf("userSvc.List failed: %v", err)
	}
	if meta.Total < 1 || len(list) < 1 {
		t.Errorf("expected at least 1 user in list, got %d", len(list))
	}

	// 7. Update user
	newEmail := fmt.Sprintf("updated_%d@faceclock.local", time.Now().UnixNano())
	updated, err := userSvc.Update(ctx, targetUserID, UpdateParams{
		Email: &newEmail,
	})
	if err != nil {
		t.Fatalf("userSvc.Update failed: %v", err)
	}
	if updated.Email != newEmail {
		t.Errorf("expected updated email %s, got %s", newEmail, updated.Email)
	}

	// 8. UpdateStatus (Deactivate then Reactivate)
	deactivated, err := userSvc.UpdateStatus(ctx, targetUserID, false)
	if err != nil {
		t.Fatalf("userSvc.UpdateStatus (deactivate) failed: %v", err)
	}
	if deactivated.IsActive {
		t.Errorf("expected user to be inactive")
	}

	reactivated, err := userSvc.UpdateStatus(ctx, targetUserID, true)
	if err != nil {
		t.Fatalf("userSvc.UpdateStatus (reactivate) failed: %v", err)
	}
	if !reactivated.IsActive {
		t.Errorf("expected user to be active")
	}

	// 9. AssignRoles
	err = userSvc.AssignRoles(ctx, superActor, targetUserID, []uuid.UUID{adminRoleID})
	if err != nil {
		t.Fatalf("userSvc.AssignRoles failed: %v", err)
	}
	fetchedWithRole, err := userSvc.GetByID(ctx, targetUserID)
	if err != nil {
		t.Fatalf("failed fetching user after role assignment: %v", err)
	}
	hasAdminRole := false
	for _, r := range fetchedWithRole.Roles {
		if r == "admin" {
			hasAdminRole = true
			break
		}
	}
	if !hasAdminRole {
		t.Errorf("expected user to have admin role, got %v", fetchedWithRole.Roles)
	}

	// 10. ResetPassword
	newTempPwd, err := userSvc.ResetPassword(ctx, targetUserID, nil)
	if err != nil {
		t.Fatalf("userSvc.ResetPassword failed: %v", err)
	}
	if newTempPwd == "" {
		t.Errorf("expected non-empty reset temporary password")
	}

	customReset := "NewSecurePass99!"
	_, err = userSvc.ResetPassword(ctx, targetUserID, &customReset)
	if err != nil {
		t.Fatalf("userSvc.ResetPassword custom failed: %v", err)
	}

	// 11. Delete user
	err = userSvc.Delete(ctx, targetUserID)
	if err != nil {
		t.Fatalf("userSvc.Delete failed: %v", err)
	}
	// Verify deleted_at is set (GetByID should return 404)
	_, err = userSvc.GetByID(ctx, targetUserID)
	if err == nil {
		t.Fatalf("expected deleted user to return not found")
	}

	// 12. Handler tests
	hEmail := fmt.Sprintf("huser_%d@faceclock.local", time.Now().UnixNano())
	hBody, _ := json.Marshal(CreateParams{
		Email:   hEmail,
		RoleIDs: []uuid.UUID{employeeRoleID},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(hBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req.WithContext(rbac.WithPrincipal(context.Background(), superActor)))
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for handler.Create, got %d body: %s", w.Code, w.Body.String())
	}
	var hCreateResp struct {
		Data CreateResult `json:"data"`
	}
	_ = json.NewDecoder(w.Body).Decode(&hCreateResp)
	hUserID := hCreateResp.Data.User.ID

	// Handler List
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&per_page=10&is_active=true", nil)
	w = httptest.NewRecorder()
	handler.List(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for handler.List, got %d", w.Code)
	}

	// Handler GetByID
	rCtx := chi.NewRouteContext()
	rCtx.URLParams.Add("id", hUserID.String())
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/"+hUserID.String(), nil).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.GetByID(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for handler.GetByID, got %d", w.Code)
	}

	// Handler Update
	hUpEmail := fmt.Sprintf("hup_%d@faceclock.local", time.Now().UnixNano())
	upBody, _ := json.Marshal(UpdateParams{Email: &hUpEmail})
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+hUserID.String(), bytes.NewReader(upBody)).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rCtx))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.Update(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for handler.Update, got %d", w.Code)
	}

	// Handler UpdateStatus
	statBody, _ := json.Marshal(UpdateStatusRequest{IsActive: false})
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+hUserID.String()+"/status", bytes.NewReader(statBody)).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rCtx))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.UpdateStatus(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for handler.UpdateStatus, got %d", w.Code)
	}

	// Handler AssignRoles
	assignBody, _ := json.Marshal(AssignRolesRequest{RoleIDs: []uuid.UUID{employeeRoleID}})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/users/"+hUserID.String()+"/roles", bytes.NewReader(assignBody)).
		WithContext(context.WithValue(rbac.WithPrincipal(context.Background(), superActor), chi.RouteCtxKey, rCtx))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.AssignRoles(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for handler.AssignRoles, got %d", w.Code)
	}

	// Handler ResetPassword
	rstBody, _ := json.Marshal(ResetPasswordRequest{})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/"+hUserID.String()+"/reset-password", bytes.NewReader(rstBody)).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rCtx))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ResetPassword(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for handler.ResetPassword, got %d", w.Code)
	}

	// Handler Delete
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+hUserID.String(), nil).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.Delete(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for handler.Delete, got %d", w.Code)
	}

	// 13. Last Super Admin Protection Tests
	var soleSuperAdminID uuid.UUID
	err = pool.QueryRow(ctx, `
		SELECT u.id FROM users u
		JOIN user_roles ur ON u.id = ur.user_id
		JOIN roles r ON ur.role_id = r.id
		WHERE r.name = 'super_admin' AND u.is_active = true AND u.deleted_at IS NULL
		ORDER BY u.created_at ASC
		LIMIT 1
	`).Scan(&soleSuperAdminID)
	if err == nil {
		// Ensure soleSuperAdminID is indeed the only active super admin for this assertion
		var otherSuperIDs []uuid.UUID
		rows, qErr := pool.Query(ctx, `
			SELECT u.id FROM users u
			JOIN user_roles ur ON u.id = ur.user_id
			JOIN roles r ON ur.role_id = r.id
			WHERE r.name = 'super_admin' AND u.is_active = true AND u.deleted_at IS NULL AND u.id != $1
		`, soleSuperAdminID)
		if qErr == nil {
			for rows.Next() {
				var oid uuid.UUID
				if rows.Scan(&oid) == nil {
					otherSuperIDs = append(otherSuperIDs, oid)
				}
			}
			rows.Close()
		}
		if len(otherSuperIDs) > 0 {
			_, _ = pool.Exec(ctx, `UPDATE users SET is_active = false WHERE id = ANY($1)`, otherSuperIDs)
			defer func() {
				_, _ = pool.Exec(context.Background(), `UPDATE users SET is_active = true WHERE id = ANY($1)`, otherSuperIDs)
			}()
		}

		// Attempt to deactivate last super admin
		_, err = userSvc.UpdateStatus(ctx, soleSuperAdminID, false)
		if err == nil {
			t.Errorf("expected last super admin deactivation to fail")
		}
		if appErr, ok := err.(*httpx.AppError); !ok || appErr.Code != httpx.CodeLastSuperAdmin {
			t.Errorf("expected CodeLastSuperAdmin, got %v", err)
		}

		// Attempt to update is_active to false via Update
		f := false
		_, err = userSvc.Update(ctx, soleSuperAdminID, UpdateParams{IsActive: &f})
		if err == nil {
			t.Errorf("expected last super admin update to inactive to fail")
		}

		// Attempt to delete last super admin
		err = userSvc.Delete(ctx, soleSuperAdminID)
		if err == nil {
			t.Errorf("expected last super admin deletion to fail")
		}

		// Attempt to strip super_admin role
		err = userSvc.AssignRoles(ctx, superActor, soleSuperAdminID, []uuid.UUID{employeeRoleID})
		if err == nil {
			t.Errorf("expected stripping super_admin role from last super admin to fail")
		}
	}

	// 14. Privilege Escalation Prevention Tests
	adminActor := &rbac.Principal{
		UserID:      uuid.New(),
		Roles:       []string{"admin"},
		Permissions: map[string]struct{}{"user.create": {}, "user.update": {}},
	}
	// Admin trying to assign super_admin role during user creation
	_, err = userSvc.Create(ctx, adminActor, CreateParams{
		Email:   fmt.Sprintf("esc_%d@faceclock.local", time.Now().UnixNano()),
		RoleIDs: []uuid.UUID{superAdminRoleID},
	})
	if err == nil {
		t.Errorf("expected privilege escalation error on Create")
	}

	// Admin trying to assign super_admin role via AssignRoles
	err = userSvc.AssignRoles(ctx, adminActor, targetUserID, []uuid.UUID{superAdminRoleID})
	if err == nil {
		t.Errorf("expected privilege escalation error on AssignRoles")
	}

	// 15. Handler Error Paths (400, 401, 404, 422)
	badRCtx := chi.NewRouteContext()
	badRCtx.URLParams.Add("id", "invalid-uuid")

	// GetByID 400
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/bad", nil).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, badRCtx))
	w = httptest.NewRecorder()
	handler.GetByID(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad id in GetByID, got %d", w.Code)
	}

	// Update 400
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/users/bad", bytes.NewReader([]byte(`{}`))).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, badRCtx))
	w = httptest.NewRecorder()
	handler.Update(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad id in Update, got %d", w.Code)
	}

	// Delete 400
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/users/bad", nil).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, badRCtx))
	w = httptest.NewRecorder()
	handler.Delete(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad id in Delete, got %d", w.Code)
	}

	// ResetPassword 400
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/bad/reset-password", bytes.NewReader([]byte(`{}`))).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, badRCtx))
	w = httptest.NewRecorder()
	handler.ResetPassword(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad id in ResetPassword, got %d", w.Code)
	}

	// UpdateStatus 400
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/users/bad/status", bytes.NewReader([]byte(`{}`))).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, badRCtx))
	w = httptest.NewRecorder()
	handler.UpdateStatus(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad id in UpdateStatus, got %d", w.Code)
	}

	// AssignRoles 400
	req = httptest.NewRequest(http.MethodPut, "/api/v1/users/bad/roles", bytes.NewReader([]byte(`{}`))).
		WithContext(context.WithValue(rbac.WithPrincipal(context.Background(), superActor), chi.RouteCtxKey, badRCtx))
	w = httptest.NewRecorder()
	handler.AssignRoles(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad id in AssignRoles, got %d", w.Code)
	}

	// Create 401 (unauthenticated)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(hBody))
	w = httptest.NewRecorder()
	handler.Create(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated Create, got %d", w.Code)
	}

	// AssignRoles 401 (unauthenticated)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/users/"+hUserID.String()+"/roles", bytes.NewReader(assignBody)).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.AssignRoles(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated AssignRoles, got %d", w.Code)
	}

	// Service Delete not found
	err = userSvc.Delete(ctx, uuid.New())
	if err == nil {
		t.Errorf("expected error deleting non-existent user")
	}

	// Service Update not found
	_, err = userSvc.Update(ctx, uuid.New(), UpdateParams{})
	if err == nil {
		t.Errorf("expected error updating non-existent user")
	}

	// Service UpdateStatus not found
	_, err = userSvc.UpdateStatus(ctx, uuid.New(), true)
	if err == nil {
		t.Errorf("expected error updating status of non-existent user")
	}

	// Service ResetPassword not found
	_, err = userSvc.ResetPassword(ctx, uuid.New(), nil)
	if err == nil {
		t.Errorf("expected error resetting password of non-existent user")
	}
}
