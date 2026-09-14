package repository

import (
	"context"

	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

// TenantRepository defines storage operations for Tenant entities.
type TenantRepository interface {
	Create(ctx context.Context, tenant *shared.Tenant) error
	GetByID(ctx context.Context, id shared.ID) (*shared.Tenant, error)
	GetByCode(ctx context.Context, code string) (*shared.Tenant, error)
	List(ctx context.Context, limit, offset int32) ([]shared.Tenant, int64, error)
	Update(ctx context.Context, tenant *shared.Tenant) error
}
