package contact

import (
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

type AddressType string

const (
	AddressTypeGeneral   AddressType = "general"
	AddressTypeBilling   AddressType = "billing"
	AddressTypeShipping  AddressType = "shipping"
	AddressTypeOffice    AddressType = "office"
	AddressTypeWarehouse AddressType = "warehouse"
	AddressTypeHome      AddressType = "home"
)

type Address struct {
	ID          shared.ID   `json:"id"`
	TenantID    shared.ID   `json:"tenant_id"`
	Type        AddressType `json:"type"`
	Label       string      `json:"label,omitempty"`
	Line1       string      `json:"line1"`
	Line2       string      `json:"line2,omitempty"`
	City        string      `json:"city"`
	State       string      `json:"state,omitempty"`
	PostalCode  string      `json:"postal_code,omitempty"`
	CountryCode string      `json:"country_code"`
	Latitude    *float64    `json:"latitude,omitempty"`
	Longitude   *float64    `json:"longitude,omitempty"`
	IsPrimary   bool        `json:"is_primary,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}
