// Command api is the faceclock-api entry point: load config, connect
// dependencies, run migrations (dev only), serve HTTP, shut down gracefully
// (Fase 0 § 5.1, § 5.3).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

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
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/platform/postgres"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/role"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/seeder"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/settings"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/storage"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/user"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/version"
)

func main() {
	// config.Load() failing is exactly E1: report every missing/invalid
	// env var at once and exit non-zero, before any dependency is touched.
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "faceclock-api: fatal config error:", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel, os.Stdout)
	log.Info("starting faceclock-api",
		slog.String("version", version.Version),
		slog.String("commit", version.Commit),
		slog.String("app_env", cfg.AppEnv),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Connect(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns, log)
	if err != nil {
		log.Error("could not connect to postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()
	log.Info("connected to postgres")

	if cfg.AppEnv == "development" {
		if err := runMigrations(cfg.DatabaseURL, log); err != nil {
			log.Error("migration failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
		if err := seeder.Run(ctx, pool, seeder.Options{
			AppEnv:            cfg.AppEnv,
			SeedAdminEmail:    os.Getenv("SEED_ADMIN_EMAIL"),
			SeedAdminPassword: os.Getenv("SEED_ADMIN_PASSWORD"),
			Logger:            log,
		}); err != nil {
			log.Error("seeder failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	var store storage.Store
	if cfg.StorageDriver == "s3" {
		store, err = storage.NewS3Store(ctx, storage.S3Config{
			Endpoint:             cfg.StorageS3Endpoint,
			Region:               cfg.StorageS3Region,
			Bucket:               cfg.StorageS3BucketFace,
			AccessKey:            cfg.StorageS3AccessKey,
			SecretKey:            cfg.StorageS3SecretKey,
			ForcePathStyle:       cfg.StorageS3ForcePathStyle,
			ServerSideEncryption: cfg.StorageS3SSE,
		})
	} else {
		store, err = storage.NewLocalStore(cfg.StorageLocalPath)
	}
	if err != nil {
		log.Error("could not initialize storage", slog.String("error", err.Error()))
		os.Exit(1)
	}

	faceEngine := face.NewRESTEngine(cfg.InferenceBaseURL, cfg.InferenceToken, "buffalo_l", face.DefaultCosineThreshold, cfg.InferenceTimeout)

	inferenceClient := inference.New(cfg.InferenceBaseURL, cfg.InferenceToken, cfg.InferenceTimeout)

	healthHandler := &health.Handler{DB: pool, Inference: inferenceClient}

	auditRecorder := audit.NewRecorder(pool)

	settingsSvc := settings.NewService(pool)
	settingsHandler := settings.NewHandler(settingsSvc, auditRecorder, inferenceClient)

	rbacCache := rbac.NewCache(30 * time.Second)
	rbacSvc := rbac.NewService(pool, rbacCache)

	tokenMgr, err := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTAccessTTL)
	if err != nil {
		log.Error("could not initialize token manager", slog.String("error", err.Error()))
		os.Exit(1)
	}

	authSvc := auth.NewService(pool, cfg, tokenMgr, settingsSvc, rbacSvc, auditRecorder)
	authHandler := auth.NewHandler(authSvc, auditRecorder)
	authMw := auth.Authenticate(tokenMgr, rbacSvc, pool)

	empSvc := employee.NewService(pool)
	empSvc.SetFaceBiometrics(faceEngine, store)
	empHandler := employee.NewHandler(empSvc, auditRecorder)

	attendanceSvc := attendance.NewService(pool, faceEngine, store, settingsSvc)
	attendanceSvc.SetAuditRecorder(auditRecorder)
	attendanceHandler := attendance.NewHandler(attendanceSvc, auditRecorder)

	userSvc := user.NewService(pool, rbacSvc)
	userHandler := user.NewHandler(userSvc, auditRecorder)

	roleSvc := role.NewService(pool, rbacSvc)
	roleHandler := role.NewHandler(roleSvc, auditRecorder)

	auditHandler := audit.NewHandler(auditRecorder)

	// Fase 3: Biometric Consent Subsystem (UU PDP No. 27/2022)
	consentRepo := consent.NewRepository(pool)
	consentCache := consent.NewCache(5 * time.Minute)
	consentSvc := consent.NewService(consentRepo, consentCache, settingsSvc, auditRecorder)
	consentHandler := consent.NewHandler(consentSvc)

	// Fase 3: Multi-Photo Face Enrollment Sessions Subsystem
	enrollRepo := enrollment.NewRepository(pool)
	enrollSvc := enrollment.NewService(enrollRepo, store, inferenceClient, settingsSvc, consentSvc, auditRecorder, log)
	enrollHandler := enrollment.NewHandler(enrollSvc)

	// Fase 3: Face Reference Lifecycle Management Subsystem
	refRepo := reference.NewPostgresRepository(pool)
	refSvc := reference.NewService(refRepo, store, settingsSvc, consentSvc, auditRecorder, log)
	refHandler := reference.NewHandler(refSvc)

	// Fase 3: Face Reindex Jobs & Background Worker
	reindexRepo := reindex.NewPostgresRepository(pool)
	reindexSvc := reindex.NewService(reindexRepo, inferenceClient, settingsSvc, auditRecorder, log)
	reindexHandler := reindex.NewHandler(reindexSvc)
	reindexWorker := reindex.NewWorker(reindexRepo, store, inferenceClient, settingsSvc, auditRecorder, log, 10*time.Second)
	go reindexWorker.Start(ctx)

	// Fase 4: Office Locations & Attendance Retention Worker
	locSvc := location.NewService(pool, settingsSvc, auditRecorder)
	locHandler := location.NewHandler(locSvc, auditRecorder)

	attendanceRetention := attendance.NewRetentionJob(pool, store, settingsSvc, auditRecorder, log)
	attendanceRetention.Start(ctx, 1*time.Hour)

	// Housekeeping job for expired refresh tokens (§ 3.7)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_, err := pool.Exec(context.Background(), "DELETE FROM refresh_tokens WHERE expires_at < NOW() - INTERVAL '30 days'")
				if err != nil {
					log.Warn("cleaning expired refresh tokens", slog.String("error", err.Error()))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

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
			AuthLogin:                 authHandler.Login,
			AuthRefresh:               authHandler.Refresh,
			AuthLogout:                authHandler.Logout,
			AuthLogoutAll:             authHandler.RevokeAll,
			AuthGetMe:                 authHandler.GetMe,
			AuthChangePassword:        authHandler.ChangePassword,
			EmployeeList:              empHandler.List,
			EmployeeCreate:            empHandler.Create,
			EmployeeGetMe:             empHandler.GetMe,
			EmployeeCompleteOwn:       empHandler.CompleteOwnProfile,
			EmployeeGetByID:           empHandler.GetByID,
			EmployeeUpdate:            empHandler.Update,
			EmployeeDelete:            empHandler.Delete,
			UserList:                  userHandler.List,
			UserCreate:                userHandler.Create,
			UserRegisterEmployee:      userHandler.RegisterEmployee,
			UserGetByID:               userHandler.GetByID,
			UserUpdate:                userHandler.Update,
			UserDelete:                userHandler.Delete,
			UserUpdateStatus:          userHandler.UpdateStatus,
			UserAssignRoles:           userHandler.AssignRoles,
			UserResetPassword:         userHandler.ResetPassword,
			RoleList:                  roleHandler.ListRoles,
			RoleCreate:                roleHandler.CreateRole,
			RoleGetByID:               roleHandler.GetRoleByID,
			RoleUpdate:                roleHandler.UpdateRole,
			RoleDelete:                roleHandler.DeleteRole,
			RoleAssignPermissions:     roleHandler.AssignPermissions,
			PermissionList:            roleHandler.ListPermissions,
			ModuleList:                roleHandler.ListModules,
			SettingsList:              settingsHandler.List,
			SettingsGetByKey:          settingsHandler.GetByKey,
			SettingsUpdate:            settingsHandler.Update,
			SettingsFaceQualityStatus: settingsHandler.FaceQualityStatus,
			AuditQuery:                auditHandler.List,
			EmployeeFaceEnroll:        empHandler.FaceEnroll,
			AttendanceClockIn:         attendanceHandler.ClockIn,
			AttendanceClockOut:        attendanceHandler.ClockOut,

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
			LocationList:    locHandler.List,
			LocationCreate:  locHandler.Create,
			LocationGetByID: locHandler.GetByID,
			LocationUpdate:  locHandler.Update,
			LocationDelete:  locHandler.Delete,

			// Fase 5: Admin Panel & Configuration (#71 - #73)
			AttendanceSummary: attendanceHandler.GetSummary,
			AttendanceExport:  attendanceHandler.Export,
		},
	})

	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("listening", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		log.Error("server error", slog.String("error", err.Error()))
		os.Exit(1)
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	// Fase 0 § 5.3: drain in-flight requests for up to 15s, then give up.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Warn("graceful shutdown deadline exceeded", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log.Info("shutdown complete")
}

func runMigrations(databaseURL string, log *slog.Logger) error {
	// golang-migrate's pgx/v5 driver wants a database/sql handle, not a
	// pgxpool.Pool — stdlib.OpenDB adapts a pgx connection config to one
	// without pulling in lib/pq as a second Postgres driver.
	connConfig, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("migrate: parse DATABASE_URL: %w", err)
	}
	sqlDB := stdlib.OpenDB(*connConfig)
	defer sqlDB.Close()

	driver, err := migratepgx.WithInstance(sqlDB, &migratepgx.Config{})
	if err != nil {
		return fmt.Errorf("migrate: open driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "pgx", driver)
	if err != nil {
		return fmt.Errorf("migrate: init: %w", err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("migrations: no change")
			return nil
		}
		// E5: a missing Postgres extension (e.g. `vector` not baked into
		// the image) surfaces here — make the image mismatch explicit
		// rather than a bare driver error.
		return fmt.Errorf("migrate up failed — if this mentions extension \"vector\", "+
			"confirm the postgres image is pgvector/pgvector:pg17, not a plain postgres image: %w", err)
	}

	log.Info("migrations applied")
	return nil
}
