package httpx

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx/middleware"
)

// Handlers contains all the HTTP handler functions for Fase 1, 2, and 3.
type Handlers struct {
	// Auth
	AuthLogin          http.HandlerFunc
	AuthRefresh        http.HandlerFunc
	AuthLogout         http.HandlerFunc
	AuthLogoutAll      http.HandlerFunc
	AuthGetMe          http.HandlerFunc
	AuthChangePassword http.HandlerFunc

	// Employees
	EmployeeList    http.HandlerFunc
	EmployeeCreate  http.HandlerFunc
	EmployeeGetMe   http.HandlerFunc
	EmployeeGetByID http.HandlerFunc
	EmployeeUpdate  http.HandlerFunc
	EmployeeDelete  http.HandlerFunc

	// Users
	UserList          http.HandlerFunc
	UserCreate        http.HandlerFunc
	UserGetByID       http.HandlerFunc
	UserUpdate        http.HandlerFunc
	UserDelete        http.HandlerFunc
	UserUpdateStatus  http.HandlerFunc
	UserAssignRoles   http.HandlerFunc
	UserResetPassword http.HandlerFunc

	// Roles & Permissions
	RoleList              http.HandlerFunc
	RoleCreate            http.HandlerFunc
	RoleGetByID           http.HandlerFunc
	RoleUpdate            http.HandlerFunc
	RoleDelete            http.HandlerFunc
	RoleAssignPermissions http.HandlerFunc
	PermissionList        http.HandlerFunc

	// Settings
	SettingsList     http.HandlerFunc
	SettingsGetByKey http.HandlerFunc
	SettingsUpdate   http.HandlerFunc

	// Audit Logs
	AuditQuery http.HandlerFunc

	// Fase 2: Biometrics & Attendance (Legacy Stubs)
	EmployeeFaceEnroll http.HandlerFunc
	AttendanceClockIn  http.HandlerFunc
	AttendanceClockOut http.HandlerFunc

	// Fase 3: Biometric Consent (UU PDP No. 27/2022)
	ConsentGetDocument http.HandlerFunc
	ConsentGetMe       http.HandlerFunc
	ConsentGrant       http.HandlerFunc
	ConsentWithdraw    http.HandlerFunc
	ConsentGetEmployee http.HandlerFunc
	ConsentAdminRecord http.HandlerFunc

	// Fase 3: Multi-Photo Face Enrollment Sessions
	FaceEnrollmentCreate      http.HandlerFunc
	FaceEnrollmentGet         http.HandlerFunc
	FaceEnrollmentUploadPhoto http.HandlerFunc
	FaceEnrollmentDeletePhoto http.HandlerFunc
	FaceEnrollmentCommit      http.HandlerFunc
	FaceEnrollmentCancel      http.HandlerFunc

	// Fase 3: Face References & Lifecycle Management
	FaceReferenceListByEmployee http.HandlerFunc
	FaceReferenceGetPhoto       http.HandlerFunc
	FaceReferenceDeactivate     http.HandlerFunc
	FaceReferenceDeleteAll      http.HandlerFunc
	FaceEnrollmentStatusMe      http.HandlerFunc

	// Fase 3: Face Reindex Jobs
	FaceReindexCreateJob http.HandlerFunc
	FaceReindexListJobs  http.HandlerFunc
	FaceReindexGetJob    http.HandlerFunc
	FaceReindexCancelJob http.HandlerFunc

	// Fase 4: Attendance Engine (#53 - #65)
	AttendancesClockIn   http.HandlerFunc
	AttendancesClockOut  http.HandlerFunc
	AttendanceContext    http.HandlerFunc
	AttendanceMe         http.HandlerFunc
	AttendanceMeToday    http.HandlerFunc
	AttendanceGetByID    http.HandlerFunc
	AttendanceGetPhoto   http.HandlerFunc
	AttendanceList       http.HandlerFunc
	AttendancePending    http.HandlerFunc
	AttendanceApprove    http.HandlerFunc
	AttendanceReject     http.HandlerFunc
	AttendanceBulkReview http.HandlerFunc
	AttendanceAttempts   http.HandlerFunc

	// Fase 4: Office Locations (#66 - #70)
	LocationList    http.HandlerFunc
	LocationCreate  http.HandlerFunc
	LocationGetByID http.HandlerFunc
	LocationUpdate  http.HandlerFunc
	LocationDelete  http.HandlerFunc

	// Fase 5: Admin Panel & Configuration (#71 - #73)
	AttendanceSummary         http.HandlerFunc
	AttendanceExport          http.HandlerFunc
	SettingsFaceQualityStatus http.HandlerFunc
}

