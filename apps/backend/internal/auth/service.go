package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/config"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/settings"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LoginInput holds user credentials and device info.
type LoginInput struct {
	Email      string  `json:"email" validate:"required,email"`
	Password   string  `json:"password" validate:"required"`
	DeviceInfo *string `json:"device_info,omitempty"`
	IPAddress  string  `json:"-"`
	UserAgent  string  `json:"-"`
}

// UserProfile represents the authenticated user's profile summary.
type UserProfile struct {
	ID                 uuid.UUID  `json:"id"`
	Email              string     `json:"email"`
	EmployeeID         *uuid.UUID `json:"employee_id,omitempty"`
	EmployeeName       *string    `json:"employee_name,omitempty"`
	EmployeeNumber     *string    `json:"employee_number,omitempty"`
	Roles              []string   `json:"roles"`
	Permissions        []string   `json:"permissions"`
	MustChangePassword bool       `json:"must_change_password"`
}

// LoginResult contains tokens and user summary.
type LoginResult struct {
	AccessToken        string      `json:"access_token"`
	RefreshToken       string      `json:"refresh_token"`
	TokenType          string      `json:"token_type"`
	ExpiresIn          int64       `json:"expires_in"`
	MustChangePassword bool        `json:"must_change_password"`
	User               UserProfile `json:"user"`
}

// RefreshResult contains a rotated token pair.
type RefreshResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// Service provides authentication operations.
type Service struct {
	db           *pgxpool.Pool
	cfg          *config.Config
	tokenManager *TokenManager
	settingsSvc  *settings.Service
	rbacSvc      *rbac.Service
	audit        *audit.Recorder
}

// NewService creates a new auth service.
func NewService(
	db *pgxpool.Pool,
	cfg *config.Config,
	tokenManager *TokenManager,
	settingsSvc *settings.Service,
	rbacSvc *rbac.Service,
	audit *audit.Recorder,
) *Service {
	return &Service{
		db:           db,
		cfg:          cfg,
		tokenManager: tokenManager,
		settingsSvc:  settingsSvc,
		rbacSvc:      rbacSvc,
		audit:        audit,
	}
}

// IsDev returns true if APP_ENV is development.
func (s *Service) IsDev() bool {
	return s != nil && s.cfg != nil && s.cfg.AppEnv == "development"
}

