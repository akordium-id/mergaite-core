package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/mergiate-core/internal/core/domain/contact"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	"github.com/akordium-id/mergiate-core/internal/core/repository/postgres/sqlc"
)

type addressContactRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewAddressContactRepository creates a new PostgreSQL Address & Contact Repository.
func NewAddressContactRepository(pool *pgxpool.Pool) contact.Repository {
	return &addressContactRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *addressContactRepository) CreateAddress(ctx context.Context, addr *contact.Address) error {
	var label *string
	if addr.Label != "" {
		label = &addr.Label
	}
	var line2 *string
	if addr.Line2 != "" {
		line2 = &addr.Line2
	}
	var state *string
	if addr.State != "" {
		state = &addr.State
	}
	var postalCode *string
	if addr.PostalCode != "" {
		postalCode = &addr.PostalCode
	}

	params := sqlc.CreateAddressParams{
		ID:          shared.ToPgUUID(addr.ID),
		TenantID:    shared.ToPgUUID(addr.TenantID),
		Type:        string(addr.Type),
		Label:       label,
		Line1:       addr.Line1,
		Line2:       line2,
		City:        addr.City,
		State:       state,
		PostalCode:  postalCode,
		CountryCode: addr.CountryCode,
		Latitude:    floatToNumeric(addr.Latitude),
		Longitude:   floatToNumeric(addr.Longitude),
		CreatedAt:   pgtype.Timestamptz{Time: addr.CreatedAt, Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: addr.UpdatedAt, Valid: true},
	}

	row, err := r.queries.CreateAddress(ctx, params)
	if err != nil {
		return err
	}

	*addr = *toDomainAddress(&row)
	return nil
}