// RouterDeps is every dependency the router needs to wire the middleware
// chain and mount all endpoints.
type RouterDeps struct {
	Logger             *slog.Logger
	CORSAllowedOrigins []string
	RequestTimeout     time.Duration
	MaxBodyBytes       int64

	HealthzHandler http.HandlerFunc
	ReadyzHandler  http.HandlerFunc
	VersionHandler http.HandlerFunc

	AuthMiddleware    func(http.Handler) http.Handler
	RBACMiddleware    func(permission string) func(http.Handler) http.Handler
	RBACAnyMiddleware func(permissions ...string) func(http.Handler) http.Handler
	ConsentMiddleware func(resolve func(*http.Request) (uuid.UUID, error)) func(http.Handler) http.Handler

	Handlers Handlers
}

// NewRouter builds the chi router with the middleware chain locked in
// Fase 0 § 2.6:
//
//	RequestID → RealIP → StructuredLogger → Recoverer → CORS
//	  → Timeout(30s) → BodyLimit(10MB) → RateLimit
//	    → [Fase 1] Authenticate → [Fase 1] RequirePermission(...) → [Fase 3] RequireConsent(...)
//	      → handler
//
// The bracketed steps do not exist yet — Fase 0 intentionally stops at
// RateLimit. Route groups that will carry Authenticate/RequirePermission
// starting Fase 1 are prepared below (see mountAPIv1) so that fase only has
// to add middleware to an existing group, not invent the mount point.
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.StructuredLogger(deps.Logger))
	r.Use(middleware.Recoverer(deps.Logger, writePanicResponse))
	r.Use(middleware.CORS(deps.CORSAllowedOrigins))
	r.Use(middleware.CSRFProtection(deps.CORSAllowedOrigins, writeCSRFFail))
	r.Use(middleware.Timeout(deps.RequestTimeout))
	r.Use(middleware.BodyLimit(deps.MaxBodyBytes))
	r.Use(middleware.RateLimit(defaultRateLimit, defaultRateLimitWindow, writeRateLimitResponse))

	r.NotFound(notFoundHandler)
	r.MethodNotAllowed(methodNotAllowedHandler)

	// Liveness/readiness are intentionally outside /api/v1 and outside any
	// future auth group — an orchestrator must be able to reach them with
	// zero credentials (Fase 0 § 4).
	r.Get("/healthz", deps.HealthzHandler)
	r.Get("/readyz", deps.ReadyzHandler)

	mountAPIv1(r, deps)

	return r
}

// defaultRateLimit is deliberately generous for Fase 0 — it exists to prove
// the middleware slot works, not to tune production traffic shaping (that
// happens per-route starting Fase 1, e.g. login attempts).
const (
	defaultRateLimit       = 300
	defaultRateLimitWindow = time.Minute
)

