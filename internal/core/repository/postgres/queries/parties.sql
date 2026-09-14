-- name: CreateParty :one
INSERT INTO parties (
    id,
    tenant_id,
    type,
    code,
    name,
    legal_name,
    tax_id,
    status,
    metadata,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetPartyByID :one
SELECT * FROM parties
WHERE tenant_id = $1 AND id = $2 LIMIT 1;

-- name: ListParties :many
SELECT * FROM parties
WHERE tenant_id = $1
  AND (sqlc.narg('type')::varchar IS NULL OR type = sqlc.narg('type'))
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountParties :one
SELECT COUNT(*) FROM parties
WHERE tenant_id = $1
  AND (sqlc.narg('type')::varchar IS NULL OR type = sqlc.narg('type'));

-- name: ListPartiesByRole :many
SELECT p.* FROM parties p
JOIN party_roles pr ON pr.party_id = p.id AND pr.tenant_id = p.tenant_id
WHERE p.tenant_id = $1 AND pr.role_type = $2 AND pr.status = 'active'
ORDER BY p.created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountPartiesByRole :one
SELECT COUNT(DISTINCT p.id) FROM parties p
JOIN party_roles pr ON pr.party_id = p.id AND pr.tenant_id = p.tenant_id
WHERE p.tenant_id = $1 AND pr.role_type = $2 AND pr.status = 'active';

-- name: UpdateParty :one
UPDATE parties
SET
    name = COALESCE(sqlc.narg('name'), name),
    legal_name = COALESCE(sqlc.narg('legal_name'), legal_name),
    tax_id = COALESCE(sqlc.narg('tax_id'), tax_id),
    status = COALESCE(sqlc.narg('status'), status),
    metadata = COALESCE(sqlc.narg('metadata'), metadata),
    updated_at = NOW()
WHERE tenant_id = sqlc.arg('tenant_id') AND id = sqlc.arg('id')
RETURNING *;

-- name: CreatePartyRole :one
INSERT INTO party_roles (
    id,
    tenant_id,
    party_id,
    organization_id,
    role_type,
    status,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: ListRolesByPartyID :many
SELECT * FROM party_roles
WHERE tenant_id = $1 AND party_id = $2
ORDER BY created_at ASC;

-- name: DeletePartyRole :exec
DELETE FROM party_roles
WHERE tenant_id = $1 AND id = $2;
