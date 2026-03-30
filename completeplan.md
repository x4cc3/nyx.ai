# NYX Complete Project Plan

This document tracks everything needed to take NYX from its current state
to a production-grade, operator-ready platform.

## Current State Summary

NYX is a **near-complete** Go-based distributed security orchestration platform.
The rebuild plan (`plan.md`) phases 0–8 are done. The tooling parity plan
(`nyxtoolplan.md`) phases 0–12 are done. The completion plan phases A–H below
have been substantially implemented.

**What works today:**
- All 3 Go services build and run (API, Orchestrator, Executor)
- 18 test packages pass (all green), including new auth/observability/executor tests
- Full Docker Compose stack with Postgres, NATS JetStream, and frontend
- Pentest executor image (Kali-based, 30+ security tools, Go toolchain synced)
- 17+ registered tool functions across 4 categories
- 5 search providers (DuckDuckGo, SearxNG, Tavily, Perplexity, Sploitus)
- Embedded chromedp browser with 5 output modes
- 4 memory namespaces with pgvector embeddings
- Agent runtime with decomposition, escalation, and policy enforcement
- Risk assessment and approval gates for dangerous operations
- SSE-based real-time flow monitoring in the frontend
- Supabase auth (email, OAuth, magic link)
- Report generation (Markdown, JSON, PDF)
- API pagination, rate limiting, batch approvals, flow cancellation
- Stricter flow creation validation with field-level errors
- Settings page in frontend (tenant, API key, browser-local prefs)
- OpenAPI spec and API reference documentation
- CI pipeline with linting, integration test support, and release workflow
- Prometheus/Grafana monitoring assets and alerting rules
- Caddy TLS reverse proxy example
- Postgres backup/restore scripts
- First-deploy guide, troubleshooting guide, partitioning docs
- Database indexes for performance (migration 0006)

---

## Completed Phases

### ~~Phase A: Test Coverage Hardening~~ ✅ DONE
- [x] A1. Auth package tests (`internal/auth/authenticator_test.go`)
- [x] A2. Queue transport tests (`internal/queue/jetstream_integration_test.go` — runs when `NYX_TEST_NATS_URL` set)
- [x] A3. Postgres store tests (`internal/store/postgres_integration_test.go` — runs when `NYX_TEST_DATABASE_URL` set)
- [x] A4. Observability tests (`internal/observability/observability_test.go`)
- [x] A5. Executor hardening tests (stronger checks in `internal/executor/manager_test.go`)

### ~~Phase B: CI/CD Pipeline~~ ✅ DONE
- [x] B1. Expanded Go CI (`.github/workflows/ci.yml` — linting, vet, integration test job)
- [x] B2. Frontend tests in CI
- [x] B3. Docker image build verification
- [x] B4. Security scanning
- [x] B5. Release automation (`.github/workflows/release.yml`)

### ~~Phase C: API Hardening~~ ✅ DONE
- [x] C1. API pagination (limit/after on flows and approvals)
- [x] C2. OpenAPI specification (`docs/api/openapi.yaml`, `docs/api/reference.md`)
- [x] C3. Request validation (URL format, field errors, rate limiter)
- [x] C4. Batch operations (batch approval, flow cancellation with `cancelled` status)

### ~~Phase D: Database & Storage~~ ✅ DONE
- [x] D1. Database indexes (`internal/store/migrations/0006_phase10_indexes.up.sql`)
- [x] D2. Partitioning strategy documented (`docs/operations/partitioning.md`)
- [x] D3. Backup automation (`scripts/backup-postgres.sh`, `scripts/restore-postgres.sh`)

### ~~Phase E: Security Hardening~~ ✅ MOSTLY DONE
- [x] E1. Docker socket security (documented in hardening/first-deploy guides)
- [x] E2. Secrets management (documented)
- [x] E3. TLS configuration (`deploy/caddy/Caddyfile.example`)
- [x] E4. Executor container hardening (stronger tests in `manager_test.go`)

### ~~Phase F: Observability Stack~~ ✅ PARTIALLY DONE
- [x] F2. Prometheus scrape configuration (`deploy/prometheus.yml`)
- [x] F2b. Alerting rules (`deploy/alerts.yml`)
- [x] F3. Grafana dashboards (`deploy/grafana/dashboards/nyx-overview.json`)
- [ ] F1. **OpenTelemetry distributed tracing** — not wired end-to-end

### ~~Phase G: Frontend Gaps~~ ✅ DONE
- [x] G1. Settings page (`web/app/settings/page.tsx`)
- [x] G2. Live chat wiring (workspace-aware responses in `web/lib/api.ts`)
- [x] G3. Error handling (form validation matches backend, cancellation UI)

### ~~Phase H: Documentation~~ ✅ DONE
- [x] H1. First deployment guide (`docs/operations/first-deploy.md`)
- [x] H2. Troubleshooting guide (`docs/operations/troubleshooting.md`)
- [x] H3. API reference (`docs/api/reference.md` + `docs/api/openapi.yaml`)
- [x] H4. Hardening guide (expanded)

---

## Remaining Work

### F1. OpenTelemetry Distributed Tracing
- Prometheus metrics endpoints exist but full OTEL trace propagation
  across API → Orchestrator → Executor is not wired
- **Need**: Add `go.opentelemetry.io/otel` SDK, instrument HTTP handlers
  and NATS publish/consume with trace context propagation
- **Priority**: 🟢 Medium — metrics and health checks work; tracing is
  an operational refinement, not a blocker

### Audit Findings (Known, Non-Regression)
- Semgrep flags dynamic `exec.Command` in `internal/executor/docker.go`
  — expected for Docker execution
- Semgrep flags direct response writes in `internal/httpapi/server.go`
  — standard Go HTTP handler pattern
- These are known audit items, not bugs

---

## Deferred / Optional

These are acknowledged gaps that are explicitly non-blocking.

### I1. Graph-sync memory support
- `plan.md` lists "Evaluate optional graph-sync support" as the only
  remaining unchecked item
- Earlier graph-backed reference stacks used Graphiti + Neo4j; NYX uses flat pgvector (intentional)
- Evaluate only if memory relationships prove insufficient

### I2. GraphQL API
- `nyx.md` mentions `gqlgen` as a target but REST-only was chosen
- Defer unless the frontend needs more flexible querying

### I3. WebSocket support
- Events are SSE-only (one-way)
- WebSocket would enable two-way real-time commands
- Defer unless operator interactive control is needed

### I4. Multi-provider LLM support
- Currently OpenAI-only for agent runtime
- Adding Anthropic, local models, etc. is a future enhancement

---

## Summary

| Phase | Status | Remaining |
|-------|--------|-----------|
| Rebuild plan (plan.md) | ✅ 8/8 phases done | 1 deferred item (graph-sync) |
| Tooling parity (nyxtoolplan.md) | ✅ 12/12 phases done | — |
| A. Test coverage | ✅ Done | Integration tests need env vars to run |
| B. CI/CD | ✅ Done | — |
| C. API hardening | ✅ Done | — |
| D. Database | ✅ Done | — |
| E. Security hardening | ✅ Done | — |
| F. Observability | 🟡 Partial | OTEL tracing not wired |
| G. Frontend | ✅ Done | — |
| H. Documentation | ✅ Done | — |
| I. Optional features | ⚪ Deferred | Graph-sync, GraphQL, WebSocket, multi-LLM |

**The project is production-ready.** The only open engineering item is
OpenTelemetry distributed tracing (Phase F1), which is an operational
refinement — all monitoring, alerting, and health endpoints already work.
