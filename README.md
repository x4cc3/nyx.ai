# NYX v2

NYX has been rebuilt as a Go-first autonomous security orchestration scaffold.

This repo now keeps only:

- `nyx.md` as the rebuild blueprint
- the Go implementation at the repo root
- the Next.js operator workspace in `web/`

## What exists now

This rebuild slice gives NYX:

- a Go API service
- a separate Go orchestrator service
- a Postgres-backed repository for runtime state
- a dedicated Go migration command
- a JetStream-backed flow dispatch path with poll fallback
- JetStream-backed action request/result transport
- event fanout and dead-letter subjects in JetStream
- a container-oriented executor manager with local and Docker modes
- per-profile execution policies with isolated per-action workspaces
- structured stdout/stderr capture, exit codes, and retry metadata for executor runs
- a supervised agent-runtime service with role prompts, task decomposition, retries, and escalation
- an OpenAI-backed planner plus action-policy path over the Responses API
- a real Go browser service with `chromedp` automation, screenshots, HTML snapshots, and session replay inputs
- a real Go search service behind `search_web` with DuckDuckGo HTML or SearxNG providers
- OpenAI embeddings-backed memory ranking with pgvector persistence and hash fallback for offline/test runs
- expanded REST workspace APIs with tenant scoping and approval review flows
- a legacy built-in operator workspace fallback served from `/workspace`
- structured logging, trace IDs, service metrics, and dedicated observability endpoints
- report exports in Markdown, JSON, and PDF
- config validation and startup sanity checks for services and queue settings
- migration rollback support through `cmd/migrate -rollback N`
- standard end-to-end coverage plus build-tagged Postgres/NATS integration coverage
- a root `Makefile`, container build recipe, and CI workflow for repeatable packaging
- a standalone Next.js operator UI in `web/`
- a workflow-oriented domain model:
  - `flows`
  - `tasks`
  - `subtasks`
  - `actions`
  - `artifacts`
  - `memories`
  - `findings`
  - `agents`
  - `executions`
- persisted SSE event streaming across processes
- a named function gateway
- a real Go browser service
- a semantic Go memory service
- `sqlc`-generated query code in `internal/dbgen`
- migration-backed schema and sqlc-compatible query files

## API

### Health

`GET /healthz`

### List functions

`GET /api/v1/functions`

### Create flow

`POST /api/v1/flows`

Example body:

```json
{
  "name": "Acme external assessment",
  "target": "https://app.example.com",
  "objective": "Map the workspace architecture and validate the execution loop."
}
```

### List flows

`GET /api/v1/flows`

### Get flow detail

`GET /api/v1/flows/{flow_id}`

### Start flow

`POST /api/v1/flows/{flow_id}/start`

When `NYX_REQUIRE_FLOW_APPROVAL=true`, this creates a pending approval instead of queueing immediately.

### Stream flow events

`GET /api/v1/flows/{flow_id}/events`

### Observability and reports

- `GET /metrics`
- `GET /readyz`
- `GET /api/v1/flows/{flow_id}/report?format=markdown|json|pdf`

### Workspace and approvals

- `GET /api/v1/approvals`
- `GET /api/v1/approvals/{approval_id}`
- `POST /api/v1/approvals/{approval_id}/review`
- `GET /api/v1/flows/{flow_id}/tasks`
- `GET /api/v1/flows/{flow_id}/subtasks`
- `GET /api/v1/flows/{flow_id}/actions`
- `GET /api/v1/flows/{flow_id}/artifacts`
- `GET /api/v1/flows/{flow_id}/findings`
- `GET /api/v1/flows/{flow_id}/agents`
- `GET /api/v1/flows/{flow_id}/memories`
- `GET /api/v1/flows/{flow_id}/executions`
- `GET /api/v1/flows/{flow_id}/approvals`
- `GET /api/v1/flows/{flow_id}/workspace`
- `GET /workspace`
- `GET /workspace/flows/{flow_id}`
  Legacy fallback UI routes served by the Go API when the old workspace bundle exists.

## Run

Start Postgres and NATS:

```bash
make infra-up
cp .env.example .env
```

Apply migrations:

```bash
set -a; source .env; set +a
make migrate
```

Rollback the most recent migration in a disposable or staging environment:

```bash
set -a; source .env; set +a
make rollback
```

Run the API:

```bash
set -a; source .env; set +a
make run-api
```

Run the orchestrator in another terminal:

```bash
set -a; source .env; set +a
make run-orchestrator
```

Run the executor worker in a third terminal:

```bash
set -a; source .env; set +a
make run-executor
```

Build stamped binaries:

```bash
make build VERSION=v0.1.0
```

Build container images:

```bash
make docker-build-api VERSION=v0.1.0
make docker-build-orchestrator VERSION=v0.1.0
make docker-build-executor VERSION=v0.1.0
make docker-build-executor-pentest
make docker-build-web
make docker-build-stack
```

Turn on the full containerized stack:

```bash
cp deploy/.env.compose.example deploy/.env
make compose-prod-up
```

`make compose-prod-up` builds the pentest worker image first, then starts Postgres, NATS, API, orchestrator, executor, and the Next.js operator UI.

The executor startup path now fails fast if the Docker socket is missing, if `NYX_EXECUTOR_IMAGE_FOR_PENTEST` is not built locally, or if `NYX_EXECUTOR_NETWORK_MODE=custom` points at a missing Docker network.

