# Round 8 — Admin + Ops Dashboard

**Trigger:** loop wake (round 7 duplicate skipped)

## Findings (record only)

| ID | Sev | Finding | Location |
|----|-----|---------|----------|
| LBA-P2-026 | P2 | Admin video `RoutingEvents` `limit` query unbounded (cross-ref video R-P3-4) | `admin/video_handler.go:249-256` |
| LBA-P2-027 | P2 | Ops WS `OriginPolicyPermissive` allows broader cross-origin admin WS | `ops_ws_handler.go:40,538+` |
| LBA-P3-016 | P3 | Ops dashboard endpoints gated by `RequireMonitoringEnabled` — good; expensive queries bounded by `parseOpsTimeRange` default | `ops_dashboard_handler.go:22-27` |
| LBA-P3-017 | P3 | Ops error export allows `include_detail` query — admin-only detail leak risk in shared sessions | `ops_handler.go:367` |

## Round 8 outcome

+4 records (2 new cross-refs + 2 ops); open ~158+
