package auth

import (
	"net/http"
)

const (
	// AccessTokenCookieName is the cookie name for JWT access token.
	AccessTokenCookieName = "access_token"

	// RefreshTokenCookieName is the cookie name for opaque refresh token.
	RefreshTokenCookieName = "refresh_token"

	// AccessTokenTTL is 900 seconds (15 minutes).
	AccessTokenTTL = 900

	// RefreshTokenTTL is 2592000 seconds (30 days).
	RefreshTokenTTL = 2592000
)

// SetAuthCookies writes HttpOnly cookies for access_token and refresh_token to the response.
// If isDev is true, Secure flag is set to false to accommodate local HTTP development.
// Otherwise, Secure flag is set to true for production HTTPS environments.
func SetAuthCookies(w http.ResponseWriter, accessToken, refreshToken string, isDev bool) {
	secure := !isDev

	if accessToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     AccessTokenCookieName,
			Value:    accessToken,
			Path:     "/",
			MaxAge:   AccessTokenTTL,
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
		})
	}

	if refreshToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     RefreshTokenCookieName,
			Value:    refreshToken,
			Path:     "/api/v1/auth",
			MaxAge:   RefreshTokenTTL,
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

// ClearAuthCookies invalidates both access_token and refresh_token cookies by setting MaxAge to -1.
func ClearAuthCookies(w http.ResponseWriter, isDev bool) {
	secure := !isDev

	http.SetCookie(w, &http.Cookie{
		Name:     AccessTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
