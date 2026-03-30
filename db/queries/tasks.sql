-- name: CreateTask :one
INSERT INTO tasks (id, flow_id, name, description, agent_role, status)
VALUES (sqlc.arg(id), sqlc.arg(flow_id), sqlc.arg(name), sqlc.arg(description), sqlc.arg(agent_role), sqlc.arg(status))
RETURNING id, flow_id, name, description, agent_role, status, created_at, updated_at;

-- name: UpdateTaskStatus :exec
UPDATE tasks
SET status = sqlc.arg(status), updated_at = NOW()
WHERE id = sqlc.arg(id);

-- name: ListTasksByFlow :many
SELECT id, flow_id, name, description, agent_role, status, created_at, updated_at
FROM tasks
WHERE flow_id = sqlc.arg(flow_id)
ORDER BY created_at ASC;

-- name: CreateSubtask :one
INSERT INTO subtasks (id, flow_id, task_id, name, description, agent_role, status)
VALUES (sqlc.arg(id), sqlc.arg(flow_id), sqlc.arg(task_id), sqlc.arg(name), sqlc.arg(description), sqlc.arg(agent_role), sqlc.arg(status))
RETURNING id, task_id, flow_id, name, description, agent_role, status, created_at, updated_at;

-- name: UpdateSubtaskStatus :exec
UPDATE subtasks
SET status = sqlc.arg(status), updated_at = NOW()
WHERE id = sqlc.arg(id);

-- name: ListSubtasksByFlow :many
SELECT id, task_id, flow_id, name, description, agent_role, status, created_at, updated_at
FROM subtasks
WHERE flow_id = sqlc.arg(flow_id)
ORDER BY created_at ASC;
