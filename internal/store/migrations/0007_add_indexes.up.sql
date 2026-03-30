CREATE INDEX IF NOT EXISTS idx_flows_status_created ON flows(status, created_at);
CREATE INDEX IF NOT EXISTS idx_findings_flow_severity ON findings(flow_id, severity);
CREATE INDEX IF NOT EXISTS idx_subtasks_task_id ON subtasks(task_id);
CREATE INDEX IF NOT EXISTS idx_artifacts_flow_id ON artifacts(flow_id);
CREATE INDEX IF NOT EXISTS idx_executions_action_id ON executions(action_id);
CREATE INDEX IF NOT EXISTS idx_actions_subtask_id ON actions(subtask_id);
