# NYX Compose Runbook

This runbook is the clean-room bring-up path for the full containerized NYX stack.

For first-time production setup, start with [docs/operations/first-deploy.md](first-deploy.md). For incident handling, use [docs/operations/troubleshooting.md](troubleshooting.md).

It covers:

- Postgres
- NATS JetStream
- API
- orchestrator
- executor
- Next.js operator UI

NYX does not run a separate scraper sidecar today. Browser automation stays inside the services through `chromedp`.

## Requirements

- Docker Engine with a working Unix socket at `/var/run/docker.sock`
- Docker Compose v2
- enough local disk for the pentest worker image and browser artifacts

The executor container needs all of the following for real tool-backed terminal execution:

- Docker CLI inside the container
- `/var/run/docker.sock` mounted from the host
- a built `NYX_EXECUTOR_IMAGE_FOR_PENTEST` image
- a pre-created Docker network when `NYX_EXECUTOR_NETWORK_MODE=custom`

## Build Order

1. Copy the production compose environment file.
2. Build the pentest worker image.
3. Start the stack.
4. Verify health checks.

The shortest supported path is:

```bash
cp deploy/.env.compose.example deploy/.env
make compose-prod-up
```

`make compose-prod-up` runs `make docker-build-executor-pentest` first so the executor startup preflight can verify the tooling image.

If you want the explicit manual sequence instead:

```bash
cp deploy/.env.compose.example deploy/.env
make docker-build-executor-pentest
docker compose --env-file deploy/.env -f deploy/docker-compose.prod.yml up --build -d
```

If you also want every app image built ahead of time:

```bash
make docker-build-stack
```

## Compose Environment

The production example in [deploy/.env.compose.example](../../deploy/.env.compose.example) defaults to:

- `NYX_EXECUTOR_MODE=docker`
- `NYX_EXECUTOR_IMAGE_FOR_PENTEST=nyx-executor-pentest:latest`
- `NYX_BROWSER_MODE=chromedp`
- `NYX_BROWSER_EXECUTABLE=/usr/bin/chromium`
- `NYX_WEB_API_BASE_URL=http://api:8080/api`

Change at least:

- `POSTGRES_PASSWORD`
- `OPENAI_API_KEY`
- `NYX_API_KEY`

## Custom Network Mode

If you want nested tool containers to join an existing Docker network:

```bash
docker network create nyx-targets
```

Then set in `deploy/.env`:

```bash
NYX_EXECUTOR_NETWORK_MODE=custom
NYX_EXECUTOR_NETWORK_NAME=nyx-targets
```

The executor startup preflight fails closed if the named network does not exist.

## Health Checks

After startup, verify the containers report healthy:

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.prod.yml ps
```

Key endpoints:

- API: `http://127.0.0.1:8080/healthz`
- API observability: `http://127.0.0.1:9080/healthz`
- orchestrator observability: `http://127.0.0.1:9081/healthz`
- executor observability: `http://127.0.0.1:9082/healthz`
- frontend: `http://127.0.0.1:3000/`

## Failure Modes

If the executor does not stay up, inspect its logs first:

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.prod.yml logs executor
```

Common causes:

- `/var/run/docker.sock` was not mounted
- `nyx-executor-pentest:latest` was not built
- `NYX_EXECUTOR_NETWORK_MODE=custom` references a missing network

## Shutdown

```bash
make compose-prod-down
```