// Login authenticates a user and generates tokens.
func (s *Service) Login(ctx context.Context, input LoginInput) (*LoginResult, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))

	const findUserQ = `
		SELECT u.id, u.employee_id, u.email, u.password_hash, u.is_active,
		       u.must_change_password, u.failed_login_count, u.locked_until,
		       u.token_version, u.deleted_at,
		       e.full_name, e.employee_number
		FROM users u
		LEFT JOIN employees e ON u.employee_id = e.id
		WHERE u.email = $1
	`
	var (
		id                 uuid.UUID
		employeeID         *uuid.UUID
		userEmail          string
		passwordHash       string
		isActive           bool
		mustChangePassword bool
		failedLoginCount   int
		lockedUntil        *time.Time
		tokenVersion       int
		deletedAt          *time.Time
		employeeName       *string
		employeeNumber     *string
	)

	err := s.db.QueryRow(ctx, findUserQ, email).Scan(
		&id,
		&employeeID,
		&userEmail,
		&passwordHash,
		&isActive,
		&mustChangePassword,
		&failedLoginCount,
		&lockedUntil,
		&tokenVersion,
		&deletedAt,
		&employeeName,
		&employeeNumber,
	)

	now := time.Now().UTC()

	// Constant-time mitigation against timing attacks if user does not exist
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_, _ = VerifyPassword(input.Password, DummyHash)
			s.recordLoginAudit(ctx, nil, email, false, "user_not_found", input)
			return nil, httpx.NewAppError(httpx.CodeInvalidCredentials, "invalid email or password")
		}
		return nil, fmt.Errorf("querying user: %w", err)
	}

	// Check if account is deleted or inactive
	if deletedAt != nil || !isActive {
		_, _ = VerifyPassword(input.Password, passwordHash)
		s.recordLoginAudit(ctx, &id, email, false, "account_deactivated", input)
		return nil, httpx.NewAppError(httpx.CodeInvalidCredentials, "invalid email or password")
	}

	// Check lockout (security: return generic 401 INVALID_CREDENTIALS)
	if lockedUntil != nil && lockedUntil.After(now) {
		s.recordLoginAudit(ctx, &id, email, false, "account_locked", input)
		return nil, httpx.NewAppError(httpx.CodeInvalidCredentials, "invalid email or password")
	}

	// Verify password
	ok, _ := VerifyPassword(input.Password, passwordHash)
	if !ok {
		newFailCount := failedLoginCount + 1
		maxAttempts := s.settingsSvc.GetInt(ctx, "security.max_failed_logins", 5)
		if maxAttempts <= 0 {
			maxAttempts = 5
		}
		lockoutMinutes := s.settingsSvc.GetInt(ctx, "security.lockout_minutes", 15)
		if lockoutMinutes <= 0 {
			lockoutMinutes = 15
		}

		var newLockUntil *time.Time
		if newFailCount >= maxAttempts {
			t := now.Add(time.Duration(lockoutMinutes) * time.Minute)
			newLockUntil = &t
		}

		const updateFailQ = `
			UPDATE users
			SET failed_login_count = $1, locked_until = $2, updated_at = NOW()
			WHERE id = $3
		`
		_, _ = s.db.Exec(ctx, updateFailQ, newFailCount, newLockUntil, id)

		s.recordLoginAudit(ctx, &id, email, false, "invalid_password", input)
		return nil, httpx.NewAppError(httpx.CodeInvalidCredentials, "invalid email or password")
	}

	// Password OK - reset failed attempts
	const resetFailQ = `
		UPDATE users
		SET failed_login_count = 0, locked_until = NULL, updated_at = NOW()
		WHERE id = $1
	`
	_, _ = s.db.Exec(ctx, resetFailQ, id)

	// Generate Access Token
	accessToken, _, err := s.tokenManager.GenerateAccessToken(id, employeeID, tokenVersion)
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	// Generate Refresh Token
	rawRefreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}
	tokenHash := HashRefreshToken(rawRefreshToken)

	familyID := uuid.New()
	expiresAt := now.Add(s.cfg.JWTRefreshTTL)

	var ipParam any
	if parsedIP := net.ParseIP(input.IPAddress); parsedIP != nil {
		ipParam = parsedIP.String()
	}

	const insertRefreshTokenQ = `
		INSERT INTO refresh_tokens (
			user_id, token_hash, family_id, parent_id, is_revoked, expires_at, created_at, ip_address, user_agent
		) VALUES (
			$1, $2, $3, NULL, false, $4, NOW(), $5, $6
		)
	`
	if _, err := s.db.Exec(ctx, insertRefreshTokenQ, id, tokenHash, familyID, expiresAt, ipParam, input.UserAgent); err != nil {
		return nil, fmt.Errorf("storing refresh token: %w", err)
	}

	// Fetch roles & permissions
	principal, err := s.rbacSvc.GetPrincipal(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching principal permissions: %w", err)
	}

	permList := make([]string, 0, len(principal.Permissions))
	for p := range principal.Permissions {
		permList = append(permList, p)
	}
	sort.Strings(permList)

	s.recordLoginAudit(ctx, &id, email, true, "success", input)

	return &LoginResult{
		AccessToken:        accessToken,
		RefreshToken:       rawRefreshToken,
		TokenType:          "Bearer",
		ExpiresIn:          int64(s.cfg.JWTAccessTTL.Seconds()),
		MustChangePassword: mustChangePassword,
		User: UserProfile{
			ID:                 id,
			Email:              userEmail,
			EmployeeID:         employeeID,
			EmployeeName:       employeeName,
			EmployeeNumber:     employeeNumber,
			Roles:              principal.Roles,
			Permissions:        permList,
			MustChangePassword: mustChangePassword,
		},
	}, nil
}

