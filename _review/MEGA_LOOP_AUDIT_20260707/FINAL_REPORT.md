# MEGA LOOP Final Report — Sub2API Bug Audit

**Date:** 2026-07-07 (UTC+8)
**Branch:** `wujie/video-capture-moat-20260702` @ `a3adde7b`
**Scope:** 35 discovery rounds + 15 adversarial passes · record-only · no code changes
**Output dir:** `_review/MEGA_LOOP_AUDIT_20260707/`

---

## Executive Summary

Sub2API is in materially better shape than the Jul 4 LOOP audit: **all 6 exploitable LBA P0s are fixed** with regression tests; S3 CNY/billing claim/production gate fixes hold. **No new P0 security bypass** was confirmed.

**7 confirmed bugs** remain (5 P1 backend/frontend, 2 P2), plus **12 likely** items and **intentional/deferred** gaps (late-payment manual recovery, phase-2 budget guard).

**Highest priority:** payment webhook retry storm on stale RECHARGING (`MLA-P1-001`), then frontend token refresh unification and payment flow fixes.

---

## Gate Status (Phase 0)

| Check | Result |
|-------|--------|
| `go test ./...` | PASS |
| `go test -tags=integration ./internal/repository/...` | PASS |
| `golangci-lint run ./...` | PASS (0 issues) |
| `vitest run` | PASS (105 files / 610 tests) |
| `eslint` + `vue-tsc` | PASS |
| `secret-scan` | PASS |

---

## Confirmed Bugs (fix recommended)

### MLA-P1-001 [P1] Stale RECHARGING webhook → HTTP 500

- **位置：** `backend/internal/handler/payment_webhook_handler.go:155-157`, `payment_fulfillment.go:228-238`
- **根因：** `ErrPaymentFulfillmentStale` 未列入 2xx sentinel，提供商无限重试
- **最小修复：**
  1. 在 handler 增加 `errors.Is(err, service.ErrPaymentFulfillmentStale)` 分支
  2. 记录 ERROR + 指标/告警
  3. 返回 2xx 停止重试
  4. 可选：暴露 admin 恢复 API
- **回归测试：** `TestWebhook_StaleRecharging_Acks2xx`
- **置信度：** 94% · **误判风险：** 低

---

### MLA-P1-005 [P1] 双轨 token 刷新竞态

- **位置：** `frontend/src/api/client.ts:165-222`, `frontend/src/stores/auth.ts:168-217`
- **根因：** Axios 401 拦截器与 store 定时器各自独立调用 refresh
- **最小修复：**
  1. 抽取 `refreshAccessToken()` 单例 mutex
  2. interceptor 与 store 共用
  3. refresh token 旋转时保证串行
- **回归测试：** Vitest 并发 401 + timer 场景
- **置信度：** 90% · **误判风险：** 低

---

### MLA-P1-006 [P1] 拦截器刷新后 Pinia 不同步

- **位置：** `frontend/src/api/client.ts:208-214`
- **根因：** 仅写 localStorage，未更新 auth store
- **最小修复：** 刷新成功后 `authStore.$patch({ token, refreshTokenValue, tokenExpiresAt })`
- **回归测试：** 与 MLA-P1-005 合并
- **置信度：** 91% · **误判风险：** 低

---

### MLA-P1-007 [P1] 微信 JSAPI 失败创建重复订单

- **位置：** `frontend/src/views/user/PaymentView.vue:792-827,918-934`
- **根因：** fallback 路径 `resetPayment` 后再次 `createOrder`
- **最小修复：** 恢复轮询原订单或显式取消，禁止二次 create
- **回归测试：** PaymentView spec mock JSAPI fail
- **置信度：** 88% · **误判风险：** 低

---

### MLA-P1-008 [P1] backend_mode 阻断 Stripe 回跳

- **位置：** `frontend/src/router/index.ts:794,880-884`
- **根因：** Stripe 路由不在 `BACKEND_MODE_ALLOWED_PATHS`，登录重定向丢失 query
- **最小修复：**
  1. 添加 `/payment/stripe`, `/payment/stripe-popup` 到 allowlist
  2. backend-mode 登录保留 `redirect=to.fullPath`
- **回归测试：** `guards.spec.ts` 扩展
- **置信度：** 89% · **误判风险：** 低

---

### MLA-P2-001 [P2] Drama 列表分页总数错误

- **位置：** `backend/internal/service/drama_gateway_service.go:410-429`
- **根因：** 内存过滤后 `total=len(out)`，非 DB 总数
- **最小修复：** 过滤条件下推 SQL + `COUNT(*)`
- **回归测试：** `TestListDramaTasks_FilteredPagination`
- **置信度：** 96% · **误判风险：** 低

---

### MLA-P2-007 [P2] KeysView 错误信息丢失

- **位置：** `frontend/src/views/user/KeysView.vue:1640+`
- **根因：** 期望 Axios `error.response`，拦截器抛出 plain object
- **最小修复：** 统一读 `error.message` 或 `code`
- **回归测试：** KeysView handleSubmit error mock
- **置信度：** 84% · **误判风险：** 低

---

## Likely Bugs (monitor / P2 queue)

| ID | Summary | Confidence |
|----|---------|------------|
| MLA-P1-002 | alreadyProcessed default nil ack | 72% |
| MLA-P1-009 | Quota race without atomic reservation | 65% |
| MLA-P1-010 | Model scope enforcement gap | 60% |
| MLA-P2-002 | Stripe webhook out_trade_no extraction | 70% |
| MLA-P2-003 | Fulfillment lock silent nil | 68% |
| MLA-P2-005..010 | Auth/session/settings edge cases | 65-75% |

---

## Misjudgments / Intentional (do not treat as bugs)

| Item | Reason |
|------|--------|
| LBA-P0-004 | Late payment after expiry — 2xx ack + manual recovery by design |
| LBA-P1-019 | `budget==nil` — phase-2A documented |
| LBA-P1-020 | StaticBudgetGuard.Charge no-op — phase-2B |
| LBA-P1-003 legacy | Fixed with out_trade_no binding |
| S3 CNY / claim / gate | Verified fixed — see TRUTH_MATRIX |

---

## Historical LBA Status

| Category | Count |
|----------|-------|
| P0 fixed | 6 |
| P0 intentional | 1 (P0-004) |
| P1 fixed since Jul 4 | ~24 |
| P1 partial/open | 3 (019, 020, 027 partial) |

Full matrix: [TRUTH_MATRIX.md](./TRUTH_MATRIX.md)

---

## Recommended Fix Order

1. **MLA-P1-001** — payment ops / provider retry (1 day)
2. **MLA-P1-005 + MLA-P1-006** — auth stability (1-2 days)
3. **MLA-P1-007 + MLA-P1-008** — payment UX (1 day)
4. **MLA-P2-001** — admin drama list (1-2 days)
5. **MLA-P2-007** — keys UX (hours)
6. Coverage: stripe_test, payment_order_expiry_test, KeysView.spec

---

## Artifacts Index

| File | Purpose |
|------|---------|
| `FINDINGS.md` | Rolling ledger |
| `TRUTH_MATRIX.md` | LBA + S3 re-verify |
| `TRIAGE.md` | Phase 2 dedup |
| `ROUND_01.md` … `ROUND_35.md` | Discovery rounds |
| `ADVERSARIAL_01.md` … `ADVERSARIAL_15.md` | Adversarial passes |
| `REPRO_CATALOG.md` | Repro steps |

---

## Boundaries Observed

- No production code modified
- No push / deploy / real provider calls
- No `.env` / secrets read
- Plan file not edited

**Audit status: COMPLETE**
