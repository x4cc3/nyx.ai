# NYX Capability Checklist

Legend:

- ✅ implemented and verified
- 🟡 implemented with follow-up refinement possible

## Runtime And Execution

| Capability | Evidence | Status |
| --- | --- | --- |
| Split API, orchestrator, and executor services | `cmd/api`, `cmd/orchestrator`, `cmd/executor` | ✅ |
| Docker-backed isolated execution | `internal/executor/docker.go` | ✅ |
| Tool-aware worker image selection | executor config + runtime metadata | ✅ |
| Configurable network policy | executor network config and compose env | ✅ |
| Approval gating for risky actions | `internal/functions/risk.go`, API approval handlers | ✅ |

## Operator Surface

| Capability | Evidence | Status |
| --- | --- | --- |
| Flow, approval, and workspace APIs | `internal/httpapi` | ✅ |
| Next.js operator UI | `web/` | ✅ |
| Error boundaries and safer frontend fetch behavior | `web/app/*/error.tsx`, `web/lib/api.ts` | ✅ |
| Report export endpoints | `internal/reports`, API handlers | ✅ |

## Search, Browser, And Memory

| Capability | Evidence | Status |
| --- | --- | --- |
| Browser automation | `internal/services/browser` | ✅ |
| Multi-provider search | `internal/services/search` | ✅ |
| Semantic memory retrieval | `internal/services/memory`, `internal/memvec` | ✅ |

## Operations

| Capability | Evidence | Status |
| --- | --- | --- |
| Health and metrics endpoints | `internal/observability`, service healthz | ✅ |
| Production compose baseline | `deploy/docker-compose.prod.yml` | ✅ |
| Pentest worker image build | `Dockerfile.executor-pentest` | ✅ |
| Docker socket access for non-root services | Dockerfile + compose `group_add` wiring | ✅ |