// mountAPIv1 mounts all versioned API routes under /api/v1.
func mountAPIv1(r chi.Router, deps RouterDeps) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/version", deps.VersionHandler)

		// Public Auth
		if deps.Handlers.AuthLogin != nil {
			r.Post("/auth/login", deps.Handlers.AuthLogin)
		}
		if deps.Handlers.AuthRefresh != nil {
			r.Post("/auth/refresh", deps.Handlers.AuthRefresh)
		}

		// Authenticated Routes Group
		r.Group(func(r chi.Router) {
			if deps.AuthMiddleware != nil {
				r.Use(deps.AuthMiddleware)
			}

			// Auth (Authenticated)
			if deps.Handlers.AuthLogout != nil {
				r.Post("/auth/logout", deps.Handlers.AuthLogout)
			}
			if deps.Handlers.AuthLogoutAll != nil {
				r.Post("/auth/logout-all", deps.Handlers.AuthLogoutAll)
			}
			if deps.Handlers.AuthGetMe != nil {
				r.Get("/auth/me", deps.Handlers.AuthGetMe)
			}
			if deps.Handlers.AuthChangePassword != nil {
				r.Post("/auth/change-password", deps.Handlers.AuthChangePassword)
			}

			// Employees
			if deps.Handlers.EmployeeList != nil {
				r.With(rbacGuard(deps, "employee.read")).Get("/employees", deps.Handlers.EmployeeList)
			}
			if deps.Handlers.EmployeeCreate != nil {
				r.With(rbacGuard(deps, "employee.create")).Post("/employees", deps.Handlers.EmployeeCreate)
			}
			if deps.Handlers.EmployeeGetMe != nil {
				r.With(rbacGuard(deps, "employee.read_self")).Get("/employees/me", deps.Handlers.EmployeeGetMe)
			}
			if deps.Handlers.EmployeeGetByID != nil {
				r.With(rbacAnyGuard(deps, "employee.read", "employee.read_self")).Get("/employees/{id}", deps.Handlers.EmployeeGetByID)
			}
			if deps.Handlers.EmployeeUpdate != nil {
				r.With(rbacGuard(deps, "employee.update")).Patch("/employees/{id}", deps.Handlers.EmployeeUpdate)
			}
			if deps.Handlers.EmployeeDelete != nil {
				r.With(rbacGuard(deps, "employee.delete")).Delete("/employees/{id}", deps.Handlers.EmployeeDelete)
			}

			// Users
			if deps.Handlers.UserList != nil {
				r.With(rbacGuard(deps, "user.read")).Get("/users", deps.Handlers.UserList)
			}
			if deps.Handlers.UserCreate != nil {
				r.With(rbacGuard(deps, "user.create")).Post("/users", deps.Handlers.UserCreate)
			}
			if deps.Handlers.UserGetByID != nil {
				r.With(rbacGuard(deps, "user.read")).Get("/users/{id}", deps.Handlers.UserGetByID)
			}
			if deps.Handlers.UserUpdate != nil {
				r.With(rbacGuard(deps, "user.update")).Patch("/users/{id}", deps.Handlers.UserUpdate)
			}
			if deps.Handlers.UserDelete != nil {
				r.With(rbacGuard(deps, "user.delete")).Delete("/users/{id}", deps.Handlers.UserDelete)
			}
			if deps.Handlers.UserUpdateStatus != nil {
				r.With(rbacGuard(deps, "user.update")).Patch("/users/{id}/status", deps.Handlers.UserUpdateStatus)
			}
			if deps.Handlers.UserAssignRoles != nil {
				r.With(rbacGuard(deps, "user.assign_role")).Put("/users/{id}/roles", deps.Handlers.UserAssignRoles)
			}
			if deps.Handlers.UserResetPassword != nil {
				r.With(rbacGuard(deps, "user.reset_password")).Post("/users/{id}/reset-password", deps.Handlers.UserResetPassword)
			}

			// Roles & Permissions
			if deps.Handlers.RoleList != nil {
				r.With(rbacGuard(deps, "role.read")).Get("/roles", deps.Handlers.RoleList)
			}
			if deps.Handlers.RoleCreate != nil {
				r.With(rbacGuard(deps, "role.create")).Post("/roles", deps.Handlers.RoleCreate)
			}
			if deps.Handlers.RoleGetByID != nil {
				r.With(rbacGuard(deps, "role.read")).Get("/roles/{id}", deps.Handlers.RoleGetByID)
			}
			if deps.Handlers.RoleUpdate != nil {
				r.With(rbacGuard(deps, "role.update")).Patch("/roles/{id}", deps.Handlers.RoleUpdate)
			}
			if deps.Handlers.RoleDelete != nil {
				r.With(rbacGuard(deps, "role.delete")).Delete("/roles/{id}", deps.Handlers.RoleDelete)
			}
			if deps.Handlers.RoleAssignPermissions != nil {
				r.With(rbacGuard(deps, "role.assign_permission")).Put("/roles/{id}/permissions", deps.Handlers.RoleAssignPermissions)
			}
			if deps.Handlers.PermissionList != nil {
				r.With(rbacGuard(deps, "permission.read")).Get("/permissions", deps.Handlers.PermissionList)
			}

			// Settings
			if deps.Handlers.SettingsFaceQualityStatus != nil {
				r.With(rbacGuard(deps, "settings.read")).Get("/settings/face-quality-status", deps.Handlers.SettingsFaceQualityStatus)
			}
			if deps.Handlers.SettingsList != nil {
				r.With(rbacGuard(deps, "settings.read")).Get("/settings", deps.Handlers.SettingsList)
			}
			if deps.Handlers.SettingsGetByKey != nil {
				r.With(rbacGuard(deps, "settings.read")).Get("/settings/{key}", deps.Handlers.SettingsGetByKey)
			}
			if deps.Handlers.SettingsUpdate != nil {
				r.With(rbacGuard(deps, "settings.update")).Put("/settings/{key}", deps.Handlers.SettingsUpdate)
			}

			// Audit Logs
			if deps.Handlers.AuditQuery != nil {
				r.With(rbacGuard(deps, "audit.read")).Get("/audit-logs", deps.Handlers.AuditQuery)
			}

			// Fase 2: Face Biometrics & Attendance
			if deps.Handlers.EmployeeFaceEnroll != nil {
				r.With(rbacAnyGuard(deps, "face.enroll_any", "face.enroll_self")).Post("/employees/{id}/face-enroll", deps.Handlers.EmployeeFaceEnroll)
			}
			if deps.Handlers.AttendanceClockIn != nil {
				r.With(rbacGuard(deps, "attendance.checkin")).Post("/attendance/clock-in", deps.Handlers.AttendanceClockIn)
			}
			if deps.Handlers.AttendanceClockOut != nil {
				r.With(rbacGuard(deps, "attendance.checkin")).Post("/attendance/clock-out", deps.Handlers.AttendanceClockOut)
			}

			// Fase 3: Biometric Consent (UU PDP No. 27/2022)
			if deps.Handlers.ConsentGetDocument != nil {
				r.Get("/consents/document", deps.Handlers.ConsentGetDocument)
			}
			if deps.Handlers.ConsentGetMe != nil {
				r.Get("/consents/me", deps.Handlers.ConsentGetMe)
			}
			if deps.Handlers.ConsentGrant != nil {
				r.With(rbacGuard(deps, "face.enroll_self")).Post("/consents", deps.Handlers.ConsentGrant)
			}
			if deps.Handlers.ConsentWithdraw != nil {
				r.Post("/consents/withdraw", deps.Handlers.ConsentWithdraw)
			}
			if deps.Handlers.ConsentGetEmployee != nil {
				r.With(rbacAnyGuard(deps, "face.read_any", "face.read_self")).Get("/employees/{id}/consent", deps.Handlers.ConsentGetEmployee)
			}
			if deps.Handlers.ConsentAdminRecord != nil {
				r.With(rbacGuard(deps, "face.enroll_any")).Post("/employees/{id}/consent", deps.Handlers.ConsentAdminRecord)
			}

			// Fase 3: Multi-Photo Face Enrollment Sessions
			if deps.Handlers.FaceEnrollmentCreate != nil {
				r.With(rbacAnyGuard(deps, "face.enroll_self", "face.enroll_any")).Post("/face/enrollments", deps.Handlers.FaceEnrollmentCreate)
			}
			if deps.Handlers.FaceEnrollmentGet != nil {
				r.With(rbacAnyGuard(deps, "face.enroll_self", "face.enroll_any")).Get("/face/enrollments/{id}", deps.Handlers.FaceEnrollmentGet)
			}
			if deps.Handlers.FaceEnrollmentUploadPhoto != nil {
				r.With(rbacAnyGuard(deps, "face.enroll_self", "face.enroll_any")).Post("/face/enrollments/{id}/photos", deps.Handlers.FaceEnrollmentUploadPhoto)
			}
			if deps.Handlers.FaceEnrollmentDeletePhoto != nil {
				r.With(rbacAnyGuard(deps, "face.enroll_self", "face.enroll_any")).Delete("/face/enrollments/{id}/photos/{pid}", deps.Handlers.FaceEnrollmentDeletePhoto)
			}
			if deps.Handlers.FaceEnrollmentCommit != nil {
				r.With(rbacAnyGuard(deps, "face.enroll_self", "face.enroll_any")).Post("/face/enrollments/{id}/commit", deps.Handlers.FaceEnrollmentCommit)
			}
			if deps.Handlers.FaceEnrollmentCancel != nil {
				r.With(rbacAnyGuard(deps, "face.enroll_self", "face.enroll_any")).Delete("/face/enrollments/{id}", deps.Handlers.FaceEnrollmentCancel)
			}

			// Fase 3: Face References & Lifecycle Management
			if deps.Handlers.FaceReferenceListByEmployee != nil {
				r.With(rbacAnyGuard(deps, "face.read_any", "face.read_self")).Get("/employees/{id}/face-references", deps.Handlers.FaceReferenceListByEmployee)
			}
			if deps.Handlers.FaceReferenceGetPhoto != nil {
				r.With(rbacAnyGuard(deps, "face.read_any", "face.read_self")).Get("/face/references/{id}/photo", deps.Handlers.FaceReferenceGetPhoto)
			}
			if deps.Handlers.FaceReferenceDeactivate != nil {
				r.With(rbacGuard(deps, "face.delete_any")).Patch("/face/references/{id}", deps.Handlers.FaceReferenceDeactivate)
			}
			if deps.Handlers.FaceReferenceDeleteAll != nil {
				r.With(rbacGuard(deps, "face.delete_any")).Delete("/employees/{id}/face-data", deps.Handlers.FaceReferenceDeleteAll)
			}
			if deps.Handlers.FaceEnrollmentStatusMe != nil {
				r.With(rbacGuard(deps, "face.read_self")).Get("/face/enrollment-status/me", deps.Handlers.FaceEnrollmentStatusMe)
			}

			// Fase 3: Face Reindex Jobs
			if deps.Handlers.FaceReindexCreateJob != nil {
				r.With(rbacGuard(deps, "face.reindex")).Post("/face/reindex-jobs", deps.Handlers.FaceReindexCreateJob)
			}
			if deps.Handlers.FaceReindexListJobs != nil {
				r.With(rbacGuard(deps, "face.reindex")).Get("/face/reindex-jobs", deps.Handlers.FaceReindexListJobs)
			}
			if deps.Handlers.FaceReindexGetJob != nil {
				r.With(rbacGuard(deps, "face.reindex")).Get("/face/reindex-jobs/{id}", deps.Handlers.FaceReindexGetJob)
			}
			if deps.Handlers.FaceReindexCancelJob != nil {
				r.With(rbacGuard(deps, "face.reindex")).Post("/face/reindex-jobs/{id}/cancel", deps.Handlers.FaceReindexCancelJob)
			}

			// Fase 4: Attendance Engine (#53 - #65)
			if deps.Handlers.AttendancesClockIn != nil {
				r.With(rbacGuard(deps, "attendance.checkin")).Post("/attendances/clock-in", deps.Handlers.AttendancesClockIn)
			}
			if deps.Handlers.AttendancesClockOut != nil {
				r.With(rbacGuard(deps, "attendance.checkin")).Post("/attendances/clock-out", deps.Handlers.AttendancesClockOut)
			}
			if deps.Handlers.AttendanceContext != nil {
				r.With(rbacGuard(deps, "attendance.checkin")).Get("/attendances/context", deps.Handlers.AttendanceContext)
			}
			if deps.Handlers.AttendanceMe != nil {
				r.With(rbacGuard(deps, "attendance.read_self")).Get("/attendances/me", deps.Handlers.AttendanceMe)
			}
			if deps.Handlers.AttendanceMeToday != nil {
				r.With(rbacGuard(deps, "attendance.read_self")).Get("/attendances/me/today", deps.Handlers.AttendanceMeToday)
			}
			if deps.Handlers.AttendanceSummary != nil {
				r.With(rbacGuard(deps, "attendance.read_all")).Get("/attendances/summary", deps.Handlers.AttendanceSummary)
			}
			if deps.Handlers.AttendanceExport != nil {
				r.With(rbacGuard(deps, "attendance.export")).Get("/attendances/export", deps.Handlers.AttendanceExport)
			}
			if deps.Handlers.AttendanceGetByID != nil {
				r.With(rbacAnyGuard(deps, "attendance.read_self", "attendance.read_all")).Get("/attendances/{id}", deps.Handlers.AttendanceGetByID)
			}
			if deps.Handlers.AttendanceGetPhoto != nil {
				r.With(rbacAnyGuard(deps, "attendance.read_self", "attendance.read_all")).Get("/attendances/{id}/photo", deps.Handlers.AttendanceGetPhoto)
			}
			if deps.Handlers.AttendanceList != nil {
				r.With(rbacGuard(deps, "attendance.read_all")).Get("/attendances", deps.Handlers.AttendanceList)
			}
			if deps.Handlers.AttendancePending != nil {
				r.With(rbacGuard(deps, "attendance.review")).Get("/attendances/pending", deps.Handlers.AttendancePending)
			}
			if deps.Handlers.AttendanceApprove != nil {
				r.With(rbacGuard(deps, "attendance.review")).Post("/attendances/{id}/approve", deps.Handlers.AttendanceApprove)
			}
			if deps.Handlers.AttendanceReject != nil {
				r.With(rbacGuard(deps, "attendance.review")).Post("/attendances/{id}/reject", deps.Handlers.AttendanceReject)
			}
			if deps.Handlers.AttendanceBulkReview != nil {
				r.With(rbacGuard(deps, "attendance.review")).Post("/attendances/reviews", deps.Handlers.AttendanceBulkReview)
			}
			if deps.Handlers.AttendanceAttempts != nil {
				r.With(rbacGuard(deps, "attendance.read_all")).Get("/attendances/attempts", deps.Handlers.AttendanceAttempts)
			}

			// Fase 4: Office Locations (#66 - #70)
			if deps.Handlers.LocationList != nil {
				r.With(rbacGuard(deps, "location.read")).Get("/locations", deps.Handlers.LocationList)
			}
			if deps.Handlers.LocationCreate != nil {
				r.With(rbacGuard(deps, "location.create")).Post("/locations", deps.Handlers.LocationCreate)
			}
			if deps.Handlers.LocationGetByID != nil {
				r.With(rbacGuard(deps, "location.read")).Get("/locations/{id}", deps.Handlers.LocationGetByID)
			}
			if deps.Handlers.LocationUpdate != nil {
				r.With(rbacGuard(deps, "location.update")).Patch("/locations/{id}", deps.Handlers.LocationUpdate)
			}
			if deps.Handlers.LocationDelete != nil {
				r.With(rbacGuard(deps, "location.delete")).Delete("/locations/{id}", deps.Handlers.LocationDelete)
			}
		})
	})
}

