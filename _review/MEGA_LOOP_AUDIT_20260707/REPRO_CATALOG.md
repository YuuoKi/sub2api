# Repro Catalog — MEGA LOOP 2026-07-07

Record-only. Temporary repro scripts not committed per plan.

---

## MLA-P1-001 — Stale RECHARGING Webhook 500

**Type:** Integration test design
**Steps:**
1. Insert payment order status RECHARGING with `updated_at` > 2min ago
2. POST webhook success payload with matching out_trade_no
3. Assert HTTP 500 today; after fix assert 200 + audit/metric

**File ref:** `payment_fulfillment.go:228-238`, `payment_webhook_handler.go:155-157`

---

## MLA-P1-005 — Dual Token Refresh

**Type:** Vitest + mock axios
**Steps:**
1. Mount app with valid refresh_token
2. Fire two API calls returning 401 simultaneously
3. Fire `scheduleTokenRefresh` via timer
4. Count `POST /auth/refresh` — expect 1; observe 2+ today

**File ref:** `client.ts:165-222`, `auth.ts:168-217`

---

## MLA-P1-007 — Duplicate Payment Order

**Type:** Manual / component test
**Steps:**
1. Open PaymentView mobile WeChat path
2. Mock `wx.chooseWXPay` rejection
3. Network: second `POST /payment/orders` after fallback

**File ref:** `PaymentView.vue:792-827,918-934`

---

## MLA-P1-008 — Backend Mode Stripe Return

**Type:** Router unit test
**Steps:**
1. Set `backend_mode_enabled` true, unauthenticated
2. Navigate to `/payment/stripe?order_id=1&resume_token=x`
3. Observe redirect to `/login` without `redirect` query

**File ref:** `router/index.ts:794,880-884`

---

## MLA-P2-001 — Drama List Pagination

**Type:** Go unit test with memory repo
**Steps:**
1. Create 15 drama tasks, 5 match filter `drama_type=X`
2. `ListDramaTasks(page=1, size=10, filter=X)`
3. Assert `total` should be 5; today may return fewer rows with `total=len(page)`

**File ref:** `drama_gateway_service.go:410-429`

---

## Existing Regression Tests (reference)

- `TestMarkCompleted_RejectsWhenNotRecharging`
- `TestHandlePaymentNotification_LegacyFallback_RejectsOutTradeNoMismatch`
- `ContentWall.spec.ts` CNY no double convert
- `video_gateway_repo_test.go` balance_charged_at claim
