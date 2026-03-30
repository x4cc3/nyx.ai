# NYX Troubleshooting

## API returns `401`

- Confirm the frontend proxy is sending either `Authorization: Bearer ...` or `X-NYX-API-Key`.
- Check the browser cookies `nyx_api_key` and `nyx_tenant`.
- Verify `NYX_API_KEY`, `SUPABASE_URL`, and `SUPABASE_JWT_AUDIENCE`.

## Flow creation returns `invalid_flow`

- The target must be a valid `http://` or `https://` URL.
- `name`, `target`, and `objective` are all required now.
- The API returns field-level validation errors under `error.field_errors`.

## Flow start stays pending

- Check `GET /api/v1/approvals` for a `flow.start` approval.
- Review the approval through `POST /api/v1/approvals/{id}/review` or `POST /api/v1/approvals/batch`.
- Confirm the flow was not cancelled before approval.

## Queue transport is failing

- Inspect `docker compose ... logs nats orchestrator executor`.
- Verify `NATS_URL` resolves and JetStream is enabled.
- Confirm the `NYX_TEST_NATS_URL` integration tests pass in CI or locally.

## Executor fails preflight

- Confirm `/var/run/docker.sock` is mounted.
- Confirm `nyx-executor-pentest:latest` exists or `NYX_EXECUTOR_IMAGE_FOR_PENTEST` points to a built image.
- If `NYX_EXECUTOR_NETWORK_MODE=custom`, verify the named network exists before startup.

## Workspace shows stale state

- Check `/api/v1/flows/{id}/events` SSE output.
- Inspect `/api/v1/flows/{id}/workspace`.
- Confirm the request rate limiter is not returning `429`.

## Postgres issues

- Run `pg_isready`.
- Inspect migration state with `cmd/migrate`.
- Restore from the latest known-good backup with [scripts/restore-postgres.sh](/home/xacce/dev/nyx.ai/scripts/restore-postgres.sh).

## Useful commands

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.prod.yml logs api
docker compose --env-file deploy/.env -f deploy/docker-compose.prod.yml logs orchestrator
docker compose --env-file deploy/.env -f deploy/docker-compose.prod.yml logs executor
curl -s http://127.0.0.1:8080/healthz | jq .
curl -s http://127.0.0.1:9080/metrics
```
