-- name: RecordEvent :one
INSERT INTO events (id, flow_id, event_type, message, payload)
VALUES (sqlc.arg(id), sqlc.arg(flow_id), sqlc.arg(event_type), sqlc.arg(message), sqlc.arg(payload)::jsonb)
RETURNING seq, id, flow_id, event_type, message, payload, created_at;

-- name: ListEvents :many
SELECT seq, id, flow_id, event_type, message, payload, created_at
FROM events
WHERE flow_id = sqlc.arg(flow_id) AND seq > sqlc.arg(seq)
ORDER BY seq ASC;
