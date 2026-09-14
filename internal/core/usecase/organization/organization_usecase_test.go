package organization_test

import (
	"context"
	"errors"
	"testing"

	"github.com/akordium-id/mergaite-core/internal/core/domain/organization"
	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
	organizationuc "github.com/akordium-id/mergaite-core/internal/core/usecase/organization"
)

type mockOrgRepo struct {
	orgs   map[shared.ID]*organization.Organization
	byCode map[string]*organization.Organization
}

func newMockOrgRepo() *mockOrgRepo {
	return &mockOrgRepo{
		orgs:   make(map[shared.ID]*organization.Organization),
		byCode: make(map[string]*organization.Organization),
	}
}

func (m *mockOrgRepo) Create(ctx context.Context, org *organization.Organization) error {
	m.orgs[org.ID] = org
	m.byCode[org.Code] = org
	return nil
}

func (m *mockOrgRepo) GetByID(ctx context.Context, tenantID, id shared.ID) (*organization.Organization, error) {
	org, ok := m.orgs[id]
	if !ok || org.TenantID != tenantID {
		return nil, shared.ErrNotFound
	}
	return org, nil
}

func (m *mockOrgRepo) GetByCode(ctx context.Context, tenantID shared.ID, code string) (*organization.Organization, error) {
	org, ok := m.byCode[code]
	if !ok || org.TenantID != tenantID {
		return nil, shared.ErrNotFound
	}
	return org, nil
}

func (m *mockOrgRepo) ListByTenant(ctx context.Context, tenantID shared.ID) ([]organization.Organization, error) {
	var res []organization.Organization
	for _, o := range m.orgs {
		if o.TenantID == tenantID {
			res = append(res, *o)
		}
	}
	return res, nil
}

func (m *mockOrgRepo) ListChildren(ctx context.Context, tenantID, parentID shared.ID) ([]organization.Organization, error) {
	var res []organization.Organization
	for _, o := range m.orgs {
		if o.TenantID == tenantID && o.ParentID != nil && *o.ParentID == parentID {
			res = append(res, *o)
		}
	}
	return res, nil
}

func (m *mockOrgRepo) Update(ctx context.Context, org *organization.Organization) error {
	m.orgs[org.ID] = org
	m.byCode[org.Code] = org
	return nil
}

func TestOrganizationUsecase_CreateAndHierarchy(t *testing.T) {
	repo := newMockOrgRepo()
	uc := organizationuc.NewUsecase(repo)

	tenantID := shared.MustNewID()
	ctx := shared.WithTenantID(context.Background(), tenantID)

	// 1. Create Root Org (Holding)
	holding, err := uc.Create(ctx, organizationuc.CreateCommand{
		Code: "holding-akordium",
		Name: "PT Akordium Group",
		Type: organization.TypeCompany,
	})
	if err != nil {
		t.Fatalf("unexpected error creating holding: %v", err)
	}

	// 2. Duplicate Code
	_, err = uc.Create(ctx, organizationuc.CreateCommand{
		Code: "holding-akordium",
		Name: "Duplicate Holding",
	})
	if !errors.Is(err, shared.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}

	// 3. Create Child Org (Branch)
	branch, err := uc.Create(ctx, organizationuc.CreateCommand{
		ParentID: &holding.ID,
		Code:     "branch-sby",
		Name:     "Surabaya Branch",
		Type:     organization.TypeBranch,
	})
	if err != nil {
		t.Fatalf("unexpected error creating branch: %v", err)
	}
	if branch.ParentID == nil || *branch.ParentID != holding.ID {
		t.Fatalf("expected parent ID %v, got %v", holding.ID, branch.ParentID)
	}

	// 4. Update Holding to make branch its parent (Circular reference!)
	_, err = uc.Update(ctx, organizationuc.UpdateCommand{
		ID:       holding.ID,
		ParentID: &branch.ID,
	})
	if !errors.Is(err, shared.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for circular reference, got %v", err)
	}
}
