package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestJWT_GenerationAndValidation(t *testing.T) {
	secret := "a-very-strong-secret-key-that-is-at-least-32-chars-long!"
	mgr, err := NewTokenManager(secret, 15*time.Minute)
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}

	userID := uuid.New()
	employeeID := uuid.New()
	tokenVersion := 1

	tokenStr, exp, err := mgr.GenerateAccessToken(userID, &employeeID, tokenVersion)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}
	if tokenStr == "" {
		t.Errorf("expected non-empty token string")
	}
	if exp.Before(time.Now().UTC()) {
		t.Errorf("expiration time should be in the future")
	}

	claims, err := mgr.ParseAccessToken(tokenStr)
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected user id %s, got %s", userID, claims.UserID)
	}
	if claims.EmployeeID == nil || *claims.EmployeeID != employeeID {
		t.Errorf("expected employee id %s, got %v", employeeID, claims.EmployeeID)
	}
	if claims.TokenVersion != tokenVersion {
		t.Errorf("expected token version %d, got %d", tokenVersion, claims.TokenVersion)
	}
	if claims.TokenType != TypeAccess {
		t.Errorf("expected token type access, got %s", claims.TokenType)
	}
}

func TestJWT_RejectsShortSecret(t *testing.T) {
	_, err := NewTokenManager("short", 15*time.Minute)
	if err == nil {
		t.Errorf("expected error when secret length < 32")
	}
}

func TestJWT_RejectsInvalidTokens(t *testing.T) {
	secret := "a-very-strong-secret-key-that-is-at-least-32-chars-long!"
	mgr, err := NewTokenManager(secret, 15*time.Minute)
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}

	userID := uuid.New()

	// 1. Rejects unsigned / alg:none token
	unsignedToken := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": userID.String(),
		"iss": Issuer,
		"aud": Audience,
		"typ": TypeAccess,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	unsignedStr, _ := unsignedToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	_, err = mgr.ParseAccessToken(unsignedStr)
	if err == nil {
		t.Errorf("expected error parsing alg:none token")
	}

	// 2. Rejects wrong audience
	wrongAudToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID.String(),
		"iss": Issuer,
		"aud": "wrong-audience",
		"typ": TypeAccess,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	wrongAudStr, _ := wrongAudToken.SignedString([]byte(secret))
	_, err = mgr.ParseAccessToken(wrongAudStr)
	if err == nil {
		t.Errorf("expected error parsing token with wrong audience")
	}

	// 3. Rejects expired token beyond leeway
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID.String(),
		"iss": Issuer,
		"aud": Audience,
		"typ": TypeAccess,
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
	})
	expiredStr, _ := expiredToken.SignedString([]byte(secret))
	_, err = mgr.ParseAccessToken(expiredStr)
	if err == nil {
		t.Errorf("expected error parsing expired token")
	}

	// 4. Rejects wrong token type (e.g. refresh token used as bearer)
	wrongTypeToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID.String(),
		"iss": Issuer,
		"aud": Audience,
		"typ": "refresh",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	wrongTypeStr, _ := wrongTypeToken.SignedString([]byte(secret))
	_, err = mgr.ParseAccessToken(wrongTypeStr)
	if err == nil {
		t.Errorf("expected error parsing token with wrong typ claim")
	}

	// 5. Rejects token signed with different secret
	diffSecret := "another-different-secret-key-at-least-32-chars-long!"
	diffSecretToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID.String(),
		"iss": Issuer,
		"aud": Audience,
		"typ": TypeAccess,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	diffSecretStr, _ := diffSecretToken.SignedString([]byte(diffSecret))
	_, err = mgr.ParseAccessToken(diffSecretStr)
	if err == nil {
		t.Errorf("expected error parsing token signed with different secret")
	}
}

func TestRefreshTokens(t *testing.T) {
	tok1, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error generating refresh token: %v", err)
	}
	tok2, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error generating second refresh token: %v", err)
	}
	if tok1 == tok2 {
		t.Errorf("consecutive tokens should not match")
	}

	h1 := HashRefreshToken(tok1)
	h2 := HashRefreshToken(tok1)
	if h1 != h2 {
		t.Errorf("hashing same token should yield identical hash")
	}
	if len(h1) != 64 {
		t.Errorf("sha256 hex string should be 64 characters, got %d", len(h1))
	}
}
