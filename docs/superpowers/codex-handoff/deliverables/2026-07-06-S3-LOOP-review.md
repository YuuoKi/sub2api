# 审查包：S3 Loop CNY 执行收口

> 执行者：Codex  
> 完成时间：2026-07-06 00:38  
> 关联任务书：`docs/superpowers/codex-handoff/CODEX_TASK_S3_LOOP_CNY.md`  
> 状态：`partial`（业务代码与测试门禁通过；golangci 本机基线异常已阻塞）

---

## 1. 本任务做了什么

- 后端 dashboard/generation-content 增加 `usd_cny_rate` 与样本 `currency`，weekly 成本统一折算成 USD 聚合，前端再按汇率展示人民币。
- 视频扣费改为带 claim 时间戳的 compare-and-clear；扣费失败只释放本次 claim，并新增 succeeded 未扣费任务的 retry/reconciliation。
- Seedance 普通 admin 生产创建要求 provider metadata `production_authorized=true`；tiny-real/trial/smoke 仍走原有限流、allowlist、redaction gate。
- 前端成本、图表、usage table、account stats、video/generation records 统一展示 `¥`；余额、开卡额度、quota、默认余额和余额阈值仍按 USD 存储/输入，并增加约人民币提示。

---

## 2. 改了哪些文件

| 文件 | 变更摘要 |
|---|---|
| `backend/internal/handler/admin/dashboard_handler.go` | dashboard stats 注入 `usd_cny_rate`，读取失败回退 7.2 |
| `backend/internal/repository/generation_content_repo.go` | recent samples 返回 currency；weekly total_cost_estimate 按汇率统一 USD |
| `backend/internal/service/video_gateway_billing.go` | 扣费 claim 返回 claimedAt，失败 compare-clear 本次 claim |
| `backend/internal/service/video_gateway_worker.go` | worker 增加 succeeded 未扣费任务 retry |
| `backend/internal/repository/video_gateway_repo.go` | 新增 compare-clear 与 uncharged succeeded 查询 |
| `backend/internal/service/video_gateway_service.go` / `backend/internal/handler/video_handler.go` | admin production path 显式要求 Seedance production 授权 |
| `frontend/src/composables/useDisplayCurrency.ts` | 新增人民币/美元账户金额统一 formatter |
| `frontend/src/views/admin/**` / `frontend/src/components/**` | dashboard、console、usage、account stats、generation/video records 接入人民币展示 |
| `frontend/src/api/admin/*.ts` / `frontend/src/types/index.ts` | 增加 `usd_cny_rate`、`currency` 类型 |
| `*_test.go` / `*.spec.ts` | 覆盖汇率字段、混币聚合、扣费 retry、production gate、前端显示 helper |

---

## 3. Phase 验收结果

| Phase | 结果 | 证据 |
|---|---|---|
| S3-0 baseline | pass + blocked | `go test ./...` pass；frontend `npx.cmd eslint` / `vue-tsc` / `vitest` pass；`golangci-lint run ./...` 基线即失败 `no go files to analyze` |
| S3-1 汇率与生成内容 | pass | handler/repository targeted Go tests pass；full backend tests最终 pass |
| S3-2 扣费可靠性 | pass | billing/repository targeted Go tests pass；full backend tests最终 pass |
| S3-3 Seedance production gate | pass | 新增 production unauthorized 与 tiny trial allowed 测试；首次全量发现服务层默认误触，已在第 1 修复轮改为显式 admin production flag |
| S3-4 前端人民币展示 | pass | focused specs 26 pass；`npx.cmd eslint` pass；`npx.cmd vue-tsc --noEmit` pass；full vitest 103 files / 605 tests pass |
| S3-5 收口 | partial | `go test ./...` pass；`git diff --check` pass；secret scan pass；golangci 仍 blocked |

---

## 4. 验证命令与结果