// Refresh rotates a refresh token and generates a new token pair with 30s reuse grace window.
func (s *Service) Refresh(ctx context.Context, rawRefreshToken, ipAddress, userAgent string) (*RefreshResult, error) {
	if rawRefreshToken == "" {
		return nil, httpx.NewAppError(httpx.CodeInvalidRefreshToken, "refresh token is required")
	}

	tokenHash := HashRefreshToken(rawRefreshToken)
	now := time.Now().UTC()

	const findTokenQ = `
		SELECT rt.id, rt.user_id, rt.family_id, rt.is_revoked, rt.revoked_reason,
		       rt.expires_at, rt.created_at,
		       u.is_active, u.deleted_at, u.employee_id, u.token_version
		FROM refresh_tokens rt
		JOIN users u ON rt.user_id = u.id
		WHERE rt.token_hash = $1
	`
	var (
		tokenID       uuid.UUID
		userID        uuid.UUID
		familyID      uuid.UUID
		isRevoked     bool
		revokedReason *string
		expiresAt     time.Time
		createdAt     time.Time
		isActive      bool
		deletedAt     *time.Time
		employeeID    *uuid.UUID
		tokenVersion  int
	)

	err := s.db.QueryRow(ctx, findTokenQ, tokenHash).Scan(
		&tokenID,
		&userID,
		&familyID,
		&isRevoked,
		&revokedReason,
		&expiresAt,
		&createdAt,
		&isActive,
		&deletedAt,
		&employeeID,
		&tokenVersion,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NewAppError(httpx.CodeInvalidRefreshToken, "invalid refresh token")
		}
		return nil, fmt.Errorf("querying refresh token: %w", err)
	}

	if deletedAt != nil || !isActive {
		return nil, httpx.NewAppError(httpx.CodeInvalidRefreshToken, "account is inactive or deactivated")
	}

	// Handle revoked token: Grace Window vs Invalidation
	if isRevoked {
		graceSec := s.settingsSvc.GetInt(ctx, "auth.refresh_token_reuse_grace_seconds", 30)

		// Check if a child token exists and was created recently (within graceSec)
		const findChildQ = `
			SELECT id, created_at
			FROM refresh_tokens
			WHERE parent_id = $1
			ORDER BY created_at DESC
			LIMIT 1
		`
		var childID uuid.UUID
		var childCreatedAt time.Time
		childErr := s.db.QueryRow(ctx, findChildQ, tokenID).Scan(&childID, &childCreatedAt)

		if childErr == nil && now.Sub(childCreatedAt) <= time.Duration(graceSec)*time.Second {
			// Inside grace window: create new pair off this family without invalidation
			newRawRefresh, err := GenerateRefreshToken()
			if err != nil {
				return nil, fmt.Errorf("generating grace refresh token: %w", err)
			}
			newHash := HashRefreshToken(newRawRefresh)
			newExpiresAt := now.Add(s.cfg.JWTRefreshTTL)
			const insGraceQ = `
				INSERT INTO refresh_tokens (
					user_id, token_hash, family_id, parent_id, is_revoked, expires_at, created_at, ip_address, user_agent
				) VALUES (
					$1, $2, $3, $4, false, $5, NOW(), $6, $7
				)
			`
			var ipParam any
			if parsedIP := net.ParseIP(ipAddress); parsedIP != nil {
				ipParam = parsedIP.String()
			}
			if _, err := s.db.Exec(ctx, insGraceQ, userID, newHash, familyID, childID, newExpiresAt, ipParam, userAgent); err != nil {
				return nil, fmt.Errorf("storing grace token: %w", err)
			}

			newAccessToken, _, err := s.tokenManager.GenerateAccessToken(userID, employeeID, tokenVersion)
			if err != nil {
				return nil, fmt.Errorf("generating access token: %w", err)
			}

			return &RefreshResult{
				AccessToken:  newAccessToken,
				RefreshToken: newRawRefresh,
				TokenType:    "Bearer",
				ExpiresIn:    int64(s.cfg.JWTAccessTTL.Seconds()),
			}, nil
		}

		// Beyond grace window -> TOKEN REUSE DETECTED: INVALIDATE WHOLE FAMILY!
		const invalidateFamilyQ = `
			UPDATE refresh_tokens
			SET is_revoked = true, revoked_reason = 'token_family_invalidated'
			WHERE family_id = $1 AND is_revoked = false
		`
		_, _ = s.db.Exec(ctx, invalidateFamilyQ, familyID)

		// Bump token version
		const bumpVersionQ = `UPDATE users SET token_version = token_version + 1, updated_at = NOW() WHERE id = $1`
		_, _ = s.db.Exec(ctx, bumpVersionQ, userID)
		s.rbacSvc.InvalidateUser(userID)

		resID := userID.String()
		_ = s.audit.Record(ctx, audit.LogEntry{
			ActorUserID:  &userID,
			Action:       "auth.token_reuse_detected",
			ResourceType: "user",
			ResourceID:   &resID,
			IP:           ipAddress,
			UserAgent:    userAgent,
			Metadata: map[string]any{
				"family_id": familyID.String(),
			},
		})

		return nil, httpx.NewAppError(httpx.CodeRefreshTokenReused, "refresh token reuse detected; session has been invalidated")
	}

	// Check expiration
	if expiresAt.Before(now) {
		return nil, httpx.NewAppError(httpx.CodeInvalidRefreshToken, "refresh token expired")
	}

	// Rotate token
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const revokeOldQ = `
		UPDATE refresh_tokens
		SET is_revoked = true, revoked_reason = 'rotated'
		WHERE id = $1
	`
	if _, err := tx.Exec(ctx, revokeOldQ, tokenID); err != nil {
		return nil, fmt.Errorf("revoking old token: %w", err)
	}

	newRawRefresh, err := GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}
	newHash := HashRefreshToken(newRawRefresh)
	newExpiresAt := now.Add(s.cfg.JWTRefreshTTL)

	var ipParam any
	if parsedIP := net.ParseIP(ipAddress); parsedIP != nil {
		ipParam = parsedIP.String()
	}

	const insNewTokenQ = `
		INSERT INTO refresh_tokens (
			user_id, token_hash, family_id, parent_id, is_revoked, expires_at, created_at, ip_address, user_agent
		) VALUES (
			$1, $2, $3, $4, false, $5, NOW(), $6, $7
		)
	`
	if _, err := tx.Exec(ctx, insNewTokenQ, userID, newHash, familyID, tokenID, newExpiresAt, ipParam, userAgent); err != nil {
		return nil, fmt.Errorf("inserting rotated token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing rotated token: %w", err)
	}

	newAccessToken, _, err := s.tokenManager.GenerateAccessToken(userID, employeeID, tokenVersion)
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	return &RefreshResult{
		AccessToken:  newAccessToken,
		RefreshToken: newRawRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.cfg.JWTAccessTTL.Seconds()),
	}, nil
}

