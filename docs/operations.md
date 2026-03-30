# NYX Operations

## Observability

Each service can expose a dedicated observability port:

- `NYX_API_OBSERVE_ADDR`
- `NYX_ORCHESTRATOR_OBSERVE_ADDR`
- `NYX_EXECUTOR_OBSERVE_ADDR`

Those servers expose:

- `GET /healthz`
- `GET /readyz`
- `GET /metrics`

The main API process also exposes:

- `GET /healthz`
- `GET /readyz`
- `GET /metrics`

## Logging

Structured logs are controlled with:

- `NYX_LOG_FORMAT=json|text`
- `NYX_LOG_LEVEL=debug|info|warn|error`

API responses include `X-NYX-Trace-ID`, and request logs capture that same trace id so operators can correlate HTTP requests with downstream events.

## Reports

Flow reports can be exported from:

- `GET /api/v1/flows/{flow_id}/report?format=markdown`
- `GET /api/v1/flows/{flow_id}/report?format=json`
- `GET /api/v1/flows/{flow_id}/report?format=pdf`

## Deployment Profiles

Container bring-up runbook:

- use [compose-runbook.md](/home/xacce/dev/nyx.ai/docs/operations/compose-runbook.md) for the full NYX stack, image build order, Docker socket requirements, and custom-network notes

Local profile:

- use [deploy/docker-compose.local.yml](/home/xacce/dev/nyx.ai/deploy/docker-compose.local.yml)
- enable the example observability ports
- mount local volumes for Postgres and NATS state

Production profile:

- use [deploy/docker-compose.prod.yml](/home/xacce/dev/nyx.ai/deploy/docker-compose.prod.yml) as a baseline
- set explicit secrets through environment or secret stores
- bind observability ports to private networks only
- run API, orchestrator, and executor as separate services

## Backup And Restore

Postgres backup:

```bash
set -a; source .env; set +a
pg_dump "$DATABASE_URL" > backups/nyx-$(date +%Y%m%d-%H%M%S).sql
```

Postgres restore:

```bash
set -a; source .env; set +a
psql "$DATABASE_URL" < backups/nyx-YYYYMMDD-HHMMSS.sql
```

NATS JetStream backup:

```bash
docker compose exec nats tar -C /data -czf - . > backups/nats-jetstream-$(date +%Y%m%d-%H%M%S).tgz
```

NATS JetStream restore:

```bash
docker compose down nats
tar -C /tmp/nyx-nats-restore -xzf backups/nats-jetstream-YYYYMMDD-HHMMSS.tgz
docker run --rm -v nyx_nats-data:/data -v /tmp/nyx-nats-restore:/restore alpine sh -lc 'cp -R /restore/. /data/'
docker compose up -d nats
```

## Operational Checks

Before starting a production flow:

- verify `GET /readyz` is healthy for API, orchestrator, and executor
- verify there are no stale pending approvals in `GET /api/v1/approvals`
- confirm Postgres migration state with `go run ./cmd/migrate`
- verify JetStream is enabled when using split action execution
