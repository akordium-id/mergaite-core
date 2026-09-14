package shared

import (
	"context"
	"time"
)

type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusArchived  TenantStatus = "archived"
)

// Tenant represents the top-level isolation and administrative boundary in Mergiate.
type Tenant struct {
	ID        ID             `json:"id"`
	Code      string         `json:"code"`
	Name      string         `json:"name"`
	Status    TenantStatus   `json:"status"`
	Settings  map[string]any `json:"settings"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

func (t Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}

type contextKey string

const (
	tenantIDContextKey contextKey = "mergiate.tenant_id"
)

// WithTenantID stores the Tenant ID in the given context.
func WithTenantID(ctx context.Context, id ID) context.Context {
	return context.WithValue(ctx, tenantIDContextKey, id)
}

// GetTenantID retrieves the Tenant ID from the context if present.
func GetTenantID(ctx context.Context) (ID, bool) {
	val := ctx.Value(tenantIDContextKey)
	if val == nil {
		return NilID(), false
	}
	id, ok := val.(ID)
	if !ok || id == NilID() {
		return NilID(), false
	}
	return id, true
}

// RequireTenantID retrieves the Tenant ID from context, returning ErrTenantRequired if missing.
func RequireTenantID(ctx context.Context) (ID, error) {
	id, ok := GetTenantID(ctx)
	if !ok {
		return NilID(), ErrTenantRequired
	}
	return id, nil
}
