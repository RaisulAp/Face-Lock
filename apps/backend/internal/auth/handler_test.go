package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
)

func TestAuthHandler_LoginAndRefresh(t *testing.T) {
	pool := getTestDB(t)
	_, handler, _ := setupTestServices(t, pool)

	email := fmt.Sprintf("test_hlogin_%d@faceclock.local", time.Now().UnixNano())
	password := "SecretPass123!"
	createTestUser(t, pool, email, password, true)

	// 1. Invalid payload
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString("{invalid-json"))
	req.Header.Set("Content-Type", "application/json")
	handler.Login(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}

	// 2. Successful login
	loginBody, _ := json.Marshal(LoginRequest{
		Email:    email,
		Password: password,
	})
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	handler.Login(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid login, got %d, body: %s", w.Code, w.Body.String())
	}

	var loginResp struct {
		Data LoginResult `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&loginResp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}
	if loginResp.Data.AccessToken == "" || loginResp.Data.RefreshToken == "" {
		t.Fatalf("expected access & refresh tokens")
	}

	// Verify login set cookies
	cookies := w.Result().Cookies()
	var foundAccess, foundRefresh bool
	for _, c := range cookies {
		if c.Name == AccessTokenCookieName && c.Value == loginResp.Data.AccessToken {
			foundAccess = true
		}
		if c.Name == RefreshTokenCookieName && c.Value == loginResp.Data.RefreshToken {
			foundRefresh = true
		}
	}
	if !foundAccess || !foundRefresh {
		t.Errorf("login did not set expected cookies: foundAccess=%v, foundRefresh=%v", foundAccess, foundRefresh)
	}

	// 3. Refresh token via Cookie (no body!)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  RefreshTokenCookieName,
		Value: loginResp.Data.RefreshToken,
	})
	handler.Refresh(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for refresh via cookie, got %d, body: %s", w.Code, w.Body.String())
	}

	// 4. Logout (clears cookies)
	principal := &rbac.Principal{
		UserID: loginResp.Data.User.ID,
		Email:  email,
		Roles:  []string{"employee"},
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  RefreshTokenCookieName,
		Value: loginResp.Data.RefreshToken,
	})
	req = req.WithContext(rbac.WithPrincipal(req.Context(), principal))
	handler.Logout(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for logout, got %d", w.Code)
	}
	for _, c := range w.Result().Cookies() {
		if (c.Name == AccessTokenCookieName || c.Name == RefreshTokenCookieName) && c.MaxAge != -1 {
			t.Errorf("expected cookie %s to have MaxAge -1, got %d", c.Name, c.MaxAge)
		}
	}
}

func TestAuthHandler_AuthenticatedEndpoints(t *testing.T) {
	pool := getTestDB(t)
	svc, handler, tokenMgr := setupTestServices(t, pool)

	email := fmt.Sprintf("test_hauth_%d@faceclock.local", time.Now().UnixNano())
	password := "SecretPass123!"
	userID := createTestUser(t, pool, email, password, true)

	principal := &rbac.Principal{
		UserID:       userID,
		Email:        email,
		TokenVersion: 1,
		Roles:        []string{"employee"},
		Permissions:  map[string]struct{}{"attendance.create": {}},
	}
	ctx := rbac.WithPrincipal(context.Background(), principal)

	// 1. GetMe
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil).WithContext(ctx)
	handler.GetMe(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for GetMe, got %d, body: %s", w.Code, w.Body.String())
	}

	var meResp struct {
		Data UserProfile `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&meResp); err != nil {
		t.Fatalf("failed to decode GetMe response: %v", err)
	}
	if meResp.Data.Email != email {
		t.Errorf("expected email %s, got %s", email, meResp.Data.Email)
	}

	// 2. Change Password
	cpBody, _ := json.Marshal(ChangePasswordRequest{
		CurrentPassword: password,
		NewPassword:     "UpdatedPass456!",
	})
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewReader(cpBody)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	handler.ChangePassword(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for ChangePassword, got %d, body: %s", w.Code, w.Body.String())
	}

	// 3. Revoke All
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/revoke-all", nil).WithContext(ctx)
	handler.RevokeAll(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for RevokeAll, got %d, body: %s", w.Code, w.Body.String())
	}

	_ = svc
	_ = tokenMgr
}

func TestAuthHandler_Unauthenticated(t *testing.T) {
	pool := getTestDB(t)
	_, handler, _ := setupTestServices(t, pool)

	// Logout without principal
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader([]byte("{}")))
	w := httptest.NewRecorder()
	handler.Logout(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated logout, got %d", w.Code)
	}

	// ChangePassword without principal (with valid body so decode succeeds)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewReader([]byte(`{"current_password":"OldPassword123!","new_password":"NewPassword123!"}`)))
	w = httptest.NewRecorder()
	handler.ChangePassword(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated change-password, got %d", w.Code)
	}

	// ChangePassword with invalid JSON
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewReader([]byte(`{invalid`)))
	w = httptest.NewRecorder()
	handler.ChangePassword(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad json in change-password, got %d", w.Code)
	}

	// Login with invalid JSON
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{invalid`)))
	w = httptest.NewRecorder()
	handler.Login(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad json in login, got %d", w.Code)
	}

	// Login with invalid credentials
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{"email":"nonexistent@faceclock.local","password":"WrongPassword123!"}`)))
	w = httptest.NewRecorder()
	handler.Login(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong credentials in login, got %d", w.Code)
	}

	// Refresh with invalid token
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader([]byte(`{"refresh_token":"invalid.refresh.token"}`)))
	w = httptest.NewRecorder()
	handler.Refresh(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid refresh token, got %d", w.Code)
	}

	// Refresh with validation error
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader([]byte(`{}`)))
	w = httptest.NewRecorder()
	handler.Refresh(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for validation error in refresh, got %d", w.Code)
	}

	// Refresh with invalid JSON
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader([]byte(`{invalid`)))
	w = httptest.NewRecorder()
	handler.Refresh(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad json in refresh, got %d", w.Code)
	}

	// GetMe without principal
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w = httptest.NewRecorder()
	handler.GetMe(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated get-me, got %d", w.Code)
	}

	// RevokeAll without principal
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/revoke-all", nil)
	w = httptest.NewRecorder()
	handler.RevokeAll(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated revoke-all, got %d", w.Code)
	}
}
