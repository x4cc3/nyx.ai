# NYX First Deployment Guide

This is the shortest production-first path to a working NYX deployment.

## 1. Prepare the host

- Install Docker Engine and Docker Compose v2.
- Create DNS for the NYX UI and API entrypoints.
- Provision TLS certificates or prepare ACME through Caddy or nginx.
- Keep `/var/run/docker.sock` available only on hardened hosts. Prefer rootless Docker, AppArmor/SELinux, and a dedicated executor host.

## 2. Configure secrets

Copy the example environment first:

```bash
cp deploy/.env.compose.example deploy/.env
```

Set at least these values in `deploy/.env`:

- `POSTGRES_PASSWORD`
- `OPENAI_API_KEY`
- `NYX_API_KEY`
- `NEXT_PUBLIC_SUPABASE_URL`
- `NEXT_PUBLIC_SUPABASE_ANON_KEY`
- `SUPABASE_URL`

For production, prefer secret injection through Docker secrets, Vault, or your platform secret store instead of committed `.env` files.

## 3. Pre-create infrastructure

If you want networked nested tool containers:

```bash
docker network create nyx-targets
```

Then set:

```bash
NYX_EXECUTOR_NETWORK_MODE=custom
NYX_EXECUTOR_NETWORK_NAME=nyx-targets
```

## 4. Build and launch

```bash
make docker-build-stack
docker compose --env-file deploy/.env -f deploy/docker-compose.prod.yml up -d
```

## 5. Configure TLS

Use [deploy/caddy/Caddyfile.example](/home/xacce/dev/nyx.ai/deploy/caddy/Caddyfile.example) as the default reverse-proxy entrypoint. Terminate TLS before the frontend and API, then forward:

- frontend to `frontend:3000`
- API to `api:8080`

## 6. Verify the platform

Check container health:

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.prod.yml ps
```

Check endpoints:

- `http://127.0.0.1:8080/healthz`
- `http://127.0.0.1:9080/metrics`
- `http://127.0.0.1:9081/healthz`
- `http://127.0.0.1:9082/healthz`
- `http://127.0.0.1:3000/`

## 7. Enable backups and observability

- Load [deploy/prometheus.yml](/home/xacce/dev/nyx.ai/deploy/prometheus.yml) and [deploy/alerts.yml](/home/xacce/dev/nyx.ai/deploy/alerts.yml) into your monitoring stack.
- Import [deploy/grafana/dashboards/nyx-overview.json](/home/xacce/dev/nyx.ai/deploy/grafana/dashboards/nyx-overview.json) into Grafana.
- Schedule [scripts/backup-postgres.sh](/home/xacce/dev/nyx.ai/scripts/backup-postgres.sh) and test [scripts/restore-postgres.sh](/home/xacce/dev/nyx.ai/scripts/restore-postgres.sh).
