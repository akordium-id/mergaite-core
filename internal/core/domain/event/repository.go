package event

import (
	"context"
	"time"

	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

// OutboxRepository defines persistence operations for the Transactional Outbox.
type OutboxRepository interface {
	Create(ctx context.Context, evt *OutboxEvent) error
	FetchPending(ctx context.Context, maxRetries, limit int32) ([]OutboxEvent, error)
	MarkPublished(ctx context.Context, id shared.ID, publishedAt time.Time) error
	MarkFailed(ctx context.Context, id shared.ID, errMsg string) error
}
