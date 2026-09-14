-- name: CreateCustomFieldDefinition :one
INSERT INTO custom_field_definitions (
    id, tenant_id, entity_type, code, name, description,
    data_type, options, is_required, default_value, validation_rules,
    sort_order, is_active, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11,
    $12, $13, $14, $15
) RETURNING *;

-- name: GetCustomFieldDefinitionByID :one
SELECT * FROM custom_field_definitions
WHERE tenant_id = $1 AND id = $2;

-- name: GetCustomFieldDefinitionByCode :one
SELECT * FROM custom_field_definitions
WHERE tenant_id = $1 AND entity_type = $2 AND code = $3;

-- name: ListCustomFieldDefinitions :many
SELECT * FROM custom_field_definitions
WHERE tenant_id = $1 AND entity_type = $2
ORDER BY sort_order ASC, code ASC;

-- name: ListActiveCustomFieldDefinitions :many
SELECT * FROM custom_field_definitions
WHERE tenant_id = $1 AND entity_type = $2 AND is_active = true
ORDER BY sort_order ASC, code ASC;

-- name: UpdateCustomFieldDefinition :one
UPDATE custom_field_definitions
SET name = $3,
    description = $4,
    options = $5,
    is_required = $6,
    default_value = $7,
    validation_rules = $8,
    sort_order = $9,
    is_active = $10,
    updated_at = $11
WHERE tenant_id = $1 AND id = $2
RETURNING *;

-- name: DeleteCustomFieldDefinition :exec
DELETE FROM custom_field_definitions
WHERE tenant_id = $1 AND id = $2;

-- name: UpsertEntityCustomFields :one
INSERT INTO entity_custom_fields (
    id, tenant_id, entity_type, entity_id, values, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
ON CONFLICT (tenant_id, entity_type, entity_id)
DO UPDATE SET
    values = EXCLUDED.values,
    updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: GetEntityCustomFields :one
SELECT * FROM entity_custom_fields
WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3;

-- name: DeleteEntityCustomFields :exec
DELETE FROM entity_custom_fields
WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3;

-- name: FindEntitiesByCustomFieldMatch :many
SELECT entity_id, values FROM entity_custom_fields
WHERE tenant_id = $1 AND entity_type = $2 AND values @> sqlc.arg('match_values')::jsonb;
