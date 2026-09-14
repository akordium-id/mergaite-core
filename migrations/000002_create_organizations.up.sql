CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES organizations(id) ON DELETE RESTRICT,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    legal_name VARCHAR(255),
    type VARCHAR(30) NOT NULL DEFAULT 'company',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_organizations_tenant_code UNIQUE (tenant_id, code)
);

CREATE INDEX IF NOT EXISTS idx_organizations_tenant_id ON organizations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_organizations_parent_id ON organizations(parent_id);
CREATE INDEX IF NOT EXISTS idx_organizations_type ON organizations(type);
CREATE INDEX IF NOT EXISTS idx_organizations_status ON organizations(status);
