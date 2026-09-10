package seeder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/auth"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Options holds configuration for running the seeder.
type Options struct {
	AppEnv            string
	SeedAdminEmail    string
	SeedAdminPassword string
	Logger            *slog.Logger
}

type appSettingSeed struct {
	Key         string
	Value       any
	ValueType   string
	Description string
	IsPublic    bool
}

// Initial 11 app_settings defined in Fase 1 (§ 3.9) with REV-SET-04 description reconciliation.
var initialAppSettings = []appSettingSeed{
	{
		Key:         "face.similarity_threshold",
		Value:       0.45,
		ValueType:   "number",
		Description: "Threshold kemiripan wajah untuk verifikasi biometrik",
		IsPublic:    false,
	},
	{
		Key:         "face.min_reference_photos",
		Value:       3,
		ValueType:   "number",
		Description: "Jumlah minimum foto referensi untuk enrollment wajah",
		IsPublic:    true,
	},
	{
		Key:         "face.model_version",
		Value:       "unset",
		ValueType:   "string",
		Description: "Versi model machine learning biometrik aktif",
		IsPublic:    false,
	},
	{
		Key:         "attendance.geofence_enabled",
		Value:       true,
		ValueType:   "boolean",
		Description: "Status aktif validasi geofence pada absensi",
		IsPublic:    true,
	},
	{
		Key:         "attendance.max_distance_meter",
		Value:       100,
		ValueType:   "number",
		Description: "Batas atas dan nilai default radius geofence lokasi kantor dalam meter",
		IsPublic:    true,
	},
	{
		Key:         "attendance.workday_start",
		Value:       "08:00",
		ValueType:   "string",
		Description: "Waktu mulai jam kerja standar (HH:mm)",
		IsPublic:    true,
	},
	{
		Key:         "attendance.workday_end",
		Value:       "17:00",
		ValueType:   "string",
		Description: "Waktu selesai jam kerja standar (HH:mm)",
		IsPublic:    true,
	},
	{
		Key:         "attendance.timezone",
		Value:       "Asia/Jakarta",
		ValueType:   "string",
		Description: "Zona waktu sistem untuk pencatatan absensi",
		IsPublic:    true,
	},
	{
		Key:         "security.password_min_length",
		Value:       10,
		ValueType:   "number",
		Description: "Panjang minimum karakter password",
		IsPublic:    true,
	},
	{
		Key:         "security.max_failed_login",
		Value:       5,
		ValueType:   "number",
		Description: "Maksimum percobaan gagal login sebelum akun terkunci",
		IsPublic:    false,
	},
	{
		Key:         "security.lockout_minutes",
		Value:       15,
		ValueType:   "number",
		Description: "Durasi penguncian akun setelah gagal login (menit)",
		IsPublic:    false,
	},
}

