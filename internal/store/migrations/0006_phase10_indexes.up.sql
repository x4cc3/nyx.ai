CREATE INDEX IF NOT EXISTS idx_actions_flow_status
    ON actions(flow_id, status);

CREATE INDEX IF NOT EXISTS idx_memories_flow_created_at
    ON memories(flow_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_tasks_flow_id_created_at
    ON tasks(flow_id, created_at ASC);
