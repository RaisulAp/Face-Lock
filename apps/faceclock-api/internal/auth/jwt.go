package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	Issuer     = "faceclock-api"
	Audience   = "faceclock"
	TypeAccess = "access"
)

// AccessClaims defines the payload inside FaceClock JWT access tokens.
type AccessClaims struct {
	UserID       uuid.UUID  `json:"sub"`
	EmployeeID   *uuid.UUID `json:"eid,omitempty"`
	TokenVersion int        `json:"tv"`
	TokenType    string     `json:"typ"`
	jwt.RegisteredClaims
}

// TokenManager handles JWT issuance and parsing.
type TokenManager struct {
	secret   []byte
	duration time.Duration
}

// NewTokenManager creates a new TokenManager.
func NewTokenManager(secret string, duration time.Duration) (*TokenManager, error) {
	if len(secret) < 32 {
		return nil, errors.New("jwt secret must be at least 32 characters long")
	}
	if duration <= 0 {
		duration = 15 * time.Minute
	}
	return &TokenManager{
		secret:   []byte(secret),
		duration: duration,
	}, nil
}

// GenerateAccessToken signs a new HS256 access token with claims.
func (m *TokenManager) GenerateAccessToken(userID uuid.UUID, employeeID *uuid.UUID, tokenVersion int) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.duration)

	claims := AccessClaims{
		UserID:       userID,
		EmployeeID:   employeeID,
		TokenVersion: tokenVersion,
		TokenType:    TypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Audience:  jwt.ClaimStrings{Audience},
			Subject:   userID.String(),
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("signing access token: %w", err)
	}

	return signed, expiresAt, nil
}

// ParseAccessToken validates the token signature, algorithm, claims, and 30s clock leeway.
func (m *TokenManager) ParseAccessToken(tokenStr string) (*AccessClaims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithIssuer(Issuer),
		jwt.WithAudience(Audience),
		jwt.WithLeeway(30*time.Second),
	)

	var claims AccessClaims
	token, err := parser.ParseWithClaims(tokenStr, &claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("token is invalid")
	}

	if claims.TokenType != TypeAccess {
		return nil, errors.New("invalid token type")
	}

	if claims.UserID == uuid.Nil {
		return nil, errors.New("missing user id in token claims")
	}

	return &claims, nil
}
