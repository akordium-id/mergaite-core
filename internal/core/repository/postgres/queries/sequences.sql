-- name: CreateSequence :one
INSERT INTO number_sequences (
    id, tenant_id, code, name, entity_type, sub_type,
    prefix, suffix, template, padding, start_value,
    increment_by, current_value, reset_policy,
    last_number, last_reset_at, is_active, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11,
    $12, $13, $14,
    $15, $16, $17, $18, $19
) RETURNING *;

-- name: GetSequenceByID :one
SELECT * FROM number_sequences
WHERE tenant_id = $1 AND id = $2;

-- name: GetSequenceByCode :one
SELECT * FROM number_sequences
WHERE tenant_id = $1 AND code = $2;

-- name: GetSequenceByEntity :one
SELECT * FROM number_sequences
WHERE tenant_id = $1 AND entity_type = $2 AND sub_type = $3 AND is_active = true;

-- name: ListSequences :many
SELECT * FROM number_sequences
WHERE tenant_id = $1
ORDER BY entity_type ASC, sub_type ASC, code ASC;

-- name: UpdateSequence :one
UPDATE number_sequences
SET name = $3,
    prefix = $4,
    suffix = $5,
    template = $6,
    padding = $7,
    reset_policy = $8,
    is_active = $9,
    updated_at = $10
WHERE tenant_id = $1 AND id = $2
RETURNING *;

-- name: DeleteSequence :exec
DELETE FROM number_sequences
WHERE tenant_id = $1 AND id = $2;

-- name: GetSequenceForUpdate :one
SELECT * FROM number_sequences
WHERE tenant_id = $1 AND entity_type = $2 AND sub_type = $3 AND is_active = true
FOR UPDATE;

-- name: GetSequenceByIDForUpdate :one
SELECT * FROM number_sequences
WHERE tenant_id = $1 AND id = $2 AND is_active = true
FOR UPDATE;

-- name: UpdateSequenceValue :exec
UPDATE number_sequences
SET current_value = $3,
    last_number = $4,
    last_reset_at = $5,
    updated_at = $6
WHERE tenant_id = $1 AND id = $2;
