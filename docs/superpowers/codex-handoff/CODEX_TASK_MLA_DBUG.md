# CODEX_TASK_MLA_DBUG — MEGA LOOP 已复核 Bug 分段修复（Dbug 彻夜执行）

> **执行模式：Dbug（Debug-in-Phases）。** 你是 Sub2API 仓库的执行代理（Codex）。
> **一次只做一个 Phase**：先复现/红测 → 最小修复 → 绿测 → 写审查包 → **单独 commit** → 再进入下一 Phase。
> **禁止**在一个 Phase 里顺手修其他 Phase 的项；**禁止**跳过红测直接改代码。
>
> **本任务书已由老板授权：允许修改 `backend/**`、`frontend/**`、`.cursor/hooks/**`（仅 DBug-8）。**
>
> **证据来源（只读，修复前先读）：**
> - 终审：[`_review/MEGA_LOOP_AUDIT_20260707/FINAL_REPORT.md`](../../_review/MEGA_LOOP_AUDIT_20260707/FINAL_REPORT.md)
> - 复核：[`_review/MEGA_LOOP_AUDIT_20260707/REPRO_CATALOG.md`](../../_review/MEGA_LOOP_AUDIT_20260707/REPRO_CATALOG.md)
> - 台账：[`_review/MEGA_LOOP_AUDIT_20260707/FINDINGS.md`](../../_review/MEGA_LOOP_AUDIT_20260707/FINDINGS.md)

---

## 0. 为什么用 Dbug 而不是一次修完

| 原则 | 说明 |
|------|------|
| **一段一段** | 每个 Phase = 1 个 bug（或强耦合的 2 个，如 token 双轨） |
| **先证后修** | 每 Phase 必须先写/跑失败测试或复现步骤，再改生产代码 |
| **最小 diff** | 只改本 Phase 列出的文件；禁止 refactor `gateway_service.go` / `SettingsView.vue` |
| **可回滚** | 每 Phase 一个 commit，message 带 `DBug-N` 与 MLA-ID |
| **可中断** | 任意 Phase 完成后可停止；审查包记录进度 |

**不要修的（复核已排除 / 有意设计）：**

- LBA-P0-004 过期订单迟到支付（人工恢复，2xx ack 为设计）
- MLA-P1-003 / LBA-P1-019 `budget==nil`（phase-2A 文档化）
- LBA-P1-020 StaticBudgetGuard.Charge no-op（phase-2B）
- MLA-P1-010 model scope（gateway 已有过滤，复核误判）
- MLA-P1-004 / LBA-P1-021 Seedance smoke gate 为 env/account 级（非鉴权 bypass，不单开 Phase）

---

## 1. Dbug 循环协议（每个 Phase 重复）

```text
Phase DBug-N:
  1. 读 REPRO_CATALOG + 对应源码，确认 bug 仍存在于当前 HEAD
  2. 写/跑红测（Test 或 Vitest）— 必须 FAIL 或记录「无法用单测复现」及原因
  3. 最小修复（仅本 Phase 文件列表）
  4. 跑本 Phase 窄测 + 相关包测试
  5. while 不绿: 修 → 重跑（同一失败最多 3 轮）
  6. 写 deliverables/2026-07-07-DBug-N-*.md（用 DELIVERABLE_TEMPLATE.md）
  7. git commit -m "fix(DBug-N): <MLA-ID> <一句话>"
  8. 进入 DBug-(N+1)；若遇停止条件 → 写 blocked 审查包并结束
```

**Windows 门禁（不要用 `pnpm run lint:check`，会缺 `sh`）：**

```powershell
# 后端（在 backend/）
$env:GOCACHE='D:\sub2api-trunk\.cache\go-build'
go test ./...                                    # 全量：仅 Phase 收尾或老板要求时
go test ./internal/handler/... -run <TestName> -count=1
golangci-lint run ./...

# 前端（在 frontend/）
npx eslint . --ext .ts,.vue --max-warnings=0
npx vue-tsc --noEmit
npx vitest run <相关spec路径> --reporter=basic
```

---

## 2. 红线

