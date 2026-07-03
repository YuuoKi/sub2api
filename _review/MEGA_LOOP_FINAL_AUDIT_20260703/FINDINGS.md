# Sub2API W5 终局 FINDINGS

日期：2026-07-03 Asia/Shanghai

## 结论

终局 open P0 = 0。

Sub2API 包内质量门禁全绿；Phase A' tiny real 三证与 B1 generation-content 账本未回归。后续真实供应商调用仍为 `已冻结`。

## 已确认通过

| 对象 | 结论 | 证据 |
|---|---|---|
| Go 全量测试 | 通过 | `go test ./...` exit 0 |
| Go 静态检查 | 通过 | `golangci-lint run` exit 0，0 issues |
| 前端 lint | 通过 | `pnpm run lint` exit 0 |
| 前端 typecheck | 通过 | `pnpm run typecheck` exit 0 |
| 前端测试 | 通过 | `pnpm run test:run` exit 0；100 files / 592 tests passed |
| 前端 build | 通过 | `pnpm run build` exit 0 |
| 安全扫描 | 通过 | `tools\secret_scan.py --include-untracked` exit 0，无 high-confidence findings |
| Phase A' 三证 | 未回归 | `success_result.json` 仍为受控单次 tiny real 证据 |
| B1 账本 | 未回归 | adoption / weekly report / Admin ContentWall 路径仍存在并通过相关门禁 |

## 仍需后续处理的非 P0

| ID | 严重度 | 状态 | 说明 |
|---|---:|---|---|
| W4-P1-003 | P1 | 待后续授权修复 | payment webhook 在非 wxpay provider lookup error 场景可能返回 2xx 成功，存在真实支付回调被上游认为已处理的语义风险。当前未触发真实支付链路，不构成 W5 P0。 |
| W4-P2-001 | P2 | 待后续优化 | weekly report 窗口仅校验 start < end；当前为 admin-only 且索引可控。 |

## B2 update 2026-07-04

- W4-P1-003 已修复待复核：`PaymentWebhookHandler.handleNotify` 在 `GetWebhookProviders` lookup failure 时统一返回 `400 verify failed`，不再对非 wxpay provider ack 2xx；保留 `ErrOrderNotFound` 2xx ack 语义。
- 已新增 unit 覆盖：非 wxpay lookup failure、wxpay lookup failure；未知订单 ack 仍由 `TestUnknownOrderWebhookAcksWithSuccess` 锁定。

## 边界

- 未读取 `.env`、key、token、cookie。
- 未触发真实 provider。
- 未 push。
- 未改 AUTH。
- 未 merge mockchain。
