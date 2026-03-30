-- name: CreateAction :one
INSERT INTO actions (id, flow_id, task_id, subtask_id, agent_role, function_name, input, status, execution_mode)
VALUES (
    sqlc.arg(id),
    sqlc.arg(flow_id),
    sqlc.arg(task_id),
    sqlc.arg(subtask_id),
    sqlc.arg(agent_role),
    sqlc.arg(function_name),
    sqlc.arg(input)::jsonb,
    sqlc.arg(status),
    sqlc.arg(execution_mode)
)
RETURNING id, flow_id, task_id, subtask_id, agent_role, function_name, input, output, status, execution_mode, created_at, updated_at;

-- name: CompleteAction :one
UPDATE actions
SET status = sqlc.arg(status), output = sqlc.arg(output)::jsonb, updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, flow_id, task_id, subtask_id, agent_role, function_name, input, output, status, execution_mode, created_at, updated_at;

-- name: ListActionsByFlow :many
SELECT id, flow_id, task_id, subtask_id, agent_role, function_name, input, output, status, execution_mode, created_at, updated_at
FROM actions
WHERE flow_id = sqlc.arg(flow_id)
ORDER BY created_at ASC;

-- name: AddArtifact :one
INSERT INTO artifacts (id, flow_id, action_id, kind, name, content, metadata)
VALUES (sqlc.arg(id), sqlc.arg(flow_id), sqlc.arg(action_id), sqlc.arg(kind), sqlc.arg(name), sqlc.arg(content), sqlc.arg(metadata)::jsonb)
RETURNING id, flow_id, action_id, kind, name, content, metadata, created_at;

-- name: ListArtifactsByFlow :many
SELECT id, flow_id, action_id, kind, name, content, metadata, created_at
FROM artifacts
WHERE flow_id = sqlc.arg(flow_id)
ORDER BY created_at ASC;

-- name: CreateExecution :one
INSERT INTO executions (id, flow_id, action_id, profile, runtime, metadata, status)
VALUES (sqlc.arg(id), sqlc.arg(flow_id), sqlc.arg(action_id), sqlc.arg(profile), sqlc.arg(runtime), sqlc.arg(metadata)::jsonb, sqlc.arg(status))
RETURNING id, flow_id, action_id, profile, runtime, metadata, status, started_at, completed_at;

-- name: CompleteExecution :exec
UPDATE executions
SET profile = sqlc.arg(profile),
    runtime = sqlc.arg(runtime),
    metadata = sqlc.arg(metadata)::jsonb,
    status = sqlc.arg(status),
    completed_at = NOW()
WHERE id = sqlc.arg(id);

-- name: ListExecutionsByFlow :many
SELECT id, flow_id, action_id, profile, runtime, metadata, status, started_at, completed_at
FROM executions
WHERE flow_id = sqlc.arg(flow_id)
ORDER BY started_at ASC;
