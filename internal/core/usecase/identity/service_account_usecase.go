package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/identity"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

// Usecase input DTOs
type CreateServiceAccountInput struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	RoleID      *shared.ID `json:"role_id,omitempty"`
}

type UpdateServiceAccountInput struct {
	Name        string                        `json:"name"`
	Description string                        `json:"description,omitempty"`
	RoleID      *shared.ID                    `json:"role_id,omitempty"`
	Status      identity.ServiceAccountStatus `json:"status"`
}

type CreateAPIKeyInput struct {
	Name        string     `json:"name"`
	Scopes      []string   `json:"scopes,omitempty"`
	IPAllowlist []string   `json:"ip_allowlist,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// ServiceAccountUsecase defines application business logic for Service Accounts and API Keys.
type ServiceAccountUsecase interface {
	identity.APIKeyValidator

	CreateServiceAccount(ctx context.Context, input CreateServiceAccountInput) (*identity.ServiceAccount, error)
	GetServiceAccount(ctx context.Context, id shared.ID) (*identity.ServiceAccount, error)
	ListServiceAccounts(ctx context.Context) ([]identity.ServiceAccount, error)
	UpdateServiceAccount(ctx context.Context, id shared.ID, input UpdateServiceAccountInput) (*identity.ServiceAccount, error)
	DeleteServiceAccount(ctx context.Context, id shared.ID) error

	CreateAPIKey(ctx context.Context, serviceAccountID shared.ID, input CreateAPIKeyInput) (*identity.APIKeyWithSecret, error)
	ListAPIKeys(ctx context.Context, serviceAccountID shared.ID) ([]identity.APIKey, error)
	RevokeAPIKey(ctx context.Context, serviceAccountID, keyID shared.ID) error
}

type serviceAccountUsecase struct {
	repo identity.ServiceAccountRepository
}

// NewServiceAccountUsecase constructs a ServiceAccountUsecase.
func NewServiceAccountUsecase(repo identity.ServiceAccountRepository) ServiceAccountUsecase {
	return &serviceAccountUsecase{repo: repo}
}

func (u *serviceAccountUsecase) CreateServiceAccount(ctx context.Context, input CreateServiceAccountInput) (*identity.ServiceAccount, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", shared.ErrInvalidInput)
	}

	id, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	sa := &identity.ServiceAccount{
		ID:          id,
		TenantID:    tenantID,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		RoleID:      input.RoleID,
		Status:      identity.ServiceAccountStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := u.repo.CreateServiceAccount(ctx, sa); err != nil {
		return nil, err
	}
	return sa, nil
}

func (u *serviceAccountUsecase) GetServiceAccount(ctx context.Context, id shared.ID) (*identity.ServiceAccount, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	return u.repo.GetServiceAccountByID(ctx, tenantID, id)
}

func (u *serviceAccountUsecase) ListServiceAccounts(ctx context.Context) ([]identity.ServiceAccount, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	return u.repo.ListServiceAccounts(ctx, tenantID)
}

func (u *serviceAccountUsecase) UpdateServiceAccount(ctx context.Context, id shared.ID, input UpdateServiceAccountInput) (*identity.ServiceAccount, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	sa, err := u.repo.GetServiceAccountByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	if name != "" {
		sa.Name = name
	}
	sa.Description = strings.TrimSpace(input.Description)
	sa.RoleID = input.RoleID

	if input.Status != "" {
		sa.Status = input.Status
	}
	sa.UpdatedAt = time.Now().UTC()

	if err := u.repo.UpdateServiceAccount(ctx, sa); err != nil {
		return nil, err
	}
	return sa, nil
}

func (u *serviceAccountUsecase) DeleteServiceAccount(ctx context.Context, id shared.ID) error {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return err
	}
	return u.repo.DeleteServiceAccount(ctx, tenantID, id)
}

func (u *serviceAccountUsecase) CreateAPIKey(ctx context.Context, serviceAccountID shared.ID, input CreateAPIKeyInput) (*identity.APIKeyWithSecret, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	// Verify service account exists and belongs to tenant
	sa, err := u.repo.GetServiceAccountByID(ctx, tenantID, serviceAccountID)
	if err != nil {
		return nil, err
	}
	if !sa.IsActive() {
		return nil, fmt.Errorf("%w: service account is not active", identity.ErrInvalidServiceAccount)
	}

	keyName := strings.TrimSpace(input.Name)
	if keyName == "" {
		keyName = fmt.Sprintf("%s-key", sa.Name)
	}

	rawKey, prefix, hash, err := identity.GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	keyID, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	apiKey := &identity.APIKey{
		ID:               keyID,
		TenantID:         tenantID,
		ServiceAccountID: serviceAccountID,
		Name:             keyName,
		KeyPrefix:        prefix,
		KeyHash:          hash,
		Scopes:           input.Scopes,
		IPAllowlist:      input.IPAllowlist,
		Status:           identity.APIKeyStatusActive,
		ExpiresAt:        input.ExpiresAt,
		CreatedAt:        now,
	}

	if apiKey.Scopes == nil {
		apiKey.Scopes = make([]string, 0)
	}

	if err := u.repo.CreateAPIKey(ctx, apiKey); err != nil {
		return nil, err
	}

	return &identity.APIKeyWithSecret{
		APIKey:    *apiKey,
		SecretKey: rawKey,
	}, nil
}

func (u *serviceAccountUsecase) ListAPIKeys(ctx context.Context, serviceAccountID shared.ID) ([]identity.APIKey, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	return u.repo.ListAPIKeysByServiceAccount(ctx, tenantID, serviceAccountID)
}

func (u *serviceAccountUsecase) RevokeAPIKey(ctx context.Context, serviceAccountID, keyID shared.ID) error {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return err
	}

	// Verify key belongs to this service account
	key, err := u.repo.GetAPIKeyByID(ctx, tenantID, keyID)
	if err != nil {
		return err
	}
	if key.ServiceAccountID != serviceAccountID {
		return shared.ErrNotFound
	}

	return u.repo.RevokeAPIKey(ctx, tenantID, keyID)
}

func (u *serviceAccountUsecase) ValidateAPIKey(ctx context.Context, rawKey, clientIP string) (*shared.AuthClaims, error) {
	rawKey = strings.TrimSpace(rawKey)
	if !strings.HasPrefix(rawKey, identity.APIKeyPrefix) {
		return nil, fmt.Errorf("%w: invalid key prefix", identity.ErrInvalidAPIKey)
	}

	hash := identity.HashAPIKey(rawKey)
	key, sa, effectivePermissions, err := u.repo.GetAPIKeyByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, fmt.Errorf("%w: key not found", identity.ErrInvalidAPIKey)
		}
		return nil, err
	}

	now := time.Now().UTC()
	if !key.IsActive(now) {
		if key.Status == identity.APIKeyStatusRevoked {
			return nil, identity.ErrAPIKeyRevoked
		}
		if key.ExpiresAt != nil && now.After(*key.ExpiresAt) {
			return nil, identity.ErrAPIKeyExpired
		}
		return nil, identity.ErrInvalidAPIKey
	}

	if !sa.IsActive() {
		return nil, fmt.Errorf("%w: service account status %s", identity.ErrInvalidServiceAccount, sa.Status)
	}

	if !identity.ValidateIP(clientIP, key.IPAllowlist) {
		return nil, identity.ErrIPNotAllowed
	}

	// Record usage asynchronously without blocking the request
	go func(keyID shared.ID, ip string, t time.Time) {
		_ = u.repo.UpdateAPIKeyUsage(context.Background(), keyID, ip, t)
	}(key.ID, clientIP, now)

	var roles []string
	if sa.RoleCode != "" {
		roles = append(roles, sa.RoleCode)
	}

	claims := &shared.AuthClaims{
		UserID:           sa.ID,
		TenantID:         sa.TenantID,
		Email:            fmt.Sprintf("%s@serviceaccount.mergiate.local", sa.Name),
		Name:             sa.Name,
		Roles:            roles,
		Permissions:      effectivePermissions,
		ActorType:        shared.ActorTypeAPIKey,
		ServiceAccountID: &sa.ID,
		APIKeyID:         &key.ID,
	}

	return claims, nil
}
