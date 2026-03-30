-- name: AddFinding :one
INSERT INTO findings (id, flow_id, title, severity, description)
VALUES (sqlc.arg(id), sqlc.arg(flow_id), sqlc.arg(title), sqlc.arg(severity), sqlc.arg(description))
RETURNING id, flow_id, title, severity, description, created_at;

-- name: ListFindingsByFlow :many
SELECT id, flow_id, title, severity, description, created_at
FROM findings
WHERE flow_id = sqlc.arg(flow_id)
ORDER BY created_at ASC;
