package identity

import (
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

// Role represents a bundle of permissions defined within a specific tenant.
type Role struct {
	ID          shared.ID    `json:"id"`
	TenantID    shared.ID    `json:"tenant_id"`
	Code        string       `json:"code"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	IsSystem    bool         `json:"is_system"`
	Permissions []Permission `json:"permissions,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// Permission represents an individual action authorization key in the system catalog.
type Permission struct {
	ID          shared.ID `json:"id"`
	Code        string    `json:"code"` // e.g. "document:create", "document:approve"
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Description string    `json:"description,omitempty"`
}
