# NYX Hardening Review

This document records the current Phase 8 hardening posture for the rebuilt NYX platform.

## Executor isolation review

- Terminal and file actions only run through the executor-manager boundary in `internal/executor`.
- Each action attempt receives an isolated workspace rooted under `NYX_EXECUTOR_WORKSPACE_ROOT`.
- File actions are constrained to their action workspace and reject path escapes.
- Execution profiles define timeout, retry, and resource policy separately for `terminal` and `file`.
- Docker mode exists for stronger runtime isolation; local mode remains available for tests and constrained environments.

Residual risk:

- Local executor mode still shares the host namespace and should be treated as a development-only profile.
- Production deployments should prefer Docker-backed execution with locked-down images and host-level sandboxing.

Recommended remediation:

1. Run the executor on a dedicated host or VM.
2. Prefer rootless Docker when nested containers are required.
3. Enforce AppArmor or SELinux on the executor host.
4. Keep the nested worker filesystem read-only and only expose `/tmp` plus the bound workspace.
5. Audit `docker.sock` access regularly and restrict host group membership.

## Queue input review

- Flow, action, result, event, and dead-letter subjects are explicit and validated at startup when `NATS_URL` is set.
- Queue payloads use typed message structs before execution reaches the function gateway.
- Dead-letter routing exists for malformed or exhausted messages.
- Action results are correlated by `flow_id` and `action_id` before orchestration state is updated.

Residual risk:

- Message authentication and per-tenant queue partitioning are not yet implemented.
- Production NATS deployments should use credentials, TLS, and stream-level retention limits.

Recommended remediation:

1. Enable NATS credentials and rotate them on a schedule.
2. Terminate TLS for NATS and Postgres, even on internal networks.
3. Use per-environment or per-tenant stream naming once transport isolation becomes a requirement.
4. Review dead-letter queues during incident response and before retries.

## API and runtime safety review

- Tenant scoping is enforced at the repository and HTTP layers.
- Approval-gated flow start is available through `NYX_REQUIRE_FLOW_APPROVAL=true`.
- API key enforcement is available through `NYX_API_KEY`.
- Config validation now rejects invalid execution, browser, logging, and queue settings before service startup.
- Migrations support paired rollback SQL and `cmd/migrate -rollback N`.

Recommended remediation:

1. Keep `NYX_REQUIRE_RISKY_APPROVAL=true` in production.
2. Keep the HTTP rate limiter enabled unless you are behind an upstream gateway with stronger controls.
3. Rotate API keys and Supabase secrets through external secret management.
4. Publish the OpenAPI contract and pin client generation to tagged releases.

## Test coverage added in Phase 8

- Standard end-to-end coverage for in-memory flow creation, dispatch, execution, reporting, and SSE streaming.
- Build-tagged integration coverage for Postgres + NATS + orchestrator + executor worker.
- Config validation tests.
- Migration pair and rollback-loading tests.

## Recommended next hardening steps

1. Add signed or authenticated queue publishers/consumers for multi-service production deployments.
2. Add per-tenant queue subjects or stream partitioning if tenant isolation must extend to transport.
3. Run live Docker executor smoke tests under CI once container access is available.
4. Keep TLS termination examples under version control. A starter Caddy config now exists at [deploy/caddy/Caddyfile.example](/home/xacce/dev/nyx.ai/deploy/caddy/Caddyfile.example).
5. Schedule backup and restore drills with [scripts/backup-postgres.sh](/home/xacce/dev/nyx.ai/scripts/backup-postgres.sh) and [scripts/restore-postgres.sh](/home/xacce/dev/nyx.ai/scripts/restore-postgres.sh).