// Logout revokes the given refresh token.
func (s *Service) Logout(ctx context.Context, userID uuid.UUID, rawRefreshToken string) error {
	if rawRefreshToken == "" {
		return nil
	}

	tokenHash := HashRefreshToken(rawRefreshToken)
	const q = `
		UPDATE refresh_tokens
		SET is_revoked = true, revoked_reason = 'logout'
		WHERE token_hash = $1 AND user_id = $2
	`
	_, err := s.db.Exec(ctx, q, tokenHash, userID)
	return err
}

// ChangePassword verifies old password and updates to new password.
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	const getPwdQ = `SELECT password_hash FROM users WHERE id = $1 AND deleted_at IS NULL`
	var currentHash string
	err := s.db.QueryRow(ctx, getPwdQ, userID).Scan(&currentHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.NewAppError(httpx.CodeNotFound, "user not found")
		}
		return fmt.Errorf("querying user: %w", err)
	}

	if ok, _ := VerifyPassword(currentPassword, currentHash); !ok {
		return httpx.NewAppError(httpx.CodeInvalidCredentials, "current password is incorrect")
	}

	if ok, _ := VerifyPassword(newPassword, currentHash); ok {
		return httpx.NewAppError(httpx.CodeConflict, "new password cannot be the same as current password")
	}

	if err := ValidatePassword(newPassword, 10); err != nil {
		return httpx.NewAppError(httpx.CodeValidationError, err.Error())
	}

	newHash, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hashing new password: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const updatePwdQ = `
		UPDATE users
		SET password_hash = $1, must_change_password = false, token_version = token_version + 1, updated_at = NOW()
		WHERE id = $2
	`
	if _, err := tx.Exec(ctx, updatePwdQ, newHash, userID); err != nil {
		return fmt.Errorf("updating password: %w", err)
	}

	const revokeTokensQ = `
		UPDATE refresh_tokens
		SET is_revoked = true, revoked_reason = 'password_changed'
		WHERE user_id = $1 AND is_revoked = false
	`
	if _, err := tx.Exec(ctx, revokeTokensQ, userID); err != nil {
		return fmt.Errorf("revoking refresh tokens: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing password change: %w", err)
	}

	s.rbacSvc.InvalidateUser(userID)
	return nil
}

