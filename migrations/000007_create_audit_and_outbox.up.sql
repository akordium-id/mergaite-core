-- 000007_create_audit_and_outbox.up.sql

-- ----------------------------------------------------------------------------
-- 1. Audit Logs (Immutable Append-Only Audit Trail)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_id UUID,
    actor_type VARCHAR(50) NOT NULL DEFAULT 'user', -- 'user', 'system', 'api_key'
    action VARCHAR(50) NOT NULL,                    -- 'create', 'update', 'delete', 'transition'
    entity_type VARCHAR(100) NOT NULL,              -- 'document', 'party', 'product', 'organization', 'tenant'
    entity_id UUID NOT NULL,
    changes JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,    -- ip, user_agent, request_id
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_entity 
    ON audit_logs(tenant_id, entity_type, entity_id);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_created 
    ON audit_logs(tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_actor 
    ON audit_logs(tenant_id, actor_id, created_at DESC) 
    WHERE actor_id IS NOT NULL;

-- ----------------------------------------------------------------------------
-- 2. Outbox Events (Transactional Outbox Pattern)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,               -- e.g. 'document.created', 'document.transitioned'
    aggregate_type VARCHAR(100) NOT NULL,           -- e.g. 'document', 'party', 'product'
    aggregate_id UUID NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- 'pending', 'processing', 'published', 'failed'
    retry_count INT NOT NULL DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMPTZ
);

-- Index for background poller/worker fetching pending events
CREATE INDEX IF NOT EXISTS idx_outbox_events_pending 
    ON outbox_events(status, created_at ASC) 
    WHERE status IN ('pending', 'processing');

CREATE INDEX IF NOT EXISTS idx_outbox_events_tenant_aggregate 
    ON outbox_events(tenant_id, aggregate_type, aggregate_id);
