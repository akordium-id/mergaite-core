package identity_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akordium-id/mergiate-core/internal/core/domain/identity"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	identityusecase "github.com/akordium-id/mergiate-core/internal/core/usecase/identity"
)

type mockServiceAccountRepo struct {
	accounts map[string]*identity.ServiceAccount
	keys     map[string]*identity.APIKey     // keyed by ID.String()
	keysHash map[string]*identity.APIKey     // keyed by KeyHash
	keyPerms map[string][]string             // keyed by KeyHash
	lastUsed map[string]string               // keyID -> ip
}

func newMockSARepo() *mockServiceAccountRepo {
	return &mockServiceAccountRepo{
		accounts: make(map[string]*identity.ServiceAccount),
		keys:     make(map[string]*identity.APIKey),
		keysHash: make(map[string]*identity.APIKey),
		keyPerms: make(map[string][]string),
		lastUsed: make(map[string]string),
	}
}

func (m *mockServiceAccountRepo) CreateServiceAccount(ctx context.Context, sa *identity.ServiceAccount) error {
	m.accounts[sa.ID.String()] = sa
	return nil
}

func (m *mockServiceAccountRepo) GetServiceAccountByID(ctx context.Context, tenantID, id shared.ID) (*identity.ServiceAccount, error) {
	sa, ok := m.accounts[id.String()]
	if !ok || sa.TenantID != tenantID {
		return nil, shared.ErrNotFound
	}
	return sa, nil
}

func (m *mockServiceAccountRepo) ListServiceAccounts(ctx context.Context, tenantID shared.ID) ([]identity.ServiceAccount, error) {
	var list []identity.ServiceAccount
	for _, sa := range m.accounts {
		if sa.TenantID == tenantID {
			list = append(list, *sa)
		}
	}
	return list, nil
}

func (m *mockServiceAccountRepo) UpdateServiceAccount(ctx context.Context, sa *identity.ServiceAccount) error {
	if _, ok := m.accounts[sa.ID.String()]; !ok {
		return shared.ErrNotFound
	}
	m.accounts[sa.ID.String()] = sa
	return nil
}

func (m *mockServiceAccountRepo) DeleteServiceAccount(ctx context.Context, tenantID, id shared.ID) error {
	if sa, ok := m.accounts[id.String()]; ok && sa.TenantID == tenantID {
		delete(m.accounts, id.String())
		return nil
	}
	return shared.ErrNotFound
}

func (m *mockServiceAccountRepo) CreateAPIKey(ctx context.Context, key *identity.APIKey) error {
	m.keys[key.ID.String()] = key
	m.keysHash[key.KeyHash] = key
	return nil
}

func (m *mockServiceAccountRepo) GetAPIKeyByID(ctx context.Context, tenantID, id shared.ID) (*identity.APIKey, error) {
	k, ok := m.keys[id.String()]
	if !ok || k.TenantID != tenantID {
		return nil, shared.ErrNotFound
	}
	return k, nil
}

func (m *mockServiceAccountRepo) GetAPIKeyByHash(ctx context.Context, keyHash string) (*identity.APIKey, *identity.ServiceAccount, []string, error) {
	k, ok := m.keysHash[keyHash]
	if !ok {
		return nil, nil, nil, shared.ErrNotFound
	}
	sa, ok := m.accounts[k.ServiceAccountID.String()]
	if !ok {
		return nil, nil, nil, shared.ErrNotFound
	}

	perms := k.Scopes
	if len(perms) == 0 {
		perms = m.keyPerms[keyHash]
	}
	return k, sa, perms, nil
}

func (m *mockServiceAccountRepo) ListAPIKeysByServiceAccount(ctx context.Context, tenantID, serviceAccountID shared.ID) ([]identity.APIKey, error) {
	var list []identity.APIKey
	for _, k := range m.keys {
		if k.TenantID == tenantID && k.ServiceAccountID == serviceAccountID {
			list = append(list, *k)
		}
	}
	return list, nil
}

func (m *mockServiceAccountRepo) RevokeAPIKey(ctx context.Context, tenantID, id shared.ID) error {
	k, ok := m.keys[id.String()]
	if !ok || k.TenantID != tenantID {
		return shared.ErrNotFound
	}
	k.Status = identity.APIKeyStatusRevoked
	now := time.Now().UTC()
	k.RevokedAt = &now
	return nil
}

func (m *mockServiceAccountRepo) UpdateAPIKeyUsage(ctx context.Context, id shared.ID, clientIP string, usedAt time.Time) error {
	m.lastUsed[id.String()] = clientIP
	return nil
}

