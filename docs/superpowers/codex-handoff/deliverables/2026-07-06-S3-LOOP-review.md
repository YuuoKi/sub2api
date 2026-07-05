# S3 Loop CNY 修复复查包

> 仓库：`D:\sub2api-trunk`
> 分支：`wujie/video-capture-moat-20260702`
> 基线提交：`31af790e S3 loop CNY billing and display`
> 本轮状态：`内部可用` / `待人工浏览器复核`
> 禁止项遵守：未 push，未 deploy，未读取 `.env`/token/cookie，未触发真实付费 provider。

## 1. 本轮复查结论

上一轮复查发现的问题已经开修并复核完成：

| 问题 | 修复结果 |
|---|---|
| Usage / Account / Generation / Video 部分页面没有使用设置里的 `usd_cny_rate`，仍可能按默认 7.2 展示 | 已修复。相关后端接口补齐 `usd_cny_rate`，前端显式透传到卡片、表格、图表和样本墙 |
| `currency=CNY` 样本需要原样展示人民币，不能再乘汇率 | 已加回归测试，`CNY` 原样 `¥5.0094`，`USD` 按汇率一次转换 |
| `UsageView.spec.ts` mock 未包含真实调用的 `dashboard.getModelStats`，Vitest stderr 有噪音 | 已修复 mock，目标用例不再因缺 mock 打印该错误 |
| VideoTasks 列表没有独立汇率字段 | 已新增只读 composable 从现有 dashboard stats 拉取 `usd_cny_rate`，失败回退 7.2，不改视频列表分页契约 |
| 旧导出的 `components/account/AccountStatsModal.vue` 仍保留 `$` / USD 展示 | 已同步修复，避免未来通过 barrel export 重新暴露美元显示 |

## 2. 代码逻辑说明

### 后端

- 新增 `backend/internal/handler/admin/display_currency.go`
  - `resolveUSDCNYRate(ctx, settingService)` 统一读取设置汇率。
  - `settingService == nil` 或读取异常时沿用 `service.DefaultUSDCNYRate`，当前默认 `7.2`。

- `GET /api/v1/admin/usage/stats`
  - `usagestats.UsageStats` 新增 `usd_cny_rate`。
  - `admin.UsageHandler` 注入 `SettingService` 后在返回前填充汇率。

- `GET /api/v1/admin/accounts/:id/stats`
  - `usagestats.AccountUsageStatsResponse` 新增 `usd_cny_rate`。
  - `admin.AccountHandler` 注入 `SettingService` 后在账户统计返回前填充汇率。

- `GET /api/v1/admin/generation-content/samples`
  - 顶层响应新增 `usd_cny_rate`。
  - sample 的 `currency` 字段沿用上一轮实现，用于前端判断是否原生 CNY。

- `GET /api/v1/admin/generation-content/weekly-report`
  - 顶层响应新增 `usd_cny_rate`。
  - weekly `total_cost_estimate` 仍保持后端 USD 归一聚合，前端展示时只乘一次汇率。

- `backend/cmd/server/wire_gen.go`
  - 将现有 `settingService` 注入 `AccountHandler`、`UsageHandler`、`GenerationContentHandler`。
  - 其他测试构造函数通过 variadic 参数保持兼容。

### 前端

- 新增 `frontend/src/composables/useAdminDisplayCurrencyRate.ts`
  - 调用 `adminAPI.dashboard.getStats()` 读取 `usd_cny_rate`。
  - 无效值或接口失败时回退 `DEFAULT_USD_CNY_RATE = 7.2`。

- `formatByCurrency(amount, currency, rate)`
  - `currency=CNY`：原样人民币格式化，不乘汇率。
  - `currency=USD` 或缺省：按传入汇率乘一次后展示为 `¥`。

- `GenerationContentView`
  - 从 samples / weekly 响应接收汇率。
  - 只有有效正数汇率才覆盖当前值，避免旧响应缺字段时把已取得的汇率重置为默认 7.2。
  - `ContentWall` 显式接收 `usdCnyRate`。