```text
cd backend
$env:GOCACHE='D:\sub2api-trunk\.cache\go-build'; go test ./...
=> PASS

golangci-lint run ./...
=> BLOCKED: Running error: context loading failed: no go files to analyze

cd frontend
npx.cmd eslint . --ext .ts,.vue --max-warnings=0
=> PASS

npx.cmd vue-tsc --noEmit
=> PASS

npx.cmd vitest run --reporter=basic
=> PASS: 103 files / 605 tests

git diff --check
=> PASS（仅提示 CODEX_START_HERE.md 既有 LF/CRLF warning）

C:\Users\浩臣移动工作站\.cache\codex-runtimes\codex-primary-runtime\dependencies\python\python.exe tools/secret_scan.py --include-untracked
=> PASS: no high-confidence tracked-plus-untracked findings
```

截图说明：本轮未启动前端 dev server，也未做浏览器截图；原因是变更为数据展示/类型/组件层改造，已用 component specs、full vitest、eslint、vue-tsc 覆盖。浏览器验收路径见第 8 节。

---

## 5. 接口与字段契约

- `GET /api/v1/admin/dashboard/stats`：`data.usd_cny_rate: number`
- `GET /api/v1/admin/generation-content/samples`：sample 增加 `currency?: string`
- `SystemSettings.usd_cny_rate: number`
- `UpdateSettingsRequest.usd_cny_rate?: number`
- 视频 repository/service：`ClaimVideoBalanceCharge` 返回 claimedAt；新增 `ClearVideoBalanceChargeIfClaimedAt` 与 `ListUnchargedSucceededVideoTasks`

---

## 6. 双重换算保护

- 后端 weekly generation report 只输出 USD 归一聚合：CNY video cost 通过 `usd_cny_rate` 折回 USD，USD 原值保留。
- 前端 `formatByCurrency(amount, currency, rate)` 对 `currency=CNY` 原样显示人民币；`USD/缺省` 才乘一次汇率。
- `users.balance`、`api_keys.quota`、默认余额、余额阈值继续保留 USD 主语义；仅显示“约人民币”提示。

---

## 7. 风险、阻塞与回滚

- 未解决问题：`golangci-lint run ./...` 在 `backend/` 复现本机基线异常 `no go files to analyze`，本轮代码未能让该工具门禁变绿，状态标 `已阻塞`。
- 外部风险：没有触发真实 provider 调用；Seedance production gate 只在 admin 普通创建路径生效，tiny-real/trial/smoke 保持原 gate。
- 回滚方案：本地 commit 后可 `git revert <commit>`；若只回滚前端展示，可优先回滚 `frontend/src/composables/useDisplayCurrency.ts` 及调用点；若只回滚 production gate，可回滚 `video_handler.go` 与 `video_gateway_service.go` 中显式 flag。

---

## 8. 老板浏览器验收路径

1. 管理后台打开设置页，进入用户/默认值区域，确认 `USD/CNY 汇率` 可编辑，默认余额与余额阈值仍是 USD 输入，并显示约人民币提示。
2. 打开老板概览/Usage/Dashboard，看成本、图表 tooltip、usage table、账户统计弹窗是否显示 `¥`；余额、开卡额度、quota 仍显示 `$`。
3. 打开视频任务和生成内容墙，确认 USD 样本折算 `¥`，CNY 样本不二次换算；后台创建 Seedance 普通生产任务时未授权账号返回 `VIDEO_PRODUCTION_NOT_AUTHORIZED`。

---

## 9. 后续提示词

```text
继续在 D:\sub2api-trunk 审查 S3 Loop CNY 收口结果。
重点复核：
1. golangci-lint 本机 no go files to analyze 是否为工具/模块环境问题；
2. Seedance production_authorized gate 是否仅影响 admin 普通生产创建；
3. 前端是否还有成本字段误显示 $，以及余额/quota 字段是否仍保持 USD 主语义。
禁止 push/deploy/读取 .env。
```
