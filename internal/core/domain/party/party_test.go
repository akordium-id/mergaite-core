package party_test

import (
	"testing"

	"github.com/akordium-id/mergaite-core/internal/core/domain/party"
	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

func TestParty_HasRole(t *testing.T) {
	p := party.Party{
		ID:       shared.MustNewID(),
		TenantID: shared.MustNewID(),
		Type:     party.TypeOrganization,
		Name:     "PT Maju Jaya",
		Status:   party.StatusActive,
		Roles: []party.PartyRole{
			{
				RoleType: party.RoleCustomer,
				Status:   party.StatusActive,
			},
			{
				RoleType: party.RoleSupplier,
				Status:   party.StatusInactive,
			},
		},
	}

	if !p.HasRole(party.RoleCustomer) {
		t.Fatal("expected party to have active Customer role")
	}
	if p.HasRole(party.RoleSupplier) {
		t.Fatal("expected party NOT to have active Supplier role (status inactive)")
	}
	if p.HasRole(party.RolePartner) {
		t.Fatal("expected party NOT to have Partner role")
	}
}