- `AccountStatsModal`
  - 总成本、今日成本、最高成本日、趋势 tooltip、Model/Endpoint chart 均使用接口汇率。
  - admin 组件和旧 barrel export 组件同步修复。

- `VideoTasksView`
  - 挂载时并行拉取任务列表和 dashboard 汇率。
  - 任务 `currency=CNY` 不二次换算，USD 按 dashboard 汇率展示。

## 3. 新增/更新测试

### 后端

- `TestAdminUsageStatsIncludesUSDCNYRate`
  - 验证 admin usage stats 响应的 `data.usd_cny_rate` 使用设置值 `7.41`。

- `TestGenerationContentHandlerSamplesIncludesUSDCNYRate`
  - 验证 generation samples 顶层响应带 `usd_cny_rate`。

- `TestGenerationContentHandlerWeeklyReportIncludesUSDCNYRate`
  - 验证 weekly report 顶层响应带 `usd_cny_rate`。

### 前端

- `useAdminDisplayCurrencyRate.spec.ts`
  - 覆盖 dashboard 汇率读取和无效汇率回退。

- `ContentWall.spec.ts`
  - 覆盖 USD 按自定义汇率转换、CNY 不二次转换。

- `GenerationContentView.spec.ts`
  - 覆盖 samples 响应汇率向样本墙和 weekly cost 显示透传。
  - 覆盖 weekly 缺字段时不覆盖 samples 的有效汇率。

- `AccountStatsModal.spec.ts`
  - 覆盖账户统计卡片与分布图表使用接口汇率。

- `UsageView.spec.ts`
  - 补齐 `dashboard.getModelStats` mock，移除本轮相关测试噪音。

## 4. 红测记录

| 命令 | 预期失败证据 |
|---|---|
| `npx.cmd vitest run src/components/admin/generation-content/__tests__/ContentWall.spec.ts src/views/admin/__tests__/GenerationContentView.spec.ts src/components/admin/account/__tests__/AccountStatsModal.spec.ts src/views/admin/__tests__/UsageView.spec.ts --reporter=basic` | 红：ContentWall 仍显示 `¥7.20`；GenerationContentView 没传 `rate=7.5`；AccountStatsModal 没显示 `¥7.50` |
| `npx.cmd vitest run src/composables/__tests__/useAdminDisplayCurrencyRate.spec.ts --reporter=basic` | 红：`../useAdminDisplayCurrencyRate` 不存在 |
| `$env:GOCACHE='D:\sub2api-trunk\tmp\gocache'; go test ./internal/handler/admin -run "TestAdminUsageStatsIncludesUSDCNYRate|TestGenerationContentHandler.*USDCNYRate" -count=1` | 红：构造器尚不支持 `SettingService` 注入；后续修正测试响应包裹层后进入绿测 |

## 5. 绿测与全量门禁

| 命令 | 结果 |
|---|---|
| `$env:GOCACHE='D:\sub2api-trunk\tmp\gocache'; go test ./internal/handler/admin -run "TestAdminUsageStatsIncludesUSDCNYRate|TestGenerationContentHandler.*USDCNYRate" -count=1` | PASS |
| `npx.cmd vitest run src/composables/__tests__/useAdminDisplayCurrencyRate.spec.ts src/components/admin/generation-content/__tests__/ContentWall.spec.ts src/views/admin/__tests__/GenerationContentView.spec.ts src/components/admin/account/__tests__/AccountStatsModal.spec.ts src/views/admin/__tests__/UsageView.spec.ts --reporter=basic` | PASS：5 files / 15 tests |
| `$env:GOCACHE='D:\sub2api-trunk\tmp\gocache'; go test ./...` | PASS |
| `$env:GOCACHE='D:\sub2api-trunk\tmp\gocache'; golangci-lint run ./...` | PASS：`0 issues`；仍有本机用户目录 golangci cache 写入 warning，不影响 lint 结果 |
| `npx.cmd eslint . --ext .ts,.vue --max-warnings=0` | PASS |
| `npx.cmd vue-tsc --noEmit` | PASS |
| `npx.cmd vitest run --reporter=basic` | PASS：105 files / 610 tests |
| `git diff --check` | PASS；仅提示既有 `CODEX_START_HERE.md` LF/CRLF warning |
| `C:\Users\浩臣移动工作站\.cache\codex-runtimes\codex-primary-runtime\dependencies\python\python.exe tools/secret_scan.py --include-untracked` | PASS：no high-confidence tracked-plus-untracked findings |

