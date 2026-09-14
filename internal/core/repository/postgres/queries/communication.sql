-- name: CreateComment :one
INSERT INTO comments (
    id, tenant_id, entity_type, entity_id, author_id, type,
    content, mentions, parent_id, is_pinned, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetCommentByID :one
SELECT * FROM comments
WHERE tenant_id = $1 AND id = $2;

-- name: ListCommentsByEntity :many
SELECT 
    c.id,
    c.tenant_id,
    c.entity_type,
    c.entity_id,
    c.author_id,
    c.type,
    c.content,
    c.mentions,
    c.parent_id,
    c.is_pinned,
    c.metadata,
    c.created_at,
    c.updated_at,
    u.name AS author_name,
    u.email AS author_email
FROM comments c
JOIN users u ON c.author_id = u.id
WHERE c.tenant_id = $1 AND c.entity_type = $2 AND c.entity_id = $3
ORDER BY c.is_pinned DESC, c.created_at ASC;

-- name: DeleteComment :exec
DELETE FROM comments
WHERE tenant_id = $1 AND id = $2;

-- name: CreateNotification :one
INSERT INTO notifications (
    id, tenant_id, user_id, actor_id, type, title, message,
    entity_type, entity_id, is_read, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetNotificationByID :one
SELECT * FROM notifications
WHERE tenant_id = $1 AND id = $2;

-- name: ListNotificationsByUser :many
SELECT * FROM notifications
WHERE tenant_id = $1 AND user_id = $2
  AND (CASE WHEN $3::boolean THEN is_read = false ELSE true END)
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: MarkNotificationRead :exec
UPDATE notifications
SET is_read = true, read_at = NOW()
WHERE tenant_id = $1 AND user_id = $2 AND id = $3;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications
SET is_read = true, read_at = NOW()
WHERE tenant_id = $1 AND user_id = $2 AND is_read = false;

-- name: CountUnreadNotifications :one
SELECT COUNT(*) FROM notifications
WHERE tenant_id = $1 AND user_id = $2 AND is_read = false;
