package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/mergiate-core/internal/core/domain/customfield"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	"github.com/akordium-id/mergiate-core/internal/core/repository/postgres/sqlc"
)

type customFieldRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewCustomFieldRepository creates a PostgreSQL implementation of customfield.Repository.
func NewCustomFieldRepository(pool *pgxpool.Pool) customfield.Repository {
	return &customFieldRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// ----------------------------------------------------------------------------
// Definitions
// ----------------------------------------------------------------------------

func (r *customFieldRepository) CreateDefinition(ctx context.Context, def *customfield.Definition) error {
	optionsBytes, _ := json.Marshal(def.Options)
	defaultBytes, _ := json.Marshal(def.DefaultValue)
	rulesBytes, _ := json.Marshal(def.ValidationRules)

	var desc *string
	if def.Description != "" {
		desc = &def.Description
	}

	params := sqlc.CreateCustomFieldDefinitionParams{
		ID:              shared.ToPgUUID(def.ID),
		TenantID:        shared.ToPgUUID(def.TenantID),
		EntityType:      string(def.EntityType),
		Code:            def.Code,
		Name:            def.Name,
		Description:     desc,
		DataType:        string(def.DataType),
		Options:         optionsBytes,
		IsRequired:      def.IsRequired,
		DefaultValue:    defaultBytes,
		ValidationRules: rulesBytes,
		SortOrder:       def.SortOrder,
		IsActive:        def.IsActive,
		CreatedAt:       pgtype.Timestamptz{Time: def.CreatedAt, Valid: true},
		UpdatedAt:       pgtype.Timestamptz{Time: def.UpdatedAt, Valid: true},
	}

	row, err := r.queries.CreateCustomFieldDefinition(ctx, params)
	if err != nil {
		return err
	}

	def.ID = shared.FromPgUUID(row.ID)
	def.CreatedAt = row.CreatedAt.Time
	def.UpdatedAt = row.UpdatedAt.Time
	return nil
}

func (r *customFieldRepository) GetDefinitionByID(ctx context.Context, tenantID, id shared.ID) (*customfield.Definition, error) {
	row, err := r.queries.GetCustomFieldDefinitionByID(ctx, sqlc.GetCustomFieldDefinitionByIDParams{
		TenantID: shared.ToPgUUID(tenantID),
		ID:       shared.ToPgUUID(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}

	return toDomainDefinition(row)
}

func (r *customFieldRepository) GetDefinitionByCode(ctx context.Context, tenantID shared.ID, entityType customfield.EntityType, code string) (*customfield.Definition, error) {
	row, err := r.queries.GetCustomFieldDefinitionByCode(ctx, sqlc.GetCustomFieldDefinitionByCodeParams{
		TenantID:   shared.ToPgUUID(tenantID),
		EntityType: string(entityType),
		Code:       code,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}

	return toDomainDefinition(row)
}

func (r *customFieldRepository) ListDefinitions(ctx context.Context, tenantID shared.ID, entityType customfield.EntityType) ([]customfield.Definition, error) {
	rows, err := r.queries.ListCustomFieldDefinitions(ctx, sqlc.ListCustomFieldDefinitionsParams{
		TenantID:   shared.ToPgUUID(tenantID),
		EntityType: string(entityType),
	})
	if err != nil {
		return nil, err
	}

	defs := make([]customfield.Definition, len(rows))
	for i, row := range rows {
		d, err := toDomainDefinition(row)
		if err != nil {
			return nil, err
		}
		defs[i] = *d
	}
	return defs, nil
}

func (r *customFieldRepository) ListActiveDefinitions(ctx context.Context, tenantID shared.ID, entityType customfield.EntityType) ([]customfield.Definition, error) {
	rows, err := r.queries.ListActiveCustomFieldDefinitions(ctx, sqlc.ListActiveCustomFieldDefinitionsParams{
		TenantID:   shared.ToPgUUID(tenantID),
		EntityType: string(entityType),
	})
	if err != nil {
		return nil, err
	}

	defs := make([]customfield.Definition, len(rows))
	for i, row := range rows {
		d, err := toDomainDefinition(row)
		if err != nil {
			return nil, err
		}
		defs[i] = *d
	}
	return defs, nil
}

func (r *customFieldRepository) UpdateDefinition(ctx context.Context, def *customfield.Definition) error {
	optionsBytes, _ := json.Marshal(def.Options)
	defaultBytes, _ := json.Marshal(def.DefaultValue)
	rulesBytes, _ := json.Marshal(def.ValidationRules)

	var desc *string
	if def.Description != "" {
		desc = &def.Description
	}

	params := sqlc.UpdateCustomFieldDefinitionParams{
		TenantID:        shared.ToPgUUID(def.TenantID),
		ID:              shared.ToPgUUID(def.ID),
		Name:            def.Name,
		Description:     desc,
		Options:         optionsBytes,
		IsRequired:      def.IsRequired,
		DefaultValue:    defaultBytes,
		ValidationRules: rulesBytes,
		SortOrder:       def.SortOrder,
		IsActive:        def.IsActive,
		UpdatedAt:       pgtype.Timestamptz{Time: def.UpdatedAt, Valid: true},
	}

	row, err := r.queries.UpdateCustomFieldDefinition(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return shared.ErrNotFound
		}
		return err
	}

	def.UpdatedAt = row.UpdatedAt.Time
	return nil
}

func (r *customFieldRepository) DeleteDefinition(ctx context.Context, tenantID, id shared.ID) error {
	return r.queries.DeleteCustomFieldDefinition(ctx, sqlc.DeleteCustomFieldDefinitionParams{
		TenantID: shared.ToPgUUID(tenantID),
		ID:       shared.ToPgUUID(id),
	})
}

// ----------------------------------------------------------------------------
// Entity Custom Field Values
// ----------------------------------------------------------------------------

func (r *customFieldRepository) UpsertEntityValues(ctx context.Context, vals *customfield.EntityCustomFields) error {
	valuesBytes, err := json.Marshal(vals.Values)
	if err != nil {
		return err
	}

	params := sqlc.UpsertEntityCustomFieldsParams{
		ID:         shared.ToPgUUID(vals.ID),
		TenantID:   shared.ToPgUUID(vals.TenantID),
		EntityType: string(vals.EntityType),
		EntityID:   shared.ToPgUUID(vals.EntityID),
		Values:     valuesBytes,
		CreatedAt:  pgtype.Timestamptz{Time: vals.CreatedAt, Valid: true},
		UpdatedAt:  pgtype.Timestamptz{Time: vals.UpdatedAt, Valid: true},
	}

	row, err := r.queries.UpsertEntityCustomFields(ctx, params)
	if err != nil {
		return err
	}

	vals.ID = shared.FromPgUUID(row.ID)
	vals.CreatedAt = row.CreatedAt.Time
	vals.UpdatedAt = row.UpdatedAt.Time
	return nil
}

func (r *customFieldRepository) GetEntityValues(ctx context.Context, tenantID shared.ID, entityType customfield.EntityType, entityID shared.ID) (*customfield.EntityCustomFields, error) {
	row, err := r.queries.GetEntityCustomFields(ctx, sqlc.GetEntityCustomFieldsParams{
		TenantID:   shared.ToPgUUID(tenantID),
		EntityType: string(entityType),
		EntityID:   shared.ToPgUUID(entityID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}

	var values map[string]any
	if len(row.Values) > 0 {
		_ = json.Unmarshal(row.Values, &values)
	}
	if values == nil {
		values = make(map[string]any)
	}

	return &customfield.EntityCustomFields{
		ID:         shared.FromPgUUID(row.ID),
		TenantID:   shared.FromPgUUID(row.TenantID),
		EntityType: customfield.EntityType(row.EntityType),
		EntityID:   shared.FromPgUUID(row.EntityID),
		Values:     values,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}, nil
}

func (r *customFieldRepository) DeleteEntityValues(ctx context.Context, tenantID shared.ID, entityType customfield.EntityType, entityID shared.ID) error {
	return r.queries.DeleteEntityCustomFields(ctx, sqlc.DeleteEntityCustomFieldsParams{
		TenantID:   shared.ToPgUUID(tenantID),
		EntityType: string(entityType),
		EntityID:   shared.ToPgUUID(entityID),
	})
}

func (r *customFieldRepository) FindEntitiesByMatch(ctx context.Context, tenantID shared.ID, entityType customfield.EntityType, match map[string]any) ([]shared.ID, error) {
	matchBytes, err := json.Marshal(match)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.FindEntitiesByCustomFieldMatch(ctx, sqlc.FindEntitiesByCustomFieldMatchParams{
		TenantID:    shared.ToPgUUID(tenantID),
		EntityType:  string(entityType),
		MatchValues: matchBytes,
	})
	if err != nil {
		return nil, err
	}

	ids := make([]shared.ID, len(rows))
	for i, row := range rows {
		ids[i] = shared.FromPgUUID(row.EntityID)
	}
	return ids, nil
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

func toDomainDefinition(row sqlc.CustomFieldDefinition) (*customfield.Definition, error) {
	var options []string
	if len(row.Options) > 0 {
		_ = json.Unmarshal(row.Options, &options)
	}

	var defaultValue any
	if len(row.DefaultValue) > 0 {
		_ = json.Unmarshal(row.DefaultValue, &defaultValue)
	}

	var rules customfield.ValidationRules
	if len(row.ValidationRules) > 0 {
		_ = json.Unmarshal(row.ValidationRules, &rules)
	}

	return &customfield.Definition{
		ID:              shared.FromPgUUID(row.ID),
		TenantID:        shared.FromPgUUID(row.TenantID),
		EntityType:      customfield.EntityType(row.EntityType),
		Code:            row.Code,
		Name:            row.Name,
		Description:     strFromPtr(row.Description),
		DataType:        customfield.DataType(row.DataType),
		Options:         options,
		IsRequired:      row.IsRequired,
		DefaultValue:    defaultValue,
		ValidationRules: rules,
		SortOrder:       row.SortOrder,
		IsActive:        row.IsActive,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}, nil
}
