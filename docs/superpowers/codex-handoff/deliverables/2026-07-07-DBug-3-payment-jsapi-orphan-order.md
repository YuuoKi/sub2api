# 审查包：DBug-3 — MLA-P1-007 支付 JSAPI Fallback 孤儿订单

> 执行者：Codex
> 完成时间：2026-07-08 01:41 +08:00
> 关联规划：[CODEX_TASK_MLA_DBUG.md](../CODEX_TASK_MLA_DBUG.md)
> 状态：`done`

---

## 1. 本任务做了什么（给 Claude / 老板看）

- 修复微信 JSAPI 调起失败后在已有 `order_id` 时仍 `resetPayment` + 二次 `createOrder`，导致首单 pending 成为孤儿订单的问题。
- 采用任务书 Option A：JSAPI invoke 失败且订单已创建时，**保留原订单**继续轮询/恢复，不再走桌面 QR 二次下单。
- `WECHAT_H5_NOT_AUTHORIZED` 路径不变：首单 create 失败时仍允许二次 create（H5→QR 为产品需求，见 `PaymentView.spec.ts:374-417`）。
- 新增红测：`JSAPI invoke fail` 场景断言 `createOrder` 仅调用 1 次且 recovery snapshot 保留。

---

## 2. 改了哪些文件

| 文件 | 变更摘要 |
|------|----------|
| `frontend/src/views/user/PaymentView.vue` | JSAPI fail 不 reset；`shouldFallbackToDesktopQr` 在有 `existingOrderId` 时跳过 `WECHAT_JSAPI_FAILED` 二次 create |
| `frontend/src/views/user/__tests__/PaymentView.spec.ts` | 新增 JSAPI invoke fail 单测 |
| `docs/superpowers/codex-handoff/deliverables/2026-07-07-MLA-DBUG-PROGRESS.md` | DBug-3 → done |

---

## 3. 验收结果（必须可核对）

| 验收项 | 结果 | 证据 |
|--------|------|------|
| H5→QR 二次 create 仍 PASS | pass | `falls back to QR flow when mobile WeChat payment is unavailable` |
| JSAPI fail 不二次 create | pass | 新测 `createOrder` times = 1 |
| recovery 保留原订单 | pass | localStorage 含 `sub2_jsapi_123` |
| `vitest PaymentView.spec.ts` | pass | 7 tests |
| `eslint` + `vue-tsc` | pass | exit 0 |
| 未改后端 | pass | 仅 `PaymentView.vue` |

---

## 4. 验证命令与结果

```text
cd D:\sub2api-trunk\frontend

npx.cmd vitest run src/views/user/__tests__/PaymentView.spec.ts --reporter=basic
# Test Files  1 passed (1)
# Tests  7 passed (7)

npx.cmd eslint src/views/user/PaymentView.vue --ext .vue,.ts --max-warnings=0
# exit 0

npx.cmd vue-tsc --noEmit
# exit 0
```

---

## 5. 给 Claude 的前端接口说明（如有）

无 API 变更。行为变化：

- **JSAPI 已下单后 invoke 失败：** 保留 `paymentState` / recovery，展示场景错误，用户可在 `PaymentStatusPanel` 继续轮询或取消原单。
- **H5 未授权（create 失败）：** 仍二次 `createOrder` 拉桌面 QR（不变）。

---

## 6. 风险与遗留

- JSAPI 失败后用户需在原订单上完成支付或手动取消；不再自动切换 QR（避免孤儿单）。
- 建议下一任务：**DBug-4** — MLA-P1-008 `backend_mode` Stripe 回跳（`router/index.ts`）。

---

## 7. 阻塞项（若 status=blocked）

无。
