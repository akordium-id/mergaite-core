package auth_test

import (
	"testing"
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	"github.com/akordium-id/mergiate-core/pkg/auth"
)

func TestPasswordHashing(t *testing.T) {
	raw := "SecureP@ssw0rd!"

	hash, err := auth.HashPassword(raw)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if hash == raw {
		t.Errorf("hash must not equal raw password")
	}

	if !auth.CheckPasswordHash(raw, hash) {
		t.Errorf("expected valid password check, got false")
	}

	if auth.CheckPasswordHash("WrongPassword!", hash) {
		t.Errorf("expected wrong password to fail check, got true")
	}
}

func TestJWTTokenManager(t *testing.T) {
	mgr := auth.NewTokenManager("my-super-secret-key-1234567890", "test-issuer")

	userID, _ := shared.NewID()
	tenantID, _ := shared.NewID()
	email := "user@example.com"
	name := "Alice"
	roles := []string{"admin", "finance"}
	perms := []string{"document:create", "document:approve"}

	tokenStr, err := mgr.GenerateToken(userID, tenantID, email, name, roles, perms, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := mgr.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected UserID %v, got %v", userID, claims.UserID)
	}
	if claims.TenantID != tenantID {
		t.Errorf("expected TenantID %v, got %v", tenantID, claims.TenantID)
	}
	if claims.Email != email {
		t.Errorf("expected Email %s, got %s", email, claims.Email)
	}
	if len(claims.Permissions) != 2 || claims.Permissions[1] != "document:approve" {
		t.Errorf("expected permissions to match, got %v", claims.Permissions)
	}

	// Test expired token
	expiredToken, _ := mgr.GenerateToken(userID, tenantID, email, name, roles, perms, -1*time.Hour)
	_, err = mgr.ValidateToken(expiredToken)
	if err == nil {
		t.Errorf("expected error for expired token, got nil")
	}
}