func TestServiceAccountUsecase_LifecycleAndAPIKeys(t *testing.T) {
	repo := newMockSARepo()
	uc := identityusecase.NewServiceAccountUsecase(repo)

	tenantID, _ := shared.NewID()
	ctx := shared.WithTenantID(context.Background(), tenantID)

	// 1. Create Service Account
	sa, err := uc.CreateServiceAccount(ctx, identityusecase.CreateServiceAccountInput{
		Name:        "GPS Ingest Daemon",
		Description: "Background worker for IoT telematics",
	})
	require.NoError(t, err)
	assert.Equal(t, "GPS Ingest Daemon", sa.Name)
	assert.Equal(t, identity.ServiceAccountStatusActive, sa.Status)

	// 2. Generate API Key with Scopes and IP Allowlist
	keyRes, err := uc.CreateAPIKey(ctx, sa.ID, identityusecase.CreateAPIKeyInput{
		Name:        "Primary Ingest Key",
		Scopes:      []string{"document:create", "product:read"},
		IPAllowlist: []string{"192.168.1.0/24", "10.0.0.5"},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, keyRes.SecretKey)
	assert.Contains(t, keyRes.SecretKey, identity.APIKeyPrefix)
	assert.Equal(t, "Primary Ingest Key", keyRes.Name)
	assert.Equal(t, identity.APIKeyStatusActive, keyRes.Status)

	// 3. Validate API Key from allowed IP
	claims, err := uc.ValidateAPIKey(ctx, keyRes.SecretKey, "192.168.1.50")
	require.NoError(t, err)
	assert.Equal(t, shared.ActorTypeAPIKey, claims.ActorType)
	assert.Equal(t, sa.ID, claims.UserID)
	assert.Equal(t, sa.ID, *claims.ServiceAccountID)
	assert.Equal(t, keyRes.ID, *claims.APIKeyID)
	assert.Equal(t, []string{"document:create", "product:read"}, claims.Permissions)

	// 4. Validate API Key from disallowed IP
	_, err = uc.ValidateAPIKey(ctx, keyRes.SecretKey, "203.0.113.195")
	require.ErrorIs(t, err, identity.ErrIPNotAllowed)

	// 5. Revoke API Key
	err = uc.RevokeAPIKey(ctx, sa.ID, keyRes.ID)
	require.NoError(t, err)

	// 6. Validate Revoked Key should fail
	_, err = uc.ValidateAPIKey(ctx, keyRes.SecretKey, "192.168.1.50")
	require.ErrorIs(t, err, identity.ErrAPIKeyRevoked)
}

func TestServiceAccountUsecase_ExpiredKey(t *testing.T) {
	repo := newMockSARepo()
	uc := identityusecase.NewServiceAccountUsecase(repo)

	tenantID, _ := shared.NewID()
	ctx := shared.WithTenantID(context.Background(), tenantID)

	sa, err := uc.CreateServiceAccount(ctx, identityusecase.CreateServiceAccountInput{
		Name: "Temporary Sync Bot",
	})
	require.NoError(t, err)

	// Expired yesterday
	yesterday := time.Now().UTC().Add(-24 * time.Hour)
	keyRes, err := uc.CreateAPIKey(ctx, sa.ID, identityusecase.CreateAPIKeyInput{
		Name:      "Expiring Key",
		ExpiresAt: &yesterday,
	})
	require.NoError(t, err)

	_, err = uc.ValidateAPIKey(ctx, keyRes.SecretKey, "127.0.0.1")
	require.ErrorIs(t, err, identity.ErrAPIKeyExpired)
}

func TestServiceAccountUsecase_SuspendedAccount(t *testing.T) {
	repo := newMockSARepo()
	uc := identityusecase.NewServiceAccountUsecase(repo)

	tenantID, _ := shared.NewID()
	ctx := shared.WithTenantID(context.Background(), tenantID)

	sa, err := uc.CreateServiceAccount(ctx, identityusecase.CreateServiceAccountInput{
		Name: "External Connector",
	})
	require.NoError(t, err)

	keyRes, err := uc.CreateAPIKey(ctx, sa.ID, identityusecase.CreateAPIKeyInput{
		Name: "Valid Key",
	})
	require.NoError(t, err)

	// Suspend the service account
	_, err = uc.UpdateServiceAccount(ctx, sa.ID, identityusecase.UpdateServiceAccountInput{
		Status: identity.ServiceAccountStatusSuspended,
	})
	require.NoError(t, err)

	_, err = uc.ValidateAPIKey(ctx, keyRes.SecretKey, "127.0.0.1")
	require.ErrorIs(t, err, identity.ErrInvalidServiceAccount)
}
