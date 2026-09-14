-- ----------------------------------------------------------------------------
-- 1. Custom Field Definitions
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS custom_field_definitions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL, -- 'party', 'product', 'document', 'organization'
    code VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    data_type VARCHAR(30) NOT NULL, -- 'text', 'number', 'boolean', 'date', 'select', 'multi_select', 'json'
    options JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_required BOOLEAN NOT NULL DEFAULT FALSE,
    default_value JSONB,
    validation_rules JSONB NOT NULL DEFAULT '{}'::jsonb,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_custom_field_defs_tenant_entity_code UNIQUE (tenant_id, entity_type, code)
);

CREATE INDEX IF NOT EXISTS idx_custom_field_defs_lookup 
    ON custom_field_definitions(tenant_id, entity_type, is_active);

-- ----------------------------------------------------------------------------
-- 2. Entity Custom Field Values (JSONB per Entity with GIN indexing)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS entity_custom_fields (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    values JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_entity_custom_fields_tenant_entity UNIQUE (tenant_id, entity_type, entity_id)
);

CREATE INDEX IF NOT EXISTS idx_entity_custom_fields_values 
    ON entity_custom_fields USING GIN (values);

CREATE INDEX IF NOT EXISTS idx_entity_custom_fields_entity 
    ON entity_custom_fields(tenant_id, entity_type, entity_id);

-- ----------------------------------------------------------------------------
-- 3. Seed Custom Field Permission
-- ----------------------------------------------------------------------------
INSERT INTO permissions (id, code, name, category, description) VALUES
    (gen_random_uuid(), 'custom_field:manage', 'Manage Custom Fields', 'custom_field', 'Permission to define and manage custom fields schema and values')
ON CONFLICT (code) DO NOTHING;
