-- name: CreateUser :one
INSERT INTO users (
    id, email, password_hash, name, status, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: AddTenantMember :one
INSERT INTO tenant_users (
    id, tenant_id, user_id, status, joined_at
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetTenantMember :one
SELECT * FROM tenant_users 
WHERE tenant_id = $1 AND user_id = $2;

-- name: ListUserTenants :many
SELECT 
    t.id AS tenant_id,
    t.code AS tenant_code,
    t.name AS tenant_name,
    t.status AS tenant_status,
    tu.status AS membership_status,
    tu.joined_at
FROM tenant_users tu
JOIN tenants t ON t.id = tu.tenant_id
WHERE tu.user_id = $1
ORDER BY tu.joined_at ASC;

-- name: ListTenantUsers :many
SELECT 
    u.id AS user_id,
    u.email,
    u.name,
    u.status AS user_status,
    tu.status AS membership_status,
    tu.joined_at
FROM tenant_users tu
JOIN users u ON u.id = tu.user_id
WHERE tu.tenant_id = $1
ORDER BY tu.joined_at ASC;

-- name: CreateRole :one
INSERT INTO roles (
    id, tenant_id, code, name, description, is_system, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetRoleByID :one
SELECT * FROM roles WHERE tenant_id = $1 AND id = $2;

-- name: GetRoleByCode :one
SELECT * FROM roles WHERE tenant_id = $1 AND code = $2;

-- name: ListRoles :many
SELECT * FROM roles WHERE tenant_id = $1 ORDER BY name ASC;

-- name: ListPermissions :many
SELECT * FROM permissions ORDER BY category ASC, code ASC;

-- name: AssignPermissionToRole :exec
INSERT INTO role_permissions (role_id, permission_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: ClearRolePermissions :exec
DELETE FROM role_permissions WHERE role_id = $1;

-- name: ListRolePermissions :many
SELECT p.id, p.code, p.name, p.category, p.description
FROM role_permissions rp
JOIN permissions p ON p.id = rp.permission_id
WHERE rp.role_id = $1
ORDER BY p.code ASC;

-- name: AssignUserRole :exec
INSERT INTO user_roles (tenant_id, user_id, role_id)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: RemoveUserRole :exec
DELETE FROM user_roles
WHERE tenant_id = $1 AND user_id = $2 AND role_id = $3;

-- name: ListUserRolesInTenant :many
SELECT r.id, r.code, r.name, r.description, r.is_system
FROM user_roles ur
JOIN roles r ON r.id = ur.role_id
WHERE ur.tenant_id = $1 AND ur.user_id = $2
ORDER BY r.name ASC;

-- name: ListUserPermissionsInTenant :many
SELECT DISTINCT p.code
FROM user_roles ur
JOIN roles r ON r.id = ur.role_id
JOIN role_permissions rp ON rp.role_id = r.id
JOIN permissions p ON p.id = rp.permission_id
WHERE ur.tenant_id = $1 AND ur.user_id = $2;
