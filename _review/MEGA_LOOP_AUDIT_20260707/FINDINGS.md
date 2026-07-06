# MEGA LOOP Findings — 2026-07-07

**Session:** MLA-20260707
**Mode:** record-only (no code fixes)
**Progress:** 35 discovery rounds + 15 adversarial passes COMPLETE
**Final:** See [FINAL_REPORT.md](./FINAL_REPORT.md)

## Summary Counts

| Severity | New (MLA) | Open from LBA | Fixed/Intentional |
|----------|-----------|---------------|-------------------|
| P0 | 0 | 0 (P0-004 = intentional) | 6 |
| P1 | 8 | 3 partial/open | 20+ |
| P2 | 12 | — | — |
| P3 | 9 | — | — |

---

## P0 — Critical

*None new. LBA-P0-004 classified Intentional (manual recovery for late payments).*

---

## P1 — High

| ID | Location | Description | Round | Status |
|----|----------|-------------|-------|--------|
| MLA-P1-001 | `payment_webhook_handler.go:155-157` | `ErrPaymentFulfillmentStale` → HTTP 500 → provider retry storm on stuck RECHARGING | R2, AV3 | **Confirmed** (94%) |
| MLA-P1-002 | `payment_fulfillment.go:261-262` | `alreadyProcessed` default returns nil for unhandled statuses | R2, AV2 | **Likely** (72%) |
| MLA-P1-003 | `video_gateway_service.go:52,787` | No production billing brake wired (`budget==nil`) | R3, R26 | **Intentional** (LBA-P1-019) |
| MLA-P1-004 | `video_gateway_service.go:708+` | Seedance smoke gate env/account-wide not per-user | R3 | **Open** (LBA-P1-021) |
| MLA-P1-005 | `client.ts:165-222` + `auth.ts:168-217` | Dual token refresh race (interceptor vs store timer) | R11, AV8 | **Confirmed** (90%) |
| MLA-P1-006 | `client.ts:208-214` | Interceptor refresh doesn't sync Pinia store | R11, AV7 | **Confirmed** (91%) |
| MLA-P1-007 | `PaymentView.vue:792-827` | WeChat JSAPI fallback creates duplicate order | R12, AV3 | **Confirmed** (88%) |
| MLA-P1-008 | `router/index.ts:794,880-884` | Backend mode blocks Stripe return without redirect preservation | R13, AV3 | **Confirmed** (89%) |
| MLA-P1-009 | `api_key_service.go:825-853` | Fresh quota check mitigates but no atomic reservation at request time | R5, R28 | **Partial** (LBA-P1-027) |
| MLA-P1-010 | gateway model scope enforcement | SupportedModelScopes may not block all gateway paths | R28 | **Likely** |

---

## P2 — Medium

| ID | Location | Description | Round | Status |
|----|----------|-------------|-------|--------|
| MLA-P2-001 | `drama_gateway_service.go:410-429` | Drama list post-filter pagination: wrong totals, sparse pages | R3, AV3 | **Confirmed** (96%) |
| MLA-P2-002 | `payment_webhook_handler.go:163-186` | Stripe/Airwallex cannot extract out_trade_no for instance pinning | R2 | **Likely** |
| MLA-P2-003 | `payment_fulfillment.go:295-296` | Fulfillment lock race returns nil silently | R2 | **Likely** |
| MLA-P2-004 | `video_gateway_service.go:684-690` | Safe-demo fails if mock provider unavailable | R3 | **Likely** |
| MLA-P2-005 | `client.ts:251-272` | Auth 401 on login clears existing valid session tokens | R11 | **Likely** |
| MLA-P2-006 | `KeysView.vue:1478-1488` | Status toggle treats exhausted/expired as enable-to-active | R14 | **Likely** |
| MLA-P2-007 | `KeysView.vue:1640+` | Error handling expects Axios shape; interceptor rejects plain object | R14 | **Confirmed** (84%) |
| MLA-P2-008 | `SettingsView.vue:7227+` | Parallel load/save race can overwrite form with stale data | R15 | **Likely** |
| MLA-P2-009 | `auth.ts:177-187` | Invalid token_expires_at → NaN schedule | R11 | **Likely** |
| MLA-P2-010 | `paymentFlow.ts:345-347` | Strict payAmount validation drops legacy recovery snapshots | R12 | **Likely** |
| MLA-P2-011 | `payment_order_expiry_service.go` | Background expiry worker untested | R18 | **CoverageGap** |
| MLA-P2-012 | `internal/payment/provider/stripe.go` | No stripe_test.go | R21 | **CoverageGap** |
| MLA-P2-013 | `.cursor/hooks/secret-scan.ps1:22-29` | Pre-push secret scan fails open when Python/scanner missing (`exit 0`) | R1 Bugbot+Security | **Confirmed** (88%) |

---

## P3 — Low

| ID | Location | Description |
|----|----------|-------------|
| MLA-P3-001 | `drama_gateway_service.go:353-392` | Silent AddTaskEvent failures |
| MLA-P3-002 | `payment_fulfillment.go:467-528` | Affiliate failure audit outside rolled-back tx |
| MLA-P3-003 | `payment_webhook_handler.go:104-107` | Truncated webhook body in verify logs |
| MLA-P3-004 | `video_handler.go:213-214` | Auth subject ok ignored (mitigated by middleware) |
| MLA-P3-005 | `KeysView.vue:1848` | No onUnmounted abort for loadApiKeys |
| MLA-P3-006 | `OAuthCallbackView.vue:242` | Duplicate sanitizeRedirectPath copy |
| MLA-P3-007 | `SettingsView.vue:7547+` | OAuth redirect URLs not validated in UI |
| MLA-P3-008 | golangci G104 exclude | Some swallowed errors intentional |
| MLA-P3-009 | Vitest stderr noise | Intentional error logs in passing tests |
