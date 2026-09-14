package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/mergiate-core/internal/core/domain/identity"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

type serviceAccountRepository struct {
	pool *pgxpool.Pool
}

// NewServiceAccountRepository constructs a PostgreSQL repository for Service Accounts and API Keys.
func NewServiceAccountRepository(pool *pgxpool.Pool) identity.ServiceAccountRepository {
	return &serviceAccountRepository{pool: pool}
}

// ----------------------------------------------------------------------------
// Service Account Operations
// ----------------------------------------------------------------------------

func (r *serviceAccountRepository) CreateServiceAccount(ctx context.Context, sa *identity.ServiceAccount) error {
	query := `
		INSERT INTO service_accounts (
			id, tenant_id, name, description, role_id, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	var roleUUID *pgtype.UUID
	if sa.RoleID != nil && *sa.RoleID != shared.NilID() {
		u := shared.ToPgUUID(*sa.RoleID)
		roleUUID = &u
	}

	_, err := r.pool.Exec(ctx, query,
		shared.ToPgUUID(sa.ID),
		shared.ToPgUUID(sa.TenantID),
		sa.Name,
		sa.Description,
		roleUUID,
		string(sa.Status),
		sa.CreatedAt,
		sa.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert service account: %w", err)
	}
	return nil
}

func (r *serviceAccountRepository) GetServiceAccountByID(ctx context.Context, tenantID, id shared.ID) (*identity.ServiceAccount, error) {
	query := `
		SELECT 
			s.id, s.tenant_id, s.name, s.description, s.role_id, 
			COALESCE(ro.code, ''), COALESCE(ro.name, ''),
			s.status, s.created_at, s.updated_at
		FROM service_accounts s
		LEFT JOIN roles ro ON ro.id = s.role_id
		WHERE s.tenant_id = $1 AND s.id = $2
	`
	var (
		saID, tID pgtype.UUID
		roleID    pgtype.UUID
		desc      *string
		roleCode  string
		roleName  string
		status    string
		createdAt time.Time
		updatedAt time.Time
		name      string
	)

	err := r.pool.QueryRow(ctx, query, shared.ToPgUUID(tenantID), shared.ToPgUUID(id)).Scan(
		&saID, &tID, &name, &desc, &roleID,
		&roleCode, &roleName,
		&status, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get service account: %w", err)
	}

	sa := &identity.ServiceAccount{
		ID:        shared.FromPgUUID(saID),
		TenantID:  shared.FromPgUUID(tID),
		Name:      name,
		Status:    identity.ServiceAccountStatus(status),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	if desc != nil {
		sa.Description = *desc
	}
	if roleID.Valid {
		rid := shared.FromPgUUID(roleID)
		sa.RoleID = &rid
		sa.RoleCode = roleCode
		sa.RoleName = roleName
	}
	return sa, nil
}

func (r *serviceAccountRepository) ListServiceAccounts(ctx context.Context, tenantID shared.ID) ([]identity.ServiceAccount, error) {
	query := `
		SELECT 
			s.id, s.tenant_id, s.name, s.description, s.role_id, 
			COALESCE(ro.code, ''), COALESCE(ro.name, ''),
			s.status, s.created_at, s.updated_at
		FROM service_accounts s
		LEFT JOIN roles ro ON ro.id = s.role_id
		WHERE s.tenant_id = $1
		ORDER BY s.created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, shared.ToPgUUID(tenantID))
	if err != nil {
		return nil, fmt.Errorf("failed to list service accounts: %w", err)
	}
	defer rows.Close()

	var list []identity.ServiceAccount
	for rows.Next() {
		var (
			saID, tID pgtype.UUID
			roleID    pgtype.UUID
			desc      *string
			roleCode  string
			roleName  string
			status    string
			createdAt time.Time
			updatedAt time.Time
			name      string
		)
		if err := rows.Scan(&saID, &tID, &name, &desc, &roleID, &roleCode, &roleName, &status, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan service account row: %w", err)
		}

		sa := identity.ServiceAccount{
			ID:        shared.FromPgUUID(saID),
			TenantID:  shared.FromPgUUID(tID),
			Name:      name,
			Status:    identity.ServiceAccountStatus(status),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}
		if desc != nil {
			sa.Description = *desc
		}
		if roleID.Valid {
			rid := shared.FromPgUUID(roleID)
			sa.RoleID = &rid
			sa.RoleCode = roleCode
			sa.RoleName = roleName
		}
		list = append(list, sa)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading service accounts: %w", err)
	}
	return list, nil
}

