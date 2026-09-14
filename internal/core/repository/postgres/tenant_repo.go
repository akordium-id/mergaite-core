package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/mergaite-core/internal/core/domain/repository"
	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
	"github.com/akordium-id/mergaite-core/internal/core/repository/postgres/sqlc"
)

type tenantRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewTenantRepository creates a new PostgreSQL TenantRepository.
func NewTenantRepository(pool *pgxpool.Pool) repository.TenantRepository {
	return &tenantRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *tenantRepository) Create(ctx context.Context, t *shared.Tenant) error {
	settingsJSON, err := json.Marshal(t.Settings)
	if err != nil {
		return fmt.Errorf("%w: invalid tenant settings json", shared.ErrInvalidInput)
	}

	params := sqlc.CreateTenantParams{
		ID:        shared.ToPgUUID(t.ID),
		Code:      t.Code,
		Name:      t.Name,
		Status:    string(t.Status),
		Settings:  settingsJSON,
		CreatedAt: pgtype.Timestamptz{Time: t.CreatedAt, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: t.UpdatedAt, Valid: true},
	}

	row, err := r.queries.CreateTenant(ctx, params)
	if err != nil {
		return err
	}

	*t = *toDomainTenant(&row)
	return nil
}

func (r *tenantRepository) GetByID(ctx context.Context, id shared.ID) (*shared.Tenant, error) {
	row, err := r.queries.GetTenantByID(ctx, shared.ToPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toDomainTenant(&row), nil
}

func (r *tenantRepository) GetByCode(ctx context.Context, code string) (*shared.Tenant, error) {
	row, err := r.queries.GetTenantByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toDomainTenant(&row), nil
}

func (r *tenantRepository) List(ctx context.Context, limit, offset int32) ([]shared.Tenant, int64, error) {
	total, err := r.queries.CountTenants(ctx)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.ListTenants(ctx, sqlc.ListTenantsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	tenants := make([]shared.Tenant, len(rows))
	for i, row := range rows {
		tenants[i] = *toDomainTenant(&row)
	}

	return tenants, total, nil
}

func (r *tenantRepository) Update(ctx context.Context, t *shared.Tenant) error {
	var settingsJSON []byte
	if t.Settings != nil {
		var err error
		settingsJSON, err = json.Marshal(t.Settings)
		if err != nil {
			return fmt.Errorf("%w: invalid tenant settings json", shared.ErrInvalidInput)
		}
	}

	statusStr := string(t.Status)
	params := sqlc.UpdateTenantParams{
		ID:       shared.ToPgUUID(t.ID),
		Name:     &t.Name,
		Status:   &statusStr,
		Settings: settingsJSON,
	}

	row, err := r.queries.UpdateTenant(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return shared.ErrNotFound
		}
		return err
	}

	*t = *toDomainTenant(&row)
	return nil
}

func toDomainTenant(row *sqlc.Tenant) *shared.Tenant {
	var settings map[string]any
	if len(row.Settings) > 0 {
		_ = json.Unmarshal(row.Settings, &settings)
	}
	if settings == nil {
		settings = make(map[string]any)
	}

	return &shared.Tenant{
		ID:        shared.FromPgUUID(row.ID),
		Code:      row.Code,
		Name:      row.Name,
		Status:    shared.TenantStatus(row.Status),
		Settings:  settings,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
