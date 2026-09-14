-- 000008_create_identity_and_access.up.sql

-- ----------------------------------------------------------------------------
-- 1. Users (Global Identity)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- 'active', 'suspended', 'deactivated'
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- ----------------------------------------------------------------------------
-- 2. Tenant Memberships
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS tenant_users (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- 'active', 'invited', 'suspended'
    joined_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_tenant_users_tenant ON tenant_users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_users_user ON tenant_users(user_id);

-- ----------------------------------------------------------------------------
-- 3. Roles (Per-Tenant)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, code)
);

CREATE INDEX IF NOT EXISTS idx_roles_tenant ON roles(tenant_id);

-- ----------------------------------------------------------------------------
-- 4. Permissions (Global System Catalog)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY,
    code VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    description TEXT
);

-- ----------------------------------------------------------------------------
-- 5. Role Permissions (Pivot)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- ----------------------------------------------------------------------------
-- 6. User Roles (Per-Tenant Role Assignment)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS user_roles (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (tenant_id, user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_lookup ON user_roles(tenant_id, user_id);

-- ----------------------------------------------------------------------------
-- 7. Seed Core Permissions
-- ----------------------------------------------------------------------------
INSERT INTO permissions (id, code, name, category, description) VALUES
    (gen_random_uuid(), 'document:create', 'Create Documents', 'document', 'Permission to create new business documents'),
    (gen_random_uuid(), 'document:read', 'Read Documents', 'document', 'Permission to view business documents'),
    (gen_random_uuid(), 'document:update', 'Update Documents', 'document', 'Permission to update draft business documents'),
    (gen_random_uuid(), 'document:transition', 'Transition Document Status', 'document', 'Permission to change document workflow status'),
    (gen_random_uuid(), 'document:delete', 'Delete Documents', 'document', 'Permission to cancel or delete documents'),
    (gen_random_uuid(), 'party:create', 'Create Parties', 'party', 'Permission to create customers, suppliers, and partners'),
    (gen_random_uuid(), 'party:read', 'Read Parties', 'party', 'Permission to view party directory'),
    (gen_random_uuid(), 'party:update', 'Update Parties', 'party', 'Permission to modify party details'),
    (gen_random_uuid(), 'party:delete', 'Delete Parties', 'party', 'Permission to deactivate parties'),
    (gen_random_uuid(), 'product:create', 'Create Products', 'product', 'Permission to create catalog products and units'),
    (gen_random_uuid(), 'product:read', 'Read Products', 'product', 'Permission to view product catalog and units'),
    (gen_random_uuid(), 'product:update', 'Update Products', 'product', 'Permission to modify products and prices'),
    (gen_random_uuid(), 'audit:read', 'Read Audit Logs', 'audit', 'Permission to view immutable system audit trail'),
    (gen_random_uuid(), 'iam:manage', 'Manage IAM & Roles', 'iam', 'Permission to manage users, invite members, and configure roles')
ON CONFLICT (code) DO NOTHING;
