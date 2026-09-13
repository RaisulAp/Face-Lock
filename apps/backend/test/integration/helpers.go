package integration

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/attendance"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/auth"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/config"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/consent"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/employee"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/face"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/face/enrollment"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/face/reference"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/face/reindex"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/health"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/inference"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/location"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/platform/logger"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/role"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/seeder"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/settings"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/storage"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/user"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/version"
	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

type TestApp struct {
	Pool          *pgxpool.Pool
	Router        http.Handler
	Config        *config.Config
	TokenMgr      *auth.TokenManager
	AuthSvc       *auth.Service
	RBACSvc       *rbac.Service
	UserSvc       *user.Service
	EmployeeSvc   *employee.Service
	AttendanceSvc *attendance.Service
	FaceEngine    *face.MockEngine
	Store         storage.Store
	RoleSvc       *role.Service
	SettingsSvc   *settings.Service
	AuditRec      *audit.Recorder
	ConsentSvc    *consent.Service
	EnrollmentSvc *enrollment.Service
	FaceRefSvc    *reference.Service
	ReindexSvc    *reindex.Service
	ReindexRepo   reindex.Repository
	MockInfEngine *inference.MockFaceEngine
}

func SetupTestApp(t *testing.T) *TestApp {
	t.Helper()

	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://faceclock:devpassword123@localhost:5434/faceclock?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Skipf("skipping integration tests: database unreachable: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skipping integration tests: ping failed: %v", err)
	}

	// Auto-run migrations for test database
	migPath := "file://../../migrations"
	if _, err := os.Stat("../../migrations"); os.IsNotExist(err) {
		migPath = "file://migrations"
	}
	if connConfig, err := pgx.ParseConfig(connStr); err == nil {
		sqlDB := stdlib.OpenDB(*connConfig)
		if driver, err := migratepgx.WithInstance(sqlDB, &migratepgx.Config{}); err == nil {
			if m, err := migrate.NewWithDatabaseInstance(migPath, "pgx", driver); err == nil {
				_ = m.Up()
			}
		}
		_ = sqlDB.Close()
	}

	log := logger.New("error", io.Discard)

	cfg := &config.Config{
		AppEnv:             "development",
		AppPort:            "8080",
		LogLevel:           "error",
		DatabaseURL:        connStr,
		DatabaseMaxConns:   10,
		JWTSecret:          "01234567890123456789012345678901",
		JWTAccessTTL:       15 * time.Minute,
		JWTRefreshTTL:      30 * 24 * time.Hour,
		StorageLocalPath:   t.TempDir(),
		InferenceBaseURL:   "http://localhost:8001",
		InferenceTimeout:   5 * time.Second,
		CORSAllowedOrigins: []string{"*"},
		RequestTimeout:     30 * time.Second,
		MaxBodyBytes:       1048576,
	}

	// Ensure seeder has run
	_ = seeder.Run(context.Background(), pool, seeder.Options{
		AppEnv:            cfg.AppEnv,
		SeedAdminEmail:    "admin@faceclock.local",
		SeedAdminPassword: "SeedAdminPassword123!",
		Logger:            log,
	})

	// Ensure app_settings has calibrated face.model_version for testing
	_, _ = pool.Exec(context.Background(), `UPDATE app_settings SET value = '"buffalo_l@v1"' WHERE key = 'face.model_version'`)
	_, _ = pool.Exec(context.Background(), `
		DELETE FROM face_reindex_items;
		DELETE FROM face_reindex_jobs;
		DELETE FROM face_references;
		DELETE FROM face_enrollment_photos;
		DELETE FROM face_enrollment_sessions;
		DELETE FROM biometric_consents;
	`)

	tokenMgr, err := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTAccessTTL)
	if err != nil {
		t.Fatalf("failed to init token manager: %v", err)
	}

	rbacCache := rbac.NewCache(30 * time.Second)
	rbacSvc := rbac.NewService(pool, rbacCache)
	auditRec := audit.NewRecorder(pool)
	settingsSvc := settings.NewService(pool)
	authSvc := auth.NewService(pool, cfg, tokenMgr, settingsSvc, rbacSvc, auditRec)

	store, err := storage.NewLocalStore(cfg.StorageLocalPath)
	if err != nil {
		t.Fatalf("failed to init local store: %v", err)
	}
	mockFaceEngine := face.NewMockEngine()

	empSvc := employee.NewService(pool)
	empSvc.SetFaceBiometrics(mockFaceEngine, store)

	attendanceSvc := attendance.NewService(pool, mockFaceEngine, store, settingsSvc)
	attendanceHandler := attendance.NewHandler(attendanceSvc, auditRec)

	locationSvc := location.NewService(pool, settingsSvc, auditRec)
	locationHandler := location.NewHandler(locationSvc, auditRec)

	userSvc := user.NewService(pool, rbacSvc)
	roleSvc := role.NewService(pool, rbacSvc)

	authHandler := auth.NewHandler(authSvc, auditRec)
	authMw := auth.Authenticate(tokenMgr, rbacSvc, pool)
	empHandler := employee.NewHandler(empSvc, auditRec)
	userHandler := user.NewHandler(userSvc, auditRec)
	roleHandler := role.NewHandler(roleSvc, auditRec)
	settingsHandler := settings.NewHandler(settingsSvc, auditRec)
	auditHandler := audit.NewHandler(auditRec)
	healthHandler := &health.Handler{DB: pool}

	mockInfEngine := inference.NewMockFaceEngine()

	consentRepo := consent.NewRepository(pool)
	consentCache := consent.NewCache(5 * time.Minute)
	consentSvc := consent.NewService(consentRepo, consentCache, settingsSvc, auditRec)
	consentHandler := consent.NewHandler(consentSvc)

	enrollRepo := enrollment.NewRepository(pool)
	enrollSvc := enrollment.NewService(enrollRepo, store, mockInfEngine, settingsSvc, consentSvc, auditRec, log)
	enrollHandler := enrollment.NewHandler(enrollSvc)

	refRepo := reference.NewPostgresRepository(pool)
	refSvc := reference.NewService(refRepo, store, settingsSvc, consentSvc, auditRec, log)
	refHandler := reference.NewHandler(refSvc)

	reindexRepo := reindex.NewPostgresRepository(pool)
	reindexSvc := reindex.NewService(reindexRepo, mockInfEngine, settingsSvc, auditRec, log)
	reindexHandler := reindex.NewHandler(reindexSvc)

	router := httpx.NewRouter(httpx.RouterDeps{
		Logger:             log,
		CORSAllowedOrigins: cfg.CORSAllowedOrigins,
		RequestTimeout:     cfg.RequestTimeout,
		MaxBodyBytes:       cfg.MaxBodyBytes,
		HealthzHandler:     healthHandler.Healthz,
		ReadyzHandler:      healthHandler.Readyz,
		VersionHandler:     version.Handler,

		AuthMiddleware:    authMw,
		RBACMiddleware:    rbac.RequirePermission,
		RBACAnyMiddleware: rbac.RequireAnyPermission,
		ConsentMiddleware: func(resolve func(*http.Request) (uuid.UUID, error)) func(http.Handler) http.Handler {
			return consent.RequireConsent(consentSvc, resolve)
		},

		Handlers: httpx.Handlers{
			AuthLogin:             authHandler.Login,
			AuthRefresh:           authHandler.Refresh,
			AuthLogout:            authHandler.Logout,
			AuthLogoutAll:         authHandler.RevokeAll,
			AuthGetMe:             authHandler.GetMe,
			AuthChangePassword:    authHandler.ChangePassword,
			EmployeeList:          empHandler.List,
			EmployeeCreate:        empHandler.Create,
			EmployeeGetMe:         empHandler.GetMe,
			EmployeeGetByID:       empHandler.GetByID,
			EmployeeUpdate:        empHandler.Update,
			EmployeeDelete:        empHandler.Delete,
			UserList:              userHandler.List,
			UserCreate:            userHandler.Create,
			UserGetByID:           userHandler.GetByID,
			UserUpdate:            userHandler.Update,
			UserDelete:            userHandler.Delete,
			UserUpdateStatus:      userHandler.UpdateStatus,
			UserAssignRoles:       userHandler.AssignRoles,
			UserResetPassword:     userHandler.ResetPassword,
			RoleList:              roleHandler.ListRoles,
			RoleCreate:            roleHandler.CreateRole,
			RoleGetByID:           roleHandler.GetRoleByID,
			RoleUpdate:            roleHandler.UpdateRole,
			RoleDelete:            roleHandler.DeleteRole,
			RoleAssignPermissions: roleHandler.AssignPermissions,
			PermissionList:        roleHandler.ListPermissions,
			SettingsList:          settingsHandler.List,
			SettingsGetByKey:      settingsHandler.GetByKey,
			SettingsUpdate:        settingsHandler.Update,
			AuditQuery:            auditHandler.List,
			EmployeeFaceEnroll:    empHandler.FaceEnroll,
			AttendanceClockIn:     attendanceHandler.ClockIn,
			AttendanceClockOut:    attendanceHandler.ClockOut,

			// Fase 3: Biometric Consent (UU PDP No. 27/2022)
			ConsentGetDocument: consentHandler.GetActiveDocument,
			ConsentGetMe:       consentHandler.GetMyConsent,
			ConsentGrant:       consentHandler.GrantConsent,
			ConsentWithdraw:    consentHandler.WithdrawConsent,
			ConsentGetEmployee: consentHandler.GetEmployeeConsent,
			ConsentAdminRecord: consentHandler.AdminRecordConsent,

			// Fase 3: Multi-Photo Face Enrollment
			FaceEnrollmentCreate:      enrollHandler.CreateSession,
			FaceEnrollmentGet:         enrollHandler.GetSession,
			FaceEnrollmentUploadPhoto: enrollHandler.UploadPhoto,
			FaceEnrollmentDeletePhoto: enrollHandler.DeletePhoto,
			FaceEnrollmentCommit:      enrollHandler.CommitSession,
			FaceEnrollmentCancel:      enrollHandler.CancelSession,

			// Fase 3: Face References & Lifecycle
			FaceReferenceListByEmployee: refHandler.ListReferences,
			FaceReferenceGetPhoto:       refHandler.GetPhoto,
			FaceReferenceDeactivate:     refHandler.DeactivateReference,
			FaceReferenceDeleteAll:      refHandler.DeleteFaceData,
			FaceEnrollmentStatusMe:      refHandler.GetEnrollmentStatusMe,

			// Fase 3: Face Reindex Jobs
			FaceReindexCreateJob: reindexHandler.CreateJob,
			FaceReindexListJobs:  reindexHandler.ListJobs,
			FaceReindexGetJob:    reindexHandler.GetJob,
			FaceReindexCancelJob: reindexHandler.CancelJob,

			// Fase 4: Attendance Engine (#53 - #65)
			AttendancesClockIn:   attendanceHandler.ClockIn,
			AttendancesClockOut:  attendanceHandler.ClockOut,
			AttendanceContext:    attendanceHandler.GetContext,
			AttendanceMe:         attendanceHandler.GetMe,
			AttendanceMeToday:    attendanceHandler.GetMeToday,
			AttendanceGetByID:    attendanceHandler.GetRecordByID,
			AttendanceGetPhoto:   attendanceHandler.GetPhoto,
			AttendanceList:       attendanceHandler.GetAdminRecords,
			AttendancePending:    attendanceHandler.GetPendingRecords,
			AttendanceApprove:    attendanceHandler.ApproveRecord,
			AttendanceReject:     attendanceHandler.RejectRecord,
			AttendanceBulkReview: attendanceHandler.BulkReviewRecords,
			AttendanceAttempts:   attendanceHandler.GetAttempts,

			// Fase 4: Office Locations (#66 - #70)
			LocationList:    locationHandler.List,
			LocationCreate:  locationHandler.Create,
			LocationGetByID: locationHandler.GetByID,
			LocationUpdate:  locationHandler.Update,
			LocationDelete:  locationHandler.Delete,
		},
	})

	return &TestApp{
		Pool:          pool,
		Router:        router,
		Config:        cfg,
		TokenMgr:      tokenMgr,
		AuthSvc:       authSvc,
		RBACSvc:       rbacSvc,
		UserSvc:       userSvc,
		EmployeeSvc:   empSvc,
		AttendanceSvc: attendanceSvc,
		FaceEngine:    mockFaceEngine,
		Store:         store,
		RoleSvc:       roleSvc,
		SettingsSvc:   settingsSvc,
		AuditRec:      auditRec,
		ConsentSvc:    consentSvc,
		EnrollmentSvc: enrollSvc,
		FaceRefSvc:    refSvc,
		ReindexSvc:    reindexSvc,
		ReindexRepo:   reindexRepo,
		MockInfEngine: mockInfEngine,
	}
}

