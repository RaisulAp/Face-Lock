package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
)

func TestAuthenticateMiddleware(t *testing.T) {
	pool := getTestDB(t)
	_, _, tokenMgr := setupTestServices(t, pool)
	rbacCache := rbac.NewCache(30 * time.Second)
	rbacSvc := rbac.NewService(pool, rbacCache)
	authMiddleware := Authenticate(tokenMgr, rbacSvc, pool)

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := rbac.GetPrincipal(r.Context())
		if !ok || p == nil {
			http.Error(w, "no principal", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(p.Email))
	})
	handlerToTest := authMiddleware(dummyHandler)

	// 1. Missing Authorization header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handlerToTest.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing header, got %d", w.Code)
	}

	// 2. Malformed Authorization header (not Bearer)
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Basic 12345")
	w = httptest.NewRecorder()
	handlerToTest.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for non-Bearer auth, got %d", w.Code)
	}

	// 3. Invalid token string
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-token")
	w = httptest.NewRecorder()
	handlerToTest.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid token, got %d", w.Code)
	}

	// Create a test user in DB
	email := fmt.Sprintf("test_mid_%d@faceclock.local", time.Now().UnixNano())
	password := "SecretPass123!"
	userID := createTestUser(t, pool, email, password, true)

	// 4. Valid token (Bearer header)
	validToken, _, err := tokenMgr.GenerateAccessToken(userID, nil, 1)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	w = httptest.NewRecorder()
	handlerToTest.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Body.String() != email {
		t.Errorf("expected 200 with email, got %d, body: %s", w.Code, w.Body.String())
	}

	// 4b. Valid token (HttpOnly Cookie, NO Authorization header)
	reqCookie := httptest.NewRequest(http.MethodGet, "/test", nil)
	reqCookie.AddCookie(&http.Cookie{
		Name:  AccessTokenCookieName,
		Value: validToken,
	})
	wCookie := httptest.NewRecorder()
	handlerToTest.ServeHTTP(wCookie, reqCookie)
	if wCookie.Code != http.StatusOK || wCookie.Body.String() != email {
		t.Errorf("expected 200 with email via cookie, got %d, body: %s", wCookie.Code, wCookie.Body.String())
	}

	// 5. Token version mismatch (stale token)
	staleToken, _, err := tokenMgr.GenerateAccessToken(userID, nil, 0)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+staleToken)
	w = httptest.NewRecorder()
	handlerToTest.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for stale token version, got %d", w.Code)
	}

	// 6. Inactive account
	inactiveEmail := fmt.Sprintf("test_mid_inact_%d@faceclock.local", time.Now().UnixNano())
	inactiveID := createTestUser(t, pool, inactiveEmail, password, false)
	inactiveToken, _, _ := tokenMgr.GenerateAccessToken(inactiveID, nil, 1)
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+inactiveToken)
	w = httptest.NewRecorder()
	handlerToTest.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for inactive user, got %d", w.Code)
	}

	// 7. Temporarily locked account
	lockedEmail := fmt.Sprintf("test_mid_lock_%d@faceclock.local", time.Now().UnixNano())
	lockedID := createTestUser(t, pool, lockedEmail, password, true)
	_, err = pool.Exec(context.Background(), `UPDATE users SET locked_until = NOW() + INTERVAL '10 minutes' WHERE id = $1`, lockedID)
	if err != nil {
		t.Fatalf("failed to lock user: %v", err)
	}
	lockedToken, _, _ := tokenMgr.GenerateAccessToken(lockedID, nil, 1)
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+lockedToken)
	w = httptest.NewRecorder()
	handlerToTest.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for locked user, got %d", w.Code)
	}
}
