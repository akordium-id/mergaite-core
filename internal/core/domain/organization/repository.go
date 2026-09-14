package organization

import (
	"context"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

// Repository defines storage operations for Organization aggregate.
type Repository interface {
	Create(ctx context.Context, org *Organization) error
	GetByID(ctx context.Context, tenantID, id shared.ID) (*Organization, error)
	GetByCode(ctx context.Context, tenantID shared.ID, code string) (*Organization, error)
	ListByTenant(ctx context.Context, tenantID shared.ID) ([]Organization, error)
	ListChildren(ctx context.Context, tenantID, parentID shared.ID) ([]Organization, error)
	Update(ctx context.Context, org *Organization) error
}
