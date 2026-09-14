-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (
    id, tenant_id, event_type, aggregate_type, aggregate_id, payload, status, retry_count, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: FetchPendingOutboxEvents :many
SELECT 
    id, tenant_id, event_type, aggregate_type, aggregate_id, payload, status, retry_count, error_message, created_at, published_at
FROM outbox_events
WHERE status IN ('pending', 'failed')
  AND retry_count < $1
ORDER BY created_at ASC
LIMIT $2
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxEventPublished :exec
UPDATE outbox_events
SET status = 'published',
    published_at = $2,
    error_message = NULL
WHERE id = $1;

-- name: MarkOutboxEventFailed :exec
UPDATE outbox_events
SET status = 'failed',
    retry_count = retry_count + 1,
    error_message = $2
WHERE id = $1;
