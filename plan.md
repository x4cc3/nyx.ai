# NYX Historical Build Plan

This is an archived planning document from the earlier platform migration. Keep it for project history; use [README.md](README.md) and [docs/architecture.md](docs/architecture.md) for current project documentation.

## Phase 0: Foundation

- [x] Delete the legacy Python/Next.js NYX implementation
- [x] Convert the repo into a Go module
- [x] Establish the core domain model:
  - [x] `flows`
  - [x] `tasks`
  - [x] `subtasks`
  - [x] `actions`
  - [x] `artifacts`
  - [x] `memories`
  - [x] `findings`
  - [x] `agents`
  - [x] `executions`
- [x] Create split service entrypoints for `api` and `orchestrator`
- [x] Add baseline docs in `README.md` and `docs/architecture.md`

Exit criteria:
- The repo builds as a Go project.
- The API and orchestrator are separate binaries.

## Phase 1: Persistence

- [x] Introduce a shared repository interface
- [x] Add an in-memory store for fast tests
- [x] Add a Postgres-backed store for runtime state
- [x] Add versioned SQL migrations
- [x] Add a dedicated migration command
- [x] Move schema/query layout toward `sqlc` compatibility
- [x] Add guardrail tests for migration and `sqlc` config paths
- [x] Generate `internal/dbgen` from `sqlc`

Exit criteria:
- Runtime state is persisted in Postgres.
- Schema changes are migration-driven.
- Query definitions are ready for code generation.

## Phase 2: Transport And Dispatch

- [x] Add JetStream-backed flow dispatch
- [x] Keep DB-poll fallback when NATS is not configured
- [x] Publish flow start requests from the API
- [x] Consume flow work in the orchestrator
- [x] Add JetStream subjects for action execution requests
- [x] Add JetStream subjects for execution results
- [x] Add JetStream-backed event fanout beyond flow dispatch
- [x] Add dead-letter or failed-message handling policy

Exit criteria:
- Flow dispatch works over JetStream.
- The system can still run locally without NATS.

## Phase 3: Executor Manager

- [x] Introduce an executor-manager abstraction
- [x] Add `local` executor mode
- [x] Add `docker` executor mode
- [x] Route `terminal` actions through the executor manager
- [x] Route `file` actions through the executor manager
- [x] Add per-profile executor specs
- [x] Add resource limits and timeout policy per profile
- [x] Add isolated workspace mounts for action execution
- [x] Capture structured stdout/stderr and execution metadata
- [x] Add retry and failure policy for executor runs

Exit criteria:
- Terminal and file actions are isolated behind the executor boundary.
- Docker mode is usable for controlled execution profiles.

## Phase 4: Agent Runtime

- [x] Keep a deterministic planner/researcher/executor loop as a bootstrap runtime
- [x] Replace the deterministic flow with a real agent-runtime service
- [x] Add agent state transitions and supervision rules
- [x] Add task decomposition and reassignment logic
- [x] Add subtask retry and escalation rules
- [x] Add execution context assembly per agent role
- [x] Add prompt/template management for each agent type

Exit criteria:
- NYX runs a real agentic loop rather than a fixed scripted sequence.
- Agent behavior is observable and recoverable.

## Phase 5: Browser And Memory Services

- [x] Replace the browser placeholder with real Go browser automation
- [x] Add screenshots and page snapshots
- [x] Add authenticated replay/session handling
- [x] Replace plain text memory search with embeddings-backed retrieval
- [x] Add `pgvector` to the Postgres runtime
- [x] Define memory ingestion, ranking, and retention rules
- [ ] Evaluate optional graph-sync support

Exit criteria:
- Browser actions produce real observations.
- Memory search is semantic rather than plain substring matching.

## Phase 6: API And Workspace Surface

- [x] Expose basic REST endpoints for flows and events
- [x] Add richer flow detail APIs for agents, actions, artifacts, and findings
- [x] Decide between expanded REST, `chi`, or GraphQL
- [x] Add auth and multi-tenant policy
- [x] Add operator approval flows for risky execution
- [x] Build a new workspace UI for flow-centric operation
- [x] Add task, subtask, action, finding, and artifact views

Exit criteria:
- Operators can manage full NYX workspaces from the UI/API.
- Access control exists for multi-user operation.

## Phase 7: Reports, Observability, And Ops

- [x] Add structured logging across all services
- [x] Add metrics and health coverage for API, orchestrator, queue, and executor
- [x] Add tracing/observability wiring
- [x] Add report generation from findings, artifacts, and memories
- [x] Add export formats for Markdown and PDF
- [x] Add local and production deployment profiles
- [x] Add backup/restore guidance for Postgres and NATS

Exit criteria:
- NYX is operable as a long-running system.
- Flows produce usable reports and observability signals.

## Phase 8: Hardening

- [x] Add integration tests for Postgres + NATS + orchestrator
- [x] Add end-to-end tests for flow creation, dispatch, execution, and streaming
- [x] Add security review for executor isolation and queue inputs
- [x] Add config validation and startup sanity checks
- [x] Add migration rollback testing
- [x] Add release/versioning process

Exit criteria:
- The platform is safe enough to iterate on without brittle manual steps.
- Core runtime paths are covered by integration and end-to-end tests.

## Current Priority Order

1. Evaluate optional graph-sync support if memory relationships need it.
2. Add deploy-profile CI coverage for Docker executor and release artifacts.
3. Extend transport hardening with queue auth and stronger tenant isolation.

## Notes

- Completed items reflect work already landed in the current repo state.
- This file should be updated as phases move forward so it stays the source of truth.
