package tenant_test

import (
	"context"
	"errors"
	"testing"

	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
	"github.com/akordium-id/mergaite-core/internal/core/usecase/tenant"
)

type mockTenantRepo struct {
	tenants map[shared.ID]*shared.Tenant
	byCode  map[string]*shared.Tenant
}

func newMockRepo() *mockTenantRepo {
	return &mockTenantRepo{
		tenants: make(map[shared.ID]*shared.Tenant),
		byCode:  make(map[string]*shared.Tenant),
	}
}

func (m *mockTenantRepo) Create(ctx context.Context, t *shared.Tenant) error {
	m.tenants[t.ID] = t
	m.byCode[t.Code] = t
	return nil
}

func (m *mockTenantRepo) GetByID(ctx context.Context, id shared.ID) (*shared.Tenant, error) {
	t, ok := m.tenants[id]
	if !ok {
		return nil, shared.ErrNotFound
	}
	return t, nil
}

func (m *mockTenantRepo) GetByCode(ctx context.Context, code string) (*shared.Tenant, error) {
	t, ok := m.byCode[code]
	if !ok {
		return nil, shared.ErrNotFound
	}
	return t, nil
}

func (m *mockTenantRepo) List(ctx context.Context, limit, offset int32) ([]shared.Tenant, int64, error) {
	res := make([]shared.Tenant, 0, len(m.tenants))
	for _, t := range m.tenants {
		res = append(res, *t)
	}
	return res, int64(len(m.tenants)), nil
}

func (m *mockTenantRepo) Update(ctx context.Context, t *shared.Tenant) error {
	m.tenants[t.ID] = t
	m.byCode[t.Code] = t
	return nil
}

func TestTenantUsecase_Create(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	uc := tenant.NewUsecase(repo)

	// Valid creation
	created, err := uc.CreateTenant(ctx, tenant.CreateTenantCommand{
		Code: "akordium",
		Name: "Akordium Lab",
	})
	if err != nil {
		t.Fatalf("unexpected error creating tenant: %v", err)
	}
	if created.ID == shared.NilID() {
		t.Fatal("expected non-nil tenant ID")
	}
	if created.Code != "akordium" {
		t.Errorf("expected code 'akordium', got '%s'", created.Code)
	}

	// Duplicate creation
	_, err = uc.CreateTenant(ctx, tenant.CreateTenantCommand{
		Code: "akordium",
		Name: "Another Akordium",
	})
	if !errors.Is(err, shared.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestTenantUsecase_Get(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	uc := tenant.NewUsecase(repo)

	created, _ := uc.CreateTenant(ctx, tenant.CreateTenantCommand{
		Code: "pt-sumber",
		Name: "PT Sumber Makmur",
	})

	found, err := uc.GetTenant(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.Name != "PT Sumber Makmur" {
		t.Errorf("expected name 'PT Sumber Makmur', got '%s'", found.Name)
	}

	// Not found
	_, err = uc.GetTenant(ctx, shared.MustNewID())
	if !errors.Is(err, shared.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
