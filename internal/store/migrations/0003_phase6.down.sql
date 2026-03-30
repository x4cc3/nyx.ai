DROP INDEX IF EXISTS idx_approvals_flow_id;
DROP INDEX IF EXISTS idx_approvals_tenant_created_at;
DROP TABLE IF EXISTS approvals;

DROP INDEX IF EXISTS idx_flows_tenant_created_at;
ALTER TABLE flows
    DROP COLUMN IF EXISTS tenant_id;
