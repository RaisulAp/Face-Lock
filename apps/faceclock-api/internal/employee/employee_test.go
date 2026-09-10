package employee

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

func TestEmployeeServiceAndHandler(t *testing.T) {
	pool := getTestDB(t)
	svc := NewService(pool)
	auditRec := audit.NewRecorder(pool)
	handler := NewHandler(svc, auditRec)

	ctx := context.Background()

	empNum := fmt.Sprintf("EMP-%d", time.Now().UnixNano()%1000000)
	empName := "Test Employee Alpha"

	// 1. Service Create
	joinDate := "2026-01-15"
	dept := "Engineering"
	pos := "Software Engineer"
	phone := "081234567890"
	email := "alpha@example.com"

	created, err := svc.Create(ctx, CreateParams{
		EmployeeNumber:   empNum,
		FullName:         empName,
		Department:       &dept,
		Position:         &pos,
		Phone:            &phone,
		Email:            &email,
		JoinDate:         &joinDate,
		EmploymentStatus: "active",
	})
	if err != nil {
		t.Fatalf("svc.Create failed: %v", err)
	}
	if created.EmployeeNumber != empNum {
		t.Errorf("expected empNum %s, got %s", empNum, created.EmployeeNumber)
	}

	// 2. Duplicate EmployeeNumber
	_, err = svc.Create(ctx, CreateParams{
		EmployeeNumber:   empNum,
		FullName:         "Duplicate",
		Department:       &dept,
		Position:         &pos,
		JoinDate:         &joinDate,
		EmploymentStatus: "active",
	})
	if err == nil {
		t.Fatalf("expected duplicate employee number error")
	}
	appErr, ok := err.(*httpx.AppError)
	if !ok || appErr.Code != httpx.CodeEmployeeNumberTaken {
		t.Errorf("expected CodeEmployeeNumberTaken, got %v", err)
	}

	// 3. Service GetByID
	fetched, err := svc.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("svc.GetByID failed: %v", err)
	}
	if fetched.FullName != empName {
		t.Errorf("expected full name %s, got %s", empName, fetched.FullName)
	}

	// GetByID not found
	_, err = svc.GetByID(ctx, uuid.New())
	if err == nil {
		t.Fatalf("expected not found error")
	}

	// 4. Service List
	hasUserFalse := false
	list, meta, err := svc.List(ctx, Filter{
		Search:           empNum,
		Department:       "Engineering",
		EmploymentStatus: "active",
		HasUser:          &hasUserFalse,
		Page:             1,
		PerPage:          10,
	})
	if err != nil {
		t.Fatalf("svc.List failed: %v", err)
	}
	if meta.Total < 1 || len(list) < 1 {
		t.Errorf("expected at least 1 employee in list, got %d", len(list))
	}

	// 5. Service Update
	newTitle := "Lead Software Engineer"
	updated, err := svc.Update(ctx, created.ID, UpdateParams{
		Position: &newTitle,
	})
	if err != nil {
		t.Fatalf("svc.Update failed: %v", err)
	}
	if updated.Position == nil || *updated.Position != newTitle {
		t.Errorf("expected position %s, got %v", newTitle, updated.Position)
	}

	// 6. Handler List
	req := httptest.NewRequest(http.MethodGet, "/api/v1/employees?page=1&per_page=10&has_user=false", nil)
	w := httptest.NewRecorder()
	handler.List(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for handler.List, got %d", w.Code)
	}

	// 7. Handler Create
	newEmpNum := fmt.Sprintf("EMP-%d", (time.Now().UnixNano()+1)%1000000)
	hJoinDate := "2026-02-01"
	hDept := "Product"
	hPos := "PM"
	cBody, _ := json.Marshal(CreateParams{
		EmployeeNumber:   newEmpNum,
		FullName:         "Handler Test Employee",
		Department:       &hDept,
		Position:         &hPos,
		JoinDate:         &hJoinDate,
		EmploymentStatus: "inactive",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/employees", bytes.NewReader(cBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for handler.Create, got %d, body: %s", w.Code, w.Body.String())
	}
	var createdHandlerResp struct {
		Data Employee `json:"data"`
	}
	_ = json.NewDecoder(w.Body).Decode(&createdHandlerResp)
	hEmpID := createdHandlerResp.Data.ID

	// Create with invalid JSON
	req = httptest.NewRequest(http.MethodPost, "/api/v1/employees", bytes.NewReader([]byte(`{bad`)))
	w = httptest.NewRecorder()
	handler.Create(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad json in Create, got %d", w.Code)
	}

	// Create with validation error
	req = httptest.NewRequest(http.MethodPost, "/api/v1/employees", bytes.NewReader([]byte(`{}`)))
	w = httptest.NewRecorder()
	handler.Create(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for validation error in Create, got %d", w.Code)
	}

	// 8. Handler GetByID
	adminPrincipal := &rbac.Principal{
		UserID:      uuid.New(),
		Roles:       []string{"admin"},
		Permissions: map[string]struct{}{rbac.PermEmployeeRead: {}},
	}
	rCtx := chi.NewRouteContext()
	rCtx.URLParams.Add("id", hEmpID.String())
	req = httptest.NewRequest(http.MethodGet, "/api/v1/employees/"+hEmpID.String(), nil).
		WithContext(context.WithValue(rbac.WithPrincipal(context.Background(), adminPrincipal), chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.GetByID(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for handler.GetByID, got %d", w.Code)
	}

	// GetByID with invalid UUID
	rCtxBad := chi.NewRouteContext()
	rCtxBad.URLParams.Add("id", "not-a-uuid")
	req = httptest.NewRequest(http.MethodGet, "/api/v1/employees/not-a-uuid", nil).
		WithContext(context.WithValue(rbac.WithPrincipal(context.Background(), adminPrincipal), chi.RouteCtxKey, rCtxBad))
	w = httptest.NewRecorder()
	handler.GetByID(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad uuid in GetByID, got %d", w.Code)
	}

	// 9. Handler GetByID privacy check:
	// Employee with only employee.read_self trying to read another employee's record -> must get 404 (not 403, for privacy)
	otherEmpID := uuid.New()
	empPrincipal := &rbac.Principal{
		UserID:      uuid.New(),
		Roles:       []string{"employee"},
		Permissions: map[string]struct{}{rbac.PermEmployeeReadSelf: {}},
		EmployeeID:  &otherEmpID,
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/employees/"+hEmpID.String(), nil).
		WithContext(context.WithValue(rbac.WithPrincipal(context.Background(), empPrincipal), chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.GetByID(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-employee read attempt, got %d", w.Code)
	}

	// 10. Handler GetMe
	selfPrincipal := &rbac.Principal{
		UserID:      uuid.New(),
		Roles:       []string{"employee"},
		Permissions: map[string]struct{}{rbac.PermEmployeeReadSelf: {}},
		EmployeeID:  &hEmpID,
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/employees/me", nil).
		WithContext(rbac.WithPrincipal(context.Background(), selfPrincipal))
	w = httptest.NewRecorder()
	handler.GetMe(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for handler.GetMe, got %d", w.Code)
	}

	// GetMe with unlinked user
	noEmpPrincipal := &rbac.Principal{UserID: uuid.New()}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/employees/me", nil).
		WithContext(rbac.WithPrincipal(context.Background(), noEmpPrincipal))
	w = httptest.NewRecorder()
	handler.GetMe(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for GetMe unlinked employee, got %d", w.Code)
	}

	// 11. Handler Update
	upPos := "Senior PM"
	upBody, _ := json.Marshal(UpdateParams{Position: &upPos})
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/employees/"+hEmpID.String(), bytes.NewReader(upBody)).
		WithContext(context.WithValue(rbac.WithPrincipal(context.Background(), adminPrincipal), chi.RouteCtxKey, rCtx))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.Update(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for handler.Update, got %d", w.Code)
	}

	// Update with invalid UUID
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/employees/bad-uuid", bytes.NewReader(upBody)).
		WithContext(context.WithValue(rbac.WithPrincipal(context.Background(), adminPrincipal), chi.RouteCtxKey, rCtxBad))
	w = httptest.NewRecorder()
	handler.Update(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad uuid in Update, got %d", w.Code)
	}

	// 12. Handler Delete
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/employees/"+hEmpID.String(), nil).
		WithContext(context.WithValue(rbac.WithPrincipal(context.Background(), adminPrincipal), chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.Delete(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for handler.Delete, got %d", w.Code)
	}

	// Delete with invalid UUID
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/employees/bad-uuid", nil).
		WithContext(context.WithValue(rbac.WithPrincipal(context.Background(), adminPrincipal), chi.RouteCtxKey, rCtxBad))
	w = httptest.NewRecorder()
	handler.Delete(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad uuid in Delete, got %d", w.Code)
	}

	// Delete not found
	err = svc.Delete(ctx, uuid.New())
	if err == nil {
		t.Errorf("expected error deleting non-existent employee")
	}

	// Filter with has_user=true
	hasUserTrue := true
	_, _, err = svc.List(ctx, Filter{HasUser: &hasUserTrue})
	if err != nil {
		t.Fatalf("svc.List has_user=true failed: %v", err)
	}

	// 13. Service Delete with active user guard
	// Create employee and link an active user to it
	gDept := "HR"
	gPos := "HR Officer"
	gJoinDate := "2026-01-01"
	guardEmp, err := svc.Create(ctx, CreateParams{
		EmployeeNumber:   fmt.Sprintf("EMP-%d", (time.Now().UnixNano()+2)%1000000),
		FullName:         "Active User Emp",
		Department:       &gDept,
		Position:         &gPos,
		JoinDate:         &gJoinDate,
		EmploymentStatus: "active",
	})
	if err != nil {
		t.Fatalf("failed to create guard emp: %v", err)
	}
	activeUserID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, is_active, employee_id, created_at, updated_at)
		VALUES ($1, $2, 'dummy', true, $3, NOW(), NOW())`,
		activeUserID, fmt.Sprintf("user_%d@faceclock.local", time.Now().UnixNano()), guardEmp.ID)
	if err != nil {
		t.Fatalf("failed to insert active user: %v", err)
	}
	defer pool.Exec(ctx, "DELETE FROM users WHERE id = $1", activeUserID)

	err = svc.Delete(ctx, guardEmp.ID)
	if err == nil {
		t.Fatalf("expected CodeEmployeeHasActiveUser when deleting employee with active user")
	}
	appErr, ok = err.(*httpx.AppError)
	if !ok || appErr.Code != httpx.CodeEmployeeHasActiveUser {
		t.Errorf("expected CodeEmployeeHasActiveUser, got %v", err)
	}
}
