-- name: CreateOrganization :one
INSERT INTO organizations (
    id,
    tenant_id,
    parent_id,
    code,
    name,
    legal_name,
    type,
    status,
    settings,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetOrganizationByID :one
SELECT * FROM organizations
WHERE tenant_id = $1 AND id = $2 LIMIT 1;

-- name: GetOrganizationByCode :one
SELECT * FROM organizations
WHERE tenant_id = $1 AND code = $2 LIMIT 1;

-- name: ListOrganizationsByTenant :many
SELECT * FROM organizations
WHERE tenant_id = $1
ORDER BY created_at ASC;

-- name: ListOrganizationChildren :many
SELECT * FROM organizations
WHERE tenant_id = $1 AND parent_id = $2
ORDER BY name ASC;

-- name: UpdateOrganization :one
UPDATE organizations
SET
    parent_id = COALESCE(sqlc.narg('parent_id'), parent_id),
    name = COALESCE(sqlc.narg('name'), name),
    legal_name = COALESCE(sqlc.narg('legal_name'), legal_name),
    type = COALESCE(sqlc.narg('type'), type),
    status = COALESCE(sqlc.narg('status'), status),
    settings = COALESCE(sqlc.narg('settings'), settings),
    updated_at = NOW()
WHERE tenant_id = sqlc.arg('tenant_id') AND id = sqlc.arg('id')
RETURNING *;
