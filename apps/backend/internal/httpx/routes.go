package httpx

// RouteGuard represents the protection level or required permission for a route.
type RouteGuard string

const (
	GuardPublic RouteGuard = "public"
	GuardAuth   RouteGuard = "auth"
)

// RouteInfo describes an expected endpoint in the API contract.
type RouteInfo struct {
	Method string
	Path   string
	Guard  RouteGuard
}

// RouteRegistry is the authoritative catalog of all registered routes and their guards
// for Fase 0 and Fase 1 (§ 4.8 Ringkasan seluruh endpoint Fase 1).
var RouteRegistry = map[string]RouteGuard{
	// Fase 0 Base Endpoints
	"GET /healthz":        GuardPublic,
	"GET /readyz":         GuardPublic,
	"GET /api/v1/version": GuardPublic,

	// #1 - #6: Auth Endpoints
	"POST /api/v1/auth/login":           GuardPublic,
	"POST /api/v1/auth/refresh":         GuardPublic,
	"POST /api/v1/auth/logout":          GuardAuth,
	"POST /api/v1/auth/logout-all":      GuardAuth,
	"GET /api/v1/auth/me":               GuardAuth,
	"POST /api/v1/auth/change-password": GuardAuth,

	// #7 - #12: Employee Endpoints
	"GET /api/v1/employees":         "employee.read",
	"POST /api/v1/employees":        "employee.create",
	"GET /api/v1/employees/me":      "employee.read_self",
	"GET /api/v1/employees/{id}":    "employee.read|employee.read_self",
	"PATCH /api/v1/employees/{id}":  "employee.update",
	"DELETE /api/v1/employees/{id}": "employee.delete",

	// #13 - #20: User Endpoints
	"GET /api/v1/users":                      "user.read",
	"POST /api/v1/users":                     "user.create",
	"GET /api/v1/users/{id}":                 "user.read",
	"PATCH /api/v1/users/{id}":               "user.update",
	"DELETE /api/v1/users/{id}":              "user.delete",
	"PATCH /api/v1/users/{id}/status":        "user.update",
	"PUT /api/v1/users/{id}/roles":           "user.assign_role",
	"POST /api/v1/users/{id}/reset-password": "user.reset_password",

	// #21 - #27: Role & Permission Endpoints
	"GET /api/v1/roles":                  "role.read",
	"POST /api/v1/roles":                 "role.create",
	"GET /api/v1/roles/{id}":             "role.read",
	"PATCH /api/v1/roles/{id}":           "role.update",
	"DELETE /api/v1/roles/{id}":          "role.delete",
	"PUT /api/v1/roles/{id}/permissions": "role.assign_permission",
	"GET /api/v1/permissions":            "permission.read",
	"GET /api/v1/modules":                "permission.read",

	// #28 - #30, #73: Settings Endpoints
	"GET /api/v1/settings/face-quality-status": "settings.read",
	"GET /api/v1/settings":                     "settings.read",
	"GET /api/v1/settings/{key}":               "settings.read",
	"PUT /api/v1/settings/{key}":               "settings.update",

	// #31: Audit Logs Endpoint
	"GET /api/v1/audit-logs": "audit.read",

	// Fase 2: Biometrics & Attendance Endpoints (Legacy / Phase 2 Stubs)
	"POST /api/v1/employees/{id}/face-enroll": "face.enroll_any|face.enroll_self",
	"POST /api/v1/attendance/clock-in":        "attendance.checkin",
	"POST /api/v1/attendance/clock-out":       "attendance.checkin",

	// #32 - #37: Biometric Consent Endpoints (Fase 3 - UU PDP No. 27/2022)
	"GET /api/v1/consents/document":       GuardAuth,
	"GET /api/v1/consents/me":             GuardAuth,
	"POST /api/v1/consents":               "face.enroll_self",
	"POST /api/v1/consents/withdraw":      GuardAuth,
	"GET /api/v1/employees/{id}/consent":  "face.read_any|face.read_self",
	"POST /api/v1/employees/{id}/consent": "face.enroll_any",

	// #38 - #43: Multi-Photo Face Enrollment Sessions (Fase 3)
	"POST /api/v1/face/enrollments":                     "face.enroll_self|face.enroll_any",
	"GET /api/v1/face/enrollments/{id}":                 "face.enroll_self|face.enroll_any",
	"POST /api/v1/face/enrollments/{id}/photos":         "face.enroll_self|face.enroll_any",
	"DELETE /api/v1/face/enrollments/{id}/photos/{pid}": "face.enroll_self|face.enroll_any",
	"POST /api/v1/face/enrollments/{id}/commit":         "face.enroll_self|face.enroll_any",
	"DELETE /api/v1/face/enrollments/{id}":              "face.enroll_self|face.enroll_any",

	// #44 - #48: Face References & Lifecycle Management (Fase 3)
	"GET /api/v1/employees/{id}/face-references": "face.read_any|face.read_self",
	"GET /api/v1/face/references/{id}/photo":     "face.read_any|face.read_self",
	"PATCH /api/v1/face/references/{id}":         "face.delete_any",
	"DELETE /api/v1/employees/{id}/face-data":    "face.delete_any",
	"GET /api/v1/face/enrollment-status/me":      "face.read_self",

	// #49 - #52: Face Reindex Jobs (Fase 3)
	"POST /api/v1/face/reindex-jobs":             "face.reindex",
	"GET /api/v1/face/reindex-jobs":              "face.reindex",
	"GET /api/v1/face/reindex-jobs/{id}":         "face.reindex",
	"POST /api/v1/face/reindex-jobs/{id}/cancel": "face.reindex",

	// #53 - #65, #71 - #72: Attendance Engine (Fase 4 - 5)
	"POST /api/v1/attendances/clock-in":     "attendance.checkin",
	"POST /api/v1/attendances/clock-out":    "attendance.checkin",
	"GET /api/v1/attendances/context":       "attendance.checkin",
	"GET /api/v1/attendances/me":            "attendance.read_self",
	"GET /api/v1/attendances/me/today":      "attendance.read_self",
	"GET /api/v1/attendances/summary":       "attendance.read_all",
	"GET /api/v1/attendances/export":        "attendance.export",
	"GET /api/v1/attendances/{id}":          "attendance.read_self|attendance.read_all",
	"GET /api/v1/attendances/{id}/photo":    "attendance.read_self|attendance.read_all",
	"GET /api/v1/attendances":               "attendance.read_all",
	"GET /api/v1/attendances/pending":       "attendance.review",
	"POST /api/v1/attendances/{id}/approve": "attendance.review",
	"POST /api/v1/attendances/{id}/reject":  "attendance.review",
	"POST /api/v1/attendances/reviews":      "attendance.review",
	"GET /api/v1/attendances/attempts":      "attendance.read_all",

	// #66 - #70: Office Locations (Fase 4)
	"GET /api/v1/locations":         "location.read",
	"POST /api/v1/locations":        "location.create",
	"GET /api/v1/locations/{id}":    "location.read",
	"PATCH /api/v1/locations/{id}":  "location.update",
	"DELETE /api/v1/locations/{id}": "location.delete",
}
