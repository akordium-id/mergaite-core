package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/akordium-id/mergiate-core/internal/core/domain/identity"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	"github.com/akordium-id/mergiate-core/pkg/auth"
	"github.com/akordium-id/mergiate-core/pkg/response"
)

// AuthRequired validates either an M2M API Key (via X-API-Key or Bearer mrg_...) or
// a user JWT Bearer token, injecting the claims and tenant context into the request.
func AuthRequired(tokenMgr auth.TokenManager, keyValidator ...identity.APIKeyValidator) func(next http.Handler) http.Handler {
	var kv identity.APIKeyValidator
	if len(keyValidator) > 0 {
		kv = keyValidator[0]
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Check X-API-Key header
			if apiKey := r.Header.Get("X-API-Key"); apiKey != "" && kv != nil {
				claims, err := kv.ValidateAPIKey(r.Context(), apiKey, r.RemoteAddr)
				if err != nil {
					response.Err(w, http.StatusUnauthorized, "INVALID_API_KEY", err.Error())
					return
				}
				ctx := shared.WithAuthClaims(r.Context(), claims)
				if claims.TenantID != shared.NilID() {
					ctx = shared.WithTenantID(ctx, claims.TenantID)
				}
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// 2. Check Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Err(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing Authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 {
				response.Err(w, http.StatusUnauthorized, "UNAUTHORIZED", "Malformed Authorization header")
				return
			}

			scheme := parts[0]
			tokenStr := strings.TrimSpace(parts[1])

			// 2a. Check if token is an M2M API key (Bearer mrg_live_... or ApiKey mrg_live_...)
			if (strings.EqualFold(scheme, "Bearer") || strings.EqualFold(scheme, "ApiKey")) &&
				strings.HasPrefix(tokenStr, identity.APIKeyPrefix) && kv != nil {
				claims, err := kv.ValidateAPIKey(r.Context(), tokenStr, r.RemoteAddr)
				if err != nil {
					response.Err(w, http.StatusUnauthorized, "INVALID_API_KEY", err.Error())
					return
				}
				ctx := shared.WithAuthClaims(r.Context(), claims)
				if claims.TenantID != shared.NilID() {
					ctx = shared.WithTenantID(ctx, claims.TenantID)
				}
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// 2b. Validate standard user JWT bearer token
			if strings.EqualFold(scheme, "Bearer") {
				claims, err := tokenMgr.ValidateToken(tokenStr)
				if err != nil {
					response.Err(w, http.StatusUnauthorized, "INVALID_TOKEN", "Authentication token is invalid or expired")
					return
				}

				ctx := shared.WithAuthClaims(r.Context(), &shared.AuthClaims{
					UserID:      claims.UserID,
					TenantID:    claims.TenantID,
					Email:       claims.Email,
					Name:        claims.Name,
					Roles:       claims.Roles,
					Permissions: claims.Permissions,
					ActorType:   shared.ActorTypeUser,
				})

				if claims.TenantID != shared.NilID() {
					ctx = shared.WithTenantID(ctx, claims.TenantID)
				}

				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			response.Err(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unsupported authentication scheme")
		})
	}
}

// AuthOptional validates the JWT token or API Key if present, injecting claims without blocking unauthenticated requests.
func AuthOptional(tokenMgr auth.TokenManager, keyValidator ...identity.APIKeyValidator) func(next http.Handler) http.Handler {
	var kv identity.APIKeyValidator
	if len(keyValidator) > 0 {
		kv = keyValidator[0]
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check X-API-Key
			if apiKey := r.Header.Get("X-API-Key"); apiKey != "" && kv != nil {
				if claims, err := kv.ValidateAPIKey(r.Context(), apiKey, r.RemoteAddr); err == nil && claims != nil {
					ctx := shared.WithAuthClaims(r.Context(), claims)
					if claims.TenantID != shared.NilID() {
						ctx = shared.WithTenantID(ctx, claims.TenantID)
					}
					r = r.WithContext(ctx)
				}
			} else if authHeader := r.Header.Get("Authorization"); authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 {
					scheme := parts[0]
					tokenStr := strings.TrimSpace(parts[1])

					if (strings.EqualFold(scheme, "Bearer") || strings.EqualFold(scheme, "ApiKey")) &&
						strings.HasPrefix(tokenStr, identity.APIKeyPrefix) && kv != nil {
						if claims, err := kv.ValidateAPIKey(r.Context(), tokenStr, r.RemoteAddr); err == nil && claims != nil {
							ctx := shared.WithAuthClaims(r.Context(), claims)
							if claims.TenantID != shared.NilID() {
								ctx = shared.WithTenantID(ctx, claims.TenantID)
							}
							r = r.WithContext(ctx)
						}
					} else if strings.EqualFold(scheme, "Bearer") {
						if claims, err := tokenMgr.ValidateToken(tokenStr); err == nil && claims != nil {
							ctx := shared.WithAuthClaims(r.Context(), &shared.AuthClaims{
								UserID:      claims.UserID,
								TenantID:    claims.TenantID,
								Email:       claims.Email,
								Name:        claims.Name,
								Roles:       claims.Roles,
								Permissions: claims.Permissions,
								ActorType:   shared.ActorTypeUser,
							})
							if claims.TenantID != shared.NilID() {
								ctx = shared.WithTenantID(ctx, claims.TenantID)
							}
							r = r.WithContext(ctx)
						}
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