type UserWithToken struct {
	UserID     uuid.UUID
	EmployeeID *uuid.UUID
	Email      string
	Role       string
	Token      string
}

func (app *TestApp) EnsureUserWithRole(t *testing.T, roleName string) *UserWithToken {
	t.Helper()
	ctx := context.Background()

	var roleID uuid.UUID
	err := app.Pool.QueryRow(ctx, "SELECT id FROM roles WHERE name = $1 AND deleted_at IS NULL", roleName).Scan(&roleID)
	if err != nil {
		t.Fatalf("role %s not found: %v", roleName, err)
	}

	// Create a corresponding employee if role is employee or admin
	var empID *uuid.UUID
	uSuffix := uuid.New().String()[:8]
	empNum := fmt.Sprintf("EMP-%s", uSuffix)

	var createdEmpID uuid.UUID
	err = app.Pool.QueryRow(ctx, `
		INSERT INTO employees (employee_number, full_name, department, email, employment_status)
		VALUES ($1, $2, 'Testing', $3, 'active')
		RETURNING id
	`, empNum, fmt.Sprintf("Test User %s", roleName), fmt.Sprintf("user_%s_%s@faceclock.local", roleName, uSuffix)).Scan(&createdEmpID)
	if err != nil {
		t.Fatalf("failed creating employee in EnsureUserWithRole: %v", err)
	}
	empID = &createdEmpID

	email := fmt.Sprintf("%s_%s@faceclock.local", roleName, uSuffix)
	passwordHash, err := auth.HashPassword("TestPass123!")
	if err != nil {
		t.Fatalf("hash password failed: %v", err)
	}

	var userID uuid.UUID
	var tokenVersion int
	err = app.Pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, employee_id, is_active, must_change_password)
		VALUES ($1, $2, $3, true, false)
		RETURNING id, token_version
	`, email, passwordHash, empID).Scan(&userID, &tokenVersion)
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	_, err = app.Pool.Exec(ctx, "INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)", userID, roleID)
	if err != nil {
		t.Fatalf("assign role failed: %v", err)
	}

	token, _, err := app.TokenMgr.GenerateAccessToken(userID, empID, tokenVersion)
	if err != nil {
		t.Fatalf("generate token failed: %v", err)
	}

	return &UserWithToken{
		UserID:     userID,
		EmployeeID: empID,
		Email:      email,
		Role:       roleName,
		Token:      token,
	}
}

func (app *TestApp) ExecuteRequest(req *http.Request) (*httptest.ResponseRecorder, []byte) {
	if req.Header.Get("X-Skip-CSRF-Header") != "" {
		req.Header.Del("X-Skip-CSRF-Header")
	} else {
		switch req.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if req.Header.Get("X-Requested-With") == "" && req.Header.Get("X-CSRF-Token") == "" {
				req.Header.Set("X-Requested-With", "XMLHttpRequest")
			}
		}
	}

	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	body := w.Body.Bytes()
	return w, body
}

// localRoundTripper routes http.Client requests directly to an http.Handler in memory,
// capturing and managing cookies via http.CookieJar without spinning up a TCP socket.
type localRoundTripper struct {
	handler http.Handler
}

func (rt *localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("X-Skip-CSRF-Header") != "" {
		req.Header.Del("X-Skip-CSRF-Header")
	} else {
		switch req.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if req.Header.Get("X-Requested-With") == "" && req.Header.Get("X-CSRF-Token") == "" {
				req.Header.Set("X-Requested-With", "XMLHttpRequest")
			}
		}
	}

	rec := httptest.NewRecorder()
	rt.handler.ServeHTTP(rec, req)

	res := rec.Result()
	res.Request = req
	return res, nil
}

// NewTestClient creates an http.Client backed by an in-memory cookiejar and in-memory handler transport.
// This allows testing full browser session lifecycle (auto cookie persistence, rotation, clearing).
func (app *TestApp) NewTestClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Jar: jar,
		Transport: &localRoundTripper{
			handler: app.Router,
		},
	}
}

func AssertNoSecretLeak(t *testing.T, body []byte) {
	t.Helper()
	s := strings.ToLower(string(body))
	if strings.Contains(s, "password_hash") {
		t.Errorf("SECURITY LEAK: response body contains 'password_hash': %s", string(body))
	}
	if strings.Contains(s, "token_hash") {
		t.Errorf("SECURITY LEAK: response body contains 'token_hash': %s", string(body))
	}
	if strings.Contains(s, "$argon2id$") || strings.Contains(s, "$argon2i$") {
		t.Errorf("SECURITY LEAK: response body contains argon2 hash signature: %s", string(body))
	}
	if strings.Contains(s, "$2a$") || strings.Contains(s, "$2b$") {
		t.Errorf("SECURITY LEAK: response body contains bcrypt hash signature: %s", string(body))
	}
}

func RandomString(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
