package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/mergaite-core/internal/core/domain/organization"
	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
	"github.com/akordium-id/mergaite-core/internal/core/repository/postgres/sqlc"
)

type organizationRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewOrganizationRepository creates a new PostgreSQL OrganizationRepository.
func NewOrganizationRepository(pool *pgxpool.Pool) organization.Repository {
	return &organizationRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *organizationRepository) Create(ctx context.Context, o *organization.Organization) error {
	settingsJSON, err := json.Marshal(o.Settings)
	if err != nil {
		return fmt.Errorf("%w: invalid organization settings json", shared.ErrInvalidInput)
	}

	var parentID pgtype.UUID
	if o.ParentID != nil && *o.ParentID != shared.NilID() {
		parentID = shared.ToPgUUID(*o.ParentID)
	}

	var legalName *string
	if o.LegalName != "" {
		legalName = &o.LegalName
	}

	params := sqlc.CreateOrganizationParams{
		ID:        shared.ToPgUUID(o.ID),
		TenantID:  shared.ToPgUUID(o.TenantID),
		ParentID:  parentID,
		Code:      o.Code,
		Name:      o.Name,
		LegalName: legalName,
		Type:      string(o.Type),
		Status:    string(o.Status),
		Settings:  settingsJSON,
		CreatedAt: pgtype.Timestamptz{Time: o.CreatedAt, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: o.UpdatedAt, Valid: true},
	}

	row, err := r.queries.CreateOrganization(ctx, params)
	if err != nil {
		return err
	}

	*o = *toDomainOrganization(&row)
	return nil
}

func (r *organizationRepository) GetByID(ctx context.Context, tenantID, id shared.ID) (*organization.Organization, error) {
	row, err := r.queries.GetOrganizationByID(ctx, sqlc.GetOrganizationByIDParams{
		TenantID: shared.ToPgUUID(tenantID),
		ID:       shared.ToPgUUID(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toDomainOrganization(&row), nil
}

func (r *organizationRepository) GetByCode(ctx context.Context, tenantID shared.ID, code string) (*organization.Organization, error) {
	row, err := r.queries.GetOrganizationByCode(ctx, sqlc.GetOrganizationByCodeParams{
		TenantID: shared.ToPgUUID(tenantID),
		Code:     code,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toDomainOrganization(&row), nil
}

func (r *organizationRepository) ListByTenant(ctx context.Context, tenantID shared.ID) ([]organization.Organization, error) {
	rows, err := r.queries.ListOrganizationsByTenant(ctx, shared.ToPgUUID(tenantID))
	if err != nil {
		return nil, err
	}

	result := make([]organization.Organization, len(rows))
	for i, row := range rows {
		result[i] = *toDomainOrganization(&row)
	}
	return result, nil
}

func (r *organizationRepository) ListChildren(ctx context.Context, tenantID, parentID shared.ID) ([]organization.Organization, error) {
	rows, err := r.queries.ListOrganizationChildren(ctx, sqlc.ListOrganizationChildrenParams{
		TenantID: shared.ToPgUUID(tenantID),
		ParentID: shared.ToPgUUID(parentID),
	})
	if err != nil {
		return nil, err
	}

	result := make([]organization.Organization, len(rows))
	for i, row := range rows {
		result[i] = *toDomainOrganization(&row)
	}
	return result, nil
}

func (r *organizationRepository) Update(ctx context.Context, o *organization.Organization) error {
	var settingsJSON []byte
	if o.Settings != nil {
		var err error
		settingsJSON, err = json.Marshal(o.Settings)
		if err != nil {
			return fmt.Errorf("%w: invalid organization settings json", shared.ErrInvalidInput)
		}
	}

	var parentID pgtype.UUID
	if o.ParentID != nil && *o.ParentID != shared.NilID() {
		parentID = shared.ToPgUUID(*o.ParentID)
	}

	var legalName *string
	if o.LegalName != "" {
		legalName = &o.LegalName
	}

	typeStr := string(o.Type)
	statusStr := string(o.Status)

	params := sqlc.UpdateOrganizationParams{
		TenantID:  shared.ToPgUUID(o.TenantID),
		ID:        shared.ToPgUUID(o.ID),
		ParentID:  parentID,
		Name:      &o.Name,
		LegalName: legalName,
		Type:      &typeStr,
		Status:    &statusStr,
		Settings:  settingsJSON,
	}

	row, err := r.queries.UpdateOrganization(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return shared.ErrNotFound
		}
		return err
	}

	*o = *toDomainOrganization(&row)
	return nil
}

func toDomainOrganization(row *sqlc.Organization) *organization.Organization {
	var settings map[string]any
	if len(row.Settings) > 0 {
		_ = json.Unmarshal(row.Settings, &settings)
	}
	if settings == nil {
		settings = make(map[string]any)
	}

	var parentID *shared.ID
	if row.ParentID.Valid {
		pid := shared.FromPgUUID(row.ParentID)
		parentID = &pid
	}

	legalName := ""
	if row.LegalName != nil {
		legalName = *row.LegalName
	}

	return &organization.Organization{
		ID:        shared.FromPgUUID(row.ID),
		TenantID:  shared.FromPgUUID(row.TenantID),
		ParentID:  parentID,
		Code:      row.Code,
		Name:      row.Name,
		LegalName: legalName,
		Type:      organization.Type(row.Type),
		Status:    organization.Status(row.Status),
		Settings:  settings,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
