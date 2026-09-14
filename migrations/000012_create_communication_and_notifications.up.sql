-- ----------------------------------------------------------------------------
-- Milestone 8: Communication & Notifications Engine
-- ----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(30) NOT NULL DEFAULT 'comment',
    content TEXT NOT NULL,
    mentions JSONB NOT NULL DEFAULT '[]'::jsonb,
    parent_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    is_pinned BOOLEAN NOT NULL DEFAULT false,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_comments_entity ON comments(tenant_id, entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_comments_author ON comments(tenant_id, author_id);
CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_id);

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'system',
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    entity_type VARCHAR(50) NOT NULL DEFAULT '',
    entity_id UUID,
    is_read BOOLEAN NOT NULL DEFAULT false,
    read_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(tenant_id, user_id, is_read, created_at DESC);

-- Seed Permissions
INSERT INTO permissions (id, code, name, category, description) VALUES
    (gen_random_uuid(), 'comment:create', 'Create Comments', 'communication', 'Permission to add comments and internal notes'),
    (gen_random_uuid(), 'comment:read', 'Read Comments', 'communication', 'Permission to read comments and activity timeline'),
    (gen_random_uuid(), 'comment:delete', 'Delete Comments', 'communication', 'Permission to delete comments'),
    (gen_random_uuid(), 'notification:read', 'Read Notifications', 'communication', 'Permission to read and manage personal notifications')
ON CONFLICT (code) DO NOTHING;
