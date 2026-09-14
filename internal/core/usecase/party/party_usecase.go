package party

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/akordium-id/mergaite-core/internal/core/domain/contact"
	"github.com/akordium-id/mergaite-core/internal/core/domain/party"
	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

type CreatePartyCommand struct {
	Type         party.Type       `json:"type"`
	Code         string           `json:"code,omitempty"`
	Name         string           `json:"name"`
	LegalName    string           `json:"legal_name,omitempty"`
	TaxID        string           `json:"tax_id,omitempty"`
	Metadata     map[string]any   `json:"metadata,omitempty"`
	InitialRoles []party.RoleType `json:"initial_roles,omitempty"`
}

type UpdatePartyCommand struct {
	ID        shared.ID    `json:"id"`
	Name      string       `json:"name,omitempty"`
	LegalName string       `json:"legal_name,omitempty"`
	TaxID     string       `json:"tax_id,omitempty"`
	Status    party.Status `json:"status,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type AddRoleCommand struct {
	PartyID        shared.ID      `json:"party_id"`
	OrganizationID *shared.ID     `json:"organization_id,omitempty"`
	RoleType       party.RoleType `json:"role_type"`
}

type AddAddressCommand struct {
	Type        contact.AddressType `json:"type"`
	Label       string              `json:"label,omitempty"`
	Line1       string              `json:"line1"`
	Line2       string              `json:"line2,omitempty"`
	City        string              `json:"city"`
	State       string              `json:"state,omitempty"`
	PostalCode  string              `json:"postal_code,omitempty"`
	CountryCode string              `json:"country_code"`
	IsPrimary   bool                `json:"is_primary"`
}

type AddContactCommand struct {
	Type      contact.Type `json:"type"`
	Value     string       `json:"value"`
	Label     string       `json:"label,omitempty"`
	IsPrimary bool         `json:"is_primary"`
}

type PartyDetail struct {
	Party     party.Party       `json:"party"`
	Addresses []contact.Address `json:"addresses"`
	Contacts  []contact.Contact `json:"contacts"`
}

type PartyListResult struct {
	Items    []party.Party `json:"items"`
	Total    int64         `json:"total"`
	Page     int32         `json:"page"`
	PageSize int32         `json:"page_size"`
}

type Usecase interface {
	CreateParty(ctx context.Context, cmd CreatePartyCommand) (*party.Party, error)
	GetParty(ctx context.Context, id shared.ID) (*PartyDetail, error)
	ListParties(ctx context.Context, page, pageSize int32, partyType *party.Type, roleType *party.RoleType) (*PartyListResult, error)
	UpdateParty(ctx context.Context, cmd UpdatePartyCommand) (*party.Party, error)

	AddRole(ctx context.Context, cmd AddRoleCommand) (*party.PartyRole, error)
	RemoveRole(ctx context.Context, roleID shared.ID) error

	AddAddress(ctx context.Context, partyID shared.ID, cmd AddAddressCommand) (*contact.Address, error)
	AddContact(ctx context.Context, partyID shared.ID, cmd AddContactCommand) (*contact.Contact, error)
}

type usecase struct {
	partyRepo   party.Repository
	contactRepo contact.Repository
}

func NewUsecase(partyRepo party.Repository, contactRepo contact.Repository) Usecase {
	return &usecase{
		partyRepo:   partyRepo,
		contactRepo: contactRepo,
	}
}

func (u *usecase) CreateParty(ctx context.Context, cmd CreatePartyCommand) (*party.Party, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: party name is required", shared.ErrInvalidInput)
	}

	partyType := cmd.Type
	if partyType == "" {
		partyType = party.TypeOrganization
	}

	id, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	metadata := cmd.Metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}

	p := &party.Party{
		ID:        id,
		TenantID:  tenantID,
		Type:      partyType,
		Code:      strings.TrimSpace(cmd.Code),
		Name:      name,
		LegalName: strings.TrimSpace(cmd.LegalName),
		TaxID:     strings.TrimSpace(cmd.TaxID),
		Status:    party.StatusActive,
		Metadata:  metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := u.partyRepo.Create(ctx, p); err != nil {
		return nil, err
	}

	// Assign initial roles if requested
	for _, roleType := range cmd.InitialRoles {
		roleID, _ := shared.NewID()
		role := &party.PartyRole{
			ID:        roleID,
			TenantID:  tenantID,
			PartyID:   p.ID,
			RoleType:  roleType,
			Status:    party.StatusActive,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := u.partyRepo.AddRole(ctx, role); err == nil {
			p.Roles = append(p.Roles, *role)
		}
	}

	return p, nil
}

func (u *usecase) GetParty(ctx context.Context, id shared.ID) (*PartyDetail, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	p, err := u.partyRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	addresses, err := u.contactRepo.ListPartyAddresses(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	contacts, err := u.contactRepo.ListPartyContacts(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	return &PartyDetail{
		Party:     *p,
		Addresses: addresses,
		Contacts:  contacts,
	}, nil
}

func (u *usecase) ListParties(ctx context.Context, page, pageSize int32, partyType *party.Type, roleType *party.RoleType) (*PartyListResult, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	filter := party.PartyFilter{
		Type:     partyType,
		RoleType: roleType,
		Limit:    pageSize,
		Offset:   offset,
	}

	items, total, err := u.partyRepo.List(ctx, tenantID, filter)
	if err != nil {
		return nil, err
	}

	return &PartyListResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (u *usecase) UpdateParty(ctx context.Context, cmd UpdatePartyCommand) (*party.Party, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	p, err := u.partyRepo.GetByID(ctx, tenantID, cmd.ID)
	if err != nil {
		return nil, err
	}

	if cmd.Name != "" {
		p.Name = strings.TrimSpace(cmd.Name)
	}
	if cmd.LegalName != "" {
		p.LegalName = strings.TrimSpace(cmd.LegalName)
	}
	if cmd.TaxID != "" {
		p.TaxID = strings.TrimSpace(cmd.TaxID)
	}
	if cmd.Status != "" {
		p.Status = cmd.Status
	}
	if cmd.Metadata != nil {
		p.Metadata = cmd.Metadata
	}
	p.UpdatedAt = time.Now().UTC()

	if err := u.partyRepo.Update(ctx, p); err != nil {
		return nil, err
	}

	return p, nil
}

func (u *usecase) AddRole(ctx context.Context, cmd AddRoleCommand) (*party.PartyRole, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	// Verify party exists
	_, err = u.partyRepo.GetByID(ctx, tenantID, cmd.PartyID)
	if err != nil {
		return nil, err
	}

	roleID, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	role := &party.PartyRole{
		ID:             roleID,
		TenantID:       tenantID,
		PartyID:        cmd.PartyID,
		OrganizationID: cmd.OrganizationID,
		RoleType:       cmd.RoleType,
		Status:         party.StatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := u.partyRepo.AddRole(ctx, role); err != nil {
		return nil, err
	}

	return role, nil
}

func (u *usecase) RemoveRole(ctx context.Context, roleID shared.ID) error {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return err
	}
	return u.partyRepo.RemoveRole(ctx, tenantID, roleID)
}

func (u *usecase) AddAddress(ctx context.Context, partyID shared.ID, cmd AddAddressCommand) (*contact.Address, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	// Verify party
	if _, err := u.partyRepo.GetByID(ctx, tenantID, partyID); err != nil {
		return nil, err
	}

	addrID, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	countryCode := strings.ToUpper(strings.TrimSpace(cmd.CountryCode))
	if countryCode == "" {
		countryCode = "ID"
	}

	now := time.Now().UTC()
	addr := &contact.Address{
		ID:          addrID,
		TenantID:    tenantID,
		Type:        cmd.Type,
		Label:       cmd.Label,
		Line1:       cmd.Line1,
		Line2:       cmd.Line2,
		City:        cmd.City,
		State:       cmd.State,
		PostalCode:  cmd.PostalCode,
		CountryCode: countryCode,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := u.contactRepo.CreateAddress(ctx, addr); err != nil {
		return nil, err
	}

	// Link address to party
	if err := u.contactRepo.LinkPartyAddress(ctx, tenantID, partyID, addr.ID, string(cmd.Type), cmd.IsPrimary); err != nil {
		return nil, err
	}

	addr.IsPrimary = cmd.IsPrimary
	return addr, nil
}

func (u *usecase) AddContact(ctx context.Context, partyID shared.ID, cmd AddContactCommand) (*contact.Contact, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	// Verify party
	if _, err := u.partyRepo.GetByID(ctx, tenantID, partyID); err != nil {
		return nil, err
	}

	contactID, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	c := &contact.Contact{
		ID:        contactID,
		TenantID:  tenantID,
		PartyID:   &partyID,
		Type:      cmd.Type,
		Value:     cmd.Value,
		Label:     cmd.Label,
		IsPrimary: cmd.IsPrimary,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := u.contactRepo.CreateContact(ctx, c); err != nil {
		return nil, err
	}

	return c, nil
}