func (r *serviceAccountRepository) UpdateServiceAccount(ctx context.Context, sa *identity.ServiceAccount) error {
	query := `
		UPDATE service_accounts
		SET name = $1, description = $2, role_id = $3, status = $4, updated_at = $5
		WHERE tenant_id = $6 AND id = $7
	`
	var roleUUID *pgtype.UUID
	if sa.RoleID != nil && *sa.RoleID != shared.NilID() {
		u := shared.ToPgUUID(*sa.RoleID)
		roleUUID = &u
	}

	cmd, err := r.pool.Exec(ctx, query,
		sa.Name,
		sa.Description,
		roleUUID,
		string(sa.Status),
		sa.UpdatedAt,
		shared.ToPgUUID(sa.TenantID),
		shared.ToPgUUID(sa.ID),
	)
	if err != nil {
		return fmt.Errorf("failed to update service account: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return shared.ErrNotFound
	}
	return nil
}

func (r *serviceAccountRepository) DeleteServiceAccount(ctx context.Context, tenantID, id shared.ID) error {
	query := `DELETE FROM service_accounts WHERE tenant_id = $1 AND id = $2`
	cmd, err := r.pool.Exec(ctx, query, shared.ToPgUUID(tenantID), shared.ToPgUUID(id))
	if err != nil {
		return fmt.Errorf("failed to delete service account: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// ----------------------------------------------------------------------------
// API Key Operations
// ----------------------------------------------------------------------------

func (r *serviceAccountRepository) CreateAPIKey(ctx context.Context, key *identity.APIKey) error {
	query := `
		INSERT INTO api_keys (
			id, tenant_id, service_account_id, name, key_prefix, key_hash, scopes, ip_allowlist, status, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	scopesJSON, err := json.Marshal(key.Scopes)
	if err != nil {
		return fmt.Errorf("failed to marshal scopes: %w", err)
	}

	var ipJSON []byte
	if key.IPAllowlist != nil {
		ipJSON, err = json.Marshal(key.IPAllowlist)
		if err != nil {
			return fmt.Errorf("failed to marshal ip allowlist: %w", err)
		}
	}

	_, err = r.pool.Exec(ctx, query,
		shared.ToPgUUID(key.ID),
		shared.ToPgUUID(key.TenantID),
		shared.ToPgUUID(key.ServiceAccountID),
		key.Name,
		key.KeyPrefix,
		key.KeyHash,
		scopesJSON,
		ipJSON,
		string(key.Status),
		key.ExpiresAt,
		key.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert api key: %w", err)
	}
	return nil
}

func (r *serviceAccountRepository) GetAPIKeyByID(ctx context.Context, tenantID, id shared.ID) (*identity.APIKey, error) {
	query := `
		SELECT 
			id, tenant_id, service_account_id, name, key_prefix, scopes, ip_allowlist,
			status, expires_at, last_used_at, last_used_ip, created_at, revoked_at
		FROM api_keys
		WHERE tenant_id = $1 AND id = $2
	`
	var (
		kID, tID, saID   pgtype.UUID
		name, prefix     string
		scopesRaw, ipRaw []byte
		status           string
		expiresAt        *time.Time
		lastUsedAt       *time.Time
		lastUsedIP       *string
		createdAt        time.Time
		revokedAt        *time.Time
	)

	err := r.pool.QueryRow(ctx, query, shared.ToPgUUID(tenantID), shared.ToPgUUID(id)).Scan(
		&kID, &tID, &saID, &name, &prefix, &scopesRaw, &ipRaw,
		&status, &expiresAt, &lastUsedAt, &lastUsedIP, &createdAt, &revokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}

	var scopes []string
	if len(scopesRaw) > 0 {
		_ = json.Unmarshal(scopesRaw, &scopes)
	}

	var ipAllowlist []string
	if len(ipRaw) > 0 {
		_ = json.Unmarshal(ipRaw, &ipAllowlist)
	}

	return &identity.APIKey{
		ID:               shared.FromPgUUID(kID),
		TenantID:         shared.FromPgUUID(tID),
		ServiceAccountID: shared.FromPgUUID(saID),
		Name:             name,
		KeyPrefix:        prefix,
		Scopes:           scopes,
		IPAllowlist:      ipAllowlist,
		Status:           identity.APIKeyStatus(status),
		ExpiresAt:        expiresAt,
		LastUsedAt:       lastUsedAt,
		LastUsedIP:       lastUsedIP,
		CreatedAt:        createdAt,
		RevokedAt:        revokedAt,
	}, nil
}

func (r *serviceAccountRepository) GetAPIKeyByHash(ctx context.Context, keyHash string) (*identity.APIKey, *identity.ServiceAccount, []string, error) {
	query := `
		SELECT 
			k.id, k.tenant_id, k.service_account_id, k.name, k.key_prefix, k.scopes, k.ip_allowlist,
			k.status, k.expires_at, k.last_used_at, k.last_used_ip, k.created_at, k.revoked_at,
			s.id, s.tenant_id, s.name, s.description, s.role_id, s.status, s.created_at, s.updated_at,
			COALESCE(
				(
					SELECT json_agg(p.code)
					FROM role_permissions rp
					JOIN permissions p ON p.id = rp.permission_id
					WHERE rp.role_id = s.role_id
				),
				'[]'::json
			) AS role_permissions
		FROM api_keys k
		JOIN service_accounts s ON s.id = k.service_account_id
		WHERE k.key_hash = $1
	`
	var (
		kID, tID, saID   pgtype.UUID
		name, prefix     string
		scopesRaw, ipRaw []byte
		status           string
		expiresAt        *time.Time
		lastUsedAt       *time.Time
		lastUsedIP       *string
		createdAt        time.Time
		revokedAt        *time.Time

		saUUID, saTenantUUID pgtype.UUID
		saName               string
		saDesc               *string
		saRoleUUID           pgtype.UUID
		saStatus             string
		saCreatedAt          time.Time
		saUpdatedAt          time.Time
		rolePermsRaw         []byte
	)

	err := r.pool.QueryRow(ctx, query, keyHash).Scan(
		&kID, &tID, &saID, &name, &prefix, &scopesRaw, &ipRaw,
		&status, &expiresAt, &lastUsedAt, &lastUsedIP, &createdAt, &revokedAt,
		&saUUID, &saTenantUUID, &saName, &saDesc, &saRoleUUID, &saStatus, &saCreatedAt, &saUpdatedAt,
		&rolePermsRaw,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil, shared.ErrNotFound
		}
		return nil, nil, nil, fmt.Errorf("failed to lookup api key: %w", err)
	}

	var scopes []string
	if len(scopesRaw) > 0 {
		_ = json.Unmarshal(scopesRaw, &scopes)
	}

	var ipAllowlist []string
	if len(ipRaw) > 0 {
		_ = json.Unmarshal(ipRaw, &ipAllowlist)
	}

	var rolePerms []string
	if len(rolePermsRaw) > 0 {
		_ = json.Unmarshal(rolePermsRaw, &rolePerms)
	}

	apiKey := &identity.APIKey{
		ID:               shared.FromPgUUID(kID),
		TenantID:         shared.FromPgUUID(tID),
		ServiceAccountID: shared.FromPgUUID(saID),
		Name:             name,
		KeyPrefix:        prefix,
		Scopes:           scopes,
		IPAllowlist:      ipAllowlist,
		Status:           identity.APIKeyStatus(status),
		ExpiresAt:        expiresAt,
		LastUsedAt:       lastUsedAt,
		LastUsedIP:       lastUsedIP,
		CreatedAt:        createdAt,
		RevokedAt:        revokedAt,
	}

	sa := &identity.ServiceAccount{
		ID:        shared.FromPgUUID(saUUID),
		TenantID:  shared.FromPgUUID(saTenantUUID),
		Name:      saName,
		Status:    identity.ServiceAccountStatus(saStatus),
		CreatedAt: saCreatedAt,
		UpdatedAt: saUpdatedAt,
	}
	if saDesc != nil {
		sa.Description = *saDesc
	}
	if saRoleUUID.Valid {
		rid := shared.FromPgUUID(saRoleUUID)
		sa.RoleID = &rid
	}

	// Resolve effective permissions:
	// If the API key defines explicit custom scopes, use them.
	// Otherwise, inherit the service account's role permissions.
	var effectivePermissions []string
	if len(scopes) > 0 {
		effectivePermissions = scopes
	} else {
		effectivePermissions = rolePerms
	}

	return apiKey, sa, effectivePermissions, nil
}

func (r *serviceAccountRepository) ListAPIKeysByServiceAccount(ctx context.Context, tenantID, serviceAccountID shared.ID) ([]identity.APIKey, error) {
	query := `
		SELECT 
			id, tenant_id, service_account_id, name, key_prefix, scopes, ip_allowlist,
			status, expires_at, last_used_at, last_used_ip, created_at, revoked_at
		FROM api_keys
		WHERE tenant_id = $1 AND service_account_id = $2
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, shared.ToPgUUID(tenantID), shared.ToPgUUID(serviceAccountID))
	if err != nil {
		return nil, fmt.Errorf("failed to list api keys: %w", err)
	}
	defer rows.Close()

	var list []identity.APIKey
	for rows.Next() {
		var (
			kID, tID, saID   pgtype.UUID
			name, prefix     string
			scopesRaw, ipRaw []byte
			status           string
			expiresAt        *time.Time
			lastUsedAt       *time.Time
			lastUsedIP       *string
			createdAt        time.Time
			revokedAt        *time.Time
		)
		if err := rows.Scan(
			&kID, &tID, &saID, &name, &prefix, &scopesRaw, &ipRaw,
			&status, &expiresAt, &lastUsedAt, &lastUsedIP, &createdAt, &revokedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan api key: %w", err)
		}

		var scopes []string
		if len(scopesRaw) > 0 {
			_ = json.Unmarshal(scopesRaw, &scopes)
		}
		var ipAllowlist []string
		if len(ipRaw) > 0 {
			_ = json.Unmarshal(ipRaw, &ipAllowlist)
		}

		list = append(list, identity.APIKey{
			ID:               shared.FromPgUUID(kID),
			TenantID:         shared.FromPgUUID(tID),
			ServiceAccountID: shared.FromPgUUID(saID),
			Name:             name,
			KeyPrefix:        prefix,
			Scopes:           scopes,
			IPAllowlist:      ipAllowlist,
			Status:           identity.APIKeyStatus(status),
			ExpiresAt:        expiresAt,
			LastUsedAt:       lastUsedAt,
			LastUsedIP:       lastUsedIP,
			CreatedAt:        createdAt,
			RevokedAt:        revokedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading api keys: %w", err)
	}
	return list, nil
}

func (r *serviceAccountRepository) RevokeAPIKey(ctx context.Context, tenantID, id shared.ID) error {
	query := `
		UPDATE api_keys
		SET status = 'revoked', revoked_at = CURRENT_TIMESTAMP
		WHERE tenant_id = $1 AND id = $2 AND status = 'active'
	`
	cmd, err := r.pool.Exec(ctx, query, shared.ToPgUUID(tenantID), shared.ToPgUUID(id))
	if err != nil {
		return fmt.Errorf("failed to revoke api key: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return shared.ErrNotFound
	}
	return nil
}

func (r *serviceAccountRepository) UpdateAPIKeyUsage(ctx context.Context, id shared.ID, clientIP string, usedAt time.Time) error {
	query := `
		UPDATE api_keys
		SET last_used_at = $1, last_used_ip = $2
		WHERE id = $3
	`
	_, err := r.pool.Exec(ctx, query, usedAt, clientIP, shared.ToPgUUID(id))
	return err
}
