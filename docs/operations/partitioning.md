# NYX Partitioning Strategy

NYX does not partition tables today. For low-volume deployments, the current schema is enough. For multi-tenant or long-retention production use, use one of these strategies.

## Tenant-first partitioning

Use this when you have a small number of large tenants and strict isolation requirements.

- Partition `flows`, `events`, `approvals`, and `memories` by `tenant_id`.
- Keep per-tenant retention and backup policies separate.
- Prefer this when legal or operational requirements map cleanly to tenants.

## Time-first partitioning

Use this when the dominant problem is event and memory retention volume.

- Partition `events`, `memories`, and `actions` by month or week.
- Archive or drop old partitions instead of deleting row-by-row.
- Keep `flows` unpartitioned unless row counts also become problematic.

## Recommended default

Start with time-based partitioning for `events` and `memories`. Add tenant-based partitioning only when tenant isolation or index size requires it.
