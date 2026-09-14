package tenant

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/akordium-id/mergaite-core/internal/core/domain/repository"
	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

type CreateTenantCommand struct {
	Code     string         `json:"code"`
	Name     string         `json:"name"`
	Settings map[string]any `json:"settings"`
}

type UpdateTenantCommand struct {
	ID       shared.ID      `json:"id"`
	Name     string         `json:"name"`
	Status   string         `json:"status"`
	Settings map[string]any `json:"settings"`
}

type TenantListResult struct {
	Items    []shared.Tenant `json:"items"`
	Total    int64           `json:"total"`
	Page     int32           `json:"page"`
	PageSize int32           `json:"page_size"`
}

type Usecase interface {
	CreateTenant(ctx context.Context, cmd CreateTenantCommand) (*shared.Tenant, error)
	GetTenant(ctx context.Context, id shared.ID) (*shared.Tenant, error)
	GetTenantByCode(ctx context.Context, code string) (*shared.Tenant, error)
	ListTenants(ctx context.Context, page, pageSize int32) (*TenantListResult, error)
	UpdateTenant(ctx context.Context, cmd UpdateTenantCommand) (*shared.Tenant, error)
}

type usecase struct {
	tenantRepo repository.TenantRepository
}

func NewUsecase(tenantRepo repository.TenantRepository) Usecase {
	return &usecase{
		tenantRepo: tenantRepo,
	}
}

func (u *usecase) CreateTenant(ctx context.Context, cmd CreateTenantCommand) (*shared.Tenant, error) {
	code := strings.ToLower(strings.TrimSpace(cmd.Code))
	name := strings.TrimSpace(cmd.Name)

	if code == "" || name == "" {
		return nil, fmt.Errorf("%w: code and name are required", shared.ErrInvalidInput)
	}

	// Check existing tenant with same code
	existing, err := u.tenantRepo.GetByCode(ctx, code)
	if err != nil && err != shared.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: tenant code '%s' already exists", shared.ErrAlreadyExists, code)
	}

	id, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	settings := cmd.Settings
	if settings == nil {
		settings = make(map[string]any)
	}

	tenant := &shared.Tenant{
		ID:        id,
		Code:      code,
		Name:      name,
		Status:    shared.TenantStatusActive,
		Settings:  settings,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := u.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

func (u *usecase) GetTenant(ctx context.Context, id shared.ID) (*shared.Tenant, error) {
	return u.tenantRepo.GetByID(ctx, id)
}

func (u *usecase) GetTenantByCode(ctx context.Context, code string) (*shared.Tenant, error) {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" {
		return nil, fmt.Errorf("%w: code is required", shared.ErrInvalidInput)
	}
	return u.tenantRepo.GetByCode(ctx, code)
}

func (u *usecase) ListTenants(ctx context.Context, page, pageSize int32) (*TenantListResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	items, total, err := u.tenantRepo.List(ctx, pageSize, offset)
	if err != nil {
		return nil, err
	}

	return &TenantListResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (u *usecase) UpdateTenant(ctx context.Context, cmd UpdateTenantCommand) (*shared.Tenant, error) {
	tenant, err := u.tenantRepo.GetByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}

	if cmd.Name != "" {
		tenant.Name = strings.TrimSpace(cmd.Name)
	}
	if cmd.Status != "" {
		tenant.Status = shared.TenantStatus(cmd.Status)
	}
	if cmd.Settings != nil {
		tenant.Settings = cmd.Settings
	}
	tenant.UpdatedAt = time.Now().UTC()

	if err := u.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}
