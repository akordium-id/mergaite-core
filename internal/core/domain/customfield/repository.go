package customfield

import (
	"context"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

// Repository defines data access operations for Custom Field definitions and entity values.
type Repository interface {
	// Definitions
	CreateDefinition(ctx context.Context, def *Definition) error
	GetDefinitionByID(ctx context.Context, tenantID, id shared.ID) (*Definition, error)
	GetDefinitionByCode(ctx context.Context, tenantID shared.ID, entityType EntityType, code string) (*Definition, error)
	ListDefinitions(ctx context.Context, tenantID shared.ID, entityType EntityType) ([]Definition, error)
	ListActiveDefinitions(ctx context.Context, tenantID shared.ID, entityType EntityType) ([]Definition, error)
	UpdateDefinition(ctx context.Context, def *Definition) error
	DeleteDefinition(ctx context.Context, tenantID, id shared.ID) error

	// Entity Values
	UpsertEntityValues(ctx context.Context, vals *EntityCustomFields) error
	GetEntityValues(ctx context.Context, tenantID shared.ID, entityType EntityType, entityID shared.ID) (*EntityCustomFields, error)
	DeleteEntityValues(ctx context.Context, tenantID shared.ID, entityType EntityType, entityID shared.ID) error
	FindEntitiesByMatch(ctx context.Context, tenantID shared.ID, entityType EntityType, match map[string]any) ([]shared.ID, error)
}
