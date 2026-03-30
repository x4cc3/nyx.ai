# NYX Execution Model

## Overview

NYX executes flows through a split-service model:

`UI -> API -> queue -> orchestrator -> executor -> isolated workers/services -> store`

## Runtime Contract

- the API accepts and validates operator requests
- the orchestrator decomposes flow work and dispatches actions
- the executor runs actions in isolated workspaces
- search, browser, and memory services provide supporting observations
- reports and findings are derived from persisted flow state and execution evidence

## Worker Runtime

NYX supports:

- `local` execution for lightweight local operation
- `docker` execution for isolated worker containers
- tool-aware image selection for general versus pentest-capable work

## Safety Requirements

- scope validation stays mandatory
- approvals gate risky actions when configured
- startup fails closed if Docker prerequisites are missing in Docker mode
- execution metadata records image, network mode, duration, and exit status

## Evidence Requirements

Each action should preserve enough data for an operator to reconstruct:

- what ran
- where it ran
- which image was used
- which network policy applied
- what stdout, stderr, artifacts, and findings were produced
