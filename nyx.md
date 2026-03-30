# NYX v2 Blueprint

## Purpose

Build NYX as a standalone Go-first platform for autonomous security operations.
The system should keep NYX's operator-console identity while owning its own
execution model, worker images, safety controls, and reporting pipeline.

## Product Direction

- keep the NYX visual identity in the web client
- use Go for the control plane, orchestration, executor, and migrations
- keep the React/TypeScript frontend for the operator workspace
- execute tools in isolated workers with clear network and risk controls
- treat observability, evidence, and approvals as first-class platform features

## Core Services

- `api` for tenant-aware REST endpoints, reports, approvals, and event streaming
- `orchestrator` for flow lifecycle, task planning, and action dispatch
- `executor` for isolated action execution and result publishing
- `migrate` for schema rollout and rollback
- `web` for the operator-facing workspace

## Domain Model

NYX centers the following runtime objects:

- `flows`
- `tasks`
- `subtasks`
- `actions`
- `artifacts`
- `memories`
- `findings`
- `agents`
- `executions`

## System Shape

`UI -> API -> queue -> orchestrator -> executor -> isolated tools/browser/search/memory -> store/reporting`

The API owns operator interaction.
The orchestrator owns state transitions and planning.
The executor owns isolated tool execution.
Shared storage holds flow state, evidence, memory, and reports.

## Execution Principles

1. Keep execution isolation explicit and observable.
2. Prefer deterministic interfaces for tool calls and report generation.
3. Keep scope checks and approval gates fail-closed.
4. Preserve tenant boundaries throughout the request and runtime path.
5. Keep deployment simple enough for local testing and production rollout.

## Safety Model

- networked execution must be opt-in and visible in metadata
- risky actions require approval when policy demands it
- workspaces stay isolated per action attempt
- health, metrics, and logs must be available for every service
- configuration must fail early when required dependencies are missing

## Delivery Priorities

1. Keep the API, orchestrator, executor, and frontend buildable.
2. Maintain the Docker-based execution path for real worker tooling.
3. Keep report generation and evidence storage consistent across services.
4. Improve operator visibility rather than hiding runtime behavior.
5. Expand capability in NYX-native modules instead of pulling in external repos.
