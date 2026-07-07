# 审查包：DBug-4 — MLA-P1-008 backend_mode Stripe 回跳

> 执行者：Codex
> 完成时间：2026-07-08 01:43 +08:00
> 关联规划：[CODEX_TASK_MLA_DBUG.md](../CODEX_TASK_MLA_DBUG.md)
> 状态：`done`

---

## 1. 本任务做了什么（给 Claude / 老板看）

- 将 `/payment/stripe`、`/payment/stripe-popup` 加入 `BACKEND_MODE_ALLOWED_PATHS`，与 `/payment/airwallex` 同级，未登录 Stripe 回跳不再被 backend_mode 拦截。
- backend_mode 下未登录访问非白名单路由时，跳转 `/login` 携带 `redirect=to.fullPath`，与 requiresAuth 路由行为一致，支付 query（`order_id`、`resume_token`）不再丢失。

---

## 2. 改了哪些文件

| 文件 | 变更摘要 |
|------|----------|
| `frontend/src/router/index.ts` | 扩展白名单；`next({ path, query: { redirect } })` |
| `frontend/src/router/__tests__/guards.spec.ts` | 同步白名单常量；新增 Stripe 允许与 redirect 测试 |
| `docs/superpowers/codex-handoff/deliverables/2026-07-07-MLA-DBUG-PROGRESS.md` | DBug-4 → done |

---

## 3. 验收结果（必须可核对）

| 验收项 | 结果 | 证据 |
|--------|------|------|
| `/payment/stripe` backend_mode 未登录可访问 | pass | guards.spec 新测 |
| 非白名单跳转保留 fullPath | pass | `simulateBackendModePublicRedirect` 断言 |
| `vitest guards.spec.ts` | pass | 37 tests |
| eslint + vue-tsc | pass | exit 0 |

---

## 4. 验证命令与结果

```text
cd D:\sub2api-trunk\frontend
npx.cmd vitest run src/router/__tests__/guards.spec.ts --reporter=basic
# Tests  37 passed (37)
npx.cmd eslint src/router/index.ts src/router/__tests__/guards.spec.ts --ext .ts --max-warnings=0
npx.cmd vue-tsc --noEmit
```

---

## 5. 给 Claude 的前端接口说明（如有）

无 API 变更。Stripe/Airwallex 回跳 URL query 在 backend_mode 下可完整保留至登录后 `redirect` 恢复。

---

## 6. 风险与遗留

- 建议下一任务：**DBug-5** — MLA-P2-001 Drama 列表分页 total（`drama_gateway_service.go`）。

---

## 7. 阻塞项（若 status=blocked）

无。
