# NYX Release Process

This is the current lightweight release process for NYX.

## Version source

NYX binaries expose version metadata through `internal/version`:

- `Version`
- `Commit`
- `BuildDate`

Default values are development placeholders and should be overridden at build time.

## Build command

Use linker flags to stamp the binaries:

```bash
GOCACHE=$PWD/.gocache go build \
  -ldflags "-X nyx/internal/version.Version=v0.1.0 -X nyx/internal/version.Commit=$(git rev-parse --short HEAD) -X nyx/internal/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  ./cmd/api ./cmd/orchestrator ./cmd/executor ./cmd/migrate
```

Or use the packaged repo target:

```bash
make build VERSION=v0.1.0
```

## Release checklist

1. Run `GOCACHE=$PWD/.gocache go test ./...`.
2. If infrastructure is available, run `go test -tags=integration ./internal/integration/...` with `NYX_TEST_DATABASE_URL` and `NYX_TEST_NATS_URL`.
3. Apply migrations in a staging environment with `go run ./cmd/migrate`.
4. Verify rollback safety with `go run ./cmd/migrate -rollback 1` in a disposable environment.
5. Build all binaries with stamped version metadata.
6. Optionally build container images with `make docker-build-api`, `make docker-build-orchestrator`, and `make docker-build-executor`.
7. Deploy API, orchestrator, and executor with the same version string.
8. Confirm `/healthz`, `/readyz`, `/metrics`, and `/api/v1/architecture` report the expected version.

## Operational notes

- Keep migration application separate from service startup.
- Use `NYX_REQUIRE_FLOW_APPROVAL=true` in environments where risky execution must be operator-gated.
- Prefer `NYX_EXECUTOR_MODE=docker` for production deployments.
- Keep Postgres and NATS backups aligned with the guidance in `docs/operations.md`.
