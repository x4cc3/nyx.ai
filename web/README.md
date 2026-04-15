# NYX Web App

This directory contains the Next.js operator UI for NYX.

## Stack

- Next.js 16
- React 19
- Tailwind CSS 4
- App Router

## Structure

```text
web/
  app/                       # App Router pages + API proxy route
  components/                # UI components and layout blocks
  lib/                       # API client helpers and utilities
  public/                    # static assets
  proxy.ts                   # session-aware route protection middleware
```

## Local development

```bash
cd web
npm install
npm run dev
```

## Production image

```bash
docker build -t nyx-web:latest .
```

For full-stack deployment, prefer the root Compose workflow documented in [../README.md](../README.md).

## Runtime behavior

- `app/api/[...path]/route.ts` proxies `/api/*` requests to the backend API.
- `proxy.ts` refreshes Supabase sessions and protects `/dashboard` and `/scans/*` routes.
- The web app is the primary operator interface for NYX.
- The Go API still exposes `/workspace` compatibility routes for fallback use.

## Environment

- `NYX_API_BASE_URL` (defaults to `http://localhost:8080/api`)
- `NYX_API_KEY` (optional backend API key, server-side only)
- `NYX_SITE_URL` (canonical site URL used for metadata and auth redirects)
- `NEXT_PUBLIC_SITE_URL` (public site URL for browser-side metadata)
- `NEXT_PUBLIC_SUPABASE_URL` (optional Supabase project URL)
- `NEXT_PUBLIC_SUPABASE_ANON_KEY` (optional Supabase anon key)

## Notes

- UI requests are proxied through `/api/*` on the Next.js server so backend secrets stay server-side.
- If you call the backend directly from a browser, ensure its CORS policy allows your origin.