// RevokeAll revokes all refresh tokens and bumps token version.
func (s *Service) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const bumpQ = `UPDATE users SET token_version = token_version + 1, updated_at = NOW() WHERE id = $1`
	if _, err := tx.Exec(ctx, bumpQ, userID); err != nil {
		return fmt.Errorf("bumping token version: %w", err)
	}

	const revokeQ = `
		UPDATE refresh_tokens
		SET is_revoked = true, revoked_reason = 'user_revoked_all'
		WHERE user_id = $1 AND is_revoked = false
	`
	if _, err := tx.Exec(ctx, revokeQ, userID); err != nil {
		return fmt.Errorf("revoking tokens: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing revoke all: %w", err)
	}

	s.rbacSvc.InvalidateUser(userID)
	return nil
}

// GetMe fetches the full user profile including roles and permissions.
func (s *Service) GetMe(ctx context.Context, userID uuid.UUID) (*UserProfile, error) {
	const q = `
		SELECT u.id, u.email, u.employee_id, u.must_change_password,
		       e.full_name, e.employee_number
		FROM users u
		LEFT JOIN employees e ON u.employee_id = e.id
		WHERE u.id = $1 AND u.deleted_at IS NULL
	`
	var (
		id                 uuid.UUID
		email              string
		employeeID         *uuid.UUID
		mustChangePassword bool
		employeeName       *string
		employeeNumber     *string
	)
	err := s.db.QueryRow(ctx, q, userID).Scan(
		&id,
		&email,
		&employeeID,
		&mustChangePassword,
		&employeeName,
		&employeeNumber,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "user not found")
		}
		return nil, fmt.Errorf("querying user profile: %w", err)
	}

	p, err := s.rbacSvc.GetPrincipal(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("querying principal permissions: %w", err)
	}

	permList := make([]string, 0, len(p.Permissions))
	for perm := range p.Permissions {
		permList = append(permList, perm)
	}
	sort.Strings(permList)

	return &UserProfile{
		ID:                 id,
		Email:              email,
		EmployeeID:         employeeID,
		EmployeeName:       employeeName,
		EmployeeNumber:     employeeNumber,
		Roles:              p.Roles,
		Permissions:        permList,
		MustChangePassword: mustChangePassword,
	}, nil
}

func (s *Service) recordLoginAudit(ctx context.Context, userID *uuid.UUID, email string, success bool, reason string, input LoginInput) {
	action := "auth.login_failed"
	if success {
		action = "auth.login_success"
	}
	var resID *string
	if userID != nil {
		idStr := userID.String()
		resID = &idStr
	}
	_ = s.audit.Record(ctx, audit.LogEntry{
		ActorUserID:  userID,
		Action:       action,
		ResourceType: "user",
		ResourceID:   resID,
		IP:           input.IPAddress,
		UserAgent:    input.UserAgent,
		Metadata: map[string]any{
			"email":   email,
			"reason":  reason,
			"success": success,
		},
	})
}
