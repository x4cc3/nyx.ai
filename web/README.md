# NYX Web App

This folder hosts the NYX web application (UI).

Stack:
- Next.js frontend
- Tailwind CSS
- App Router-based operator UI for NYX

## Structure

```
web/
  app/                       # App Router pages + API proxy route
  components/                # UI components and layout blocks
  lib/                       # API client helpers and utilities
  hooks/                     # React hooks
  public/                    # static assets
  proxy.ts                   # UI Basic Auth proxy middleware
```

## Local Development

- `cd web && npm install`
- `npm run dev`

## Production Image

- `docker build -f Dockerfile -t nyx-web:latest .`
- `docker run --rm -p 3000:3000 --env-file ../nyx nyx-web:latest`

## Usage

- `app/api/[...path]/route.ts` forwards `/api/*` to the backend API server.
- Protected pages (`/dashboard`, `/scans/*`) enforce an authenticated session server-side via route `layout.tsx` guards.
- `proxy.ts` is active as Next.js Proxy middleware and can enforce optional Basic Auth + session checks.
- This app is the primary operator interface for NYX. The Go API's `/workspace` routes are legacy fallback behavior, not the main frontend delivery path.

## Environment

- `NYX_API_BASE_URL` (preferred; defaults to `http://localhost:8000/api`)
- `NYX_API_KEY` (preferred backend API key, server-side only)
- `NYX_SITE_URL` (canonical public site URL used for metadata/sitemap)
- `NYX_UI_USER` (optional; enables Basic Auth for UI/proxy)
- `NYX_UI_PASSWORD` (optional; enables Basic Auth for UI/proxy)
- `NYX_*` equivalents are still accepted for backward compatibility.

## Notes

- UI requests are proxied through `/api/*` on the Next.js server, so secrets stay server-side.
- If you call the backend directly, ensure it allows CORS from `http://localhost:3000`.
