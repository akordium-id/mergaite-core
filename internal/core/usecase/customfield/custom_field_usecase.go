package customfield

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/customfield"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

type CreateDefinitionCommand struct {
	TenantID        shared.ID                   `json:"tenant_id"`
	EntityType      customfield.EntityType      `json:"entity_type"`
	Code            string                      `json:"code"`
	Name            string                      `json:"name"`
	Description     string                      `json:"description,omitempty"`
	DataType        customfield.DataType        `json:"data_type"`
	Options         []string                    `json:"options,omitempty"`
	IsRequired      bool                        `json:"is_required"`
	DefaultValue    any                         `json:"default_value,omitempty"`
	ValidationRules customfield.ValidationRules `json:"validation_rules"`
	SortOrder       int32                       `json:"sort_order"`
}

type UpdateDefinitionCommand struct {
	TenantID        shared.ID                   `json:"tenant_id"`
	ID              shared.ID                   `json:"id"`
	Name            string                      `json:"name"`
	Description     string                      `json:"description,omitempty"`
	Options         []string                    `json:"options,omitempty"`
	IsRequired      bool                        `json:"is_required"`
	DefaultValue    any                         `json:"default_value,omitempty"`
	ValidationRules customfield.ValidationRules `json:"validation_rules"`
	SortOrder       int32                       `json:"sort_order"`
	IsActive        bool                        `json:"is_active"`
}

type SetEntityValuesCommand struct {
	TenantID   shared.ID              `json:"tenant_id"`
	EntityType customfield.EntityType `json:"entity_type"`
	EntityID   shared.ID              `json:"entity_id"`
	Values     map[string]any         `json:"values"`
}

type Usecase interface {
	CreateDefinition(ctx context.Context, cmd CreateDefinitionCommand) (*customfield.Definition, error)
	GetDefinition(ctx context.Context, tenantID, id shared.ID) (*customfield.Definition, error)
	ListDefinitions(ctx context.Context, tenantID shared.ID, entityType customfield.EntityType) ([]customfield.Definition, error)
	UpdateDefinition(ctx context.Context, cmd UpdateDefinitionCommand) (*customfield.Definition, error)
	DeleteDefinition(ctx context.Context, tenantID, id shared.ID) error

	SetEntityValues(ctx context.Context, cmd SetEntityValuesCommand) (*customfield.EntityCustomFields, error)
	GetEntityValues(ctx context.Context, tenantID shared.ID, entityType customfield.EntityType, entityID shared.ID) (*customfield.EntityCustomFields, error)
	FindEntitiesByMatch(ctx context.Context, tenantID shared.ID, entityType customfield.EntityType, match map[string]any) ([]shared.ID, error)
}

type usecase struct {
	repo customfield.Repository
}

// NewUsecase constructs a new Custom Field usecase.
func NewUsecase(repo customfield.Repository) Usecase {
	return &usecase{repo: repo}
}

func (u *usecase) CreateDefinition(ctx context.Context, cmd CreateDefinitionCommand) (*customfield.Definition, error) {
	if cmd.TenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}

	if !cmd.EntityType.IsValid() {
		return nil, fmt.Errorf("%w: invalid entity_type '%s'", shared.ErrInvalidInput, cmd.EntityType)
	}

	if !cmd.DataType.IsValid() {
		return nil, fmt.Errorf("%w: invalid data_type '%s'", shared.ErrInvalidInput, cmd.DataType)
	}

	code := strings.ToLower(strings.TrimSpace(cmd.Code))
	name := strings.TrimSpace(cmd.Name)

	if code == "" || name == "" {
		return nil, fmt.Errorf("%w: code and name are required", shared.ErrInvalidInput)
	}

	// For select or multi_select, options must not be empty
	if (cmd.DataType == customfield.DataTypeSelect || cmd.DataType == customfield.DataTypeMultiSelect) && len(cmd.Options) == 0 {
		return nil, fmt.Errorf("%w: options list cannot be empty for select types", shared.ErrInvalidInput)
	}

	// Check existing definition code
	existing, err := u.repo.GetDefinitionByCode(ctx, cmd.TenantID, cmd.EntityType, code)
	if err != nil && err != shared.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: custom field code '%s' already exists for %s", shared.ErrAlreadyExists, code, cmd.EntityType)
	}

	id, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	def := &customfield.Definition{
		ID:              id,
		TenantID:        cmd.TenantID,
		EntityType:      cmd.EntityType,
		Code:            code,
		Name:            name,
		Description:     cmd.Description,
		DataType:        cmd.DataType,
		Options:         cmd.Options,
		IsRequired:      cmd.IsRequired,
		DefaultValue:    cmd.DefaultValue,
		ValidationRules: cmd.ValidationRules,
		SortOrder:       cmd.SortOrder,
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := u.repo.CreateDefinition(ctx, def); err != nil {
		return nil, err
	}

	return def, nil
}

