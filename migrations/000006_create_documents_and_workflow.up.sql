CREATE TABLE IF NOT EXISTS documents (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    document_type VARCHAR(50) NOT NULL, -- 'sales_order', 'invoice', 'purchase_order', 'quotation'
    document_number VARCHAR(100) NOT NULL,
    document_date DATE NOT NULL DEFAULT CURRENT_DATE,
    party_id UUID REFERENCES parties(id) ON DELETE RESTRICT,
    status VARCHAR(30) NOT NULL DEFAULT 'draft', -- 'draft', 'submitted', 'approved', 'posted', 'completed', 'cancelled', 'rejected'
    total_amount BIGINT NOT NULL DEFAULT 0, -- Money amount in minor unit (sen/rupiah)
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',
    notes TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_documents_tenant_type_number UNIQUE (tenant_id, document_type, document_number)
);

CREATE INDEX IF NOT EXISTS idx_documents_tenant_id ON documents(tenant_id);
CREATE INDEX IF NOT EXISTS idx_documents_org_id ON documents(organization_id);
CREATE INDEX IF NOT EXISTS idx_documents_party_id ON documents(party_id);
CREATE INDEX IF NOT EXISTS idx_documents_type_status ON documents(document_type, status);

CREATE TABLE IF NOT EXISTS document_lines (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    line_number INT NOT NULL,
    product_id UUID REFERENCES products(id) ON DELETE RESTRICT,
    description VARCHAR(255) NOT NULL,
    quantity DECIMAL(18, 4) NOT NULL,
    unit_id UUID NOT NULL REFERENCES units(id) ON DELETE RESTRICT,
    unit_price BIGINT NOT NULL, -- Money amount in minor unit
    subtotal BIGINT NOT NULL,   -- Money amount in minor unit
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_document_lines_doc_line UNIQUE (document_id, line_number)
);

CREATE INDEX IF NOT EXISTS idx_doc_lines_tenant_doc ON document_lines(tenant_id, document_id);
CREATE INDEX IF NOT EXISTS idx_doc_lines_product_id ON document_lines(product_id);

CREATE TABLE IF NOT EXISTS document_transitions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    from_status VARCHAR(30) NOT NULL,
    to_status VARCHAR(30) NOT NULL,
    reason TEXT,
    actor_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_doc_transitions_tenant_doc ON document_transitions(tenant_id, document_id);