1. 不读/不打印 `.env`、API Key、JWT、cookie、provider 凭据。
2. 不 push、不 deploy、不 rebase、不 reset、不 clean。
3. 不触发真实付费 provider（Seedance/Stripe 真实扣款等）。
4. 状态词只用：`内部可用` / `可演示` / `待复核` / `已阻塞` — 禁止写「生产就绪」。
5. 遵守 `handler → service → repository` 与 depguard。
6. **MLA-P1-007**：保留 mobile→QR **fallback 能力**；只消除孤儿 pending 订单，不要删掉第二次 `createOrder` 的 H5 场景（见 `PaymentView.spec.ts:374-417`）。

---

## 3. Phase 总览（按顺序执行，不可跳号）

| Phase | MLA-ID | 域 | 优先级 | 预估 |
|-------|--------|-----|--------|------|
| **DBug-0** | — | 基线 | 必做 | 15min |
| **DBug-1** | MLA-P1-001 | 支付 webhook | P1 | 1h |
| **DBug-2** | MLA-P1-005/006 | Auth refresh | P1 | 2h |
| **DBug-3** | MLA-P1-007 | 支付 fallback | P1 | 1.5h |
| **DBug-4** | MLA-P1-008 | Router backend_mode | P1 | 45min |
| **DBug-5** | MLA-P2-001 | Drama 分页 | P2 | 2h |
| **DBug-6** | MLA-P2-007 | KeysView 错误 | P2 | 45min |
| **DBug-7** | MLA-REV-SUP1 | 视频列表轮询 | P2 | 1h |
| **DBug-8** | MLA-P2-013 | secret-scan hook | P2 | 30min |
| **DBug-9** | 可选 | 待复核项 | 有余力 | 可选 |

### 3.1 覆盖矩阵（对照 `_review/MEGA_LOOP_AUDIT_20260707/FINDINGS.md`）

**结论：所有「复核为真实 bug」的 9 项均已映射到 DBug-1～8；Likely/P3 按 Dbug 原则刻意不全修。**

| ID | 上一轮状态 | 本任务书 | 说明 |
|----|------------|----------|------|
| MLA-P1-001 | Confirmed | **DBug-1** | 必做 |
| MLA-P1-005 | Confirmed | **DBug-2** | 必做 |
| MLA-P1-006 | Confirmed | **DBug-2** | 与 005 同 PR |
| MLA-P1-007 | Confirmed | **DBug-3** | 必做；保留 H5→QR fallback |
| MLA-P1-008 | Confirmed | **DBug-4** | 必做 |
| MLA-P2-001 | Confirmed | **DBug-5** | 必做 |
| MLA-P2-007 | Confirmed | **DBug-6** | 必做（仅 handleSubmit；toggle 见 DBug-9b） |
| MLA-REV-SUP1 | 复核补充 | **DBug-7** | 视频主链路闭环 |
| MLA-P2-013 | Confirmed | **DBug-8** | 工具链 |
| MLA-P1-002 | Likely | **DBug-9a** | 可选 |
| MLA-P1-009 | Partial/LBA-P1-027 | **DBug-9c** | 可选；大改默认 skip |
| MLA-P2-006 | Likely | **DBug-9b** | 可选 |
| MLA-P2-012 | CoverageGap | **DBug-9d** | 可选；补 stripe_test 骨架 |
| MLA-P1-003 | Intentional | **不修** §0 | phase-2A |
| MLA-P1-004 | Open/LBA-P1-021 | **不修** §0 | env 级 smoke gate，非单用户 bypass |
| MLA-P1-010 | 复核误判 | **不修** §0 | gateway 已有 model_scope 过滤 |
| MLA-P2-002 | Likely | **不修/监视** §3.2 | 仅多实例 Stripe；单实例无影响 |
| MLA-P2-003 | Likely | **不修** §3.2 | 幂等并发设计 |
| MLA-P2-004 | Likely | **不修** §3.2 | safe-demo 依赖 mock 为预期 |
| MLA-P2-005 | Likely | **DBug-9e** | 可选；可与 DBug-2 同批评估 |
| MLA-P2-008 | Likely | **不修** §3.2 | SettingsView 大文件，禁止本任务 refactor |
| MLA-P2-009 | Likely | **DBug-9e** | 可选；可与 DBug-2 同批（NaN expiry） |
| MLA-P2-010 | Likely/部分误判 | **不修** §3.2 | 当前 spec 均含 payAmount |
| MLA-P2-011 | CoverageGap | **DBug-9d** | 可选；补 expiry worker 测试 |
| MLA-P3-001～009 | P3 | **不修** §3.2 | 卫生/低优，不单开 Phase |
| LBA-P0-004 | Intentional | **不修** §0 | 人工恢复流程 |
| LBA-P1-019/020 | Intentional | **不修** §0 | phase-2 规划 |

