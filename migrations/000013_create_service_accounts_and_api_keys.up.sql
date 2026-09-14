-- 000013_create_service_accounts_and_api_keys.up.sql

-- ----------------------------------------------------------------------------
-- 1. Service Accounts (Machine Actors / Daemons / Integrations)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS service_accounts (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    role_id UUID REFERENCES roles(id) ON DELETE SET NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- 'active', 'suspended', 'revoked'
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_service_accounts_tenant ON service_accounts(tenant_id);

-- ----------------------------------------------------------------------------
-- 2. API Keys (Cryptographic Keys for Service Accounts)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    service_account_id UUID NOT NULL REFERENCES service_accounts(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(32) NOT NULL,              -- e.g. "mrg_live_9f8a..." (for safe UI display & fast indexing)
    key_hash VARCHAR(128) NOT NULL UNIQUE,         -- SHA-256 hex hash of the full secret key
    scopes JSONB NOT NULL DEFAULT '[]'::JSONB,     -- Custom permissions e.g. ["document:read", "product:read"]
    ip_allowlist JSONB DEFAULT NULL,               -- Array of allowed CIDRs/IPs e.g. ["192.168.1.0/24", "10.0.0.1"]
    status VARCHAR(50) NOT NULL DEFAULT 'active',  -- 'active', 'revoked', 'expired'
    expires_at TIMESTAMPTZ,                        -- NULL = never expires
    last_used_at TIMESTAMPTZ,
    last_used_ip VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_service_account ON api_keys(service_account_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(key_prefix);

-- ----------------------------------------------------------------------------
-- 3. Seed Permissions for Service Accounts & API Keys
-- ----------------------------------------------------------------------------
INSERT INTO permissions (id, code, name, category, description) VALUES
    (gen_random_uuid(), 'service_account:create', 'Create Service Account', 'identity', 'Create machine actors and service accounts'),
    (gen_random_uuid(), 'service_account:read', 'Read Service Account', 'identity', 'View service accounts and configuration'),
    (gen_random_uuid(), 'service_account:update', 'Update Service Account', 'identity', 'Modify service account details or role assignment'),
    (gen_random_uuid(), 'service_account:delete', 'Delete Service Account', 'identity', 'Delete service account and invalidate all its keys'),
    (gen_random_uuid(), 'api_key:create', 'Create API Key', 'identity', 'Generate new API keys for service accounts'),
    (gen_random_uuid(), 'api_key:read', 'Read API Key Details', 'identity', 'View API key metadata, prefix, and usage history'),
    (gen_random_uuid(), 'api_key:revoke', 'Revoke API Key', 'identity', 'Immediately revoke and invalidate an API key')
ON CONFLICT (code) DO NOTHING;
