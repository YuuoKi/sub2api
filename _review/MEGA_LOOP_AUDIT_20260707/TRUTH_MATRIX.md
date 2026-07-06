# MEGA LOOP Truth Matrix — 2026-07-07

**Branch:** `wujie/video-capture-moat-20260702`
**Commit:** `a3adde7b` (fix S3 CNY display rate propagation)
**Baseline:** LBA 2026-07-04 + S3 loop 2026-07-06

## Gate Snapshot (Phase 0)

| Gate | Result | Notes |
|------|--------|-------|
| `go test ./...` | PASS | backend, ~169s |
| `go test -tags=integration ./internal/repository/...` | PASS | 4.9s |
| `golangci-lint run ./...` | PASS | 0 issues (cache write warning only) |
| `npx vitest run` | PASS | 105 files / 610 tests |
| `npx eslint` / `vue-tsc` | PASS | Phase 0 run |
| `make secret-scan` | PASS | no high-confidence findings |

---

## LBA P0 (7 items)

| ID | Description | Status | Evidence |
|----|-------------|--------|----------|
| LBA-P0-001 | markCompleted zero-row UPDATE | **FixedSinceLBA** | `payment_fulfillment.go:379-380` returns error; `TestMarkCompleted_RejectsWhenNotRecharging` |
| LBA-P0-002 | confirmPayment Get failure → nil | **FixedSinceLBA** | `payment_fulfillment.go:88-94`; `TestConfirmPayment_MissingOrder_ReturnsError` |
| LBA-P0-003 | Legacy sub2_{id} wrong order | **FixedSinceLBA** | `payment_fulfillment.go:96-98` out_trade_no binding; regression test |
| LBA-P0-004 | Expired-beyond-grace ack 2xx, not fulfilled | **Intentional** | `ErrPaymentAfterExpiry` + audit; manual recovery required — ops gap not code bug |
| LBA-P0-005 | JWT video bypass trial cap | **FixedSinceLBA** | `video_handler.go:231` `EnforceRealProviderTrial`; admin-only bypass |
| LBA-P0-006 | Drama safe-demo → live Seedance | **FixedSinceLBA** | `SafeDemoOnly` + mock force `video_gateway_service.go:684-690` |
| LBA-P0-007 | JWT provider_account_id pins real | **FixedSinceLBA** | Admin-only `video_handler.go:233-236` |

---

## LBA Open P1 (3 items from VERIFY.md)

| ID | Description | Status | Evidence |
|----|-------------|--------|----------|
| LBA-P1-019 | Production billing brake default | **Open** | `video_gateway_service.go` `budget == nil`; phase-2A by design |
| LBA-P1-020 | StaticBudgetGuard.Charge no-op | **Intentional** | `video_gateway_budget_guard.go:55-59` documented phase-2B |
| LBA-P1-027 | Stale quota_used in auth cache | **PartiallyFixed** | `CheckAPIKeyQuotaAndExpiryFresh` at `api_key_auth.go:160`; no atomic reservation |

---

## S3 Claimed Fixes (2026-07-06)

| Item | Status | Evidence |
|------|--------|----------|
| CNY double conversion trap | **FixedSinceLBA** | `formatByCurrency`, `currency=CNY` specs; admin `usd_cny_rate` on stats APIs |
| balance_charged_at claim + rollback | **FixedSinceLBA** | `chargeForVideo` + `releaseVideoBalanceClaim` `video_gateway_billing.go:152-197` |
| Admin production gate on JWT CreateTask | **FixedSinceLBA** | `RequireSeedanceProductionAuthorization = true` `video_handler.go:236` |

---

## LBA P1 API Keys (ROUND_05) — Re-verify

| ID | Description | Status | Evidence |
|----|-------------|--------|----------|
| LBA-P1-027 | Stale quota cache | **PartiallyFixed** | Fresh DB read when quota/expires configured |
| LBA-P1-028 | USD limits fail-open on DB error | **FixedSinceLBA** | `checkAPIKeyRateLimits` fail-closed when limits configured |
| LBA-P1-029 | USD limits skipped cache nil | **FixedSinceLBA** | `billing_cache_service.go:533-538` |
| LBA-P1-030 | RPM fail-open Redis | **FixedSinceLBA** | `checkRPM` returns `ErrBillingServiceUnavailable` on incErr |
| LBA-P1-031 | Model scopes not enforced | **NeedsReview** | gateway path — see MLA-P1-010 |
| LBA-P1-032 | Inactive group on gateway | **FixedSinceLBA** | `api_key_auth.go:112-115` group inactive check |

---

## LBA Frontend P1 (ROUND_01) — Re-verify

| ID | Description | Status | Evidence |
|----|-------------|--------|----------|
| LBA-P1-009 | Login open redirect | **FixedSinceLBA** | `sanitizeRedirectPath` `LoginView.vue:494` |
| LBA-P1-010 | EmailVerify redirect | **FixedSinceLBA** | sanitize on navigation |
| LBA-P1-011 | OAuth tokens from URL fragment | **FixedSinceLBA** | oauthFragment utility (7/4 fix commit) |
| LBA-P1-012 | Stripe client_secret in URL | **PartiallyFixed** | sessionStorage for secrets; URL may still carry order_id |
| LBA-P1-013 | clientSecret in localStorage | **FixedSinceLBA** | `paymentFlow.ts` uses sessionStorage for secrets |
