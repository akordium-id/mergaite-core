CREATE TABLE IF NOT EXISTS addresses (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    type VARCHAR(30) NOT NULL DEFAULT 'general',
    label VARCHAR(100),
    line1 VARCHAR(255) NOT NULL,
    line2 VARCHAR(255),
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100),
    postal_code VARCHAR(20),
    country_code VARCHAR(2) NOT NULL DEFAULT 'ID',
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_addresses_tenant_id ON addresses(tenant_id);

CREATE TABLE IF NOT EXISTS party_addresses (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    party_id UUID NOT NULL REFERENCES parties(id) ON DELETE CASCADE,
    address_id UUID NOT NULL REFERENCES addresses(id) ON DELETE CASCADE,
    address_type VARCHAR(30) NOT NULL DEFAULT 'general',
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (party_id, address_id, address_type)
);

CREATE INDEX IF NOT EXISTS idx_party_addresses_tenant_party ON party_addresses(tenant_id, party_id);

CREATE TABLE IF NOT EXISTS organization_addresses (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    address_id UUID NOT NULL REFERENCES addresses(id) ON DELETE CASCADE,
    address_type VARCHAR(30) NOT NULL DEFAULT 'general',
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (organization_id, address_id, address_type)
);

CREATE INDEX IF NOT EXISTS idx_org_addresses_tenant_org ON organization_addresses(tenant_id, organization_id);

CREATE TABLE IF NOT EXISTS contacts (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    party_id UUID REFERENCES parties(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    type VARCHAR(30) NOT NULL, -- 'email', 'phone', 'mobile', 'website'
    value VARCHAR(255) NOT NULL,
    label VARCHAR(100),
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_contacts_owner CHECK (party_id IS NOT NULL OR organization_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_contacts_tenant_id ON contacts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_contacts_party_id ON contacts(party_id);
CREATE INDEX IF NOT EXISTS idx_contacts_org_id ON contacts(organization_id);