// Run executes the idempotent seeder (§ 5.6).
func Run(ctx context.Context, db *pgxpool.Pool, opts Options) error {
	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}

	log.Info("Starting seeder execution...")

	// 1. UPSERT all 34 catalog permissions
	permMap := make(map[string]uuid.UUID)
	for _, p := range rbac.SystemPermissions {
		parts := strings.SplitN(p.Name, ".", 2)
		resource := parts[0]
		action := parts[1]

		var id uuid.UUID
		err := db.QueryRow(ctx, `
			INSERT INTO permissions (name, resource, action, description)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (name) DO UPDATE
			SET resource = EXCLUDED.resource,
			    action = EXCLUDED.action,
			    description = EXCLUDED.description
			RETURNING id;
		`, p.Name, resource, action, p.Description).Scan(&id)
		if err != nil {
			return fmt.Errorf("seeding permission %s: %w", p.Name, err)
		}
		permMap[p.Name] = id
	}
	log.Info("Permissions seeded successfully", "count", len(permMap))

	// 2. UPSERT 3 default system roles
	type roleDef struct {
		Name        string
		DisplayName string
		Description string
	}
	systemRoles := []roleDef{
		{"super_admin", "Super Administrator", "Akses penuh ke seluruh konfigurasi, data, dan modul sistem."},
		{"admin", "Administrator", "Pengelolaan operasional pengguna, karyawan, presensi, dan pengaturan."},
		{"employee", "Pegawai", "Akses mandiri presensi dan profil diri."},
	}

	roleMap := make(map[string]uuid.UUID)
	for _, r := range systemRoles {
		var id uuid.UUID
		err := db.QueryRow(ctx, `
			INSERT INTO roles (name, display_name, description, is_system)
			VALUES ($1, $2, $3, true)
			ON CONFLICT (name) WHERE deleted_at IS NULL DO UPDATE
			SET display_name = EXCLUDED.display_name,
			    description = EXCLUDED.description,
			    is_system = true
			RETURNING id;
		`, r.Name, r.DisplayName, r.Description).Scan(&id)
		if err != nil {
			return fmt.Errorf("seeding role %s: %w", r.Name, err)
		}
		roleMap[r.Name] = id
	}
	log.Info("System roles seeded successfully", "count", len(roleMap))

	// 3. Synchronize role_permissions mapping (§ 2.4)
	// super_admin: gets ALL permissions
	superAdminID := roleMap["super_admin"]
	for _, permID := range permMap {
		_, err := db.Exec(ctx, `
			INSERT INTO role_permissions (role_id, permission_id)
			VALUES ($1, $2)
			ON CONFLICT (role_id, permission_id) DO NOTHING;
		`, superAdminID, permID)
		if err != nil {
			return fmt.Errorf("assigning permission to super_admin: %w", err)
		}
	}

	// admin: all permissions EXCEPT role.create, role.update, role.delete, role.assign_permission, user.delete
	adminExcluded := map[string]bool{
		"role.create":            true,
		"role.update":            true,
		"role.delete":            true,
		"role.assign_permission": true,
		"user.delete":            true,
	}
	adminID := roleMap["admin"]
	for permName, permID := range permMap {
		if adminExcluded[permName] {
			continue
		}
		_, err := db.Exec(ctx, `
			INSERT INTO role_permissions (role_id, permission_id)
			VALUES ($1, $2)
			ON CONFLICT (role_id, permission_id) DO NOTHING;
		`, adminID, permID)
		if err != nil {
			return fmt.Errorf("assigning permission to admin: %w", err)
		}
	}

	// employee: employee.read_self, face.enroll_self, face.read_self, attendance.checkin, attendance.read_self, settings.read
	employeeAllowed := map[string]bool{
		"employee.read_self":   true,
		"face.enroll_self":     true,
		"face.read_self":       true,
		"attendance.checkin":   true,
		"attendance.read_self": true,
		"settings.read":        true,
	}
	employeeID := roleMap["employee"]
	for permName := range employeeAllowed {
		permID, ok := permMap[permName]
		if !ok {
			continue
		}
		_, err := db.Exec(ctx, `
			INSERT INTO role_permissions (role_id, permission_id)
			VALUES ($1, $2)
			ON CONFLICT (role_id, permission_id) DO NOTHING;
		`, employeeID, permID)
		if err != nil {
			return fmt.Errorf("assigning permission to employee: %w", err)
		}
	}
	log.Info("Role permissions synchronized successfully")

	// 4. UPSERT app_settings (INSERT-only: do not overwrite administrator tuning)
	for _, s := range initialAppSettings {
		valBytes, err := json.Marshal(s.Value)
		if err != nil {
			return fmt.Errorf("marshaling setting %s: %w", s.Key, err)
		}
		_, err = db.Exec(ctx, `
			INSERT INTO app_settings (key, value, value_type, description, is_public)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (key) DO NOTHING;
		`, s.Key, valBytes, s.ValueType, s.Description, s.IsPublic)
		if err != nil {
			return fmt.Errorf("seeding app_setting %s: %w", s.Key, err)
		}
	}
	log.Info("App settings seeded (insert-only) successfully", "count", len(initialAppSettings))

	// 5. Initial Super Admin user creation (§ 5.6)
	var userCount int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL;`).Scan(&userCount); err != nil {
		return fmt.Errorf("counting users: %w", err)
	}

	if userCount > 0 {
		log.Info("Existing users found; skipping initial super admin creation", "user_count", userCount)
		return nil
	}

	adminEmail := opts.SeedAdminEmail
	if adminEmail == "" {
		adminEmail = "admin@faceclock.local"
	}

	adminPassword := opts.SeedAdminPassword
	if adminPassword == "" {
		if opts.AppEnv == "development" {
			adminPassword = "Faceclock123!"
		} else {
			return errors.New("SEED_ADMIN_PASSWORD must be provided in non-development environment when no user exists")
		}
	}

	passHash, err := auth.HashPassword(adminPassword)
	if err != nil {
		return fmt.Errorf("hashing admin password: %w", err)
	}

	var newUserID uuid.UUID
	err = db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, is_active, must_change_password)
		VALUES ($1, $2, true, true)
		RETURNING id;
	`, adminEmail, passHash).Scan(&newUserID)
	if err != nil {
		return fmt.Errorf("creating initial super admin user: %w", err)
	}

	// Assign super_admin role
	_, err = db.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, role_id) DO NOTHING;
	`, newUserID, superAdminID)
	if err != nil {
		return fmt.Errorf("assigning super_admin role to initial user: %w", err)
	}

	// Print email to stdout (password is NEVER printed)
	fmt.Printf("Initial super admin created: %s (must change password on first login)\n", adminEmail)
	log.Info("Initial super admin created successfully", "email", adminEmail, "user_id", newUserID)

	return nil
}
