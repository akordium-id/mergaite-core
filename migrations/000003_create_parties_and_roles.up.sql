CREATE TABLE IF NOT EXISTS parties (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL, -- 'person' | 'organization'
    code VARCHAR(50),
    name VARCHAR(255) NOT NULL,
    legal_name VARCHAR(255),
    tax_id VARCHAR(50),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_parties_tenant_id ON parties(tenant_id);
CREATE INDEX IF NOT EXISTS idx_parties_type ON parties(type);
CREATE INDEX IF NOT EXISTS idx_parties_status ON parties(status);
CREATE UNIQUE INDEX IF NOT EXISTS uq_parties_tenant_code ON parties(tenant_id, code) WHERE code IS NOT NULL;

CREATE TABLE IF NOT EXISTS party_roles (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    party_id UUID NOT NULL REFERENCES parties(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    role_type VARCHAR(50) NOT NULL, -- 'customer', 'supplier', 'partner', 'employee', 'agent'
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_party_roles_tenant_id ON party_roles(tenant_id);
CREATE INDEX IF NOT EXISTS idx_party_roles_party_id ON party_roles(party_id);
CREATE INDEX IF NOT EXISTS idx_party_roles_role_type ON party_roles(role_type);
CREATE UNIQUE INDEX IF NOT EXISTS uq_party_roles_unique ON party_roles (
    tenant_id,
    party_id,
    role_type,
    COALESCE(organization_id, '00000000-0000-0000-0000-000000000000'::uuid)
);
