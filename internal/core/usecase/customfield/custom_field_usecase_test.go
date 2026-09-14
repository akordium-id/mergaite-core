package customfield_test

import (
	"context"
	"errors"
	"testing"

	"github.com/akordium-id/mergiate-core/internal/core/domain/customfield"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	usecase "github.com/akordium-id/mergiate-core/internal/core/usecase/customfield"
)

type mockCustomFieldRepo struct {
	definitions  map[string]*customfield.Definition
	entityValues map[string]*customfield.EntityCustomFields
}

func newMockRepo() *mockCustomFieldRepo {
	return &mockCustomFieldRepo{
		definitions:  make(map[string]*customfield.Definition),
		entityValues: make(map[string]*customfield.EntityCustomFields),
	}
}

func defKey(tenantID shared.ID, entityType customfield.EntityType, code string) string {
	return tenantID.String() + ":" + string(entityType) + ":" + code
}

func entityKey(tenantID shared.ID, entityType customfield.EntityType, entityID shared.ID) string {
	return tenantID.String() + ":" + string(entityType) + ":" + entityID.String()
}

func (m *mockCustomFieldRepo) CreateDefinition(ctx context.Context, def *customfield.Definition) error {
	m.definitions[defKey(def.TenantID, def.EntityType, def.Code)] = def
	return nil
}

func (m *mockCustomFieldRepo) GetDefinitionByID(ctx context.Context, tenantID, id shared.ID) (*customfield.Definition, error) {
	for _, def := range m.definitions {
		if def.TenantID == tenantID && def.ID == id {
			return def, nil
		}
	}
	return nil, shared.ErrNotFound
}

func (m *mockCustomFieldRepo) GetDefinitionByCode(ctx context.Context, tenantID shared.ID, entityType customfield.EntityType, code string) (*customfield.Definition, error) {
	def, ok := m.definitions[defKey(tenantID, entityType, code)]
	if !ok {
		return nil, shared.ErrNotFound
	}
	return def, nil
}

func (m *mockCustomFieldRepo) ListDefinitions(ctx context.Context, tenantID shared.ID, entityType customfield.EntityType) ([]customfield.Definition, error) {
	var list []customfield.Definition
	for _, def := range m.definitions {
		if def.TenantID == tenantID && def.EntityType == entityType {
			list = append(list, *def)
		}
	}
	return list, nil
}

func (m *mockCustomFieldRepo) ListActiveDefinitions(ctx context.Context, tenantID shared.ID, entityType customfield.EntityType) ([]customfield.Definition, error) {
	var list []customfield.Definition
	for _, def := range m.definitions {
		if def.TenantID == tenantID && def.EntityType == entityType && def.IsActive {
			list = append(list, *def)
		}
	}
	return list, nil
}

func (m *mockCustomFieldRepo) UpdateDefinition(ctx context.Context, def *customfield.Definition) error {
	m.definitions[defKey(def.TenantID, def.EntityType, def.Code)] = def
	return nil
}

func (m *mockCustomFieldRepo) DeleteDefinition(ctx context.Context, tenantID, id shared.ID) error {
	for k, def := range m.definitions {
		if def.TenantID == tenantID && def.ID == id {
			delete(m.definitions, k)
			return nil
		}
	}
	return shared.ErrNotFound
}

func (m *mockCustomFieldRepo) UpsertEntityValues(ctx context.Context, vals *customfield.EntityCustomFields) error {
	m.entityValues[entityKey(vals.TenantID, vals.EntityType, vals.EntityID)] = vals
	return nil
}

func (m *mockCustomFieldRepo) GetEntityValues(ctx context.Context, tenantID shared.ID, entityType customfield.EntityType, entityID shared.ID) (*customfield.EntityCustomFields, error) {
	vals, ok := m.entityValues[entityKey(tenantID, entityType, entityID)]
	if !ok {
		return nil, shared.ErrNotFound
	}
	return vals, nil
}

func (m *mockCustomFieldRepo) DeleteEntityValues(ctx context.Context, tenantID shared.ID, entityType customfield.EntityType, entityID shared.ID) error {
	delete(m.entityValues, entityKey(tenantID, entityType, entityID))
	return nil
}

func (m *mockCustomFieldRepo) FindEntitiesByMatch(ctx context.Context, tenantID shared.ID, entityType customfield.EntityType, match map[string]any) ([]shared.ID, error) {
	return nil, nil
}

