package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetAuthCookies(t *testing.T) {
	// 1. Development mode (Secure: false)
	recDev := httptest.NewRecorder()
	SetAuthCookies(recDev, "mock-access-token", "mock-refresh-token", true)

	resDev := recDev.Result()
	cookiesDev := resDev.Cookies()
	if len(cookiesDev) != 2 {
		t.Fatalf("expected 2 cookies in dev mode, got %d", len(cookiesDev))
	}

	cookieMapDev := make(map[string]*http.Cookie)
	for _, c := range cookiesDev {
		cookieMapDev[c.Name] = c
	}

	accessCookie, ok := cookieMapDev[AccessTokenCookieName]
	if !ok {
		t.Fatalf("missing access_token cookie")
	}
	if accessCookie.Value != "mock-access-token" {
		t.Errorf("expected access_token value 'mock-access-token', got %s", accessCookie.Value)
	}
	if accessCookie.Path != "/" {
		t.Errorf("expected access_token path '/', got %s", accessCookie.Path)
	}
	if accessCookie.MaxAge != AccessTokenTTL {
		t.Errorf("expected access_token MaxAge %d, got %d", AccessTokenTTL, accessCookie.MaxAge)
	}
	if !accessCookie.HttpOnly {
		t.Errorf("expected access_token HttpOnly true")
	}
	if accessCookie.Secure {
		t.Errorf("expected access_token Secure false in dev mode")
	}
	if accessCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected access_token SameSite Lax")
	}

	refreshCookie, ok := cookieMapDev[RefreshTokenCookieName]
	if !ok {
		t.Fatalf("missing refresh_token cookie")
	}
	if refreshCookie.Value != "mock-refresh-token" {
		t.Errorf("expected refresh_token value 'mock-refresh-token', got %s", refreshCookie.Value)
	}
	if refreshCookie.Path != "/api/v1/auth" {
		t.Errorf("expected refresh_token path '/api/v1/auth', got %s", refreshCookie.Path)
	}
	if refreshCookie.MaxAge != RefreshTokenTTL {
		t.Errorf("expected refresh_token MaxAge %d, got %d", RefreshTokenTTL, refreshCookie.MaxAge)
	}
	if !refreshCookie.HttpOnly {
		t.Errorf("expected refresh_token HttpOnly true")
	}
	if refreshCookie.Secure {
		t.Errorf("expected refresh_token Secure false in dev mode")
	}
	if refreshCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected refresh_token SameSite Lax")
	}

	// 2. Production mode (Secure: true)
	recProd := httptest.NewRecorder()
	SetAuthCookies(recProd, "mock-access-token", "mock-refresh-token", false)

	resProd := recProd.Result()
	cookiesProd := resProd.Cookies()
	if len(cookiesProd) != 2 {
		t.Fatalf("expected 2 cookies in prod mode, got %d", len(cookiesProd))
	}

	for _, c := range cookiesProd {
		if !c.Secure {
			t.Errorf("expected cookie %s to have Secure true in production mode", c.Name)
		}
	}
}

func TestClearAuthCookies(t *testing.T) {
	rec := httptest.NewRecorder()
	ClearAuthCookies(rec, true)

	res := rec.Result()
	cookies := res.Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cleared cookies, got %d", len(cookies))
	}

	cookieMap := make(map[string]*http.Cookie)
	for _, c := range cookies {
		cookieMap[c.Name] = c
	}

	accessCookie := cookieMap[AccessTokenCookieName]
	if accessCookie.MaxAge != -1 {
		t.Errorf("expected access_token MaxAge -1, got %d", accessCookie.MaxAge)
	}
	if accessCookie.Value != "" {
		t.Errorf("expected access_token value empty, got %s", accessCookie.Value)
	}
	if accessCookie.Path != "/" {
		t.Errorf("expected access_token path '/', got %s", accessCookie.Path)
	}

	refreshCookie := cookieMap[RefreshTokenCookieName]
	if refreshCookie.MaxAge != -1 {
		t.Errorf("expected refresh_token MaxAge -1, got %d", refreshCookie.MaxAge)
	}
	if refreshCookie.Value != "" {
		t.Errorf("expected refresh_token value empty, got %s", refreshCookie.Value)
	}
	if refreshCookie.Path != "/api/v1/auth" {
		t.Errorf("expected refresh_token path '/api/v1/auth', got %s", refreshCookie.Path)
	}
}
