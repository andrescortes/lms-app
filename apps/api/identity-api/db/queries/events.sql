-- name: CreateEvent :exec
INSERT INTO registration_events (
    id,
    event_type,
    payload,
    status,
    attempts,
    created_at,
    delivered_at
) VALUES (
    ?,
    ?,
    ?,
    'pending',
    0,
    ?,
    NULL
);

-- name: ListPendingEvents :many
SELECT
    id,
    event_type,
    payload,
    status,
    attempts,
    created_at,
    delivered_at
FROM registration_events
WHERE status = 'pending'
ORDER BY created_at ASC, id ASC
LIMIT ?;

-- name: MarkEventDelivered :exec
UPDATE registration_events
SET status = 'delivered',
    delivered_at = ?
WHERE id = ?;

-- name: IncrementEventAttempts :exec
UPDATE registration_events
SET attempts = attempts + 1
WHERE id = ?;
