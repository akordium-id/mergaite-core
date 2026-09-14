-- ----------------------------------------------------------------------------
-- 1. Number Sequences Table
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS number_sequences (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    entity_type VARCHAR(50) NOT NULL, -- 'document', 'party', 'product', etc.
    sub_type VARCHAR(50) NOT NULL,    -- 'sales_order', 'invoice', 'quotation', 'purchase_order', etc.
    prefix VARCHAR(50) NOT NULL DEFAULT '',
    suffix VARCHAR(50) NOT NULL DEFAULT '',
    template VARCHAR(255) NOT NULL DEFAULT '{PREFIX}/{YYYY}/{MM}/{SEQ:4}',
    padding INT NOT NULL DEFAULT 4,
    start_value BIGINT NOT NULL DEFAULT 1,
    increment_by INT NOT NULL DEFAULT 1,
    current_value BIGINT NOT NULL DEFAULT 0,
    reset_policy VARCHAR(20) NOT NULL DEFAULT 'never', -- 'never', 'yearly', 'monthly', 'daily'
    last_number VARCHAR(100),
    last_reset_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_number_sequences_tenant_code UNIQUE (tenant_id, code),
    CONSTRAINT uq_number_sequences_tenant_entity_subtype UNIQUE (tenant_id, entity_type, sub_type)
);

CREATE INDEX IF NOT EXISTS idx_number_sequences_lookup 
    ON number_sequences(tenant_id, entity_type, sub_type, is_active);

-- ----------------------------------------------------------------------------
-- 2. Seed System Permission
-- ----------------------------------------------------------------------------
INSERT INTO permissions (id, code, name, category, description) VALUES
    (gen_random_uuid(), 'sequence:manage', 'Manage Number Sequences', 'sequence', 'Permission to configure number sequences and auto-numbering templates')
ON CONFLICT (code) DO NOTHING;
