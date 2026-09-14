-- name: CreateDocument :one
INSERT INTO documents (
    id,
    tenant_id,
    organization_id,
    document_type,
    document_number,
    document_date,
    party_id,
    status,
    total_amount,
    currency,
    notes,
    metadata,
    created_by,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
) RETURNING *;

-- name: GetDocumentByID :one
SELECT d.*, o.name as organization_name, p.name as party_name
FROM documents d
JOIN organizations o ON o.id = d.organization_id
LEFT JOIN parties p ON p.id = d.party_id
WHERE d.tenant_id = $1 AND d.id = $2 LIMIT 1;

-- name: GetDocumentByNumber :one
SELECT d.*, o.name as organization_name, p.name as party_name
FROM documents d
JOIN organizations o ON o.id = d.organization_id
LEFT JOIN parties p ON p.id = d.party_id
WHERE d.tenant_id = $1 AND d.document_type = $2 AND d.document_number = $3 LIMIT 1;

-- name: ListDocuments :many
SELECT d.*, o.name as organization_name, p.name as party_name
FROM documents d
JOIN organizations o ON o.id = d.organization_id
LEFT JOIN parties p ON p.id = d.party_id
WHERE d.tenant_id = $1
  AND (sqlc.narg('organization_id')::uuid IS NULL OR d.organization_id = sqlc.narg('organization_id'))
  AND (sqlc.narg('document_type')::varchar IS NULL OR d.document_type = sqlc.narg('document_type'))
  AND (sqlc.narg('status')::varchar IS NULL OR d.status = sqlc.narg('status'))
ORDER BY d.document_date DESC, d.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountDocuments :one
SELECT COUNT(*) FROM documents
WHERE tenant_id = $1
  AND (sqlc.narg('organization_id')::uuid IS NULL OR organization_id = sqlc.narg('organization_id'))
  AND (sqlc.narg('document_type')::varchar IS NULL OR document_type = sqlc.narg('document_type'))
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'));

-- name: UpdateDocumentStatus :one
UPDATE documents
SET
    status = $3,
    updated_at = NOW()
WHERE tenant_id = $1 AND id = $2
RETURNING *;

-- name: UpdateDocumentTotal :one
UPDATE documents
SET
    total_amount = $3,
    currency = $4,
    updated_at = NOW()
WHERE tenant_id = $1 AND id = $2
RETURNING *;

-- name: CreateDocumentLine :one
INSERT INTO document_lines (
    id,
    tenant_id,
    document_id,
    line_number,
    product_id,
    description,
    quantity,
    unit_id,
    unit_price,
    subtotal,
    metadata,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) RETURNING *;

-- name: ListDocumentLines :many
SELECT dl.*, u.code as unit_code, u.symbol as unit_symbol, p.name as product_name
FROM document_lines dl
JOIN units u ON u.id = dl.unit_id
LEFT JOIN products p ON p.id = dl.product_id
WHERE dl.tenant_id = $1 AND dl.document_id = $2
ORDER BY dl.line_number ASC;

-- name: DeleteDocumentLines :exec
DELETE FROM document_lines
WHERE tenant_id = $1 AND document_id = $2;

-- name: CreateDocumentTransition :one
INSERT INTO document_transitions (
    id,
    tenant_id,
    document_id,
    from_status,
    to_status,
    reason,
    actor_id,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: ListDocumentTransitions :many
SELECT * FROM document_transitions
WHERE tenant_id = $1 AND document_id = $2
ORDER BY created_at ASC;