func TestCustomFieldUsecase_CreateDefinition(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	uc := usecase.NewUsecase(repo)
	tenantID := shared.MustNewID()

	// 1. Valid Creation
	def, err := uc.CreateDefinition(ctx, usecase.CreateDefinitionCommand{
		TenantID:   tenantID,
		EntityType: customfield.EntityProduct,
		Code:       "warranty_months",
		Name:       "Warranty in Months",
		DataType:   customfield.DataTypeNumber,
		IsRequired: true,
	})
	if err != nil {
		t.Fatalf("unexpected error creating definition: %v", err)
	}
	if def.Code != "warranty_months" {
		t.Errorf("expected code 'warranty_months', got '%s'", def.Code)
	}

	// 2. Duplicate code error
	_, err = uc.CreateDefinition(ctx, usecase.CreateDefinitionCommand{
		TenantID:   tenantID,
		EntityType: customfield.EntityProduct,
		Code:       "warranty_months",
		Name:       "Duplicate Field",
		DataType:   customfield.DataTypeNumber,
	})
	if !errors.Is(err, shared.ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists, got %v", err)
	}

	// 3. Select without options error
	_, err = uc.CreateDefinition(ctx, usecase.CreateDefinitionCommand{
		TenantID:   tenantID,
		EntityType: customfield.EntityProduct,
		Code:       "color",
		Name:       "Color",
		DataType:   customfield.DataTypeSelect,
		Options:    nil,
	})
	if !errors.Is(err, shared.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for empty select options, got %v", err)
	}
}

func TestCustomFieldUsecase_SetEntityValuesValidation(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	uc := usecase.NewUsecase(repo)
	tenantID := shared.MustNewID()
	productID := shared.MustNewID()

	minVal := float64(0)
	maxVal := float64(60)

	// Create definitions for product
	_, _ = uc.CreateDefinition(ctx, usecase.CreateDefinitionCommand{
		TenantID:        tenantID,
		EntityType:      customfield.EntityProduct,
		Code:            "warranty_months",
		Name:            "Warranty (Months)",
		DataType:        customfield.DataTypeNumber,
		IsRequired:      true,
		ValidationRules: customfield.ValidationRules{Min: &minVal, Max: &maxVal},
	})

	_, _ = uc.CreateDefinition(ctx, usecase.CreateDefinitionCommand{
		TenantID:     tenantID,
		EntityType:   customfield.EntityProduct,
		Code:         "origin_country",
		Name:         "Origin Country",
		DataType:     customfield.DataTypeSelect,
		Options:      []string{"ID", "JP", "US", "DE"},
		IsRequired:   false,
		DefaultValue: "ID",
	})

	_, _ = uc.CreateDefinition(ctx, usecase.CreateDefinitionCommand{
		TenantID:   tenantID,
		EntityType: customfield.EntityProduct,
		Code:       "is_organic",
		Name:       "Organic Certified",
		DataType:   customfield.DataTypeBoolean,
		IsRequired: false,
	})

	_, _ = uc.CreateDefinition(ctx, usecase.CreateDefinitionCommand{
		TenantID:   tenantID,
		EntityType: customfield.EntityProduct,
		Code:       "expiry_date",
		Name:       "Expiry Date",
		DataType:   customfield.DataTypeDate,
		IsRequired: false,
	})

	// Case 1: Missing required field warranty_months
	_, err := uc.SetEntityValues(ctx, usecase.SetEntityValuesCommand{
		TenantID:   tenantID,
		EntityType: customfield.EntityProduct,
		EntityID:   productID,
		Values:     map[string]any{"is_organic": true},
	})
	if !errors.Is(err, shared.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for missing required field, got %v", err)
	}

	// Case 2: Number out of bounds (exceeds max 60)
	_, err = uc.SetEntityValues(ctx, usecase.SetEntityValuesCommand{
		TenantID:   tenantID,
		EntityType: customfield.EntityProduct,
		EntityID:   productID,
		Values: map[string]any{
			"warranty_months": 120, // exceeds max 60
		},
	})
	if !errors.Is(err, shared.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for number out of range, got %v", err)
	}

	// Case 3: Invalid select option
	_, err = uc.SetEntityValues(ctx, usecase.SetEntityValuesCommand{
		TenantID:   tenantID,
		EntityType: customfield.EntityProduct,
		EntityID:   productID,
		Values: map[string]any{
			"warranty_months": 24,
			"origin_country":  "FR", // not in ["ID", "JP", "US", "DE"]
		},
	})
	if !errors.Is(err, shared.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for invalid select option, got %v", err)
	}

	// Case 4: Invalid date format
	_, err = uc.SetEntityValues(ctx, usecase.SetEntityValuesCommand{
		TenantID:   tenantID,
		EntityType: customfield.EntityProduct,
		EntityID:   productID,
		Values: map[string]any{
			"warranty_months": 24,
			"expiry_date":     "14-09-2026", // should be YYYY-MM-DD
		},
	})
	if !errors.Is(err, shared.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for invalid date format, got %v", err)
	}

	// Case 5: Valid payload with default value applied
	res, err := uc.SetEntityValues(ctx, usecase.SetEntityValuesCommand{
		TenantID:   tenantID,
		EntityType: customfield.EntityProduct,
		EntityID:   productID,
		Values: map[string]any{
			"warranty_months": 12,
			"is_organic":      true,
			"expiry_date":     "2026-09-14",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error on valid payload: %v", err)
	}

	// Check that default value "ID" was applied for origin_country
	if res.Values["origin_country"] != "ID" {
		t.Errorf("expected default origin_country 'ID', got '%v'", res.Values["origin_country"])
	}
	if res.Values["warranty_months"] != float64(12) {
		t.Errorf("expected warranty_months 12, got '%v'", res.Values["warranty_months"])
	}
}
