# Round 2 — Video Gateway

**Agent:** [Video gateway adversarial review](09abbef2-bd96-4e13-b747-eebbe886e04f)  
**Scope:** video_handler, admin/video_handler, video_gateway_*, drama_gateway, routes

## Summary

| Severity | Count | Top themes |
|----------|-------|------------|
| P0 | 3 | JWT create bypasses API-key daily trial cap; drama safe-demo can auto-route real; explicit provider_account_id pins real account |
| P1 | 4 | Non-atomic submit double-bill; no billing brake default; StaticBudgetGuard Charge no-op; smoke gate env-wide not per-user |
| P2 | 10 | rate_limit unenforced; upstream cancel stub; decrypt fail infinite retry; SSRF gaps; drama validation missing |
| P3 | 9 | provider inventory leak; mock-asset public enum; events verbatim; pagination bugs |

## P0 IDs (record only)

- **LBA-P0-005** — JWT `POST /video/tasks` bypasses `CreateDailyTrialTask` (`video_handler.go:184-208`, `video_gateway_service.go:585-601`)
- **LBA-P0-006** — Drama safe-demo auto-routes live Seedance (`drama_gateway_service.go:296-324`)
- **LBA-P0-007** — JWT create with explicit `provider_account_id` targets real accounts without trial (`video_handler.go:25-26`)

## P1 IDs (record only)

- **LBA-P1-018** — Worker submit non-atomic → duplicate upstream creates (`video_gateway_worker.go:331-356`)
- **LBA-P1-019** — No production billing brake by default (`video_gateway_billing.go`, `config.go`)
- **LBA-P1-020** — `StaticBudgetGuard.Charge` no-op (`video_gateway_budget_guard.go:55-59`)
- **LBA-P1-021** — Smoke gate env-wide, not per-user auth (`video_gateway_adapter.go:573-596`)

## Positive controls noted

Dedicated encryption key, PlainAPIKey redaction, Seedance SSRF/smoke gates on API-key path, task IDOR guard, usage log idempotency.
