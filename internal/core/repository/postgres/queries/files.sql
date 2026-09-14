-- name: CreateFile :one
INSERT INTO files (
    id, tenant_id, storage_driver, storage_path, filename, mime_type,
    size_bytes, sha256_hash, uploaded_by, is_public, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetFileByID :one
SELECT * FROM files
WHERE tenant_id = $1 AND id = $2;

-- name: GetFileByHash :one
SELECT * FROM files
WHERE tenant_id = $1 AND sha256_hash = $2
ORDER BY created_at DESC
LIMIT 1;

-- name: ListFilesByTenant :many
SELECT * FROM files
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteFile :exec
DELETE FROM files
WHERE tenant_id = $1 AND id = $2;

-- name: CreateAttachment :one
INSERT INTO entity_attachments (
    id, tenant_id, file_id, entity_type, entity_id, purpose, title, sort_order
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetAttachmentByID :one
SELECT * FROM entity_attachments
WHERE tenant_id = $1 AND id = $2;

-- name: ListAttachmentsByEntity :many
SELECT 
    ea.id AS attachment_id,
    ea.tenant_id,
    ea.file_id,
    ea.entity_type,
    ea.entity_id,
    ea.purpose,
    ea.title,
    ea.sort_order,
    ea.created_at AS attached_at,
    f.storage_driver,
    f.storage_path,
    f.filename,
    f.mime_type,
    f.size_bytes,
    f.sha256_hash,
    f.uploaded_by,
    f.is_public,
    f.metadata AS file_metadata,
    f.created_at AS file_created_at
FROM entity_attachments ea
JOIN files f ON ea.file_id = f.id
WHERE ea.tenant_id = $1 AND ea.entity_type = $2 AND ea.entity_id = $3
ORDER BY ea.sort_order ASC, ea.created_at ASC;

-- name: DeleteAttachment :exec
DELETE FROM entity_attachments
WHERE tenant_id = $1 AND id = $2;

-- name: CountFileReferences :one
SELECT COUNT(*) FROM entity_attachments
WHERE tenant_id = $1 AND file_id = $2;