func (r *addressContactRepository) GetAddressByID(ctx context.Context, tenantID, id shared.ID) (*contact.Address, error) {
	row, err := r.queries.GetAddressByID(ctx, sqlc.GetAddressByIDParams{
		TenantID: shared.ToPgUUID(tenantID),
		ID:       shared.ToPgUUID(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toDomainAddress(&row), nil
}

func (r *addressContactRepository) LinkPartyAddress(ctx context.Context, tenantID, partyID, addressID shared.ID, addressType string, isPrimary bool) error {
	return r.queries.LinkPartyAddress(ctx, sqlc.LinkPartyAddressParams{
		TenantID:    shared.ToPgUUID(tenantID),
		PartyID:     shared.ToPgUUID(partyID),
		AddressID:   shared.ToPgUUID(addressID),
		AddressType: addressType,
		IsPrimary:   isPrimary,
		CreatedAt:   pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})
}

func (r *addressContactRepository) LinkOrganizationAddress(ctx context.Context, tenantID, orgID, addressID shared.ID, addressType string, isPrimary bool) error {
	return r.queries.LinkOrganizationAddress(ctx, sqlc.LinkOrganizationAddressParams{
		TenantID:       shared.ToPgUUID(tenantID),
		OrganizationID: shared.ToPgUUID(orgID),
		AddressID:      shared.ToPgUUID(addressID),
		AddressType:    addressType,
		IsPrimary:      isPrimary,
		CreatedAt:      pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})
}

func (r *addressContactRepository) ListPartyAddresses(ctx context.Context, tenantID, partyID shared.ID) ([]contact.Address, error) {
	rows, err := r.queries.ListPartyAddresses(ctx, sqlc.ListPartyAddressesParams{
		TenantID: shared.ToPgUUID(tenantID),
		PartyID:  shared.ToPgUUID(partyID),
	})
	if err != nil {
		return nil, err
	}

	addrs := make([]contact.Address, len(rows))
	for i, row := range rows {
		addrs[i] = contact.Address{
			ID:          shared.FromPgUUID(row.ID),
			TenantID:    shared.FromPgUUID(row.TenantID),
			Type:        contact.AddressType(row.Type),
			Label:       strFromPtr(row.Label),
			Line1:       row.Line1,
			Line2:       strFromPtr(row.Line2),
			City:        row.City,
			State:       strFromPtr(row.State),
			PostalCode:  strFromPtr(row.PostalCode),
			CountryCode: row.CountryCode,
			Latitude:    numericToFloat(row.Latitude),
			Longitude:   numericToFloat(row.Longitude),
			IsPrimary:   row.IsPrimary,
			CreatedAt:   row.CreatedAt.Time,
			UpdatedAt:   row.UpdatedAt.Time,
		}
	}
	return addrs, nil
}

func (r *addressContactRepository) ListOrganizationAddresses(ctx context.Context, tenantID, orgID shared.ID) ([]contact.Address, error) {
	rows, err := r.queries.ListOrganizationAddresses(ctx, sqlc.ListOrganizationAddressesParams{
		TenantID:       shared.ToPgUUID(tenantID),
		OrganizationID: shared.ToPgUUID(orgID),
	})
	if err != nil {
		return nil, err
	}

	addrs := make([]contact.Address, len(rows))
	for i, row := range rows {
		addrs[i] = contact.Address{
			ID:          shared.FromPgUUID(row.ID),
			TenantID:    shared.FromPgUUID(row.TenantID),
			Type:        contact.AddressType(row.Type),
			Label:       strFromPtr(row.Label),
			Line1:       row.Line1,
			Line2:       strFromPtr(row.Line2),
			City:        row.City,
			State:       strFromPtr(row.State),
			PostalCode:  strFromPtr(row.PostalCode),
			CountryCode: row.CountryCode,
			Latitude:    numericToFloat(row.Latitude),
			Longitude:   numericToFloat(row.Longitude),
			IsPrimary:   row.IsPrimary,
			CreatedAt:   row.CreatedAt.Time,
			UpdatedAt:   row.UpdatedAt.Time,
		}
	}
	return addrs, nil
}

func (r *addressContactRepository) CreateContact(ctx context.Context, c *contact.Contact) error {
	var partyID pgtype.UUID
	if c.PartyID != nil && *c.PartyID != shared.NilID() {
		partyID = shared.ToPgUUID(*c.PartyID)
	}

	var orgID pgtype.UUID
	if c.OrganizationID != nil && *c.OrganizationID != shared.NilID() {
		orgID = shared.ToPgUUID(*c.OrganizationID)
	}

	var label *string
	if c.Label != "" {
		label = &c.Label
	}

	var verifiedAt pgtype.Timestamptz
	if c.VerifiedAt != nil {
		verifiedAt = pgtype.Timestamptz{Time: *c.VerifiedAt, Valid: true}
	}

	params := sqlc.CreateContactParams{
		ID:             shared.ToPgUUID(c.ID),
		TenantID:       shared.ToPgUUID(c.TenantID),
		PartyID:        partyID,
		OrganizationID: orgID,
		Type:           string(c.Type),
		Value:          c.Value,
		Label:          label,
		IsPrimary:      c.IsPrimary,
		VerifiedAt:     verifiedAt,
		CreatedAt:      pgtype.Timestamptz{Time: c.CreatedAt, Valid: true},
		UpdatedAt:      pgtype.Timestamptz{Time: c.UpdatedAt, Valid: true},
	}

	row, err := r.queries.CreateContact(ctx, params)
	if err != nil {
		return err
	}

	*c = *toDomainContact(&row)
	return nil
}

func (r *addressContactRepository) ListPartyContacts(ctx context.Context, tenantID, partyID shared.ID) ([]contact.Contact, error) {
	rows, err := r.queries.ListPartyContacts(ctx, sqlc.ListPartyContactsParams{
		TenantID: shared.ToPgUUID(tenantID),
		PartyID:  shared.ToPgUUID(partyID),
	})
	if err != nil {
		return nil, err
	}

	contacts := make([]contact.Contact, len(rows))
	for i, row := range rows {
		contacts[i] = *toDomainContact(&row)
	}
	return contacts, nil
}

func (r *addressContactRepository) ListOrganizationContacts(ctx context.Context, tenantID, orgID shared.ID) ([]contact.Contact, error) {
	rows, err := r.queries.ListOrganizationContacts(ctx, sqlc.ListOrganizationContactsParams{
		TenantID:       shared.ToPgUUID(tenantID),
		OrganizationID: shared.ToPgUUID(orgID),
	})
	if err != nil {
		return nil, err
	}

	contacts := make([]contact.Contact, len(rows))
	for i, row := range rows {
		contacts[i] = *toDomainContact(&row)
	}
	return contacts, nil
}

func (r *addressContactRepository) DeleteContact(ctx context.Context, tenantID, id shared.ID) error {
	return r.queries.DeleteContact(ctx, sqlc.DeleteContactParams{
		TenantID: shared.ToPgUUID(tenantID),
		ID:       shared.ToPgUUID(id),
	})
}

func toDomainAddress(row *sqlc.Address) *contact.Address {
	return &contact.Address{
		ID:          shared.FromPgUUID(row.ID),
		TenantID:    shared.FromPgUUID(row.TenantID),
		Type:        contact.AddressType(row.Type),
		Label:       strFromPtr(row.Label),
		Line1:       row.Line1,
		Line2:       strFromPtr(row.Line2),
		City:        row.City,
		State:       strFromPtr(row.State),
		PostalCode:  strFromPtr(row.PostalCode),
		CountryCode: row.CountryCode,
		Latitude:    numericToFloat(row.Latitude),
		Longitude:   numericToFloat(row.Longitude),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}

func toDomainContact(row *sqlc.Contact) *contact.Contact {
	var partyID *shared.ID
	if row.PartyID.Valid {
		pid := shared.FromPgUUID(row.PartyID)
		partyID = &pid
	}
	var orgID *shared.ID
	if row.OrganizationID.Valid {
		oid := shared.FromPgUUID(row.OrganizationID)
		orgID = &oid
	}

	var verifiedAt *string
	_ = verifiedAt

	return &contact.Contact{
		ID:             shared.FromPgUUID(row.ID),
		TenantID:       shared.FromPgUUID(row.TenantID),
		PartyID:        partyID,
		OrganizationID: orgID,
		Type:           contact.Type(row.Type),
		Value:          row.Value,
		Label:          strFromPtr(row.Label),
		IsPrimary:      row.IsPrimary,
		VerifiedAt:     timeFromPgTz(row.VerifiedAt),
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}
}

func strFromPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func timeFromPgTz(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	val := t.Time
	return &val
}

func floatToNumeric(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{Valid: false}
	}
	var num pgtype.Numeric
	_ = num.Scan(fmt.Sprintf("%.8f", *f))
	return num
}

func numericToFloat(num pgtype.Numeric) *float64 {
	if !num.Valid {
		return nil
	}
	f, _ := num.Float64Value()
	val := f.Float64
	return &val
}
