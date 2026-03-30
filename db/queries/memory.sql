-- name: AddMemory :one
INSERT INTO memories (id, flow_id, action_id, kind, content, metadata)
VALUES (sqlc.arg(id), sqlc.arg(flow_id), sqlc.arg(action_id), sqlc.arg(kind), sqlc.arg(content), sqlc.arg(metadata)::jsonb)
RETURNING id, flow_id, action_id, kind, content, metadata, created_at;

-- name: SearchMemories :many
SELECT id, flow_id, action_id, kind, content, metadata, created_at
FROM memories
WHERE flow_id = sqlc.arg(flow_id) AND (sqlc.arg(query) = '%%' OR content ILIKE sqlc.arg(query))
ORDER BY created_at DESC;

-- name: ListMemoriesByFlow :many
SELECT id, flow_id, action_id, kind, content, metadata, created_at
FROM memories
WHERE flow_id = sqlc.arg(flow_id)
ORDER BY created_at ASC;
