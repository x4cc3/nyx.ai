-- name: GetFlow :one
SELECT id, name, target, objective, status, created_at, updated_at
FROM flows
WHERE id = sqlc.arg(id);

-- name: ListFlows :many
SELECT id, name, target, objective, status, created_at, updated_at
FROM flows
ORDER BY created_at DESC;

-- name: CreateFlow :one
INSERT INTO flows (id, name, target, objective, status)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(target), sqlc.arg(objective), sqlc.arg(status))
RETURNING id, name, target, objective, status, created_at, updated_at;

-- name: QueueFlow :one
UPDATE flows
SET status = 'queued', updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, name, target, objective, status, created_at, updated_at;

-- name: UpdateFlowStatus :one
UPDATE flows
SET status = sqlc.arg(status), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, name, target, objective, status, created_at, updated_at;

-- name: ClaimNextQueuedFlow :one
WITH next_flow AS (
    SELECT id
    FROM flows
    WHERE status = 'queued'
    ORDER BY created_at ASC
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
UPDATE flows AS f
SET status = 'running', updated_at = NOW()
FROM next_flow
WHERE f.id = next_flow.id
RETURNING f.id, f.name, f.target, f.objective, f.status, f.created_at, f.updated_at;
