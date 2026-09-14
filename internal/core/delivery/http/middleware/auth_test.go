package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/delivery/http/middleware"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	"github.com/akordium-id/mergiate-core/pkg/auth"
)

func TestAuthRequiredMiddleware(t *testing.T) {
	tokenMgr := auth.NewTokenManager("test-secret-key-1234567890", "test-issuer")
	userID := shared.MustNewID()
	tenantID := shared.MustNewID()

	validToken, err := tokenMgr.GenerateToken(userID, tenantID, "test@akordium.com", "Test User", []string{"admin"}, []string{"document:create", "document:read"}, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		checkContext   bool
	}{
		{
			name:           "Missing Authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Malformed Authorization header (no Bearer)",
			authHeader:     "Basic 12345",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid JWT Token",
			authHeader:     "Bearer invalid.token.value",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Valid Bearer Token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			checkContext:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := middleware.AuthRequired(tokenMgr)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.checkContext {
					claims, ok := shared.GetAuthClaims(r.Context())
					if !ok || claims == nil {
						t.Error("expected auth claims in context")
					} else if claims.UserID != userID {
						t.Errorf("expected userID %v, got %v", userID, claims.UserID)
					}

					tID, ok := shared.GetTenantID(r.Context())
					if !ok || tID != tenantID {
						t.Errorf("expected tenantID %v, got %v", tenantID, tID)
					}
				}
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestRequirePermissionMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		userPerms     []string
		requiredPerm   string
		expectedStatus int
	}{
		{
			name:           "User has required permission",
			userPerms:      []string{"document:create", "document:read"},
			requiredPerm:   "document:create",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "User does not have permission",
			userPerms:      []string{"document:read"},
			requiredPerm:   "document:approve",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "User has wildcard permission",
			userPerms:      []string{"*"},
			requiredPerm:   "billing:charge",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := middleware.RequirePermission(tt.requiredPerm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			ctx := shared.WithAuthClaims(httptest.NewRequest(http.MethodGet, "/test", nil).Context(), &shared.AuthClaims{
				UserID:      shared.MustNewID(),
				TenantID:    shared.MustNewID(),
				Permissions: tt.userPerms,
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}
