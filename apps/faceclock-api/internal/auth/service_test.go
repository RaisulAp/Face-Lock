package auth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func createTestUser(t *testing.T, pool *pgxpool.Pool, email, password string, isActive bool) uuid.UUID {
	t.Helper()
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	userID := uuid.New()
	_, err = pool.Exec(context.Background(), `
		INSERT INTO users (id, email, password_hash, is_active, token_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 1, NOW(), NOW())
		ON CONFLICT (email) WHERE deleted_at IS NULL DO UPDATE SET password_hash = $3, is_active = $4, token_version = 1, failed_login_count = 0, locked_until = NULL`,
		userID, email, hash, isActive)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return userID
}

func TestAuthService_Login_Success(t *testing.T) {
	pool := getTestDB(t)
	svc, _, _ := setupTestServices(t, pool)

	email := fmt.Sprintf("test_login_%d@faceclock.local", time.Now().UnixNano())
	password := "SecretPass123!"
	createTestUser(t, pool, email, password, true)

	ctx := context.Background()
	res, err := svc.Login(ctx, LoginInput{
		Email:     email,
		Password:  password,
		IPAddress: "127.0.0.1",
		UserAgent: "Go-Test/1.0",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if res.AccessToken == "" {
		t.Errorf("expected non-empty access token")
	}
	if res.RefreshToken == "" {
		t.Errorf("expected non-empty refresh token")
	}
	if res.User.Email != email {
		t.Errorf("expected email %s, got %s", email, res.User.Email)
	}
	if res.ExpiresIn <= 0 {
		t.Errorf("expected positive expires_in")
	}
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	pool := getTestDB(t)
	svc, _, _ := setupTestServices(t, pool)

	ctx := context.Background()

	// Non-existent user
	_, err := svc.Login(ctx, LoginInput{
		Email:     "nonexistent@faceclock.local",
		Password:  "WrongPassword123!",
		IPAddress: "127.0.0.1",
		UserAgent: "Go-Test/1.0",
	})
	if err == nil {
		t.Fatalf("expected error for non-existent user")
	}
	appErr, ok := err.(*httpx.AppError)
	if !ok || appErr.Code != httpx.CodeInvalidCredentials {
		t.Errorf("expected CodeInvalidCredentials, got: %v", err)
	}

	// Existing user with wrong password
	email := fmt.Sprintf("test_wrong_%d@faceclock.local", time.Now().UnixNano())
	createTestUser(t, pool, email, "CorrectPassword123!", true)

	_, err = svc.Login(ctx, LoginInput{
		Email:     email,
		Password:  "WrongPassword123!",
		IPAddress: "127.0.0.1",
		UserAgent: "Go-Test/1.0",
	})
	if err == nil {
		t.Fatalf("expected error for wrong password")
	}
	appErr, ok = err.(*httpx.AppError)
	if !ok || appErr.Code != httpx.CodeInvalidCredentials {
		t.Errorf("expected CodeInvalidCredentials, got: %v", err)
	}
}

func TestAuthService_Login_InactiveUser(t *testing.T) {
	pool := getTestDB(t)
	svc, _, _ := setupTestServices(t, pool)

	email := fmt.Sprintf("test_inactive_%d@faceclock.local", time.Now().UnixNano())
	password := "SecretPass123!"
	createTestUser(t, pool, email, password, false)

	ctx := context.Background()
	_, err := svc.Login(ctx, LoginInput{
		Email:     email,
		Password:  password,
		IPAddress: "127.0.0.1",
		UserAgent: "Go-Test/1.0",
	})
	if err == nil {
		t.Fatalf("expected error for inactive user")
	}
	appErr, ok := err.(*httpx.AppError)
	if !ok || appErr.Code != httpx.CodeInvalidCredentials {
		t.Errorf("expected CodeInvalidCredentials for inactive user, got: %v", err)
	}
}

func TestAuthService_Login_AccountLockout(t *testing.T) {
	pool := getTestDB(t)
	svc, _, _ := setupTestServices(t, pool)

	email := fmt.Sprintf("test_lockout_%d@faceclock.local", time.Now().UnixNano())
	password := "CorrectPass123!"
	createTestUser(t, pool, email, password, true)

	ctx := context.Background()
	// 5 failed attempts
	for i := 0; i < 5; i++ {
		_, err := svc.Login(ctx, LoginInput{
			Email:     email,
			Password:  "WrongPass123!",
			IPAddress: "127.0.0.1",
			UserAgent: "Go-Test/1.0",
		})
		if err == nil {
			t.Fatalf("expected failed login on attempt %d", i+1)
		}
	}

	// 6th attempt should be locked out (returns CodeInvalidCredentials for security)
	_, err := svc.Login(ctx, LoginInput{
		Email:     email,
		Password:  password,
		IPAddress: "127.0.0.1",
		UserAgent: "Go-Test/1.0",
	})
	if err == nil {
		t.Fatalf("expected error when locked out")
	}
	appErr, ok := err.(*httpx.AppError)
	if !ok || appErr.Code != httpx.CodeInvalidCredentials {
		t.Errorf("expected CodeInvalidCredentials, got: %v", err)
	}
}

func TestAuthService_RefreshToken_RotationAndReuseDetection(t *testing.T) {
	pool := getTestDB(t)
	svc, _, _ := setupTestServices(t, pool)

	email := fmt.Sprintf("test_refresh_%d@faceclock.local", time.Now().UnixNano())
	password := "SecretPass123!"
	createTestUser(t, pool, email, password, true)

	ctx := context.Background()
	loginRes, err := svc.Login(ctx, LoginInput{
		Email:     email,
		Password:  password,
		IPAddress: "127.0.0.1",
		UserAgent: "Go-Test/1.0",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	origRefreshToken := loginRes.RefreshToken

	// 1. Rotate token
	refreshRes, err := svc.Refresh(ctx, origRefreshToken, "127.0.0.1", "Go-Test/1.0")
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if refreshRes.AccessToken == "" || refreshRes.RefreshToken == "" {
		t.Fatalf("expected non-empty rotated tokens")
	}
	if refreshRes.RefreshToken == origRefreshToken {
		t.Fatalf("expected new refresh token to be different from original")
	}

	// Age the family tokens beyond the 30-second grace period
	oldHash := HashRefreshToken(origRefreshToken)
	_, err = pool.Exec(ctx, `
		UPDATE refresh_tokens 
		SET created_at = NOW() - INTERVAL '45 seconds'
		WHERE family_id = (SELECT family_id FROM refresh_tokens WHERE token_hash = $1 LIMIT 1)`, oldHash)
	if err != nil {
		t.Fatalf("failed to age family tokens: %v", err)
	}

	// 2. Reuse the old refresh token -> Reuse detection triggered!
	_, err = svc.Refresh(ctx, origRefreshToken, "127.0.0.1", "Go-Test/1.0")
	if err == nil {
		t.Fatalf("expected reuse detection error")
	}
	appErr, ok := err.(*httpx.AppError)
	if !ok || appErr.Code != httpx.CodeRefreshTokenReused {
		t.Errorf("expected CodeRefreshTokenReused, got: %v", err)
	}

	// 3. Since family was invalidated, the rotated new token should now also fail!
	_, err = svc.Refresh(ctx, refreshRes.RefreshToken, "127.0.0.1", "Go-Test/1.0")
	if err == nil {
		t.Fatalf("expected rotated token to fail after family revocation")
	}
}

func TestAuthService_RefreshToken_GraceWindow(t *testing.T) {
	pool := getTestDB(t)
	svc, _, _ := setupTestServices(t, pool)

	email := fmt.Sprintf("test_grace_%d@faceclock.local", time.Now().UnixNano())
	password := "SecretPass123!"
	createTestUser(t, pool, email, password, true)

	ctx := context.Background()
	loginRes, err := svc.Login(ctx, LoginInput{
		Email:     email,
		Password:  password,
		IPAddress: "127.0.0.1",
		UserAgent: "Go-Test/1.0",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	origRefreshToken := loginRes.RefreshToken

	// 1. First rotation
	refreshRes1, err := svc.Refresh(ctx, origRefreshToken, "127.0.0.1", "Go-Test/1.0")
	if err != nil {
		t.Fatalf("first refresh failed: %v", err)
	}

	// 2. Immediate second rotation with origRefreshToken within 30s grace window (e.g. mobile network glitch)
	refreshRes2, err := svc.Refresh(ctx, origRefreshToken, "127.0.0.1", "Go-Test/1.0")
	if err != nil {
		t.Fatalf("grace refresh failed: %v", err)
	}
	if refreshRes2.AccessToken == "" || refreshRes2.RefreshToken == "" {
		t.Errorf("expected valid tokens from grace refresh")
	}
	_ = refreshRes1
}

func TestAuthService_ChangePassword(t *testing.T) {
	pool := getTestDB(t)
	svc, _, _ := setupTestServices(t, pool)

	email := fmt.Sprintf("test_cp_%d@faceclock.local", time.Now().UnixNano())
	password := "OldPassword123!"
	userID := createTestUser(t, pool, email, password, true)

	ctx := context.Background()

	// 1. Wrong current password
	err := svc.ChangePassword(ctx, userID, "WrongOldPass123!", "NewPassword123!")
	if err == nil {
		t.Fatalf("expected error on wrong current password")
	}
	appErr, ok := err.(*httpx.AppError)
	if !ok || appErr.Code != httpx.CodeInvalidCredentials {
		t.Errorf("expected CodeInvalidCredentials, got: %v", err)
	}

	// 2. Weak new password
	err = svc.ChangePassword(ctx, userID, password, "weak")
	if err == nil {
		t.Fatalf("expected error on weak new password")
	}
	appErr, ok = err.(*httpx.AppError)
	if !ok || appErr.Code != httpx.CodeValidationError {
		t.Errorf("expected CodeValidationError, got: %v", err)
	}

	// 3. Same new password as old password
	err = svc.ChangePassword(ctx, userID, password, password)
	if err == nil {
		t.Fatalf("expected error on same new password")
	}
	appErr, ok = err.(*httpx.AppError)
	if !ok || appErr.Code != httpx.CodeConflict {
		t.Errorf("expected CodeConflict, got: %v", err)
	}

	// 4. Non-existent user
	err = svc.ChangePassword(ctx, uuid.New(), "any", "NewPassword123!")
	if err == nil {
		t.Fatalf("expected error on non-existent user")
	}

	// 5. Successful change
	newPassword := "BrandNewPass456!"
	err = svc.ChangePassword(ctx, userID, password, newPassword)
	if err != nil {
		t.Fatalf("change password failed: %v", err)
	}

	// 4. Old password can no longer login
	_, err = svc.Login(ctx, LoginInput{
		Email:     email,
		Password:  password,
		IPAddress: "127.0.0.1",
		UserAgent: "Go-Test",
	})
	if err == nil {
		t.Fatalf("expected old password to fail")
	}

	// 5. New password logs in successfully
	loginRes, err := svc.Login(ctx, LoginInput{
		Email:     email,
		Password:  newPassword,
		IPAddress: "127.0.0.1",
		UserAgent: "Go-Test",
	})
	if err != nil {
		t.Fatalf("login with new password failed: %v", err)
	}
	if loginRes.AccessToken == "" {
		t.Errorf("expected valid access token")
	}
}

func TestAuthService_Revocation(t *testing.T) {
	pool := getTestDB(t)
	svc, _, _ := setupTestServices(t, pool)

	email := fmt.Sprintf("test_rev_%d@faceclock.local", time.Now().UnixNano())
	password := "SecretPass123!"
	userID := createTestUser(t, pool, email, password, true)

	ctx := context.Background()
	loginRes, err := svc.Login(ctx, LoginInput{
		Email:     email,
		Password:  password,
		IPAddress: "127.0.0.1",
		UserAgent: "Go-Test/1.0",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Revoke refresh token
	err = svc.Logout(ctx, userID, loginRes.RefreshToken)
	if err != nil {
		t.Fatalf("revoke failed: %v", err)
	}

	// Revoke all user sessions
	err = svc.RevokeAll(ctx, userID)
	if err != nil {
		t.Fatalf("revoke all sessions failed: %v", err)
	}

	// 4. GetMe non-existent user
	_, err = svc.GetMe(ctx, uuid.New())
	if err == nil {
		t.Fatalf("expected error on get me for non-existent user")
	}
}
