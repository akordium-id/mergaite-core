package middleware

import (
	"net/http"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	"github.com/akordium-id/mergiate-core/pkg/response"
)

const HeaderTenantID = "X-Tenant-ID"

// TenantRequired enforces that requests must carry a valid X-Tenant-ID header.
func TenantRequired() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If tenant is already established in context (e.g. via JWT AuthRequired), proceed
			if id, ok := shared.GetTenantID(r.Context()); ok && id != shared.NilID() {
				next.ServeHTTP(w, r)
				return
			}

			rawID := r.Header.Get(HeaderTenantID)
			if rawID == "" {
				response.Err(w, http.StatusBadRequest, "TENANT_REQUIRED", "Missing X-Tenant-ID header")
				return
			}

			tenantID, err := shared.ParseID(rawID)
			if err != nil || tenantID == shared.NilID() {
				response.Err(w, http.StatusBadRequest, "INVALID_TENANT_ID", "X-Tenant-ID must be a valid UUID")
				return
			}

			ctx := shared.WithTenantID(r.Context(), tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TenantOptional parses the X-Tenant-ID header if present, but does not block requests if absent.
func TenantOptional() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawID := r.Header.Get(HeaderTenantID)
			if rawID != "" {
				if tenantID, err := shared.ParseID(rawID); err == nil && tenantID != shared.NilID() {
					ctx := shared.WithTenantID(r.Context(), tenantID)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
