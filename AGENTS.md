# AGENTS.md — NYX Codebase Guide for AI Agents

> This file gives any AI agent immediate context about the NYX autonomous security orchestration platform.

## What NYX Is

NYX is an autonomous pentesting orchestration platform. An operator creates a **Flow** (target + objective), and NYX autonomously plans tasks, dispatches specialist agents, executes security tools in sandboxed containers, and produces findings/reports. It uses LLM-driven planning with human-in-the-loop approval gates.

## Repository Layout

```
nyx.ai/
├── cmd/                        # Go service entry points
│   ├── api/                    # HTTP API server (port 8080)
│   ├── orchestrator/           # Flow orchestration loop
│   ├── executor/               # Action execution (Docker/local)
│   └── migrate/                # Database migration runner
├── internal/                   # Go packages (not importable externally)
│   ├── agentruntime/           # LLM agent loop: planning → subtask → tool calls
│   ├── auth/                   # Supabase JWT + API key authentication
│   ├── config/                 # All configuration via env vars (config.go)
│   ├── dbgen/                  # Auto-generated sqlc code — DO NOT EDIT
│   ├── domain/                 # Core types: Flow, Task, Action, Memory, etc.
│   ├── e2e/                    # End-to-end integration tests
│   ├── events/                 # SSE event streaming to frontend
│   ├── executor/               # Docker/local command execution manager
│   ├── functions/              # Tool registry: terminal, browser, search, file, etc.
│   ├── httpapi/                # HTTP server, routes, middleware
│   ├── ids/                    # ID generation utilities
│   ├── integration/            # Tagged integration tests
│   ├── memvec/                 # In-memory vector store for embeddings
│   ├── observability/          # Prometheus metrics
│   ├── openai/                 # OpenAI API client (chat, embeddings, action policy)
│   ├── orchestrator/           # Flow lifecycle: claim → plan → dispatch → complete
│   ├── queue/                  # NATS JetStream queue abstraction
│   ├── reports/                # Report generation from flow findings
│   ├── services/
│   │   ├── browser/            # Chromedp-based browser automation
│   │   ├── memory/             # Semantic memory search service
│   │   └── search/             # Web/deep/exploit/code search backends
│   ├── store/                  # Repository interface + Postgres/in-memory impls
│   │   ├── repository.go       # Repository interface definition
│   │   ├── memory.go           # In-memory implementation (tests)
│   │   ├── postgres.go         # Postgres implementation
│   │   ├── migrations/         # SQL migration files (0001–0007)
│   │   └── migrations.go       # Migration runner
│   └── version/                # Build version info
├── db/
│   └── queries/                # sqlc query files (.sql) — source of truth for dbgen
├── web/                        # Next.js 16 frontend (React 19, TypeScript)
│   ├── app/                    # App Router pages
│   │   ├── dashboard/          # Main dashboard
│   │   ├── scans/              # Flow list, new flow, flow detail, live scan, reports
│   │   ├── settings/           # Operator settings (auth-guarded)
│   │   ├── login/              # Supabase auth login
│   │   ├── register/           # Supabase auth registration
│   │   └── api/                # API route handlers
│   ├── components/             # React components
│   │   ├── auth/               # Auth forms
│   │   ├── common/             # ErrorBoundary, shared UI
│   │   ├── dashboard/          # Dashboard widgets
│   │   ├── i18n/               # Internationalization provider
│   │   ├── layout/             # AppShell, Sidebar, Header
│   │   └── ui/                 # Primitives (Button, Card, Badge, etc.)
│   ├── hooks/                  # useAuth, useCurrentUser, useDebounce, useOnClickOutside
│   └── lib/                    # api.ts, sse.ts, supabase/, i18n.ts, types.ts
├── deploy/                     # Docker Compose files (local + prod)
├── docs/                       # Architecture, operations, parity docs
├── scripts/                    # Helper scripts
├── .github/workflows/ci.yml   # CI pipeline
├── Dockerfile                  # Multi-service Go builder
├── Dockerfile.executor-pentest # Pentest executor with security tools
├── docker-compose.yml          # Development compose
├── sqlc.yaml                   # sqlc configuration
├── Makefile                    # Build/test/deploy commands
└── go.mod                      # Go 1.26, module name: nyx
```

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.26 (module: `nyx`) |
| Frontend | Next.js 16, React 19, TypeScript, Tailwind CSS 4 |
| Database | PostgreSQL 16 with pgvector extension |
| Message queue | NATS JetStream |
| Auth | Supabase (JWT) + NYX API key fallback |
| LLM | OpenAI-compatible API (configurable model/base URL) |
| Containers | Docker (executor sandboxing) |
| Browser | chromedp (headless Chromium) |
| DB codegen | sqlc v2 |
| CI | GitHub Actions (lint, test, integration, Docker build, security scan) |