func (u *usecase) GetDefinition(ctx context.Context, tenantID, id shared.ID) (*customfield.Definition, error) {
	if tenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}
	return u.repo.GetDefinitionByID(ctx, tenantID, id)
}

func (u *usecase) ListDefinitions(ctx context.Context, tenantID shared.ID, entityType customfield.EntityType) ([]customfield.Definition, error) {
	if tenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}
	if !entityType.IsValid() {
		return nil, fmt.Errorf("%w: invalid entity_type '%s'", shared.ErrInvalidInput, entityType)
	}
	return u.repo.ListDefinitions(ctx, tenantID, entityType)
}

func (u *usecase) UpdateDefinition(ctx context.Context, cmd UpdateDefinitionCommand) (*customfield.Definition, error) {
	if cmd.TenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}

	def, err := u.repo.GetDefinitionByID(ctx, cmd.TenantID, cmd.ID)
	if err != nil {
		return nil, err
	}

	if cmd.Name != "" {
		def.Name = strings.TrimSpace(cmd.Name)
	}
	def.Description = cmd.Description
	if len(cmd.Options) > 0 {
		def.Options = cmd.Options
	}
	def.IsRequired = cmd.IsRequired
	def.DefaultValue = cmd.DefaultValue
	def.ValidationRules = cmd.ValidationRules
	def.SortOrder = cmd.SortOrder
	def.IsActive = cmd.IsActive
	def.UpdatedAt = time.Now().UTC()

	if err := u.repo.UpdateDefinition(ctx, def); err != nil {
		return nil, err
	}

	return def, nil
}

func (u *usecase) DeleteDefinition(ctx context.Context, tenantID, id shared.ID) error {
	if tenantID == shared.NilID() {
		return shared.ErrTenantRequired
	}
	return u.repo.DeleteDefinition(ctx, tenantID, id)
}

// ----------------------------------------------------------------------------
// Entity Custom Field Values
// ----------------------------------------------------------------------------

func (u *usecase) SetEntityValues(ctx context.Context, cmd SetEntityValuesCommand) (*customfield.EntityCustomFields, error) {
	if cmd.TenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}
	if !cmd.EntityType.IsValid() {
		return nil, fmt.Errorf("%w: invalid entity_type '%s'", shared.ErrInvalidInput, cmd.EntityType)
	}
	if cmd.EntityID == shared.NilID() {
		return nil, fmt.Errorf("%w: entity_id is required", shared.ErrInvalidInput)
	}

	// 1. Fetch active definitions for this entity type
	defs, err := u.repo.ListActiveDefinitions(ctx, cmd.TenantID, cmd.EntityType)
	if err != nil {
		return nil, err
	}

	// 2. Validate input values against definitions
	sanitizedValues, err := customfield.ValidateValues(defs, cmd.Values)
	if err != nil {
		return nil, err
	}

	id, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	entityFields := &customfield.EntityCustomFields{
		ID:         id,
		TenantID:   cmd.TenantID,
		EntityType: cmd.EntityType,
		EntityID:   cmd.EntityID,
		Values:     sanitizedValues,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := u.repo.UpsertEntityValues(ctx, entityFields); err != nil {
		return nil, err
	}

	return entityFields, nil
}

func (u *usecase) GetEntityValues(ctx context.Context, tenantID shared.ID, entityType customfield.EntityType, entityID shared.ID) (*customfield.EntityCustomFields, error) {
	if tenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}
	if !entityType.IsValid() {
		return nil, fmt.Errorf("%w: invalid entity_type '%s'", shared.ErrInvalidInput, entityType)
	}
	if entityID == shared.NilID() {
		return nil, fmt.Errorf("%w: entity_id is required", shared.ErrInvalidInput)
	}

	record, err := u.repo.GetEntityValues(ctx, tenantID, entityType, entityID)
	if err != nil {
		if err == shared.ErrNotFound {
			// Return empty map if no values set yet
			return &customfield.EntityCustomFields{
				TenantID:   tenantID,
				EntityType: entityType,
				EntityID:   entityID,
				Values:     make(map[string]any),
			}, nil
		}
		return nil, err
	}

	return record, nil
}

func (u *usecase) FindEntitiesByMatch(ctx context.Context, tenantID shared.ID, entityType customfield.EntityType, match map[string]any) ([]shared.ID, error) {
	if tenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}
	if !entityType.IsValid() {
		return nil, fmt.Errorf("%w: invalid entity_type '%s'", shared.ErrInvalidInput, entityType)
	}
	return u.repo.FindEntitiesByMatch(ctx, tenantID, entityType, match)
}
