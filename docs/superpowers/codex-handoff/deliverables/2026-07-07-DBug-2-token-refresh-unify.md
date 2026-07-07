# 审查包：DBug-2 — MLA-P1-005/006 统一 Token Refresh

> 执行者：Codex
> 完成时间：2026-07-08 01:26 +08:00
> 关联规划：[CODEX_TASK_MLA_DBUG.md](../CODEX_TASK_MLA_DBUG.md)
> 状态：`done`

---

## 1. 本任务做了什么（给 Claude / 老板看）

- 消除 Axios 401 拦截器与 `auth` store 定时器双轨 refresh 导致的重复 `/auth/refresh` 请求。
- 新增 `refreshAccessTokenOnce()`（singleflight），401 拦截器与 `performTokenRefresh` 共用同一路径。
- 刷新成功后同时写 localStorage **并** `$patch` Pinia（`token`、`refreshTokenValue`、`tokenExpiresAt`），修复拦截器刷新后 store 不同步问题。
- 红测覆盖：并发 refresh 只 POST 一次、401 刷新后 Pinia 与 localStorage 一致、并发 401 只触发一次 refresh。

---

## 2. 改了哪些文件

| 文件 | 变更摘要 |
|------|----------|
| `frontend/src/api/tokenRefresh.ts` | 新增 unified singleflight refresh + localStorage + Pinia sync |
| `frontend/src/api/client.ts` | 401 分支改用 `refreshAccessTokenOnce`，移除独立 isRefreshing 队列 |
| `frontend/src/stores/auth.ts` | 新增 `syncTokenRefreshResult`；`performTokenRefresh` 改用 unified helper |
| `frontend/src/api/__tests__/tokenRefresh.spec.ts` | 新增 singleflight + Pinia sync 红测 |
| `frontend/src/api/__tests__/client.spec.ts` | 扩展 401 Pinia 同步与并发 401 单 refresh 测试 |
| `docs/superpowers/codex-handoff/deliverables/2026-07-07-MLA-DBUG-PROGRESS.md` | DBug-2 → done |

---

## 3. 验收结果（必须可核对）

| 验收项 | 结果 | 证据 |
|--------|------|------|
| 红测：Pinia sync（修复前 FAIL） | pass | `tokenRefresh.spec.ts` 先 FAIL 后 PASS |
| 红测：并发 refresh 单次 POST | pass | `tokenRefresh.spec.ts` + `client.spec.ts` |
| `vitest` auth/client/tokenRefresh | pass | 36 tests passed |
| `eslint` 变更文件 | pass | exit 0 |
| `vue-tsc --noEmit` | pass | exit 0 |
| 未改后端 JWT / login / logout | pass | 仅前端 refresh 路径 |

---

## 4. 验证命令与结果

```text
cd D:\sub2api-trunk\frontend

npx.cmd vitest run src/api/__tests__/tokenRefresh.spec.ts src/api/__tests__/client.spec.ts src/stores/__tests__/auth.spec.ts --reporter=basic
# Test Files  3 passed (3)
# Tests  36 passed (36)

npx.cmd eslint src/api/client.ts src/api/tokenRefresh.ts src/stores/auth.ts --ext .ts --max-warnings=0
# exit 0

npx.cmd vue-tsc --noEmit
# exit 0
```

---

## 5. 给 Claude 的前端接口说明（如有）

无 API 契约变更。`/auth/refresh` 请求/响应格式不变。

- **行为变化：** 任意路径刷新 token 后，Pinia `useAuthStore().token` 与 localStorage `auth_token` 保持一致。
- **前端建议：** 无需改页面；依赖 store.token 的组件在 401 自动刷新后会拿到新 token。

---

## 6. 风险与遗留

- `authAPI.refreshToken()` 仍保留（经 apiClient），但 store 定时器已改走 `refreshAccessTokenOnce`；若其他地方直接调 `authAPI.refreshToken` 会走旧路径（当前仅 store 曾使用，已切换）。
- 建议下一任务：**DBug-3** — MLA-P1-007 支付 JSAPI fallback 孤儿订单（`PaymentView.vue`）。

---

## 7. 阻塞项（若 status=blocked）

无。
