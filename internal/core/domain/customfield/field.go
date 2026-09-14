package customfield

import (
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

type DataType string

const (
	DataTypeText        DataType = "text"
	DataTypeNumber      DataType = "number"
	DataTypeBoolean     DataType = "boolean"
	DataTypeDate        DataType = "date"
	DataTypeSelect      DataType = "select"
	DataTypeMultiSelect DataType = "multi_select"
	DataTypeJSON        DataType = "json"
)

func (d DataType) IsValid() bool {
	switch d {
	case DataTypeText, DataTypeNumber, DataTypeBoolean, DataTypeDate, DataTypeSelect, DataTypeMultiSelect, DataTypeJSON:
		return true
	default:
		return false
	}
}

type EntityType string

const (
	EntityParty        EntityType = "party"
	EntityProduct      EntityType = "product"
	EntityDocument     EntityType = "document"
	EntityOrganization EntityType = "organization"
)

func (e EntityType) IsValid() bool {
	switch e {
	case EntityParty, EntityProduct, EntityDocument, EntityOrganization:
		return true
	default:
		return false
	}
}

// ValidationRules specifies constraints on field values.
type ValidationRules struct {
	Min   *float64 `json:"min,omitempty"`
	Max   *float64 `json:"max,omitempty"`
	Regex string   `json:"regex,omitempty"`
}

// Definition represents the schema configuration for a dynamic custom field.
type Definition struct {
	ID              shared.ID       `json:"id"`
	TenantID        shared.ID       `json:"tenant_id"`
	EntityType      EntityType      `json:"entity_type"`
	Code            string          `json:"code"`
	Name            string          `json:"name"`
	Description     string          `json:"description,omitempty"`
	DataType        DataType        `json:"data_type"`
	Options         []string        `json:"options,omitempty"`
	IsRequired      bool            `json:"is_required"`
	DefaultValue    any             `json:"default_value,omitempty"`
	ValidationRules ValidationRules `json:"validation_rules,omitzero"`
	SortOrder       int32           `json:"sort_order"`
	IsActive        bool            `json:"is_active"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// EntityCustomFields represents the set of custom attribute values for a specific entity instance.
type EntityCustomFields struct {
	ID         shared.ID      `json:"id"`
	TenantID   shared.ID      `json:"tenant_id"`
	EntityType EntityType     `json:"entity_type"`
	EntityID   shared.ID      `json:"entity_id"`
	Values     map[string]any `json:"values"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}
