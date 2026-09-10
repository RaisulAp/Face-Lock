package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/auth"
	"github.com/google/uuid"
)

type MatrixTestCase struct {
	Index          int
	Method         string
	Path           string
	GetBody        func(app *TestApp, targetID uuid.UUID) []byte
	GetPath        func(targetID uuid.UUID) string
	ExpectAnon     int
	ExpectEmployee int
	ExpectAdmin    int
	ExpectSuper    int
}

func TestRBACMatrix(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Pool.Close()

	ctx := context.Background()

	// Prepare principals
	superUser := app.EnsureUserWithRole(t, "super_admin")
	adminUser := app.EnsureUserWithRole(t, "admin")
	empUser := app.EnsureUserWithRole(t, "employee")

	// Prepare target resources
	// 1. Target employee (other than empUser)
	var targetEmpID uuid.UUID
	err := app.Pool.QueryRow(ctx, `
		INSERT INTO employees (employee_number, full_name, department, email, employment_status)
		VALUES ($1, 'Target Employee', 'Engineering', $2, 'active')
		RETURNING id
	`, fmt.Sprintf("EMP-TGT-%d", time.Now().UnixNano()%1000000), fmt.Sprintf("tgt_emp_%d@faceclock.local", time.Now().UnixNano())).Scan(&targetEmpID)
	if err != nil {
		t.Fatalf("failed creating target employee: %v", err)
	}

	// 2. Target user (other than empUser/adminUser/superUser)
	passHash, _ := auth.HashPassword("TargetPass123!")
	var targetUserID uuid.UUID
	err = app.Pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, employee_id, is_active, must_change_password)
		VALUES ($1, $2, $3, true, false)
		RETURNING id
	`, fmt.Sprintf("tgt_user_%d@faceclock.local", time.Now().UnixNano()), passHash, targetEmpID).Scan(&targetUserID)
	if err != nil {
		t.Fatalf("failed creating target user: %v", err)
	}
	var empRoleID uuid.UUID
	_ = app.Pool.QueryRow(ctx, "SELECT id FROM roles WHERE name = 'employee'").Scan(&empRoleID)
	_, _ = app.Pool.Exec(ctx, "INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)", targetUserID, empRoleID)

	// 3. Target custom role (for role update/delete/assign permissions)
	var targetRoleID uuid.UUID
	err = app.Pool.QueryRow(ctx, `
		INSERT INTO roles (name, display_name, description, is_system)
		VALUES ($1, 'Target Custom Role', 'Description', false)
		RETURNING id
	`, fmt.Sprintf("tgt_role_%d", time.Now().UnixNano())).Scan(&targetRoleID)
	if err != nil {
		t.Fatalf("failed creating target role: %v", err)
	}

	// Get a sample permission ID for role assign permissions
	var samplePermID uuid.UUID
	_ = app.Pool.QueryRow(ctx, "SELECT id FROM permissions WHERE name = 'employee.read_self'").Scan(&samplePermID)

	// Define all 31 endpoints explicitly as data table
	testCases := []MatrixTestCase{
		{
			Index:      1,
			Method:     http.MethodPost,
			Path:       "/api/v1/auth/login",
			GetBody:    func(_ *TestApp, _ uuid.UUID) []byte { return []byte(`{}`) },
			ExpectAnon: http.StatusUnprocessableEntity,
			// For authenticated users, login route is public and handles body validation (422)
			ExpectEmployee: http.StatusUnprocessableEntity,
			ExpectAdmin:    http.StatusUnprocessableEntity,
			ExpectSuper:    http.StatusUnprocessableEntity,
		},
		{
			Index:      2,
			Method:     http.MethodPost,
			Path:       "/api/v1/auth/refresh",
			GetBody:    func(_ *TestApp, _ uuid.UUID) []byte { return []byte(`{}`) },
			ExpectAnon: http.StatusUnprocessableEntity,
			// refresh route is public and handles body validation (422)
			ExpectEmployee: http.StatusUnprocessableEntity,
			ExpectAdmin:    http.StatusUnprocessableEntity,
			ExpectSuper:    http.StatusUnprocessableEntity,
		},
		{
			Index:          3,
			Method:         http.MethodPost,
			Path:           "/api/v1/auth/logout",
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusNoContent,
			ExpectAdmin:    http.StatusNoContent,
			ExpectSuper:    http.StatusNoContent,
		},
		{
			Index:          4,
			Method:         http.MethodPost,
			Path:           "/api/v1/auth/logout-all",
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusNoContent,
			ExpectAdmin:    http.StatusNoContent,
			ExpectSuper:    http.StatusNoContent,
		},
		{
			Index:          5,
			Method:         http.MethodGet,
			Path:           "/api/v1/auth/me",
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusOK,
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:  6,
			Method: http.MethodPost,
			Path:   "/api/v1/auth/change-password",
			GetBody: func(_ *TestApp, _ uuid.UUID) []byte {
				b, _ := json.Marshal(map[string]string{
					"current_password": "TestPass123!",
					"new_password":     "NewSecretPass123!",
				})
				return b
			},
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusOK,
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:          7,
			Method:         http.MethodGet,
			Path:           "/api/v1/employees",
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:  8,
			Method: http.MethodPost,
			Path:   "/api/v1/employees",
			GetBody: func(_ *TestApp, _ uuid.UUID) []byte {
				b, _ := json.Marshal(map[string]string{
					"employee_number": fmt.Sprintf("E-CRE-%d", time.Now().UnixNano()%1000000),
					"full_name":       "New Employee Name",
					"department":      "IT",
				})
				return b
			},
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusCreated,
			ExpectSuper:    http.StatusCreated,
		},
		{
			Index:          9,
			Method:         http.MethodGet,
			Path:           "/api/v1/employees/me",
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusOK,
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:  10,
			Method: http.MethodGet,
			GetPath: func(id uuid.UUID) string {
				return fmt.Sprintf("/api/v1/employees/%s", targetEmpID)
			},
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusNotFound, // § 10.11: accessing other employee returns 404
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:  11,
			Method: http.MethodPatch,
			GetPath: func(id uuid.UUID) string {
				return fmt.Sprintf("/api/v1/employees/%s", targetEmpID)
			},
			GetBody: func(_ *TestApp, _ uuid.UUID) []byte {
				return []byte(`{"full_name":"Updated Name"}`)
			},
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:  12,
			Method: http.MethodDelete,
			GetPath: func(id uuid.UUID) string {
				// Create temporary employee to delete
				var toDeleteID uuid.UUID
				_ = app.Pool.QueryRow(ctx, `
					INSERT INTO employees (employee_number, full_name, department, email, employment_status)
					VALUES ($1, 'Delete Me', 'Sales', $2, 'inactive')
					RETURNING id
				`, fmt.Sprintf("DEL-%d", time.Now().UnixNano()%1000000), fmt.Sprintf("del_%d@faceclock.local", time.Now().UnixNano())).Scan(&toDeleteID)
				return fmt.Sprintf("/api/v1/employees/%s", toDeleteID)
			},
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusNoContent,
			ExpectSuper:    http.StatusNoContent,
		},
		{
			Index:          13,
			Method:         http.MethodGet,
			Path:           "/api/v1/users",
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:  14,
			Method: http.MethodPost,
			Path:   "/api/v1/users",
			GetBody: func(_ *TestApp, _ uuid.UUID) []byte {
				b, _ := json.Marshal(map[string]any{
					"email":    fmt.Sprintf("create_user_%d@faceclock.local", time.Now().UnixNano()),
					"role_ids": []string{empRoleID.String()},
				})
				return b
			},
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusCreated,
			ExpectSuper:    http.StatusCreated,
		},
		{
			Index:  15,
			Method: http.MethodGet,
			GetPath: func(id uuid.UUID) string {
				return fmt.Sprintf("/api/v1/users/%s", targetUserID)
			},
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:  16,
			Method: http.MethodPatch,
			GetPath: func(id uuid.UUID) string {
				return fmt.Sprintf("/api/v1/users/%s", targetUserID)
			},
			GetBody: func(_ *TestApp, _ uuid.UUID) []byte {
				return fmt.Appendf(nil, `{"email":"updated_%d@faceclock.local"}`, time.Now().UnixNano())
			},
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:  17,
			Method: http.MethodDelete,
			GetPath: func(id uuid.UUID) string {
				// Create temporary user to delete
				var toDeleteUserID uuid.UUID
				_ = app.Pool.QueryRow(ctx, `
					INSERT INTO users (email, password_hash, is_active)
					VALUES ($1, 'hash', false)
					RETURNING id
				`, fmt.Sprintf("u_del_%d@faceclock.local", time.Now().UnixNano())).Scan(&toDeleteUserID)
				return fmt.Sprintf("/api/v1/users/%s", toDeleteUserID)
			},
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusForbidden, // Admin lacks user.delete (§ 4.8 / 000006)
			ExpectSuper:    http.StatusNoContent,
		},
		{
			Index:  18,
			Method: http.MethodPatch,
			GetPath: func(id uuid.UUID) string {
				return fmt.Sprintf("/api/v1/users/%s/status", targetUserID)
			},
			GetBody: func(_ *TestApp, _ uuid.UUID) []byte {
				return []byte(`{"is_active":true}`)
			},
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:  19,
			Method: http.MethodPut,
			GetPath: func(id uuid.UUID) string {
				return fmt.Sprintf("/api/v1/users/%s/roles", targetUserID)
			},
			GetBody: func(_ *TestApp, _ uuid.UUID) []byte {
				b, _ := json.Marshal(map[string]any{
					"role_ids": []string{empRoleID.String()},
				})
				return b
			},
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:  20,
			Method: http.MethodPost,
			GetPath: func(id uuid.UUID) string {
				return fmt.Sprintf("/api/v1/users/%s/reset-password", targetUserID)
			},
			GetBody: func(_ *TestApp, _ uuid.UUID) []byte {
				return []byte(`{"password":"NewResetPassword123!"}`)
			},
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:          21,
			Method:         http.MethodGet,
			Path:           "/api/v1/roles",
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:  22,
			Method: http.MethodPost,
			Path:   "/api/v1/roles",
			GetBody: func(_ *TestApp, _ uuid.UUID) []byte {
				b, _ := json.Marshal(map[string]any{
					"name":         fmt.Sprintf("r_new_%d", time.Now().UnixNano()),
					"display_name": "New Role",
				})
				return b
			},
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusForbidden, // Admin lacks role.create
			ExpectSuper:    http.StatusCreated,
		},
		{
			Index:  23,
			Method: http.MethodGet,
			GetPath: func(id uuid.UUID) string {
				return fmt.Sprintf("/api/v1/roles/%s", targetRoleID)
			},
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:  24,
			Method: http.MethodPatch,
			GetPath: func(id uuid.UUID) string {
				return fmt.Sprintf("/api/v1/roles/%s", targetRoleID)
			},
			GetBody: func(_ *TestApp, _ uuid.UUID) []byte {
				return []byte(`{"display_name":"Updated Role Name"}`)
			},
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusForbidden, // Admin lacks role.update
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:  25,
			Method: http.MethodDelete,
			GetPath: func(id uuid.UUID) string {
				var toDeleteRoleID uuid.UUID
				_ = app.Pool.QueryRow(ctx, `
					INSERT INTO roles (name, display_name, is_system)
					VALUES ($1, 'To Delete', false)
					RETURNING id
				`, fmt.Sprintf("del_role_%d", time.Now().UnixNano())).Scan(&toDeleteRoleID)
				return fmt.Sprintf("/api/v1/roles/%s", toDeleteRoleID)
			},
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusForbidden, // Admin lacks role.delete
			ExpectSuper:    http.StatusNoContent,
		},
		{
			Index:  26,
			Method: http.MethodPut,
			GetPath: func(id uuid.UUID) string {
				return fmt.Sprintf("/api/v1/roles/%s/permissions", targetRoleID)
			},
			GetBody: func(_ *TestApp, _ uuid.UUID) []byte {
				b, _ := json.Marshal(map[string]any{
					"permission_ids": []string{samplePermID.String()},
				})
				return b
			},
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusForbidden, // Admin lacks role.assign_permission
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:          27,
			Method:         http.MethodGet,
			Path:           "/api/v1/permissions",
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:          28,
			Method:         http.MethodGet,
			Path:           "/api/v1/settings",
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusOK, // Filtered to is_public = true
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:  29,
			Method: http.MethodGet,
			// Access a non-public setting: employee receives 404, admin/super receives 200
			Path:           "/api/v1/settings/face.similarity_threshold",
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusNotFound,
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:  30,
			Method: http.MethodPut,
			Path:   "/api/v1/settings/attendance.max_distance_meter",
			GetBody: func(_ *TestApp, _ uuid.UUID) []byte {
				return []byte(`{"value":150}`)
			},
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
		{
			Index:          31,
			Method:         http.MethodGet,
			Path:           "/api/v1/audit-logs",
			ExpectAnon:     http.StatusUnauthorized,
			ExpectEmployee: http.StatusForbidden,
			ExpectAdmin:    http.StatusOK,
			ExpectSuper:    http.StatusOK,
		},
	}

	if len(testCases) != 31 {
		t.Fatalf("RBAC matrix must have exactly 31 endpoints, got %d", len(testCases))
	}

	principals := []struct {
		Name        string
		User        *UserWithToken
		GetExpected func(tc MatrixTestCase) int
	}{
		{
			Name: "Anonymous",
			User: nil,
			GetExpected: func(tc MatrixTestCase) int {
				return tc.ExpectAnon
			},
		},
		{
			Name: "Employee",
			User: empUser,
			GetExpected: func(tc MatrixTestCase) int {
				return tc.ExpectEmployee
			},
		},
		{
			Name: "Admin",
			User: adminUser,
			GetExpected: func(tc MatrixTestCase) int {
				return tc.ExpectAdmin
			},
		},
		{
			Name: "SuperAdmin",
			User: superUser,
			GetExpected: func(tc MatrixTestCase) int {
				return tc.ExpectSuper
			},
		},
	}

	// Run all combinations: 4 principals × 31 endpoints = 124 tests
	for _, p := range principals {
		t.Run(p.Name, func(t *testing.T) {
			for _, tc := range testCases {
				path := tc.Path
				if tc.GetPath != nil {
					path = tc.GetPath(targetEmpID)
				}

				testName := fmt.Sprintf("#%02d_%s_%s", tc.Index, tc.Method, path)
				t.Run(testName, func(t *testing.T) {
					var bodyBytes []byte
					if tc.GetBody != nil {
						bodyBytes = tc.GetBody(app, targetEmpID)
					}

					req, err := http.NewRequest(tc.Method, path, bytes.NewReader(bodyBytes))
					if err != nil {
						t.Fatalf("failed to create request: %v", err)
					}
					req.Header.Set("Content-Type", "application/json")
					if p.User != nil {
						activeUser := p.User
						if tc.Index == 3 || tc.Index == 4 || tc.Index == 6 {
							activeUser = app.EnsureUserWithRole(t, p.User.Role)
						}
						req.Header.Set("Authorization", "Bearer "+activeUser.Token)
						req.AddCookie(&http.Cookie{
							Name:  auth.AccessTokenCookieName,
							Value: activeUser.Token,
							Path:  "/",
						})
					}

					rec, respBody := app.ExecuteRequest(req)
					expectedStatus := p.GetExpected(tc)

					// Security check on EVERY response (DoD 12)
					AssertNoSecretLeak(t, respBody)

					if rec.Code != expectedStatus {
						t.Errorf("[%s] %s %s: expected status %d, got %d. Body: %s",
							p.Name, tc.Method, path, expectedStatus, rec.Code, string(respBody))
					}
				})
			}
		})
	}
}
