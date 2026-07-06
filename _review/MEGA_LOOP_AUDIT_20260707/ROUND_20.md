# Round 20 — Redis Cache / singleflight

**Files:** `billing_cache_service.go`, `billing_cache_service_singleflight_test.go`

## Verification

- balanceLoadSF merges concurrent loads — test PASS
- P1-028..030 fixes verified in R5/R20
- **MLA-P1-009** — quota still not atomically reserved pre-request

## Tests

`go test -tags=integration ./internal/repository/...` — PASS
