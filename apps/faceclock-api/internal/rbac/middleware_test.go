package rbac

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestRequirePermission(t *testing.T) {
	tests := []struct {
		name           string
		principal      *Principal
		requiredPerm   string
		expectedStatus int
	}{
		{
			name:           "unauthenticated request",
			principal:      nil,
			requiredPerm:   PermEmployeeRead,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "user lacks permission",
			principal: &Principal{
				UserID:      uuid.New(),
				Roles:       []string{"employee"},
				Permissions: map[string]struct{}{PermAttendanceCheckin: {}},
			},
			requiredPerm:   PermEmployeeDelete,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "user possesses permission",
			principal: &Principal{
				UserID:      uuid.New(),
				Roles:       []string{"employee"},
				Permissions: map[string]struct{}{PermEmployeeRead: {}},
			},
			requiredPerm:   PermEmployeeRead,
			expectedStatus: http.StatusOK,
		},
		{
			name: "super admin possesses all permissions",
			principal: &Principal{
				UserID: uuid.New(),
				Roles:  []string{"super_admin"},
			},
			requiredPerm:   PermSettingsUpdate,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := RequirePermission(tt.requiredPerm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.principal != nil {
				req = req.WithContext(WithPrincipal(req.Context(), tt.principal))
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestRequireAnyPermission(t *testing.T) {
	tests := []struct {
		name           string
		principal      *Principal
		requiredPerms  []string
		expectedStatus int
	}{
		{
			name:           "unauthenticated",
			principal:      nil,
			requiredPerms:  []string{PermEmployeeRead, PermEmployeeReadSelf},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "matches none",
			principal: &Principal{
				UserID:      uuid.New(),
				Roles:       []string{"employee"},
				Permissions: map[string]struct{}{PermAttendanceCheckin: {}},
			},
			requiredPerms:  []string{PermEmployeeRead, PermEmployeeReadSelf},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "matches one",
			principal: &Principal{
				UserID:      uuid.New(),
				Roles:       []string{"employee"},
				Permissions: map[string]struct{}{PermEmployeeReadSelf: {}},
			},
			requiredPerms:  []string{PermEmployeeRead, PermEmployeeReadSelf},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := RequireAnyPermission(tt.requiredPerms...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.principal != nil {
				req = req.WithContext(WithPrincipal(req.Context(), tt.principal))
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestRequireSuperAdmin(t *testing.T) {
	tests := []struct {
		name           string
		principal      *Principal
		expectedStatus int
	}{
		{
			name:           "unauthenticated",
			principal:      nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "regular user",
			principal: &Principal{
				UserID: uuid.New(),
				Roles:  []string{"admin"},
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "super admin",
			principal: &Principal{
				UserID: uuid.New(),
				Roles:  []string{"super_admin"},
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := RequireSuperAdmin()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.principal != nil {
				req = req.WithContext(WithPrincipal(req.Context(), tt.principal))
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}