func consentGuard(deps RouterDeps, resolve func(*http.Request) (uuid.UUID, error)) func(http.Handler) http.Handler {
	if deps.ConsentMiddleware != nil {
		return deps.ConsentMiddleware(resolve)
	}
	return func(next http.Handler) http.Handler { return next }
}

func rbacGuard(deps RouterDeps, perm string) func(http.Handler) http.Handler {
	if deps.RBACMiddleware != nil {
		return deps.RBACMiddleware(perm)
	}
	return func(next http.Handler) http.Handler { return next }
}

func rbacAnyGuard(deps RouterDeps, perms ...string) func(http.Handler) http.Handler {
	if deps.RBACAnyMiddleware != nil {
		return deps.RBACAnyMiddleware(perms...)
	}
	return func(next http.Handler) http.Handler { return next }
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	Fail(r.Context(), w, NewAppError(CodeNotFound, "Endpoint tidak ditemukan"))
}

func methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	// 405 has no dedicated catalog code (Fase 0 § 2.6 lists 13 codes, none of
	// them 405) — Fase 0 § 11.5 only requires the ENVELOPE shape and the
	// real HTTP status, so this bypasses StatusFor via FailWithStatus rather
	// than stretching an existing code's registered status.
	FailWithStatus(r.Context(), w, http.StatusMethodNotAllowed,
		NewAppError(CodeBadRequest, "Method tidak diizinkan untuk endpoint ini"))
}

func writePanicResponse(w http.ResponseWriter, r *http.Request) {
	Fail(r.Context(), w, NewAppError(CodeInternalError, "Terjadi kesalahan pada server"))
}

func writeCSRFFail(w http.ResponseWriter, r *http.Request, code string, msg string) {
	Fail(r.Context(), w, NewAppError(ErrorCode(code), msg))
}

func writeRateLimitResponse(w http.ResponseWriter, r *http.Request) {
	Fail(r.Context(), w, NewAppError(CodeRateLimited, "Terlalu banyak request, coba lagi nanti"))
}
