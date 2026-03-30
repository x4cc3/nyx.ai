-- name: CreateAgent :one
INSERT INTO agents (id, flow_id, role, model, status)
VALUES (sqlc.arg(id), sqlc.arg(flow_id), sqlc.arg(role), sqlc.arg(model), sqlc.arg(status))
RETURNING id, flow_id, role, model, status, created_at, updated_at;

-- name: UpdateAgentStatus :exec
UPDATE agents
SET status = sqlc.arg(status), updated_at = NOW()
WHERE id = sqlc.arg(id);

-- name: ListAgentsByFlow :many
SELECT id, flow_id, role, model, status, created_at, updated_at
FROM agents
WHERE flow_id = sqlc.arg(flow_id)
ORDER BY created_at ASC;
