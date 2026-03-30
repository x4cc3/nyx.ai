# Web App Initial Plan

## Purpose
The web app is the primary control plane for NYX. This plan describes the first implementation pass for routing, data flow, and UI wiring. UI spec lives in `interface/`.

## MVP Scope
- Dashboard, New Scan, Live Scan, Report
- Polling updates for live scans (1-3s interval)
- Cost summary UI from `/scan/{id}/cost`
- No auth in MVP (placeholder only)

## Recommended Structure (Next.js App Router)
```
web/
  app/
    layout.tsx
    page.tsx                  # Dashboard
    scans/
      new/page.tsx            # New Scan
      [scan_id]/page.tsx      # Live Scan
      [scan_id]/report/page.tsx
  components/
    layout/LeftRail.tsx
    layout/RightRail.tsx
    chat/ChatThread.tsx
    chat/MessageCard.tsx
    scan/ScanList.tsx
    scan/ScanForm.tsx
    report/FindingsList.tsx
    cost/CostSummaryCard.tsx
    cost/PhaseCostBreakdown.tsx
    cost/TopCostStepsList.tsx
  lib/
    api.ts
    types.ts
    polling.ts
    format.ts
  styles/
    tokens.css
    globals.css
```

## API Wiring
Use the endpoints defined in `overview/web_api.md`.

- `POST /scan` -> start scan
- `GET /scan/{id}` -> status header
- `GET /scan/{id}/steps` -> chat stream
- `GET /scan/{id}/report` -> report page
- `GET /scan/{id}/cost` -> cost summary

## Data Flow
1. Dashboard lists scans (poll summary endpoint or fetch on load).
2. New Scan posts target and navigates to Live Scan view.
3. Live Scan polls status + steps and appends to chat.
4. Right rail polls cost summary and status.
5. Report page fetches final report and cost summary.

## State Model (MVP)
- Scan summary: `scan_id`, `target`, `status`, `current_step`, `started_at`
- Step item: `step_id`, `phase`, `decision`, `action`, `summary`, `created_at`
- Cost summary: `total_cost_usd`, `phase_breakdown`, `top_steps`

## Error and Empty States
- No scans yet -> empty state with New Scan CTA
- Scan not found -> 404 page
- API unavailable -> inline error banner + retry

## Checklist
- [x] Routes scaffolded for the four core pages
- [x] API client wrapper with typed responses
- [x] Polling utility for steps + status + cost
- [x] Chat thread renders step types correctly
- [x] Right rail surfaces cost summary + status
- [x] Report page shows findings + cost summary
- [x] Tokens loaded from `interface/style_guide.md`
- [ ] `/scans` list endpoint aligned with backend (or remove list call)
- [ ] Empty + error state design pass for dashboard + live scan
- [ ] Form validation (URL format) and disabled states
- [ ] Live scan footer actions (Pause / Cancel / Export)
- [ ] Tool output expanders + severity badges in chat
- [ ] Report export actions (JSON / Markdown)
- [ ] Minimal auth placeholder (optional)

## Remaining Plan (Implementation Order)
1. **API alignment**: confirm `/scans` endpoint or update list fetch to match backend.
2. **Live scan UX**: add footer actions, tool output expanders, and decision badges.
3. **Report UX**: export actions + severity styling.
4. **States**: consistent empty/error/loading treatments across pages.
5. **Validation**: URL validation + inline form feedback.
6. **Polish**: animations for message reveal + hover states for cards.