## Core Domain Model

The entity hierarchy flows top-down:

```
Flow (top-level engagement: target + objective)
├── Agent (specialist: scanner, exploiter, reporter)
├── Task (high-level goal assigned to an agent role)
│   └── Subtask (atomic unit of work)
│       └── Action (tool invocation: terminal, browser, search, file)
│           ├── Execution (runtime metadata: Docker container, profile)
│           ├── Artifact (output: logs, screenshots, plans)
│           └── Memory (semantic observation stored with embeddings)
├── Finding (vulnerability discovered during the flow)
├── Approval (human gate: flow.start, action.risk_review)
└── Event (audit log: flow.status, action.started, etc.)
```

**Statuses**: `pending → queued → running → completed|failed|cancelled`

## Architecture — Service Boundaries

### API (`cmd/api`)
- HTTP server on `:8080`
- REST endpoints under `/api/v1/`
- SSE streaming at `/api/v1/flows/{id}/events`
- Middleware: auth (Supabase JWT / API key), CORS, rate limiting, request logging
- Serves workspace aggregate endpoint for frontend

### Orchestrator (`cmd/orchestrator`)
- Polls for queued flows, claims one at a time
- Runs the agent runtime loop: plan → create tasks → dispatch subtasks
- Publishes action requests to NATS, waits for results
- Uses OpenAI for LLM-driven planning and decision-making

### Executor (`cmd/executor`)
- Consumes action requests from NATS
- Executes tool calls (terminal commands, browser automation, file I/O, search)
- Supports Docker mode (sandboxed containers) and local mode
- Publishes results back via NATS

### Migrate (`cmd/migrate`)
- Runs SQL migrations up or rolls back N steps
- Usage: `go run ./cmd/migrate` or `go run ./cmd/migrate -rollback N`

## API Endpoints

```
GET    /healthz                          Health check
GET    /readyz                           Readiness check
GET    /metrics                          Prometheus metrics

GET    /api/v1/functions                 List available tool functions
GET    /api/v1/architecture              System architecture info
GET    /api/v1/flows                     List flows (paginated, tenant-scoped)
POST   /api/v1/flows                     Create a new flow
GET    /api/v1/flows/{id}                Flow detail (full aggregate)
POST   /api/v1/flows/{id}/queue          Queue a flow for execution
POST   /api/v1/flows/{id}/cancel         Cancel a running flow
GET    /api/v1/flows/{id}/events         SSE event stream
GET    /api/v1/flows/{id}/report         Generate/download report
GET    /api/v1/approvals                 List approvals (tenant-scoped)
POST   /api/v1/approvals/{id}/review     Approve or reject
GET    /api/v1/workspaces                Batch workspace data
GET    /workspace                        Single workspace by flow_id query param
```

**Auth headers**: `Authorization: Bearer <supabase-jwt>` or `X-NYX-API-Key: <key>`
**Tenant header**: `X-NYX-Tenant: <tenant-id>` (optional, defaults to config)

## Repository Interface (`internal/store/repository.go`)

```go
type Repository interface {
    Lifecycle                    // Init, Ping, Close
    FlowReader                   // GetFlow, ListFlows, FlowDetail, etc.
    FlowWriter                   // CreateFlow, QueueFlow, UpdateFlowStatus, ClaimNextQueuedFlow
    WorkUnitWriter               // CreateAgent/Task/Subtask/Action/Execution, Complete*, Update*
    ArtifactWriter               // AddArtifact, AddFinding
    MemoryReadWriter             // AddMemory, SearchMemories
    ApprovalStore                // CreateApproval, ReviewApproval, List*
    EventStore                   // RecordEvent, ListEvents
}
```

