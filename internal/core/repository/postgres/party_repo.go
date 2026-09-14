package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/mergaite-core/internal/core/domain/party"
	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
	"github.com/akordium-id/mergaite-core/internal/core/repository/postgres/sqlc"
)

type partyRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewPartyRepository creates a new PostgreSQL PartyRepository.
func NewPartyRepository(pool *pgxpool.Pool) party.Repository {
	return &partyRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *partyRepository) Create(ctx context.Context, p *party.Party) error {
	metadataJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("%w: invalid party metadata json", shared.ErrInvalidInput)
	}

	var code *string
	if p.Code != "" {
		code = &p.Code
	}
	var legalName *string
	if p.LegalName != "" {
		legalName = &p.LegalName
	}
	var taxID *string
	if p.TaxID != "" {
		taxID = &p.TaxID
	}

	params := sqlc.CreatePartyParams{
		ID:        shared.ToPgUUID(p.ID),
		TenantID:  shared.ToPgUUID(p.TenantID),
		Type:      string(p.Type),
		Code:      code,
		Name:      p.Name,
		LegalName: legalName,
		TaxID:     taxID,
		Status:    string(p.Status),
		Metadata:  metadataJSON,
		CreatedAt: pgtype.Timestamptz{Time: p.CreatedAt, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: p.UpdatedAt, Valid: true},
	}

	row, err := r.queries.CreateParty(ctx, params)
	if err != nil {
		return err
	}

	*p = *toDomainParty(&row)
	return nil
}

