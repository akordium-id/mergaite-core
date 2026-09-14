-- ----------------------------------------------------------------------------
-- Milestone 7: File & Attachment Engine
-- ----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    storage_driver VARCHAR(50) NOT NULL DEFAULT 'local',
    storage_path VARCHAR(500) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    mime_type VARCHAR(150) NOT NULL,
    size_bytes BIGINT NOT NULL,
    sha256_hash VARCHAR(64) NOT NULL,
    uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    is_public BOOLEAN NOT NULL DEFAULT false,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_files_tenant ON files(tenant_id);
CREATE INDEX IF NOT EXISTS idx_files_hash ON files(tenant_id, sha256_hash);

CREATE TABLE IF NOT EXISTS entity_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    purpose VARCHAR(50) NOT NULL DEFAULT 'attachment',
    title VARCHAR(255) NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_attachments_lookup ON entity_attachments(tenant_id, entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_attachments_file ON entity_attachments(tenant_id, file_id);

-- Seed Attachment Permissions
INSERT INTO permissions (id, code, name, category, description) VALUES
    (gen_random_uuid(), 'file:upload', 'Upload Files', 'file', 'Permission to upload physical files to storage'),
    (gen_random_uuid(), 'file:read', 'Read Files', 'file', 'Permission to download and view files'),
    (gen_random_uuid(), 'file:delete', 'Delete Files', 'file', 'Permission to delete physical files'),
    (gen_random_uuid(), 'attachment:manage', 'Manage Attachments', 'file', 'Permission to attach and detach files to domain entities')
ON CONFLICT (code) DO NOTHING;
