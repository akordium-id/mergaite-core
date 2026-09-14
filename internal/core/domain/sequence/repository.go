package sequence

import (
	"context"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

// Repository defines data access operations for Number Sequences.
type Repository interface {
	Create(ctx context.Context, seq *Sequence) error
	GetByID(ctx context.Context, tenantID, id shared.ID) (*Sequence, error)
	GetByCode(ctx context.Context, tenantID shared.ID, code string) (*Sequence, error)
	GetByEntity(ctx context.Context, tenantID shared.ID, entityType, subType string) (*Sequence, error)
	List(ctx context.Context, tenantID shared.ID) ([]Sequence, error)
	Update(ctx context.Context, seq *Sequence) error
	Delete(ctx context.Context, tenantID, id shared.ID) error

	// Atomic operation executing inside a transaction with SELECT ... FOR UPDATE row locking
	AcquireNextNumber(ctx context.Context, tenantID shared.ID, entityType, subType string, extraTokens map[string]string) (string, error)
}