When `NYX_EXECUTOR_MODE=docker`, also set `DOCKER_SOCKET_GID=$(stat -c %g /var/run/docker.sock)` in the compose env file so the non-root `nyx` user can talk to the mounted Docker socket.

For a manual bring-up sequence, use:

```bash
cp deploy/.env.compose.example deploy/.env
make docker-build-executor-pentest
docker compose --env-file deploy/.env -f deploy/docker-compose.prod.yml up --build -d
```

Browser settings are controlled with:

- `NYX_BROWSER_MODE=auto|chromedp|http`
- `NYX_BROWSER_TIMEOUT=20s`
- `NYX_BROWSER_ARTIFACTS_ROOT=/tmp/nyx-browser`
- `NYX_BROWSER_HEADLESS=true`
- `NYX_BROWSER_EXECUTABLE=/path/to/chrome`
- `NYX_SEARCH_MODE=duckduckgo|searxng|disabled`
- `NYX_SEARCH_BASE_URL=...`
- `NYX_SEARCH_TIMEOUT=12s`
- `NYX_SEARCH_RESULT_LIMIT=5`

Executor worker-image settings are controlled with:

- `NYX_EXECUTOR_IMAGE=alpine:3.20`
- `NYX_EXECUTOR_IMAGE_FOR_PENTEST=nyx-executor-pentest:latest`
- `NYX_EXECUTOR_NETWORK_MODE=none|bridge|custom`
- `NYX_EXECUTOR_NETWORK_NAME=...`
- `NYX_EXECUTOR_ENABLE_NET_RAW=false`

Container-stack runbook:

- [docs/operations/compose-runbook.md](/home/xacce/dev/nyx.ai/docs/operations/compose-runbook.md)

Smoke-test the pentest worker image after building it:

- `make executor-smoke`

Operator API settings:

- `NYX_API_KEY=...`
- `NYX_DEFAULT_TENANT=default`
- `NYX_REQUIRE_FLOW_APPROVAL=true`
- `NYX_AGENT_RUNTIME_MODE=openai|auto|deterministic`
- `OPENAI_API_KEY=...`
- `OPENAI_MODEL=gpt-5.1-codex-mini`
- `OPENAI_REASONING_EFFORT=high`
- `OPENAI_MAX_OUTPUT_TOKENS=8000`
- `NYX_MEMORY_EMBEDDINGS_MODE=auto|openai|hash`
- `OPENAI_EMBEDDING_MODEL=text-embedding-3-small`
- `OPENAI_EMBEDDING_DIMS=1536`

Requests can send:

- `X-NYX-API-Key`
- `X-NYX-Tenant`
- `X-NYX-Operator`

Workspace frontend commands:

- `make web-install`
- `make web-dev`
- `make web-build`

The canonical operator UI is the standalone Next.js app under `web/`, run with `make web-dev` for local development or `cd web && npm run start` after a Next build. The Go API still exposes `/workspace` as a legacy compatibility surface backed by `web/dist` and the older server-rendered fallback bundle.

For real autonomous execution, set `OPENAI_API_KEY` and keep `NYX_AGENT_RUNTIME_MODE=openai` or `auto`. In `openai` mode the orchestrator fails fast if the key is missing, which is the intended production behavior. The planner generates the task graph, and the action policy chooses the next concrete function call inside each subtask from live observations.

Observability settings:

- `NYX_LOG_FORMAT=json|text`
- `NYX_LOG_LEVEL=debug|info|warn|error`
- `NYX_API_OBSERVE_ADDR=:9080`
- `NYX_ORCHESTRATOR_OBSERVE_ADDR=:9081`
- `NYX_EXECUTOR_OBSERVE_ADDR=:9082`

Then in another terminal:

```bash
curl -s http://localhost:8080/healthz | jq .
```

Create and queue a flow:

```bash
curl -s -X POST http://localhost:8080/api/v1/flows \
  -H 'Content-Type: application/json' \
  -d '{"name":"Acme external assessment","target":"https://app.example.com","objective":"Operate NYX as a standalone autonomous security system"}' | jq .
```

```bash
curl -N http://localhost:8080/api/v1/flows/<flow_id>/events
```

```bash
curl -s -X POST http://localhost:8080/api/v1/flows/<flow_id>/start | jq .
```

## Architecture direction

The new code is organized toward the target shape from `nyx.md`:

- `cmd/api`
- `cmd/executor`
- `cmd/orchestrator`
- `internal/agentruntime`
- `internal/httpapi`
- `internal/orchestrator`
- `internal/functions`
- `internal/queue`
- `internal/executor`
- `internal/services/browser`
- `internal/services/memory`
- `internal/domain`
- `internal/store`
- `db/queries`
- `internal/store/migrations`

## What is still missing

This is a real rebuild start, not the full final platform yet. It still needs:

- richer JetStream consumers and policies beyond the current flow/action/event transport
- production-grade executor isolation policies and profile-specific container images
- richer agent reasoning beyond the current supervised bootstrap runtime
- optional graph-sync support for memory relationships

Operational guidance lives in [docs/operations.md](docs/operations.md), and deployment examples live under [deploy](/home/xacce/dev/nyx.ai/deploy).

Regenerate the query package with:

```bash
sqlc generate
```

Phase 8 hardening notes live in [docs/hardening.md](docs/hardening.md), and release/versioning guidance lives in [docs/release.md](docs/release.md).
CI formatting and standard test coverage now live in [.github/workflows/ci.yml](/home/xacce/dev/nyx.ai/.github/workflows/ci.yml).

See [docs/architecture.md](docs/architecture.md).
