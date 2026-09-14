package contact

import (
	"context"

	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

// Repository defines storage operations for Addresses and Contacts.
type Repository interface {
	CreateAddress(ctx context.Context, addr *Address) error
	GetAddressByID(ctx context.Context, tenantID, id shared.ID) (*Address, error)
	LinkPartyAddress(ctx context.Context, tenantID, partyID, addressID shared.ID, addressType string, isPrimary bool) error
	LinkOrganizationAddress(ctx context.Context, tenantID, orgID, addressID shared.ID, addressType string, isPrimary bool) error
	ListPartyAddresses(ctx context.Context, tenantID, partyID shared.ID) ([]Address, error)
	ListOrganizationAddresses(ctx context.Context, tenantID, orgID shared.ID) ([]Address, error)

	CreateContact(ctx context.Context, c *Contact) error
	ListPartyContacts(ctx context.Context, tenantID, partyID shared.ID) ([]Contact, error)
	ListOrganizationContacts(ctx context.Context, tenantID, orgID shared.ID) ([]Contact, error)
	DeleteContact(ctx context.Context, tenantID, id shared.ID) error
}
