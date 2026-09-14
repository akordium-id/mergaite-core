package organization

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/organization"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

type CreateCommand struct {
	ParentID  *shared.ID        `json:"parent_id,omitempty"`
	Code      string            `json:"code"`
	Name      string            `json:"name"`
	LegalName string            `json:"legal_name,omitempty"`
	Type      organization.Type `json:"type"`
	Settings  map[string]any    `json:"settings,omitempty"`
}

type UpdateCommand struct {
	ID        shared.ID          `json:"id"`
	ParentID  *shared.ID         `json:"parent_id,omitempty"`
	Name      string             `json:"name,omitempty"`
	LegalName string             `json:"legal_name,omitempty"`
	Type      organization.Type  `json:"type,omitempty"`
	Status    organization.Status`json:"status,omitempty"`
	Settings  map[string]any     `json:"settings,omitempty"`
}

type Usecase interface {
	Create(ctx context.Context, cmd CreateCommand) (*organization.Organization, error)
	GetByID(ctx context.Context, id shared.ID) (*organization.Organization, error)
	GetByCode(ctx context.Context, code string) (*organization.Organization, error)
	List(ctx context.Context) ([]organization.Organization, error)
	GetTree(ctx context.Context) ([]organization.Organization, error)
	Update(ctx context.Context, cmd UpdateCommand) (*organization.Organization, error)
}

type usecase struct {
	repo organization.Repository
}

func NewUsecase(repo organization.Repository) Usecase {
	return &usecase{repo: repo}
}

func (u *usecase) Create(ctx context.Context, cmd CreateCommand) (*organization.Organization, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	code := strings.ToLower(strings.TrimSpace(cmd.Code))
	name := strings.TrimSpace(cmd.Name)
	if code == "" || name == "" {
		return nil, fmt.Errorf("%w: code and name are required", shared.ErrInvalidInput)
	}

	orgType := cmd.Type
	if orgType == "" {
		orgType = organization.TypeCompany
	}

	// Verify code uniqueness within tenant
	existing, err := u.repo.GetByCode(ctx, tenantID, code)
	if err != nil && err != shared.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: organization code '%s' already exists for this tenant", shared.ErrAlreadyExists, code)
	}

	// Verify parent existence if provided
	if cmd.ParentID != nil && *cmd.ParentID != shared.NilID() {
		parent, err := u.repo.GetByID(ctx, tenantID, *cmd.ParentID)
		if err != nil {
			return nil, fmt.Errorf("parent organization: %w", err)
		}
		if parent.Status != organization.StatusActive {
			return nil, fmt.Errorf("%w: parent organization is not active", shared.ErrInvalidInput)
		}
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

	org := &organization.Organization{
		ID:        id,
		TenantID:  tenantID,
		ParentID:  cmd.ParentID,
		Code:      code,
		Name:      name,
		LegalName: strings.TrimSpace(cmd.LegalName),
		Type:      orgType,
		Status:    organization.StatusActive,
		Settings:  settings,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := u.repo.Create(ctx, org); err != nil {
		return nil, err
	}

	return org, nil
}

func (u *usecase) GetByID(ctx context.Context, id shared.ID) (*organization.Organization, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	return u.repo.GetByID(ctx, tenantID, id)
}

func (u *usecase) GetByCode(ctx context.Context, code string) (*organization.Organization, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	return u.repo.GetByCode(ctx, tenantID, strings.ToLower(strings.TrimSpace(code)))
}

func (u *usecase) List(ctx context.Context) ([]organization.Organization, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	return u.repo.ListByTenant(ctx, tenantID)
}

func (u *usecase) GetTree(ctx context.Context) ([]organization.Organization, error) {
	orgs, err := u.List(ctx)
	if err != nil {
		return nil, err
	}
	return organization.BuildTree(orgs), nil
}

func (u *usecase) Update(ctx context.Context, cmd UpdateCommand) (*organization.Organization, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	org, err := u.repo.GetByID(ctx, tenantID, cmd.ID)
	if err != nil {
		return nil, err
	}

	// Invariant check for hierarchy change (circular reference prevention)
	if cmd.ParentID != nil && (org.ParentID == nil || *cmd.ParentID != *org.ParentID) {
		allOrgs, err := u.repo.ListByTenant(ctx, tenantID)
		if err != nil {
			return nil, err
		}

		if err := organization.CheckCircularReference(allOrgs, org.ID, cmd.ParentID); err != nil {
			return nil, err
		}
		org.ParentID = cmd.ParentID
	}

	if cmd.Name != "" {
		org.Name = strings.TrimSpace(cmd.Name)
	}
	if cmd.LegalName != "" {
		org.LegalName = strings.TrimSpace(cmd.LegalName)
	}
	if cmd.Type != "" {
		org.Type = cmd.Type
	}
	if cmd.Status != "" {
		org.Status = cmd.Status
	}
	if cmd.Settings != nil {
		org.Settings = cmd.Settings
	}
	org.UpdatedAt = time.Now().UTC()

	if err := u.repo.Update(ctx, org); err != nil {
		return nil, err
	}

	return org, nil
}
