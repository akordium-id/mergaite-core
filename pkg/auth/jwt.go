package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

var (
	ErrInvalidToken = errors.New("invalid or expired authentication token")
)

type TokenClaims struct {
	UserID      shared.ID `json:"uid"`
	TenantID    shared.ID `json:"tid"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"perms"`
	jwt.RegisteredClaims
}

type TokenManager interface {
	GenerateToken(userID, tenantID shared.ID, email, name string, roles, permissions []string, ttl time.Duration) (string, error)
	ValidateToken(tokenStr string) (*TokenClaims, error)
}

type jwtTokenManager struct {
	secretKey []byte
	issuer    string
}

// NewTokenManager constructs a new JWT token manager.
func NewTokenManager(secretKey string, issuer string) TokenManager {
	if secretKey == "" {
		secretKey = "mergiate-default-insecure-secret-key-change-in-prod"
	}
	if issuer == "" {
		issuer = "mergiate-core"
	}
	return &jwtTokenManager{
		secretKey: []byte(secretKey),
		issuer:    issuer,
	}
}

func (m *jwtTokenManager) GenerateToken(userID, tenantID shared.ID, email, name string, roles, permissions []string, ttl time.Duration) (string, error) {
	if ttl == 0 {
		ttl = 24 * time.Hour
	}

	now := time.Now().UTC()
	claims := TokenClaims{
		UserID:      userID,
		TenantID:    tenantID,
		Email:       email,
		Name:        name,
		Roles:       roles,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

func (m *jwtTokenManager) ValidateToken(tokenStr string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &TokenClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