### 3.2 刻意不在今夜修的范围（写进 CLOSEOUT「仍 open」）

- **支付多实例：** MLA-P2-002（Stripe out_trade_no 预定位）
- **Settings 巨型页：** MLA-P2-008（需专门任务，非 Dbug 最小 diff）
- **P3 卫生项：** MLA-P3-001～009（AddTaskEvent 静默、affiliate audit、日志截断等）
- **LBA 开放：** MLA-P1-004 smoke gate 粒度、MLA-P1-009 quota 原子预留（除非老板显式开 DBug-9c）

---

## Phase DBug-0 · 基线预检（只读 + 记录）

**目标：** 确认从哪一 Phase 开始、HEAD 是否可工作。

1. `git status --short`、`git rev-parse --short HEAD`、`git branch --show-current`
2. 读 `_review/MEGA_LOOP_AUDIT_20260707/FINAL_REPORT.md` 的 Confirmed 清单
3. 跑快速门禁快照（记录 exit code，不必全量 go test 若时间紧，但 DBug-1 前必须全绿基线）：
   - `go test ./internal/service -run Payment -count=1`
   - `npx vitest run src/router/__tests__/guards.spec.ts --reporter=basic`
4. 写 `deliverables/2026-07-07-DBug-0-baseline.md`（仅基线，无代码改动）
5. **停止条件①**：若 working tree 有与 MLA 无关的大块脏改动 → 审查包标 `blocked`，列出文件，不要开始 DBug-1

**本 Phase 不 commit**（无代码变更）。

---

## Phase DBug-1 · MLA-P1-001 支付 webhook stale RECHARGING → 500

**Bug：** `ErrPaymentFulfillmentStale` 未在 webhook handler 走 2xx，provider 无限重试。

| 项 | 内容 |
|----|------|
| **文件** | `backend/internal/handler/payment_webhook_handler.go`（主改） |
| **只读** | `backend/internal/service/payment_fulfillment.go:220-238` |
| **已有单测** | `TestAlreadyProcessed_StaleRechargingOrderReturnsRetryableError`（service 层） |

**红测（必须先做）：**

- 在 `payment_webhook_handler_test.go` 新增：mock/stub 使 `HandlePaymentNotification` 返回 `ErrPaymentFulfillmentStale` → 断言 HTTP **200**（当前应 FAIL 为 500）
- 参考同文件 `TestUnknownOrderWebhookAcksWithSuccess` 的 sentinel + 2xx 模式

**最小修复：**

```go
// payment_webhook_handler.go handleNotify 内，ErrPaymentRejected 分支之后：
if errors.Is(err, service.ErrPaymentFulfillmentStale) {
    slog.Error("[Payment Webhook] stale fulfillment, acking to stop retries", ...)
    writeSuccessResponse(c, resolvedProviderKey)
    return
}
```

**验收：**

- 新 handler 测试 PASS
- `go test ./internal/handler -run Payment -count=1` PASS
- `golangci-lint run ./internal/handler/...` 0 issues

**审查包：** `deliverables/2026-07-07-DBug-1-webhook-stale-recharging.md`
**Commit：** `fix(DBug-1): MLA-P1-001 ack stale RECHARGING webhook with 2xx`

---

## Phase DBug-2 · MLA-P1-005 + MLA-P1-006 统一 token refresh

**Bug：** Axios 401 拦截器与 `auth` store 定时器双轨 refresh；拦截器成功后 Pinia 不同步。

