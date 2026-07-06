# Round 9 — Unified Billing (LLM / Image / Video)

**Files:** `billing_service.go`, `gateway_service.go`, `video_gateway_billing.go`

## Verification

- V-5 Claude fallback prices — unit tests PASS
- Video CNY→USD via `ConvertBillingAmount` + `usd_cny_rate`
- usage_log_repo USD normalization for dashboard

## Findings

- **MLA-P1-003** reaffirmed — create-time budget guard not wired
- Large `gateway_service.go` (~8.7k lines) — high regression risk; thermo review deferred to R35

## Tests

`go test -tags=unit ./internal/service -run Billing -count=1` — PASS
