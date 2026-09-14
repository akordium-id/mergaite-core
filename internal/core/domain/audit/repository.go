package audit

import (
	"context"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

// Repository defines data access operations for AuditLog.
type Repository interface {
	Create(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, tenantID shared.ID, filter Filter) ([]AuditLog, int64, error)
}