| 项 | 内容 |
|----|------|
| **文件** | `frontend/src/api/client.ts`、`frontend/src/stores/auth.ts`；可新增 `frontend/src/api/tokenRefresh.ts`（若需，保持单文件 <150 行） |
| **只读** | `frontend/src/api/auth.ts:295-310` |

**红测：**

- Vitest：模拟并发 401 + 断言 `/auth/refresh` 只调用 1 次（或 mock 计数）
- Vitest：拦截器 refresh 成功后 `useAuthStore().token` 与 localStorage 一致

**最小修复：**

1. 抽出 `refreshAccessTokenOnce(): Promise<RefreshResult>`，模块级 mutex / singleflight
2. `client.ts` 401 分支与 `auth.ts` `performTokenRefresh` **都调用它**
3. 成功时：写 localStorage **且** `$patch` auth store（token、refreshTokenValue、tokenExpiresAt）

**禁止：** 改后端 JWT 逻辑；改 login/logout 流程（除非测试证明 broken）

**验收：**

- 新/扩 vitest PASS
- `npx eslint` + `npx vue-tsc` PASS
- 现有 `auth.spec.ts` / client 相关 spec 仍 PASS

**审查包：** `deliverables/2026-07-07-DBug-2-token-refresh-unify.md`
**Commit：** `fix(DBug-2): MLA-P1-005/006 unify token refresh and sync Pinia`

---

## Phase DBug-3 · MLA-P1-007 支付 JSAPI fallback 孤儿订单

**Bug：** JSAPI 失败后 `resetPayment` + `attemptMobileQrFallback` 再 `createOrder`，首单 pending 未取消。

| 项 | 内容 |
|----|------|
| **文件** | `frontend/src/views/user/PaymentView.vue`（主改） |
| **只读** | `PaymentView.spec.ts:374-417`（H5→QR 二次 create **是产品需求**） |

**红测：**

- 区分两条路径：
  - **保留：** `WECHAT_H5_NOT_AUTHORIZED` → 二次 create（现有 spec 仍 PASS）
  - **新增：** JSAPI invoke 失败（已有 order_id）→ **不应**产生第二个 pending 无取消

**最小修复（择一，优先 A）：**

- **A（推荐）：** JSAPI 失败 fallback 时复用当前 `order_id`，仅换 `payment_type`/通道参数重新拉起支付，不 `resetPayment` 清掉 orderId
- **B：** fallback 前调用 cancel order API（若后端有）再 create
- **C：** fallback 时 `payment_source` 标记关联首单，后端幂等（改动面大，**本 Phase 禁止**）

**验收：**

- 扩展 `PaymentView.spec.ts`：JSAPI fail 路径 `createOrder` 调用次数符合预期
- `TOO_MANY_PENDING` 场景不再因 JSAPI fallback 轻易触发

**审查包：** `deliverables/2026-07-07-DBug-3-payment-jsapi-orphan-order.md`
**Commit：** `fix(DBug-3): MLA-P1-007 avoid orphan pending order on JSAPI fallback`

---

## Phase DBug-4 · MLA-P1-008 backend_mode Stripe 回跳

**Bug：** `BACKEND_MODE_ALLOWED_PATHS` 缺 stripe 路由；未登录跳转 `/login` 丢失 query。

| 项 | 内容 |
|----|------|
| **文件** | `frontend/src/router/index.ts` |
| **只读** | `frontend/src/router/__tests__/guards.spec.ts` |

**红测：**

- guards.spec：backend_mode + 未登录访问 `/payment/stripe?order_id=1&resume_token=t` → 应允许 **或** 跳 login 且 `redirect` 含 fullPath

**最小修复：**

1. `BACKEND_MODE_ALLOWED_PATHS` 增加 `'/payment/stripe'`、`'/payment/stripe-popup'`（与 `/payment/airwallex` 同级）
2. `880-884` 行 backend_mode 未登录拦截改为 `next({ path: '/login', query: { redirect: to.fullPath } })`（与 892-897 一致）

**验收：** guards.spec 扩展用例 PASS；eslint + vue-tsc PASS

**审查包：** `deliverables/2026-07-07-DBug-4-backend-mode-stripe-return.md`
**Commit：** `fix(DBug-4): MLA-P1-008 backend_mode preserve Stripe return URL`

