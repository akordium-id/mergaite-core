package party

import (
	"context"

	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

type PartyFilter struct {
	Type     *Type
	RoleType *RoleType
	Limit    int32
	Offset   int32
}

// Repository defines storage operations for Party aggregate.
type Repository interface {
	Create(ctx context.Context, party *Party) error
	GetByID(ctx context.Context, tenantID, id shared.ID) (*Party, error)
	List(ctx context.Context, tenantID shared.ID, filter PartyFilter) ([]Party, int64, error)
	Update(ctx context.Context, party *Party) error

	AddRole(ctx context.Context, role *PartyRole) error
	ListRoles(ctx context.Context, tenantID, partyID shared.ID) ([]PartyRole, error)
	RemoveRole(ctx context.Context, tenantID, roleID shared.ID) error
}
