package audit

import (
	"time"

	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

type Action string

const (
	ActionCreate     Action = "create"
	ActionUpdate     Action = "update"
	ActionDelete     Action = "delete"
	ActionTransition Action = "transition"
)

type ActorType string

const (
	ActorTypeUser   ActorType = "user"
	ActorTypeSystem ActorType = "system"
	ActorTypeAPIKey ActorType = "api_key"
)

// AuditLog represents an immutable record of a state-changing operation or event.
type AuditLog struct {
	ID         shared.ID      `json:"id"`
	TenantID   shared.ID      `json:"tenant_id"`
	ActorID    *shared.ID     `json:"actor_id,omitempty"`
	ActorType  ActorType      `json:"actor_type"`
	Action     Action         `json:"action"`
	EntityType string         `json:"entity_type"`
	EntityID   shared.ID      `json:"entity_id"`
	Changes    map[string]any `json:"changes,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// Filter defines query parameters for fetching audit records.
type Filter struct {
	EntityType *string
	EntityID   *shared.ID
	ActorID    *shared.ID
	Action     *Action
	Limit      int32
	Offset     int32
}
