package contact

import (
	"time"

	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

type Type string

const (
	TypeEmail   Type = "email"
	TypePhone   Type = "phone"
	TypeMobile  Type = "mobile"
	TypeWebsite Type = "website"
)

type Contact struct {
	ID             shared.ID  `json:"id"`
	TenantID       shared.ID  `json:"tenant_id"`
	PartyID        *shared.ID `json:"party_id,omitempty"`
	OrganizationID *shared.ID `json:"organization_id,omitempty"`
	Type           Type       `json:"type"`
	Value          string     `json:"value"`
	Label          string     `json:"label,omitempty"`
	IsPrimary      bool       `json:"is_primary"`
	VerifiedAt     *time.Time `json:"verified_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
