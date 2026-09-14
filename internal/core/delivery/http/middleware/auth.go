package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	"github.com/akordium-id/mergiate-core/pkg/auth"
	"github.com/akordium-id/mergiate-core/pkg/response"
)

// AuthRequired validates the JWT bearer token from the Authorization header,
// injecting user claims and tenant ID into request context.
func AuthRequired(tokenMgr auth.TokenManager) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Err(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing Authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				response.Err(w, http.StatusUnauthorized, "UNAUTHORIZED", "Malformed Authorization header; format must be 'Bearer <token>'")
				return
			}

			tokenStr := strings.TrimSpace(parts[1])
			claims, err := tokenMgr.ValidateToken(tokenStr)
			if err != nil {
				response.Err(w, http.StatusUnauthorized, "INVALID_TOKEN", "Authentication token is invalid or expired")
				return
			}

			// Store claims and tenant ID in context
			ctx := shared.WithAuthClaims(r.Context(), &shared.AuthClaims{
				UserID:      claims.UserID,
				TenantID:    claims.TenantID,
				Email:       claims.Email,
				Name:        claims.Name,
				Roles:       claims.Roles,
				Permissions: claims.Permissions,
			})

			// Set tenant context if tenant ID is present in claims
			if claims.TenantID != shared.NilID() {
				ctx = shared.WithTenantID(ctx, claims.TenantID)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AuthOptional validates the JWT token if present, injecting claims without blocking unauthenticated requests.
func AuthOptional(tokenMgr auth.TokenManager) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
					tokenStr := strings.TrimSpace(parts[1])
					if claims, err := tokenMgr.ValidateToken(tokenStr); err == nil && claims != nil {
						ctx := shared.WithAuthClaims(r.Context(), &shared.AuthClaims{
							UserID:      claims.UserID,
							TenantID:    claims.TenantID,
							Email:       claims.Email,
							Name:        claims.Name,
							Roles:       claims.Roles,
							Permissions: claims.Permissions,
						})
						if claims.TenantID != shared.NilID() {
							ctx = shared.WithTenantID(ctx, claims.TenantID)
						}
						r = r.WithContext(ctx)
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission verifies that the authenticated user possesses the specified permission code.
func RequirePermission(permissionCode string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !shared.HasPermission(r.Context(), permissionCode) {
				response.Err(w, http.StatusForbidden, "FORBIDDEN", fmt.Sprintf("Permission '%s' is required to perform this action", permissionCode))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission verifies that the user possesses at least one of the specified permissions.
func RequireAnyPermission(permissions ...string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, perm := range permissions {
				if shared.HasPermission(r.Context(), perm) {
					next.ServeHTTP(w, r)
					return
				}
			}
			response.Err(w, http.StatusForbidden, "FORBIDDEN", "You do not have sufficient permissions to perform this action")
		})
	}
}