---

## Phase DBug-5 · MLA-P2-001 Drama 列表分页 total

**Bug：** `ListDramaTasks` 内存过滤后 `total=len(out)`，分页错误。

| 项 | 内容 |
|----|------|
| **文件** | `backend/internal/service/drama_gateway_service.go`；可能 `backend/internal/repository/video_gateway_repo.go` |
| **只读** | `drama_gateway_service.go:410-429` |

**红测：**

- `TestListDramaTasks_FilteredPagination`：seed N 条，filter 匹配 M 条，断言 total=M 且 page 填满逻辑正确

**最小修复：**

- 将 `dramaTaskMatchesFilters` 条件下推 SQL（或 repo 层 filter + COUNT）
- 返回 DB 级 `total`，不是 `len(out)`

**验收：** 新单测 PASS；`go test ./internal/service/... -count=1`；不破坏现有 video list 测试

**审查包：** `deliverables/2026-07-07-DBug-5-drama-list-pagination.md`
**Commit：** `fix(DBug-5): MLA-P2-001 drama list filtered pagination total`

---

## Phase DBug-6 · MLA-P2-007 KeysView 错误信息

**Bug：** `handleSubmit` 读 `error.response?.data?.detail`，axios 拦截器 reject plain object。

| 项 | 内容 |
|----|------|
| **文件** | `frontend/src/views/user/KeysView.vue` |
| **参考** | 同文件 `handleDelete:1664` 已用 `error?.message`；项目 `extractApiErrorMessage` |

**红测：**

- 新建或扩展 spec：mock create key 失败，断言 toast 展示后端 message

**最小修复：**

- `handleSubmit` catch 改用 `extractApiErrorMessage(err, t, ...)` 或与 delete 一致

**验收：** vitest PASS；手动路径：错误码能显示

**审查包：** `deliverables/2026-07-07-DBug-6-keysview-error-display.md`
**Commit：** `fix(DBug-6): MLA-P2-007 KeysView submit error message`

---

## Phase DBug-7 · MLA-REV-SUP1 视频任务列表状态轮询

**Bug：** `VideoTasksView` 仅 onMounted 拉一次；`VideoTaskDetailView` 有 2s 轮询。

| 项 | 内容 |
|----|------|
| **文件** | `frontend/src/views/admin/video/VideoTasksView.vue` |
| **参考** | `VideoTaskDetailView.vue:326-330` |

**红测：**

- 可选：组件 spec mock list API 调用次数，存在 running 任务时 >1 次

**最小修复：**

- 当 `tasks` 中存在非终态（pending/running/queued 等，与 `isTerminalStatus` 对齐）时，每 3–5s `loadTasks()`；`onUnmounted` clearInterval
- 或：`document.visibilitychange` 回前台 refresh（可与轮询二选一，优先轮询）

**禁止：** 改视频创建/计费后端

**验收：** vitest（若有）PASS；eslint + vue-tsc PASS

**审查包：** `deliverables/2026-07-07-DBug-7-video-tasks-list-polling.md`
**Commit：** `fix(DBug-7): MLA-REV-SUP1 poll running tasks on video tasks list`

---

## Phase DBug-8 · MLA-P2-013 secret-scan hook fail-closed

**Bug：** Python/扫描器缺失时 hook exit 0；`failClosed: false`。

| 项 | 内容 |
|----|------|
| **文件** | `.cursor/hooks/secret-scan.ps1`、`.cursor/hooks.json` |

**红测：**

- 文档化：模拟无 Python 时 exit 应为 2（可 PowerShell 脚本自测记录于审查包）

**最小修复：**

1. `hooks.json` → `"failClosed": true`
2. `secret-scan.ps1:22-29` → 缺 Python 或 scanner → Write-Error + **exit 2**
3. 同步 `.cursor/SETUP.md` 描述（若存在且不一致）

**验收：** 有 Python 时 scan 仍 PASS；无 Python 时 push 被 block（记录于审查包，不真 push）

**审查包：** `deliverables/2026-07-07-DBug-8-secret-scan-fail-closed.md`
**Commit：** `fix(DBug-8): MLA-P2-013 secret-scan hook fail closed when tools missing`

