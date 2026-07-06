# Round 24 — Ops WebSocket + Admin High-Risk

**Files:** `ops_ws_handler.go`, `ops_ws_ticket.go`, admin handlers

## Review

- WS ticket auth path has tests (7/4 fix)
- Ops disabled 404 → frontend redirect in client.ts

## Findings

No new P1. Admin bulk operations use AbortController in several views — inconsistent (P3).
