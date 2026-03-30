ALTER TABLE flows
    ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';

CREATE INDEX IF NOT EXISTS idx_flows_tenant_created_at
    ON flows(tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS approvals (
    id TEXT PRIMARY KEY,
    flow_id TEXT NOT NULL REFERENCES flows(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    status TEXT NOT NULL,
    requested_by TEXT NOT NULL,
    reviewed_by TEXT NOT NULL DEFAULT '',
    review_note TEXT NOT NULL DEFAULT '',
    reason TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_approvals_tenant_created_at
    ON approvals(tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_approvals_flow_id
    ON approvals(flow_id, created_at ASC);
