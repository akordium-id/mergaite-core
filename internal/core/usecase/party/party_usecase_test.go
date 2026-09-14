package party_test

import (
	"context"
	"testing"

	"github.com/akordium-id/mergiate-core/internal/core/domain/contact"
	"github.com/akordium-id/mergiate-core/internal/core/domain/party"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	partyuc "github.com/akordium-id/mergiate-core/internal/core/usecase/party"
)

type mockPartyRepo struct {
	parties map[shared.ID]*party.Party
	roles   map[shared.ID]*party.PartyRole
}

func newMockPartyRepo() *mockPartyRepo {
	return &mockPartyRepo{
		parties: make(map[shared.ID]*party.Party),
		roles:   make(map[shared.ID]*party.PartyRole),
	}
}

func (m *mockPartyRepo) Create(ctx context.Context, p *party.Party) error {
	m.parties[p.ID] = p
	return nil
}

func (m *mockPartyRepo) GetByID(ctx context.Context, tenantID, id shared.ID) (*party.Party, error) {
	p, ok := m.parties[id]
	if !ok || p.TenantID != tenantID {
		return nil, shared.ErrNotFound
	}
	// populate roles
	var roles []party.PartyRole
	for _, r := range m.roles {
		if r.PartyID == p.ID && r.TenantID == tenantID {
			roles = append(roles, *r)
		}
	}
	p.Roles = roles
	return p, nil
}

func (m *mockPartyRepo) List(ctx context.Context, tenantID shared.ID, filter party.PartyFilter) ([]party.Party, int64, error) {
	var res []party.Party
	for _, p := range m.parties {
		if p.TenantID == tenantID {
			res = append(res, *p)
		}
	}
	return res, int64(len(res)), nil
}

func (m *mockPartyRepo) Update(ctx context.Context, p *party.Party) error {
	m.parties[p.ID] = p
	return nil
}

func (m *mockPartyRepo) AddRole(ctx context.Context, role *party.PartyRole) error {
	m.roles[role.ID] = role
	return nil
}

func (m *mockPartyRepo) ListRoles(ctx context.Context, tenantID, partyID shared.ID) ([]party.PartyRole, error) {
	var res []party.PartyRole
	for _, r := range m.roles {
		if r.TenantID == tenantID && r.PartyID == partyID {
			res = append(res, *r)
		}
	}
	return res, nil
}

func (m *mockPartyRepo) RemoveRole(ctx context.Context, tenantID, roleID shared.ID) error {
	delete(m.roles, roleID)
	return nil
}

type mockContactRepo struct {
	addresses []contact.Address
	contacts  []contact.Contact
}

func newMockContactRepo() *mockContactRepo {
	return &mockContactRepo{}
}

func (m *mockContactRepo) CreateAddress(ctx context.Context, addr *contact.Address) error {
	m.addresses = append(m.addresses, *addr)
	return nil
}

func (m *mockContactRepo) GetAddressByID(ctx context.Context, tenantID, id shared.ID) (*contact.Address, error) {
	for _, a := range m.addresses {
		if a.ID == id && a.TenantID == tenantID {
			return &a, nil
		}
	}
	return nil, shared.ErrNotFound
}

func (m *mockContactRepo) LinkPartyAddress(ctx context.Context, tenantID, partyID, addressID shared.ID, addressType string, isPrimary bool) error {
	return nil
}

func (m *mockContactRepo) LinkOrganizationAddress(ctx context.Context, tenantID, orgID, addressID shared.ID, addressType string, isPrimary bool) error {
	return nil
}

func (m *mockContactRepo) ListPartyAddresses(ctx context.Context, tenantID, partyID shared.ID) ([]contact.Address, error) {
	return m.addresses, nil
}

func (m *mockContactRepo) ListOrganizationAddresses(ctx context.Context, tenantID, orgID shared.ID) ([]contact.Address, error) {
	return m.addresses, nil
}

func (m *mockContactRepo) CreateContact(ctx context.Context, c *contact.Contact) error {
	m.contacts = append(m.contacts, *c)
	return nil
}

func (m *mockContactRepo) ListPartyContacts(ctx context.Context, tenantID, partyID shared.ID) ([]contact.Contact, error) {
	return m.contacts, nil
}

func (m *mockContactRepo) ListOrganizationContacts(ctx context.Context, tenantID, orgID shared.ID) ([]contact.Contact, error) {
	return m.contacts, nil
}

func (m *mockContactRepo) DeleteContact(ctx context.Context, tenantID, id shared.ID) error {
	return nil
}

func TestPartyUsecase_CreateWithRolesAndContacts(t *testing.T) {
	partyRepo := newMockPartyRepo()
	contactRepo := newMockContactRepo()
	uc := partyuc.NewUsecase(partyRepo, contactRepo)

	tenantID := shared.MustNewID()
	ctx := shared.WithTenantID(context.Background(), tenantID)

	// 1. Create Party with Customer + Supplier initial roles
	p, err := uc.CreateParty(ctx, partyuc.CreatePartyCommand{
		Type:      party.TypeOrganization,
		Code:      "SMK",
		Name:      "PT Sumber Makmur",
		LegalName: "PT Sumber Makmur Sejahtera",
		TaxID:     "01.234.567.8-901.000",
		InitialRoles: []party.RoleType{
			party.RoleCustomer,
			party.RoleSupplier,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(p.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(p.Roles))
	}
	if !p.HasRole(party.RoleCustomer) {
		t.Fatal("expected party to have Customer role")
	}
	if !p.HasRole(party.RoleSupplier) {
		t.Fatal("expected party to have Supplier role")
	}

	// 2. Add Address
	addr, err := uc.AddAddress(ctx, p.ID, partyuc.AddAddressCommand{
		Type:        contact.AddressTypeOffice,
		Line1:       "Jl. Basuki Rahmat No. 10",
		City:        "Surabaya",
		State:       "Jawa Timur",
		PostalCode:  "60271",
		CountryCode: "ID",
		IsPrimary:   true,
	})
	if err != nil {
		t.Fatalf("unexpected address error: %v", err)
	}
	if addr.City != "Surabaya" {
		t.Errorf("expected Surabaya, got %s", addr.City)
	}

	// 3. Add Contact
	c, err := uc.AddContact(ctx, p.ID, partyuc.AddContactCommand{
		Type:      contact.TypeEmail,
		Value:     "finance@sumbermakmur.com",
		Label:     "Finance Email",
		IsPrimary: true,
	})
	if err != nil {
		t.Fatalf("unexpected contact error: %v", err)
	}
	if c.Value != "finance@sumbermakmur.com" {
		t.Errorf("expected email, got %s", c.Value)
	}

	// 4. Retrieve Detail
	detail, err := uc.GetParty(ctx, p.ID)
	if err != nil {
		t.Fatalf("unexpected detail error: %v", err)
	}
	if len(detail.Party.Roles) != 2 {
		t.Errorf("expected 2 roles in detail, got %d", len(detail.Party.Roles))
	}
}
