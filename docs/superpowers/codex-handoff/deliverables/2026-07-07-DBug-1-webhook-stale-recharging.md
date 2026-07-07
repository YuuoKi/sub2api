# 审查包：DBug-1 — MLA-P1-001 Stale RECHARGING Webhook 2xx Ack

> 执行者：Codex
> 完成时间：2026-07-08 01:15 +08:00
> 关联规划：[CODEX_TASK_MLA_DBUG.md](../CODEX_TASK_MLA_DBUG.md)
> 状态：`done`

---

## 1. 本任务做了什么（给 Claude / 老板看）

- 修复支付 webhook 在订单卡在 `RECHARGING` 超过 2 分钟后仍返回 HTTP 500 的问题，导致 Stripe 等提供商无限重试。
- 在 `payment_webhook_handler.go` 为 `service.ErrPaymentFulfillmentStale` 增加与 `ErrPaymentAfterExpiry` / `ErrPaymentRejected` 同级的 2xx ack 分支，记录 ERROR 日志便于告警。
- 新增端到端 handler 测试 `TestWebhook_StaleRecharging_Acks2xx`：先红测确认修复前为 500，修复后为 200。
- 创建项目级执行代理 `.cursor/agents/mla-dbug-executor.md`，供后续 DBug-2～8 复用 Dbug 协议。
- Service 层 stale 检测逻辑（`payment_fulfillment.go:220-238`）未改动；已有 `TestAlreadyProcessed_StaleRechargingOrderReturnsRetryableError` 仍 PASS。

---

## 2. 改了哪些文件

| 文件 | 变更摘要 |
|------|----------|
| `backend/internal/handler/payment_webhook_handler.go` | `ErrPaymentFulfillmentStale` → `writeSuccessResponse` + ERROR 日志 |
| `backend/internal/handler/payment_webhook_handler_test.go` | 新增 `TestWebhook_StaleRecharging_Acks2xx` 端到端测试 |
| `.cursor/agents/mla-dbug-executor.md` | 新增 MLA Dbug 项目级 subagent |
| `docs/superpowers/codex-handoff/deliverables/2026-07-07-MLA-DBUG-PROGRESS.md` | DBug-1 → done |
| `docs/superpowers/codex-handoff/deliverables/2026-07-07-DBug-1-webhook-stale-recharging.md` | 本审查包 |

---

## 3. 验收结果（必须可核对）

| 验收项 | 结果 | 证据 |
|--------|------|------|
| 红测修复前 FAIL（500） | pass | `go test -tags=unit ... -run TestWebhook_StaleRecharging_Acks2xx` → `expected: 200 actual: 500` |
| 修复后 handler 测试 PASS | pass | 同上 → `ok .../internal/handler` |
| `go test -tags=unit ./internal/handler -run Payment` | pass | exit 0 |
| `golangci-lint run ./internal/handler/...` | pass | 0 issues |
| 仅改任务书列出的生产文件 | pass | 仅 `payment_webhook_handler.go` |
| 未读 `.env` / 未触发真实 provider | pass | sqlite 内存库 + stub provider |

---

## 4. 验证命令与结果

```text
cd D:\sub2api-trunk\backend
$env:GOCACHE='D:\sub2api-trunk\.cache\go-build'

# 红测（修复前）
go test -tags=unit ./internal/handler -run TestWebhook_StaleRecharging_Acks2xx -count=1
# FAIL: expected 200, actual 500, body="handle failed"

# 绿测（修复后）
go test -tags=unit ./internal/handler -run "Payment|TestWebhook_StaleRecharging" -count=1
# ok  	github.com/Wei-Shaw/sub2api/internal/handler	4.331s

golangci-lint run ./internal/handler/...
# 0 issues.
```

---

## 5. 给 Claude 的前端接口说明（如有）

本 Phase 无 API / 字段 / 路由变更。前端支付轮询与 `RECHARGING` 展示逻辑不变。

- **新/改接口**：无
- **前端建议改哪些页面**：无

---

## 6. 风险与遗留

- **手动恢复：** ack 2xx 后订单仍可能卡在 `RECHARGING`；需运维按既有流程人工恢复（任务书 §0 有意设计）。
- **告警：** ERROR 日志 `[Payment Webhook] stale fulfillment, acking to stop retries` 可用于监控；未在本 Phase 加指标。
- **建议下一任务：** DBug-2 — MLA-P1-005/006 统一 token refresh（`frontend/src/api/client.ts` + `stores/auth.ts`）。

---

## 7. 阻塞项（若 status=blocked）

无。
