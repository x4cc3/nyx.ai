# NYX v2 Architecture

## Goal

Rebuild NYX as a Go-first platform for autonomous security operations while keeping NYX's identity and mission.

## Current rebuilt slice

The repo now contains buildable split Go binaries:

- `api`
- `executor`
- `orchestrator`
- `agent-runtime`
- `function-gateway`
- `browser-service`
- `memory-service`
- `web` React operator workspace

The API and orchestrator now coordinate through persisted Postgres state and stored event records.
Schema changes are versioned under `internal/store/migrations`, and the repo includes a dedicated `cmd/migrate` binary for applying them outside service startup.
Flow dispatch, action execution transport, event fanout, and dead-letter routing can now move through NATS JetStream when `NATS_URL` is configured; otherwise NYX falls back to the existing DB poll loop for flows and direct local execution for actions.
The scripted planner/researcher/executor sequence has been replaced by a dedicated `internal/agentruntime` service that owns agent prompts, decomposition, retries, reassignment, supervision rules, and model-driven next-step selection.
When `NYX_AGENT_RUNTIME_MODE=openai` or `auto` with `OPENAI_API_KEY` configured, the planner stage calls the OpenAI Responses API to generate a structured NYX flow plan and the action policy calls the same API to pick the next concrete function inside each subtask from live tool results.

## Service map

### `api`

Handles:

- REST endpoints
- tenant-aware workspace APIs
- flow creation
- flow queueing
- approval review and risky execution gating
- flow detail queries
- SSE event streaming
- built-in HTML workspace pages
- serving the legacy `/workspace` compatibility surface
- architecture and function discovery
- report export and metrics surfaces

### `orchestrator`

Handles:

- flow lifecycle
- queued flow claiming or JetStream consumption
- task and subtask creation
- specialist agent sequencing
- event publication
- action request dispatch and result handling
- structured execution logs and metrics

### `executor`

Handles:

- action request consumption from JetStream
- function execution through the Go gateway
- action result publication back to the orchestrator
- dead-letter handoff on malformed or exhausted messages
- structured action logs and metrics

### `agent-runtime`

Handles:

- planner-driven task decomposition
- OpenAI-backed structured planning when configured
- prompt/template selection per role
- execution context assembly from flow state, memory, and available functions
- subtask retry policy
- reassignment and operator escalation after repeated failures
- agent/task/subtask state transitions

### API surface decision

NYX currently uses expanded REST rather than `chi` or GraphQL:

- tenant-aware collection endpoints under `/api/v1/flows/{id}/...`
- approval review endpoints under `/api/v1/approvals/...`
- a standalone Next.js operator workspace under `web/`
- a legacy `/workspace` compatibility surface in the Go API for older flows and local fallback behavior

This keeps the operator surface easy to inspect while the backend model is still moving.

### Observability

The rebuilt platform now includes:

- structured JSON or text logs across API, orchestrator, and executor
- trace IDs on HTTP requests via `X-NYX-Trace-ID`
- Prometheus-style `/metrics` output
- dedicated observability listeners for API, orchestrator, and executor
- report generation in Markdown, JSON, and PDF
- startup config validation for runtime, queue, and observability settings
- version metadata surfaced through health and architecture endpoints

### `executor-manager`

Handles:

- profile selection for terminal and file actions
- isolated workspace creation per action attempt
- timeout, retry, and resource policy selection per profile
- local fallback execution with structured stdout/stderr capture
- Docker-backed isolated execution when enabled

Current modes:

- `local`
- `docker`

Current profile behavior:

- `terminal` uses a longer timeout, two attempts, and a higher resource budget for controlled shell work
- `file` uses a shorter timeout, a single attempt, and path validation to keep writes inside the action workspace
- both profiles emit workspace path, exit code, duration, and captured stdout/stderr back through the function gateway

### `function-gateway`

Exposes named functions:

- `terminal`
- `file`
- `browser`
- `search_web`
- `search_memory`
- `done`
- `ask`

### OpenAI planner mode

The current LLM integration is now split into two runtime layers:

- OpenAI powers task decomposition and structured plan generation
- OpenAI also powers the per-subtask action policy that chooses the next function call from observations
- the Go runtime still owns tool execution, retries, event streaming, persistence, and approvals
- if the planner returns an invalid plan, NYX sanitizes it against the registered function set before execution
- if the action policy returns an invalid function, NYX falls back to the planner-provided step hint for that turn
- if `NYX_AGENT_RUNTIME_MODE=openai`, missing `OPENAI_API_KEY` is a startup error

### `search-service`

`search_web` is now backed by a real provider-driven search service:

- DuckDuckGo HTML mode by default
- optional SearxNG mode for self-hosted search
- structured summaries and top result URLs returned into the runtime memory/evidence path

### `browser-service`

Currently implemented with a real Go browser runtime:

- `chromedp` navigation in `auto` or `chromedp` mode
- full-page screenshots and saved HTML snapshots
- auth/session replay inputs for headers, cookies, local storage, and session storage
- HTTP fallback mode for constrained environments and tests

### `memory-service`

Currently backed by persisted memory records plus a provider-driven embedding pipeline:

- `text-embedding-3-small` can be used as the primary memory embedding model
- hash embeddings remain as the offline/test fallback
- pgvector-backed storage and cosine ordering in Postgres
- retention metadata applied on ingestion
- semantic ranking fallback in the in-memory repository
- optional graph sync still open

## Persistence layout

- `internal/store/migrations`
- `db/queries`
- `internal/dbgen`

`db/queries` is the source of truth for generated query code. `internal/dbgen` is produced by `sqlc`, and `internal/store` maps generated models into NYX domain types.

## Domain model

NYX v2 uses the workflow model from `nyx.md`:

- `flows`
- `tasks`
- `subtasks`
- `actions`
- `artifacts`
- `memories`
- `findings`
- `agents`
- `executions`

## Next implementation steps

1. Replace hash-based memory embeddings with a real embeddings provider.
2. Add deploy-profile CI and live Docker executor validation.
3. Extend queue transport security with stronger auth and tenant partitioning.