---

## Phase DBug-9 · 可选（彻夜有余力才做）

**不在今夜必做清单。** 每项单独 commit，仍走 DBug 协议。

| ID | 条件 | 动作 |
|----|------|------|
| MLA-P1-002 | 能构造非法 status | `alreadyProcessed` default 改返回 error + 测试 |
| MLA-P1-009 | 老板要 quota 硬保证 | 设计 pre-request reservation（**大改，默认 skip**） |
| MLA-P2-006 | 读完 backend toggle API | KeysView toggle 对 expired 状态 UX |
| stripe_test | 时间充裕 | `backend/internal/payment/provider/stripe_test.go` 骨架 |

若做 DBug-9 任一项，审查包单独命名 `DBug-9a-*`。

---

## 4. 彻夜结束收尾（全部完成的 Phase 跑完后）

1. 全量门禁：
   ```powershell
   cd backend; go test ./...; golangci-lint run ./...
   cd frontend; npx eslint . --ext .ts,.vue --max-warnings=0; npx vue-tsc --noEmit; npx vitest run --reporter=basic
   ```
2. 写总收口：`deliverables/2026-07-07-MLA-DBUG-CLOSEOUT.md`（列出完成的 DBug-N、跳过项、仍 open 的 Likely）
3. 更新 `_review/MEGA_LOOP_AUDIT_20260707/TRUTH_MATRIX.md` 中对应 MLA 行 → Fixed（仅文档，注明 commit hash）
4. **不 push**（除非老板另行指令）

---

## 5. 停止条件（遇一即停当前 Phase 并写 blocked）

| # | 条件 |
|---|------|
| ① | 基线 working tree 脏且非本任务改动 |
| ② | 红测无法写且无法手动复现 → 审查包标 `blocked`，**不要瞎修** |
| ③ | 同一 Phase 门禁失败 3 轮 |
| ④ | 修复需要改 gateway_service / 大范围 SettingsView |
| ⑤ | 需要真实 provider 凭据才能验证 |

---

## 6. 进度跟踪（Codex 自行更新）

在 `deliverables/2026-07-07-MLA-DBUG-PROGRESS.md` 维护：

```markdown
| Phase | Status | Commit | 审查包 |
|-------|--------|--------|--------|
| DBug-0 | done | — | `docs/superpowers/codex-handoff/deliverables/2026-07-07-DBug-0-baseline.md` |
| DBug-1 | done | `414ff2e9` | `docs/superpowers/codex-handoff/deliverables/2026-07-07-DBug-1-webhook-stale-recharging.md` |
| DBug-2 | done | `704c8f4c` | `docs/superpowers/codex-handoff/deliverables/2026-07-07-DBug-2-token-refresh-unify.md` |
| DBug-3 | done | `0b8eee6c` | `docs/superpowers/codex-handoff/deliverables/2026-07-07-DBug-3-payment-jsapi-orphan-order.md` |
| DBug-4 | done | `2ac31bf5` | `docs/superpowers/codex-handoff/deliverables/2026-07-07-DBug-4-backend-mode-stripe-return.md` |
| DBug-5 | done | `35d49031` | `docs/superpowers/codex-handoff/deliverables/2026-07-07-DBug-5-drama-list-pagination.md` |
| DBug-6 | done | `b5350c38` | `docs/superpowers/codex-handoff/deliverables/2026-07-07-DBug-6-keysview-error-display.md` |
| DBug-7 | done | `5f53444f` | `docs/superpowers/codex-handoff/deliverables/2026-07-07-DBug-7-video-tasks-list-polling.md` |
| DBug-8 | done | `83c71550` | `docs/superpowers/codex-handoff/deliverables/2026-07-07-DBug-8-secret-scan-fail-closed.md` |
| Closeout reinforcement | done | `42b94ca2` | `docs/superpowers/codex-handoff/deliverables/2026-07-08-MLA-DBUG-CLOSEOUT.md` |
```

每完成一 Phase 更新一行。

---

## 7. 一句话 North Star

**让已复核的真实 bug 逐个有测试、有 commit、有审查包——不追求一夜修完所有 Likely，追求每个修过的点可证明、可回滚。**
