-- name: CreateAuditLog :one
INSERT INTO audit_logs (
    id, tenant_id, actor_id, actor_type, action, entity_type, entity_id, changes, metadata, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: ListAuditLogs :many
SELECT 
    id, tenant_id, actor_id, actor_type, action, entity_type, entity_id, changes, metadata, created_at
FROM audit_logs
WHERE tenant_id = $1
  AND (sqlc.narg('entity_type')::text IS NULL OR entity_type = sqlc.narg('entity_type'))
  AND (sqlc.narg('entity_id')::uuid IS NULL OR entity_id = sqlc.narg('entity_id'))
  AND (sqlc.narg('actor_id')::uuid IS NULL OR actor_id = sqlc.narg('actor_id'))
  AND (sqlc.narg('action')::text IS NULL OR action = sqlc.narg('action'))
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountAuditLogs :one
SELECT COUNT(*) FROM audit_logs
WHERE tenant_id = $1
  AND (sqlc.narg('entity_type')::text IS NULL OR entity_type = sqlc.narg('entity_type'))
  AND (sqlc.narg('entity_id')::uuid IS NULL OR entity_id = sqlc.narg('entity_id'))
  AND (sqlc.narg('actor_id')::uuid IS NULL OR actor_id = sqlc.narg('actor_id'))
  AND (sqlc.narg('action')::text IS NULL OR action = sqlc.narg('action'));