func (r *partyRepository) GetByID(ctx context.Context, tenantID, id shared.ID) (*party.Party, error) {
	row, err := r.queries.GetPartyByID(ctx, sqlc.GetPartyByIDParams{
		TenantID: shared.ToPgUUID(tenantID),
		ID:       shared.ToPgUUID(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}

	domainParty := toDomainParty(&row)

	// Fetch roles
	roles, err := r.ListRoles(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	domainParty.Roles = roles

	return domainParty, nil
}

func (r *partyRepository) List(ctx context.Context, tenantID shared.ID, filter party.PartyFilter) ([]party.Party, int64, error) {
	limit := max(filter.Limit, 20)
	offset := max(filter.Offset, 0)

	var typeStr *string
	if filter.Type != nil {
		s := string(*filter.Type)
		typeStr = &s
	}

	if filter.RoleType != nil {
		roleStr := string(*filter.RoleType)
		total, err := r.queries.CountPartiesByRole(ctx, sqlc.CountPartiesByRoleParams{
			TenantID: shared.ToPgUUID(tenantID),
			RoleType: roleStr,
		})
		if err != nil {
			return nil, 0, err
		}

		rows, err := r.queries.ListPartiesByRole(ctx, sqlc.ListPartiesByRoleParams{
			TenantID: shared.ToPgUUID(tenantID),
			RoleType: roleStr,
			Limit:    limit,
			Offset:   offset,
		})
		if err != nil {
			return nil, 0, err
		}

		parties := make([]party.Party, len(rows))
		for i, row := range rows {
			parties[i] = *toDomainParty(&row)
		}
		return parties, total, nil
	}

	total, err := r.queries.CountParties(ctx, sqlc.CountPartiesParams{
		TenantID: shared.ToPgUUID(tenantID),
		Type:     typeStr,
	})
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.ListParties(ctx, sqlc.ListPartiesParams{
		TenantID: shared.ToPgUUID(tenantID),
		Limit:    limit,
		Offset:   offset,
		Type:     typeStr,
	})
	if err != nil {
		return nil, 0, err
	}

	parties := make([]party.Party, len(rows))
	for i, row := range rows {
		parties[i] = *toDomainParty(&row)
	}
	return parties, total, nil
}

func (r *partyRepository) Update(ctx context.Context, p *party.Party) error {
	var metadataJSON []byte
	if p.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(p.Metadata)
		if err != nil {
			return fmt.Errorf("%w: invalid party metadata json", shared.ErrInvalidInput)
		}
	}

	var legalName *string
	if p.LegalName != "" {
		legalName = &p.LegalName
	}
	var taxID *string
	if p.TaxID != "" {
		taxID = &p.TaxID
	}
	statusStr := string(p.Status)

	params := sqlc.UpdatePartyParams{
		TenantID:  shared.ToPgUUID(p.TenantID),
		ID:        shared.ToPgUUID(p.ID),
		Name:      &p.Name,
		LegalName: legalName,
		TaxID:     taxID,
		Status:    &statusStr,
		Metadata:  metadataJSON,
	}

	row, err := r.queries.UpdateParty(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return shared.ErrNotFound
		}
		return err
	}

	*p = *toDomainParty(&row)
	return nil
}

func (r *partyRepository) AddRole(ctx context.Context, role *party.PartyRole) error {
	var orgID pgtype.UUID
	if role.OrganizationID != nil && *role.OrganizationID != shared.NilID() {
		orgID = shared.ToPgUUID(*role.OrganizationID)
	}

	params := sqlc.CreatePartyRoleParams{
		ID:             shared.ToPgUUID(role.ID),
		TenantID:       shared.ToPgUUID(role.TenantID),
		PartyID:        shared.ToPgUUID(role.PartyID),
		OrganizationID: orgID,
		RoleType:       string(role.RoleType),
		Status:         string(role.Status),
		CreatedAt:      pgtype.Timestamptz{Time: role.CreatedAt, Valid: true},
		UpdatedAt:      pgtype.Timestamptz{Time: role.UpdatedAt, Valid: true},
	}

	row, err := r.queries.CreatePartyRole(ctx, params)
	if err != nil {
		return err
	}

	*role = *toDomainRole(&row)
	return nil
}

func (r *partyRepository) ListRoles(ctx context.Context, tenantID, partyID shared.ID) ([]party.PartyRole, error) {
	rows, err := r.queries.ListRolesByPartyID(ctx, sqlc.ListRolesByPartyIDParams{
		TenantID: shared.ToPgUUID(tenantID),
		PartyID:  shared.ToPgUUID(partyID),
	})
	if err != nil {
		return nil, err
	}

	roles := make([]party.PartyRole, len(rows))
	for i, row := range rows {
		roles[i] = *toDomainRole(&row)
	}
	return roles, nil
}

func (r *partyRepository) RemoveRole(ctx context.Context, tenantID, roleID shared.ID) error {
	return r.queries.DeletePartyRole(ctx, sqlc.DeletePartyRoleParams{
		TenantID: shared.ToPgUUID(tenantID),
		ID:       shared.ToPgUUID(roleID),
	})
}

func toDomainParty(row *sqlc.Party) *party.Party {
	var metadata map[string]any
	if len(row.Metadata) > 0 {
		_ = json.Unmarshal(row.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]any)
	}

	code := ""
	if row.Code != nil {
		code = *row.Code
	}
	legalName := ""
	if row.LegalName != nil {
		legalName = *row.LegalName
	}
	taxID := ""
	if row.TaxID != nil {
		taxID = *row.TaxID
	}

	return &party.Party{
		ID:        shared.FromPgUUID(row.ID),
		TenantID:  shared.FromPgUUID(row.TenantID),
		Type:      party.Type(row.Type),
		Code:      code,
		Name:      row.Name,
		LegalName: legalName,
		TaxID:     taxID,
		Status:    party.Status(row.Status),
		Metadata:  metadata,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}

func toDomainRole(row *sqlc.PartyRole) *party.PartyRole {
	var orgID *shared.ID
	if row.OrganizationID.Valid {
		id := shared.FromPgUUID(row.OrganizationID)
		orgID = &id
	}

	return &party.PartyRole{
		ID:             shared.FromPgUUID(row.ID),
		TenantID:       shared.FromPgUUID(row.TenantID),
		PartyID:        shared.FromPgUUID(row.PartyID),
		OrganizationID: orgID,
		RoleType:       party.RoleType(row.RoleType),
		Status:         party.Status(row.Status),
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}
}
