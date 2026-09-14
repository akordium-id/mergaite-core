-- name: CreateTenant :one
INSERT INTO tenants (
    id,
    code,
    name,
    status,
    settings,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetTenantByID :one
SELECT * FROM tenants
WHERE id = $1 LIMIT 1;

-- name: GetTenantByCode :one
SELECT * FROM tenants
WHERE code = $1 LIMIT 1;

-- name: ListTenants :many
SELECT * FROM tenants
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountTenants :one
SELECT COUNT(*) FROM tenants;

-- name: UpdateTenant :one
UPDATE tenants
SET
    name = COALESCE(sqlc.narg('name'), name),
    status = COALESCE(sqlc.narg('status'), status),
    settings = COALESCE(sqlc.narg('settings'), settings),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;
