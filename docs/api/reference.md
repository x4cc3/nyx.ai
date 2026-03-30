# NYX API Reference

The canonical machine-readable API definition lives at [docs/api/openapi.yaml](/home/xacce/dev/nyx.ai/docs/api/openapi.yaml).

## Authentication

- `Authorization: Bearer <token>` for Supabase-backed auth
- `X-NYX-API-Key: <key>` for static API-key auth
- `X-NYX-Tenant: <tenant>` to scope requests
- `X-NYX-Operator: <operator>` for operator attribution

## Pagination

List endpoints support:

- `limit`
- `after`

Responses include `page_info.next_after` when another page exists.

## Error shape

```json
{
  "error": {
    "code": "invalid_flow",
    "message": "Flow validation failed",
    "field_errors": {
      "target": "Target must be a valid http(s) URL."
    }
  }
}
```
