package httpx_test

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
)

func dummyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestAllRoutesHaveGuards(t *testing.T) {
	dummyMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
	dummyRBAC := func(perm string) func(http.Handler) http.Handler {
		return dummyMw
	}
	dummyRBACAny := func(perms ...string) func(http.Handler) http.Handler {
		return dummyMw
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	deps := httpx.RouterDeps{
		Logger:             logger,
		CORSAllowedOrigins: []string{"*"},
		RequestTimeout:     5 * time.Second,
		MaxBodyBytes:       1024 * 1024,
		HealthzHandler:     dummyHandler,
		ReadyzHandler:      dummyHandler,
		VersionHandler:     dummyHandler,
		AuthMiddleware:     dummyMw,
		RBACMiddleware:     dummyRBAC,
		RBACAnyMiddleware:  dummyRBACAny,
		Handlers: httpx.Handlers{
			AuthLogin:             dummyHandler,
			AuthRefresh:           dummyHandler,
			AuthLogout:            dummyHandler,
			AuthLogoutAll:         dummyHandler,
			AuthGetMe:             dummyHandler,
			AuthChangePassword:    dummyHandler,
			EmployeeList:          dummyHandler,
			EmployeeCreate:        dummyHandler,
			EmployeeGetMe:         dummyHandler,
			EmployeeGetByID:       dummyHandler,
			EmployeeUpdate:        dummyHandler,
			EmployeeDelete:        dummyHandler,
			UserList:              dummyHandler,
			UserCreate:            dummyHandler,
			UserGetByID:           dummyHandler,
			UserUpdate:            dummyHandler,
			UserDelete:            dummyHandler,
			UserUpdateStatus:      dummyHandler,
			UserAssignRoles:       dummyHandler,
			UserResetPassword:     dummyHandler,
			RoleList:              dummyHandler,
			RoleCreate:            dummyHandler,
			RoleGetByID:           dummyHandler,
			RoleUpdate:            dummyHandler,
			RoleDelete:            dummyHandler,
			RoleAssignPermissions: dummyHandler,
			PermissionList:        dummyHandler,
			SettingsList:          dummyHandler,
			SettingsGetByKey:      dummyHandler,
			SettingsUpdate:        dummyHandler,
			AuditQuery:            dummyHandler,
			EmployeeFaceEnroll:    dummyHandler,
			AttendanceClockIn:     dummyHandler,
			AttendanceClockOut:    dummyHandler,

			// Fase 3
			ConsentGetDocument:          dummyHandler,
			ConsentGetMe:                dummyHandler,
			ConsentGrant:                dummyHandler,
			ConsentWithdraw:             dummyHandler,
			ConsentGetEmployee:          dummyHandler,
			ConsentAdminRecord:          dummyHandler,
			FaceEnrollmentCreate:        dummyHandler,
			FaceEnrollmentGet:           dummyHandler,
			FaceEnrollmentUploadPhoto:   dummyHandler,
			FaceEnrollmentDeletePhoto:   dummyHandler,
			FaceEnrollmentCommit:        dummyHandler,
			FaceEnrollmentCancel:        dummyHandler,
			FaceReferenceListByEmployee: dummyHandler,
			FaceReferenceGetPhoto:       dummyHandler,
			FaceReferenceDeactivate:     dummyHandler,
			FaceReferenceDeleteAll:      dummyHandler,
			FaceEnrollmentStatusMe:      dummyHandler,
			FaceReindexCreateJob:        dummyHandler,
			FaceReindexListJobs:         dummyHandler,
			FaceReindexGetJob:           dummyHandler,
			FaceReindexCancelJob:        dummyHandler,
		},
	}

	r := httpx.NewRouter(deps)
	chiRouter, ok := r.(chi.Routes)
	if !ok {
		t.Fatalf("expected router to implement chi.Routes")
	}

	// 1. Verify every entry in RouteRegistry has a non-empty, valid guard.
	for routeKey, guard := range httpx.RouteRegistry {
		if guard == "" {
			t.Errorf("route %s has empty guard", routeKey)
		}
	}

	// 2. Walk chi routes and ensure 100% parity with RouteRegistry.
	walkedRoutes := make(map[string]bool)
	err := chi.Walk(chiRouter, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		// Clean route string: Chi appends "/*" or trailing slashes in some nested groups
		route = strings.TrimSuffix(route, "/")
		if route == "" {
			route = "/"
		}
		key := method + " " + route

		// Chi walks HEAD automatically when GET is registered; ignore HEAD if not explicitly in registry
		if method == http.MethodHead {
			return nil
		}

		walkedRoutes[key] = true
		guard, found := httpx.RouteRegistry[key]
		if !found {
			t.Errorf("unregistered route found in router: %s", key)
			return nil
		}

		if guard == "" {
			t.Errorf("route %s has an empty guard", key)
		}

		return nil
	})

	if err != nil {
		t.Fatalf("chi.Walk failed: %v", err)
	}

	// 3. Ensure all registered routes were actually mounted in the router.
	for routeKey, guard := range httpx.RouteRegistry {
		if !walkedRoutes[routeKey] {
			t.Errorf("registered route was not mounted in router: %s (guard: %s)", routeKey, guard)
		}
	}

	// 4. Validate exact count of Fase 0 + Fase 1 + Fase 2 + Fase 3 routes.
	// Total expected: 3 (base) + 31 (Fase 1 API v1) + 3 (Fase 2 API v1) + 21 (Fase 3 API v1) = 58 routes in registry
	expectedCount := 58
	if len(httpx.RouteRegistry) != expectedCount {
		t.Errorf("expected %d routes in registry, got %d", expectedCount, len(httpx.RouteRegistry))
	}
}
