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

	// #28 - #30: Settings Endpoints
	"GET /api/v1/settings":       "settings.read",
	"GET /api/v1/settings/{key}": "settings.read",
	"PUT /api/v1/settings/{key}": "settings.update",

	// #31: Audit Logs Endpoint
	"GET /api/v1/audit-logs": "audit.read",

	// #32 - #34: Fase 2 Biometrics & Attendance Endpoints
	"POST /api/v1/employees/{id}/face-enroll": "face.enroll_any|face.enroll_self",
	"POST /api/v1/attendance/clock-in":        "attendance.checkin",
	"POST /api/v1/attendance/clock-out":       "attendance.checkin",
}
