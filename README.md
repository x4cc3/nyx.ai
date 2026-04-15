# NYX

NYX is a Go-first security orchestration platform with a standalone Next.js operator UI.

It runs security workflows as **flows** (`tasks -> subtasks -> actions`), executes tool calls through a controlled function gateway, persists evidence in Postgres, and streams live events to the UI.

## Authorized use

NYX is intended for authorized security testing and research.
Only run it against systems you own or have explicit permission to assess.

## What is in this repo

- Go services at repo root (`api`, `orchestrator`, `executor`, `migrate`)
- shared runtime code under `internal/`
- database queries under `db/queries` with `sqlc` output in `internal/dbgen`
- standalone Next.js operator UI under `web/`

## Core capabilities

- flow lifecycle management (`create -> approve/start -> run -> report`)
- JetStream-backed flow/action transport with DB-poll fallback for flows
- isolated action execution through the function gateway (`terminal`, `file`, `browser`, `search_web`, `search_memory`)
- approval gates for risky actions and optional flow-start approvals
- persisted events, artifacts, memories, findings, executions, and approvals
- report exports in Markdown, JSON, and PDF
- observability endpoints (`/healthz`, `/readyz`, `/metrics`)

## Service map

- `cmd/api` — REST API, workspace endpoints, approvals, reports, SSE
- `cmd/orchestrator` — flow execution, agent runtime, action dispatch
- `cmd/executor` — consumes action requests and executes gateway functions
- `cmd/migrate` — schema migrations and rollback
- `web/` — operator workspace UI

## API quick reference

### Health

- `GET /healthz`
- `GET /readyz`

### Core flows

- `GET /api/v1/flows`
- `POST /api/v1/flows`
- `GET /api/v1/flows/{flow_id}`
- `POST /api/v1/flows/{flow_id}/start`
- `POST /api/v1/flows/{flow_id}/cancel`
- `GET /api/v1/flows/{flow_id}/events`

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

### Reports

- `GET /api/v1/flows/{flow_id}/report?format=markdown|json|pdf`

## Quickstart (local)

### 1) Start infra and env

```bash
cp .env.example .env
make infra-up
```

### 2) Pick runtime mode

For OpenAI-backed planning/policy:

- set `OPENAI_API_KEY`
- keep `NYX_AGENT_RUNTIME_MODE=openai` (or `auto`)

For local/no-key development:

- set `NYX_AGENT_RUNTIME_MODE=deterministic`

### 3) Run migrations

```bash
set -a; source .env; set +a
make migrate
```

### 4) Run backend services (three terminals)

```bash
set -a; source .env; set +a
make run-api
```

```bash
set -a; source .env; set +a
make run-orchestrator
```

```bash
set -a; source .env; set +a
make run-executor
```

### 5) Run frontend

```bash
make web-install
make web-dev
```

UI: `http://localhost:3000`
API: `http://localhost:8080`

## Minimal flow walk-through

Create a flow:

```bash
curl -s -X POST http://localhost:8080/api/v1/flows \
  -H 'Content-Type: application/json' \
  -d '{"name":"Acme external assessment","target":"https://app.example.com","objective":"Validate execution loop and capture evidence"}' | jq .
```

Start it:

```bash
curl -s -X POST http://localhost:8080/api/v1/flows/<flow_id>/start | jq .
```

Stream events:

```bash
curl -N http://localhost:8080/api/v1/flows/<flow_id>/events
```

Download report:

```bash
curl -L "http://localhost:8080/api/v1/flows/<flow_id>/report?format=markdown"
```

## Containerized stack

Start the full stack:

```bash
cp deploy/.env.compose.example deploy/.env
make compose-prod-up
```

This builds the pentest worker image first, then starts Postgres, NATS, API, orchestrator, executor, and frontend.

Stop it:

```bash
make compose-prod-down
```

## Security and operations notes

- `NYX_EXECUTOR_MODE=docker` requires Docker socket access from orchestrator/executor.
- Set `DOCKER_SOCKET_GID=$(stat -c %g /var/run/docker.sock)` in your env when using docker executor mode.
- `NYX_EXECUTOR_NETWORK_MODE=none` is the safest default. `bridge`/`custom` enable networked terminal execution.
- `NYX_CORS_ALLOWED_ORIGINS=*` is a development default. Restrict it in shared or production environments.
- `NYX_API_KEY` can gate API access when bearer auth is not configured.
- Supabase auth is optional; configure `SUPABASE_URL`, `SUPABASE_JWT_AUDIENCE`, `NEXT_PUBLIC_SUPABASE_URL`, and `NEXT_PUBLIC_SUPABASE_ANON_KEY` to enable it.

## Useful make targets

- `make fmt`
- `make test`
- `make integration-test`
- `make build`
- `make docker-build-stack`
- `make executor-smoke`
- `make rollback`

## Project policies

- License: [LICENSE](LICENSE)
- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md)
- Security reporting: [SECURITY.md](SECURITY.md)
- Code of conduct: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)

## Development references

- Architecture: [docs/architecture.md](docs/architecture.md)
- API schema: [docs/api/openapi.yaml](docs/api/openapi.yaml)
- API reference: [docs/api/reference.md](docs/api/reference.md)
- Operations: [docs/operations.md](docs/operations.md)
- Compose runbook: [docs/operations/compose-runbook.md](docs/operations/compose-runbook.md)
- Hardening notes: [docs/hardening.md](docs/hardening.md)
- Release guide: [docs/release.md](docs/release.md)
- CI: [.github/workflows/ci.yml](.github/workflows/ci.yml)

## Current scope boundaries

NYX is usable now, but still evolving. Active areas include:

- richer queue policies and transport controls
- stronger production executor isolation profiles
- deeper agent-runtime planning/reasoning behavior
- extended memory relationship/sync features
