package event

import (
	"time"

	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

// Event is the interface that all domain events in Mergiate must satisfy.
type Event interface {
	EventID() shared.ID
	TenantID() shared.ID
	EventType() string
	AggregateType() string
	AggregateID() shared.ID
	Payload() map[string]any
	OccurredAt() time.Time
}

// BaseEvent is a convenience struct implementing Event.
type BaseEvent struct {
	ID            shared.ID      `json:"id"`
	Tenant        shared.ID      `json:"tenant_id"`
	Type          string         `json:"event_type"`
	AggType       string         `json:"aggregate_type"`
	AggID         shared.ID      `json:"aggregate_id"`
	Data          map[string]any `json:"payload"`
	Timestamp     time.Time      `json:"occurred_at"`
}

func (e BaseEvent) EventID() shared.ID        { return e.ID }
func (e BaseEvent) TenantID() shared.ID       { return e.Tenant }
func (e BaseEvent) EventType() string         { return e.Type }
func (e BaseEvent) AggregateType() string     { return e.AggType }
func (e BaseEvent) AggregateID() shared.ID    { return e.AggID }
func (e BaseEvent) Payload() map[string]any   { return e.Data }
func (e BaseEvent) OccurredAt() time.Time     { return e.Timestamp }

// NewBaseEvent creates a new BaseEvent with a generated UUIDv7.
func NewBaseEvent(tenantID shared.ID, eventType, aggregateType string, aggregateID shared.ID, payload map[string]any) BaseEvent {
	id, _ := shared.NewID()
	return BaseEvent{
		ID:        id,
		Tenant:    tenantID,
		Type:      eventType,
		AggType:   aggregateType,
		AggID:     aggregateID,
		Data:      payload,
		Timestamp: time.Now().UTC(),
	}
}

type OutboxStatus string

const (
	OutboxStatusPending    OutboxStatus = "pending"
	OutboxStatusProcessing OutboxStatus = "processing"
	OutboxStatusPublished  OutboxStatus = "published"
	OutboxStatusFailed     OutboxStatus = "failed"
)

// OutboxEvent represents a transactional outbox message saved in PostgreSQL.
type OutboxEvent struct {
	ID            shared.ID      `json:"id"`
	TenantID      shared.ID      `json:"tenant_id"`
	EventType     string         `json:"event_type"`
	AggregateType string         `json:"aggregate_type"`
	AggregateID   shared.ID      `json:"aggregate_id"`
	Payload       map[string]any `json:"payload"`
	Status        OutboxStatus   `json:"status"`
	RetryCount    int32          `json:"retry_count"`
	ErrorMessage  string         `json:"error_message,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	PublishedAt   *time.Time     `json:"published_at,omitempty"`
}

// ToEvent converts an OutboxEvent into the standard Event interface.
func (o OutboxEvent) ToEvent() Event {
	return BaseEvent{
		ID:        o.ID,
		Tenant:    o.TenantID,
		Type:      o.EventType,
		AggType:   o.AggregateType,
		AggID:     o.AggregateID,
		Data:      o.Payload,
		Timestamp: o.CreatedAt,
	}
}