Vitest 全量 stderr 中仍存在既有测试主动制造的错误日志和 Vue router-link stub warning，例如 auth invalid-json、SettingsView router-link、订阅网络错误、TableLoader server error、OpsOpenAITokenStatsCard load failure；这些测试均为 PASS，且不是本轮新增问题。

## 6. 本轮修改文件索引

### 后端

- `backend/internal/handler/admin/display_currency.go`
- `backend/internal/handler/admin/usage_handler.go`
- `backend/internal/handler/admin/account_handler.go`
- `backend/internal/handler/admin/generation_content_handler.go`
- `backend/internal/pkg/usagestats/usage_log_types.go`
- `backend/cmd/server/wire_gen.go`
- `backend/internal/handler/admin/usage_handler_request_type_test.go`
- `backend/internal/handler/admin/generation_content_handler_test.go`

### 前端

- `frontend/src/composables/useAdminDisplayCurrencyRate.ts`
- `frontend/src/composables/__tests__/useAdminDisplayCurrencyRate.spec.ts`
- `frontend/src/api/admin/usage.ts`
- `frontend/src/api/admin/generation_content.ts`
- `frontend/src/types/index.ts`
- `frontend/src/views/admin/UsageView.vue`
- `frontend/src/views/admin/GenerationContentView.vue`
- `frontend/src/views/admin/video/VideoTasksView.vue`
- `frontend/src/components/admin/account/AccountStatsModal.vue`
- `frontend/src/components/account/AccountStatsModal.vue`
- `frontend/src/components/admin/account/__tests__/AccountStatsModal.spec.ts`
- `frontend/src/components/admin/generation-content/ContentWall.vue`
- `frontend/src/components/admin/generation-content/__tests__/ContentWall.spec.ts`
- `frontend/src/views/admin/__tests__/GenerationContentView.spec.ts`
- `frontend/src/views/admin/__tests__/UsageView.spec.ts`

## 7. 未纳入本轮提交的既有脏状态

- `docs/superpowers/codex-handoff/CODEX_START_HERE.md`
- `docs/superpowers/codex-handoff/CODEX_TASK_S3_LOOP_CNY.md`

这两项是任务书前置授权脏状态，本轮不回退、不覆盖、不纳入修复提交。

## 8. 截图说明

本轮未启动前端 dev server，未做浏览器截图。原因：本次修复集中在接口字段、类型契约和组件格式化逻辑，已经通过组件测试、全量 vitest、eslint、vue-tsc 覆盖。建议人工浏览器复核路径：

1. 管理后台设置页把 USD/CNY 汇率临时调整到非默认值，例如 `7.5`。
2. 打开 Usage、Accounts stats、Generation Content、Video Tasks，确认 USD 来源成本显示为 `¥` 且按新汇率变化。
3. 找一条 `currency=CNY` 的 Seedance/video 样本，确认金额不再乘汇率二次放大。

## 9. 风险与回滚

### 风险

- VideoTasks 通过 dashboard stats 拉汇率；如果 dashboard stats 接口失败，会回退 `7.2`，不会阻塞任务列表。
- 后端新增字段为向后兼容字段，不改变既有聚合金额语义。
- 未做真实 provider 调用；生产授权 gate 和付费调用仍需按受控验收路径验证。

### 回滚

- 若已提交，执行 `git revert <本轮修复提交>`。
- 若未提交，按文件索引反向还原本轮列出的 backend/frontend 修改；不要回退任务书前置脏状态。

## 10. 最终状态

本轮审查发现项已修复，新增回归测试已覆盖，后端与前端全量门禁通过。当前仍建议人工浏览器复核展示路径后再对外宣称最终可演示。
