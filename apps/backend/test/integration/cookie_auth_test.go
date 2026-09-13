package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/auth"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/platform/logger"
)

func TestCookieAuthenticationAndAntiCSRFShield(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Pool.Close()

	// Create test user
	email := fmt.Sprintf("cookie_user_%d@faceclock.local", time.Now().UnixNano())
	password := "CookieSecretPass123!"
	passHash, _ := auth.HashPassword(password)

	var userID string
	err := app.Pool.QueryRow(t.Context(), `
		INSERT INTO users (email, password_hash, is_active, must_change_password)
		VALUES ($1, $2, true, false)
		RETURNING id::text
	`, email, passHash).Scan(&userID)
	if err != nil {
		t.Fatalf("failed creating test user: %v", err)
	}

	var empRoleID string
	_ = app.Pool.QueryRow(t.Context(), "SELECT id::text FROM roles WHERE name = 'employee'").Scan(&empRoleID)
	_, _ = app.Pool.Exec(t.Context(), "INSERT INTO user_roles (user_id, role_id) VALUES ($1::uuid, $2::uuid)", userID, empRoleID)

	// 1. POST /api/v1/auth/login sets HttpOnly access_token and refresh_token cookies
	loginPayload, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})

	loginReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginPayload))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec, loginBody := app.ExecuteRequest(loginReq)

	AssertNoSecretLeak(t, loginBody)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login failed: expected 200, got %d: %s", loginRec.Code, string(loginBody))
	}

	cookies := loginRec.Result().Cookies()
	var accessCookie, refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.AccessTokenCookieName {
			accessCookie = c
		}
		if c.Name == auth.RefreshTokenCookieName {
			refreshCookie = c
		}
	}

	if accessCookie == nil {
		t.Fatalf("expected access_token cookie to be set")
	}
	if !accessCookie.HttpOnly {
		t.Errorf("access_token must have HttpOnly: true")
	}
	if accessCookie.Path != "/" {
		t.Errorf("access_token path must be '/', got '%s'", accessCookie.Path)
	}
	if accessCookie.MaxAge != auth.AccessTokenTTL {
		t.Errorf("access_token MaxAge must be %d, got %d", auth.AccessTokenTTL, accessCookie.MaxAge)
	}
	if accessCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("access_token SameSite must be Lax, got %v", accessCookie.SameSite)
	}

	if refreshCookie == nil {
		t.Fatalf("expected refresh_token cookie to be set")
	}
	if !refreshCookie.HttpOnly {
		t.Errorf("refresh_token must have HttpOnly: true")
	}
	if refreshCookie.Path != "/api/v1/auth" {
		t.Errorf("refresh_token path must be '/api/v1/auth', got '%s'", refreshCookie.Path)
	}
	if refreshCookie.MaxAge != auth.RefreshTokenTTL {
		t.Errorf("refresh_token MaxAge must be %d, got %d", auth.RefreshTokenTTL, refreshCookie.MaxAge)
	}
	if refreshCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("refresh_token SameSite must be Lax, got %v", refreshCookie.SameSite)
	}

	// 2. Pure Cookie Authentication: GET /api/v1/auth/me using ONLY access_token cookie (NO Authorization header)
	meReq, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.AddCookie(accessCookie)
	meRec, meBody := app.ExecuteRequest(meReq)

	AssertNoSecretLeak(t, meBody)
	if meRec.Code != http.StatusOK {
		t.Fatalf("cookie-only authentication failed: expected 200, got %d: %s", meRec.Code, string(meBody))
	}

	var meResp struct {
		Data struct {
			Email string `json:"email"`
		} `json:"data"`
	}
	_ = json.Unmarshal(meBody, &meResp)
	if meResp.Data.Email != email {
		t.Errorf("expected email '%s', got '%s'", email, meResp.Data.Email)
	}

	// 3. Anti-CSRF Shield on Mutating Request:
	// 3a. POST /api/v1/auth/logout without X-Requested-With or X-CSRF-Token must be rejected with 403 CSRF_HEADER_MISSING
	csrfReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader([]byte(`{}`)))
	csrfReq.Header.Set("Content-Type", "application/json")
	csrfReq.Header.Set("X-Skip-CSRF-Header", "true") // tell test harness NOT to inject X-Requested-With
	csrfReq.AddCookie(accessCookie)
	csrfRec, csrfBody := app.ExecuteRequest(csrfReq)

	if csrfRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 FORBIDDEN when CSRF headers missing, got %d: %s", csrfRec.Code, string(csrfBody))
	}

	var csrfErrResp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(csrfBody, &csrfErrResp)
	if csrfErrResp.Error.Code != "CSRF_HEADER_MISSING" {
		t.Errorf("expected error code 'CSRF_HEADER_MISSING', got '%s'", csrfErrResp.Error.Code)
	}

	// 3b. Mutating request with untrusted Origin header must be rejected with 403 CSRF_UNTRUSTED_ORIGIN
	// (Test with non-wildcard router instance to verify origin checking)
	csrfOriginReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader([]byte(`{}`)))
	csrfOriginReq.Header.Set("Content-Type", "application/json")
	csrfOriginReq.Header.Set("X-Requested-With", "XMLHttpRequest")
	csrfOriginReq.Header.Set("Origin", "https://malicious-site.com")
	csrfOriginReq.AddCookie(accessCookie)

	// In test app config CORSAllowedOrigins is ["*"]. When allowed has "*", any origin is accepted in dev.
	// But let's verify with an isolated router having strict allowed origins:
	log := logger.New("error", io.Discard)
	strictRouter := httpx.NewRouter(httpx.RouterDeps{
		Logger:             log,
		CORSAllowedOrigins: []string{"https://app.faceclock.io"},
		RequestTimeout:     app.Config.RequestTimeout,
		MaxBodyBytes:       app.Config.MaxBodyBytes,
		AuthMiddleware:     auth.Authenticate(app.TokenMgr, app.RBACSvc, app.Pool),
		Handlers: httpx.Handlers{
			AuthLogout: auth.NewHandler(app.AuthSvc, app.AuditRec).Logout,
		},
	})
	strictRec := httptest.NewRecorder()
	strictRouter.ServeHTTP(strictRec, csrfOriginReq)
	if strictRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for untrusted Origin, got %d: %s", strictRec.Code, strictRec.Body.String())
	}
	var originErrResp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(strictRec.Body.Bytes(), &originErrResp)
	if originErrResp.Error.Code != "CSRF_UNTRUSTED_ORIGIN" {
		t.Errorf("expected error code 'CSRF_UNTRUSTED_ORIGIN', got '%s'", originErrResp.Error.Code)
	}

	// 4. Token Refresh via Cookie ONLY (no body sent)
	refreshReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	refreshReq.AddCookie(refreshCookie)
	refreshRec, refreshBody := app.ExecuteRequest(refreshReq)

	AssertNoSecretLeak(t, refreshBody)
	if refreshRec.Code != http.StatusOK {
		t.Fatalf("refresh via cookie failed: expected 200, got %d: %s", refreshRec.Code, string(refreshBody))
	}

	refreshedCookies := refreshRec.Result().Cookies()
	var newAccessCookie, newRefreshCookie *http.Cookie
	for _, c := range refreshedCookies {
		if c.Name == auth.AccessTokenCookieName {
			newAccessCookie = c
		}
		if c.Name == auth.RefreshTokenCookieName {
			newRefreshCookie = c
		}
	}
	if newAccessCookie == nil || newRefreshCookie == nil {
		t.Fatalf("expected new cookies upon refresh rotation")
	}

	// 5. Logout clears both cookies with MaxAge = -1
	logoutReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutReq.AddCookie(newAccessCookie)
	logoutReq.AddCookie(newRefreshCookie)
	logoutRec, _ := app.ExecuteRequest(logoutReq)

	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for logout, got %d", logoutRec.Code)
	}

	logoutCookies := logoutRec.Result().Cookies()
	for _, c := range logoutCookies {
		if (c.Name == auth.AccessTokenCookieName || c.Name == auth.RefreshTokenCookieName) && c.MaxAge != -1 {
			t.Errorf("expected cleared cookie %s to have MaxAge -1, got %d", c.Name, c.MaxAge)
		}
	}

	// 6. Full End-to-End Browser Session Simulation with http.CookieJar
	client := app.NewTestClient()

	// 6a. Browser calls login
	clientLoginRes, err := client.Post("http://localhost/api/v1/auth/login", "application/json", bytes.NewReader(loginPayload))
	if err != nil {
		t.Fatalf("client.Post login failed: %v", err)
	}
	defer clientLoginRes.Body.Close()
	if clientLoginRes.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on client login, got %d", clientLoginRes.StatusCode)
	}

	// 6b. Browser calls GET /api/v1/auth/me without manual token handling
	clientMeRes, err := client.Get("http://localhost/api/v1/auth/me")
	if err != nil {
		t.Fatalf("client.Get me failed: %v", err)
	}
	defer clientMeRes.Body.Close()
	if clientMeRes.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on client GET me, got %d", clientMeRes.StatusCode)
	}

	// 6c. Browser calls POST /api/v1/auth/refresh without body
	clientRefreshRes, err := client.Post("http://localhost/api/v1/auth/refresh", "application/json", nil)
	if err != nil {
		t.Fatalf("client.Post refresh failed: %v", err)
	}
	defer clientRefreshRes.Body.Close()
	if clientRefreshRes.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on client POST refresh, got %d", clientRefreshRes.StatusCode)
	}

	// 6d. Browser calls POST /api/v1/auth/logout
	clientLogoutRes, err := client.Post("http://localhost/api/v1/auth/logout", "application/json", nil)
	if err != nil {
		t.Fatalf("client.Post logout failed: %v", err)
	}
	defer clientLogoutRes.Body.Close()
	if clientLogoutRes.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 on client POST logout, got %d", clientLogoutRes.StatusCode)
	}

	// 6e. Subsequent calls fail with 401 UNAUTHENTICATED
	clientAfterLogoutRes, err := client.Get("http://localhost/api/v1/auth/me")
	if err != nil {
		t.Fatalf("client.Get me after logout failed: %v", err)
	}
	defer clientAfterLogoutRes.Body.Close()
	if clientAfterLogoutRes.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d", clientAfterLogoutRes.StatusCode)
	}
	clientAfterBody, _ := io.ReadAll(clientAfterLogoutRes.Body)
	AssertNoSecretLeak(t, clientAfterBody)
}