Two implementations: `MemoryStore` (testing) and `PostgresStore` (production).

## Available Tool Functions

The agent runtime can invoke these tools during flow execution:

| Function | Category | Description |
|----------|----------|-------------|
| `done` | control | Mark subtask complete |
| `ask` | control | Escalate to operator / request input |
| `terminal` | execution | Run shell command (Docker/local) |
| `terminal_exec` | execution | Direct command execution with target scope |
| `file` / `file_read` / `file_write` | workspace | Read/write/list workspace files |
| `browser` / `browser_html` / `browser_markdown` / `browser_links` / `browser_screenshot` | browser | Headless browser automation |
| `search_web` | search | Web search (DuckDuckGo/Tavily/SearXNG) |
| `search_deep` | search | Deep research (Perplexity/Tavily) |
| `search_exploits` | search | Exploit database search (Sploitus) |
| `search_code` | search | Code search |
| `search_memory` | memory | Semantic memory search with embeddings |

## Database

- **Engine**: PostgreSQL 16 + pgvector
- **Migrations**: `internal/store/migrations/0001_init.up.sql` through `0007_add_indexes.up.sql`
- **Code generation**: sqlc v2 — queries in `db/queries/*.sql`, output in `internal/dbgen/`
- **Connection pool**: Configurable via `NYX_DB_MAX_OPEN_CONNS`, `NYX_DB_MAX_IDLE_CONNS`, `NYX_DB_CONN_MAX_LIFETIME`, `NYX_DB_CONN_MAX_IDLE_TIME`

### Key tables
`flows`, `agents`, `tasks`, `subtasks`, `actions`, `executions`, `artifacts`, `memories` (with pgvector `embedding` column), `findings`, `approvals`, `events`

## Message Queue (NATS JetStream)

Streams and subjects:
- `NYX_FLOW_RUNS` / `nyx.flows.run` — flow dispatch (orchestrator consumes)
- `NYX_ACTION_REQUESTS` / `nyx.actions.execute` — action dispatch (executor consumes)
- `NYX_ACTION_RESULTS` / `nyx.actions.result` — action results (orchestrator consumes)
- `NYX_EVENTS` / `nyx.events.flow` — event broadcast
- `NYX_DLQ` / `nyx.dlq` — dead letter queue

## Configuration

All configuration is via environment variables loaded in `internal/config/config.go`. Key groups:

**Core**: `NYX_LISTEN_ADDR`, `DATABASE_URL`, `NATS_URL`, `NYX_SERVICE_NAME`
**Auth**: `NYX_API_KEY`, `SUPABASE_URL`, `SUPABASE_JWT_AUDIENCE`
**LLM**: `OPENAI_API_KEY`, `OPENAI_BASE_URL`, `OPENAI_MODEL`, `OPENAI_REASONING_EFFORT`
**Executor**: `NYX_EXECUTOR_MODE` (local|docker|auto), `NYX_EXECUTOR_IMAGE`, `NYX_EXECUTOR_NETWORK_MODE`
**Browser**: `NYX_BROWSER_MODE` (auto|chromedp|http), `NYX_BROWSER_HEADLESS`
**Search**: `NYX_SEARCH_MODE`, `TAVILY_API_KEY`, `PERPLEXITY_API_KEY`
**Embeddings**: `NYX_MEMORY_EMBEDDINGS_MODE` (auto|hash|openai), `OPENAI_EMBEDDING_MODEL`
**Frontend**: `NEXT_PUBLIC_SUPABASE_URL`, `NEXT_PUBLIC_SUPABASE_ANON_KEY`

See `internal/config/config.go` for the full list with defaults.

## Build & Run

```bash
# Backend
make build                    # Build all Go services to bin/
make test                     # Run all Go tests
make integration-test         # Run tagged integration tests
make vet                      # go vet
make fmt                      # gofmt

# Frontend
make web-install              # npm install
make web-dev                  # Next.js dev server
make web-build                # Production build

# Docker
make docker-build-stack       # Build all Docker images
make compose-prod-up          # Start full production stack
make compose-prod-down        # Stop production stack

# Database
make migrate                  # Run migrations
make rollback                 # Rollback 1 migration step

# Infrastructure (local dev)
make infra-up                 # Start Postgres + NATS locally
make infra-down               # Stop local infra
```

