package party

import (
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

type Type string

const (
	TypePerson       Type = "person"
	TypeOrganization Type = "organization"
)

type RoleType string

const (
	RoleCustomer RoleType = "customer"
	RoleSupplier RoleType = "supplier"
	RolePartner  RoleType = "partner"
	RoleEmployee RoleType = "employee"
	RoleAgent    RoleType = "agent"
	RoleVendor   RoleType = "vendor"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

// Party is the universal business actor aggregate (individual person or legal entity).
type Party struct {
	ID        shared.ID      `json:"id"`
	TenantID  shared.ID      `json:"tenant_id"`
	Type      Type           `json:"type"`
	Code      string         `json:"code,omitempty"`
	Name      string         `json:"name"`
	LegalName string         `json:"legal_name,omitempty"`
	TaxID     string         `json:"tax_id,omitempty"`
	Status    Status         `json:"status"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Roles     []PartyRole    `json:"roles,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// PartyRole defines a commercial or operational relationship between a Party and the Tenant/Organization.
type PartyRole struct {
	ID             shared.ID  `json:"id"`
	TenantID       shared.ID  `json:"tenant_id"`
	PartyID        shared.ID  `json:"party_id"`
	OrganizationID *shared.ID `json:"organization_id,omitempty"`
	RoleType       RoleType   `json:"role_type"`
	Status         Status     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// HasRole checks if the party currently holds an active role of the specified type.
func (p *Party) HasRole(roleType RoleType) bool {
	for _, r := range p.Roles {
		if r.RoleType == roleType && r.Status == StatusActive {
			return true
		}
	}
	return false
}