## CI Pipeline (`.github/workflows/ci.yml`)

5 jobs run on every push/PR:

1. **go-quality** — gofmt, go vet, golangci-lint, unit tests with 55% coverage floor
2. **go-integration** — Postgres + NATS service containers, store + queue integration tests
3. **web-build** — npm ci, jest tests, Next.js production build
4. **docker-build** — Build all 6 Docker images (api, orchestrator, executor, migrate, executor-pentest, web)
5. **security-scan** — govulncheck + Trivy filesystem scan

## Testing Conventions

- **Unit tests**: `*_test.go` files alongside source, run with `go test ./...`
- **Integration tests**: Use build tag `integration`, need real Postgres/NATS
- **Test env vars**: `NYX_TEST_DATABASE_URL`, `NYX_TEST_NATS_URL`
- **In-memory store**: `store.MemoryStore` for unit tests (no DB needed)
- **Frontend tests**: Jest + Testing Library (`web/__tests__/`), Playwright for E2E (`web/e2e/`)
- **Error handling in tests**: Use `//nolint:errcheck` for test-only helper calls, or check errors explicitly with `t.Fatalf`

## Code Style & Conventions

- **Go**: Standard `gofmt`, `go vet`, `golangci-lint` (errcheck, staticcheck, ineffassign, unused enabled)
- **No comments** on obvious code; comment only for clarification
- **Error handling**: Always check errors. Use `defer func() { _ = x.Close() }()` for defer close calls
- **IDs**: Generated via `internal/ids` package (ULID-based)
- **Pagination**: Cursor-based with `afterID` + `limit`, ordered by `(created_at DESC, id DESC)`
- **Frontend**: TypeScript strict mode, Tailwind CSS, Prettier formatting
- **Imports**: stdlib → external → internal (Go), alphabetical within groups

## Docker Images

The main `Dockerfile` is a multi-stage build with a `SERVICE` build arg:
```bash
docker build --build-arg SERVICE=api -t nyx-api .
docker build --build-arg SERVICE=orchestrator -t nyx-orchestrator .
docker build --build-arg SERVICE=executor -t nyx-executor .
docker build --build-arg SERVICE=migrate -t nyx-migrate .
```

`Dockerfile.executor-pentest` builds a specialized image with security tools (nmap, sqlmap, nikto, etc.).

The `web/Dockerfile` builds the Next.js frontend as a standalone deployment.

## Key Files to Read First

| Purpose | File |
|---------|------|
| Domain model | `internal/domain/types.go` |
| Store interface | `internal/store/repository.go` |
| Configuration | `internal/config/config.go` |
| HTTP routes | `internal/httpapi/server.go` |
| Agent runtime | `internal/agentruntime/runtime.go` |
| Tool registry | `internal/functions/registry.go` |
| Orchestrator loop | `internal/orchestrator/orchestrator.go` |
| Executor | `internal/executor/manager.go` |
| Queue abstraction | `internal/queue/nats.go` |
| Frontend API client | `web/lib/api.ts` |
| Frontend types | `web/lib/types.ts` |
| SSE streaming | `web/lib/sse.ts` |

## Common Tasks for AI Agents

**Adding a new tool function**: Add a `ToolSpec` entry in `internal/functions/registry.go` with a `domain.FunctionDef` and handler function. If it needs a new safety profile or execution mode, update `internal/functions/risk.go` and `internal/functions/scope.go`.

**Adding a new API endpoint**: Add a `mux.HandleFunc` in `internal/httpapi/server.go` → `Handler()`. Create the handler method on `*Server`. Auth middleware is applied globally.

**Adding a new DB table**: Create a new migration file in `internal/store/migrations/`, add queries in `db/queries/`, run `sqlc generate` to update `internal/dbgen/`, then implement in `postgres.go` and `memory.go`.

**Adding a frontend page**: Create a directory under `web/app/` following Next.js App Router conventions. Use `web/lib/api.ts` for backend calls, `web/lib/sse.ts` for streaming.

**Running locally**: `make infra-up && make migrate && make run-api` (terminal 1), `make run-orchestrator` (terminal 2), `make run-executor` (terminal 3), `make web-dev` (terminal 4).
