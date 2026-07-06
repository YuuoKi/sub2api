# CODEX_TASK_S3_LOOP — 人民币统一 + 计费可靠性收口（重型循环任务书）

> **执行模式：自主循环（loop）。** 你是 Sub2API 仓库的执行代理（Codex），单会话顺序执行本任务书全部 Phase。
> 每个 Phase 完成后必须跑门禁；门禁不绿就修，修完重跑，直到全绿才进入下一 Phase。
> 老板不在电脑前逐步确认——**除了「停止条件」列出的情况，不要停下来问问题**，用工程判断做决定并把决定记入审查包。
>
> **本任务书已由老板授权：允许修改 `frontend/**`（这是本任务的核心工作之一）。**
> 规划真相源：`D:\Codex创业任务\QCanvas（无界版）\北极星\QCanvas行动手册2.0_可执行_20260705.html`

---

## 0. 任务背景（为什么做）

2026-07-05 五路审计确认：

1. 老板是中国人，控制台金额几乎全是 `$`（只有视频任务页有 `¥`），看不习惯。**目标：管理控制台展示层统一人民币。**
2. 审计发现两个真实缺陷：
   - **扣费可靠性**：`video_gateway_billing.go:151-169` — `balance_charged_at` claim 成功后若 `DeductBalance` 失败，只 `slog.Warn` 就 return，且 claim 不回滚 → 视频已交付但钱没扣，worker 重试也不会再扣（资损+对账缺口）。
   - **生产 gate 旁路**：`production_authorized` 检查只在 API Key 生产路径（`CreateAPIKeySeedanceProductionTask`）生效；管理员走 `video_handler.go:236` 的 `CreateTask` 可绕过。
3. AI 调用记录页的视频提示词样本取自 `video_tasks.cost_estimate`，**该值 Seedance 行本来就是 CNY**，但后端样本 API 不返回 currency，前端当 USD 显示 —— 这是人民币统一改造中的**头号双重换算陷阱**。

## 1. 存储与展示币种语义（改造前必须背下来）

| 数据 | 存储币种 | 说明 |
|------|---------|------|
| `users.balance`、`api_keys.quota` | **USD** | **本任务不改存储语义** |
| `usage_logs.*_cost` | USD | LLM 计费 |
| `video_usage_logs.cost_estimate` / `video_tasks.cost_estimate` | **原生币种**（Seedance=CNY，mock=USD），`currency` 列标记 | video_tasks 样本 API 目前**不带** currency |
| dashboard `unified_total_actual_cost` | USD（SQL 已把 CNY 行 ÷rate 折成 USD，见 `usage_log_repo.go:107-129`） | 前端可放心 ×rate 转 ¥ |
| `settings.usd_cny_rate` | 默认 7.20 | 已在 `GET /api/v1/admin/settings` 暴露（`setting_handler.go:77-79`），前端未消费 |

**总方案（已定，不要重新设计）：存储保持 USD 不动，展示层统一 ¥ = USD × usd_cny_rate。原生 CNY 的行（video currency=CNY）直接显示 ¥ 原值，绝不再乘汇率。**

---

## 2. 循环协议（LOOP PROTOCOL）

```text
for each Phase in [S3-0 … S3-5]:
    实现 → 自测 → 跑该 Phase 的门禁
    while 门禁不绿:
        分析失败 → 修复 → 重跑（同一失败最多重试 3 轮，仍不绿则记 blocked 并继续下一 Phase）
    在审查包草稿中记录：改动文件、决策、证据
最后：跑全量门禁 → 写正式审查包 → git commit
```

**门禁命令（本机是 Windows PowerShell，pnpm scripts 会因缺 `sh` 失败，必须用下面的直调形式）：**

```powershell
# 后端（backend/ 目录下）
go test ./...
golangci-lint run ./...

# 前端（frontend/ 目录下）—— 不要用 pnpm run lint:check / typecheck（会报 'sh' 不是内部或外部命令）
npx eslint . --ext .ts,.vue --max-warnings=0
npx vue-tsc --noEmit
npx vitest run --reporter=basic   # 至少跑 charts/__tests__ 与被改文件相关的 spec
```

## 3. 红线（违反任何一条 = 任务失败）

- 不读取、不打印 `.env`、API Key、token、密码。
- 不 push、不 rebase、不 reset、不 clean。**允许 commit**（见 §10）。
- 不改 `users.balance` / `api_keys.quota` 的存储币种语义；不写迁移把历史数据换币种。
- 遵守 `handler → service → repository` 分层与 `backend/.golangci.yml` depguard。
- 前端不新增依赖；沿用现有组件与深色风格；文案用中文。
- 禁止宣称：不写「计费全面验证」「生产就绪」——状态词只用 内部可用/可演示/待复核/已阻塞/已冻结。

---

## Phase S3-0 · 基线预检

1. `git status --short` 必须干净（当前 HEAD 应为 `bfb41558` 或其后代）。不干净 → 记录后停止（停止条件①）。
2. 跑一遍全部门禁记录基线结果（应全绿；前端两条用 npx 直调）。

## Phase S3-1 · 后端：汇率与 currency 透出

| # | 改动 | 位置 |
|---|------|------|
| 1 | `GET /api/v1/admin/dashboard/stats` 响应增加 `usd_cny_rate` 字段（从 `settingService.GetUSDCNYRate()` 读） | `backend/internal/handler/admin/dashboard_handler.go:79-134` |
| 2 | generation-content 样本 API 返回每条样本的 `currency`（联 `video_tasks`/`video_usage_logs` 的 currency 列；LLM 样本恒 USD） | `backend/internal/handler/admin/generation_content_handler.go` + `generation_content_repo.go:319-320` 附近 |
| 3 | 若周报/samples 有跨币种求和（`generation_content_repo.go:213` 的 `total_cost_estimate`），改为按 dashboard 同款 CTE 折 USD 后再加总；改不动就在响应加 `mixed_currency: true` 并在审查包记录 | 同上 |

**验收**：新字段有单测或集成测试覆盖；`go test ./...` 全绿。

## Phase S3-2 · 后端：扣费失败自愈（最高优先级修复）

现状：`video_gateway_billing.go:151-169` claim 成功 → `DeductBalance` 失败 → 只告警，claim 永久占位，重试不再扣。

**要求：**

1. `DeductBalance` 失败时**释放 claim**（把 `balance_charged_at` 清回 NULL），让 worker 下一轮重试能再次尝试扣费。注意与 `video_usage_logs` UNIQUE 幂等的先后关系：usage log 的插入是 `ON CONFLICT DO NOTHING`，重试安全。
2. 释放要么在同一事务里，要么用 compare-and-clear（只清自己刚 claim 的那条），避免并发 worker 重复扣。
3. 新增测试：a) 扣费失败 → claim 释放 → 第二次调用成功扣费且只扣一次；b) 扣费成功后重复调用不双扣（已有测试 `video_gateway_billing_test.go:273-298`，保持通过）。
4. 顺手更新 `video_gateway_service.go:51-52` 过期的 "phase-2 real billing" 注释。

## Phase S3-3 · 后端：管理员路径补 production gate

1. 管理员 `CreateTask`（`video_handler.go:236` 一路到 `video_gateway_service.go` 的普通创建路径）在目标 provider 为 Seedance 且**非 trial/smoke 模式**时，同样检查 `seedanceProductionAuthorized()`（`video_gateway_adapter.go:749`），未授权返回 `VIDEO_PRODUCTION_NOT_AUTHORIZED`。
2. 保留试跑任务（trial/smoke）路径不受影响——「系统→试跑任务」是消防演习功能，必须继续可用。
3. 补集成测试：管理员创建 Seedance 生产任务、账号未授权 → 被拦；trial 任务 → 放行。参考 `api_key_video_gateway_test.go:425` 的写法。

## Phase S3-4 · 前端：人民币统一展示

**核心机制**：新建 composable（如 `frontend/src/composables/useDisplayCurrency.ts`）——从 dashboard stats 的 `usd_cny_rate`（S3-1 新字段）取汇率，暴露 `formatCny(usd: number)` 与 `formatByCurrency(amount, currency)`。汇率取不到时回退 7.2 并在 UI 不出错。

| 文件 | 改什么 |
|------|--------|
| `views/admin/console/consoleUtils.ts:24-31` | `formatMoney` → ¥（USD×rate）；或改为接受 rate 参数由 composable 驱动 |
| `views/admin/console/BossOverviewView.vue` | 388 附近「美元」文案删除；466 Chart 轴/tooltip `$` → ¥ |
| `views/admin/console/StaffView.vue` | 今日/累计花费列改 ¥；**开卡表单 251 行保持 USD 输入**，标签改为「额度（USD，账户余额币种）」，旁边加灰字换算提示「≈¥xx」 |
| `views/admin/console/AiRecordsView.vue:166` | 样本金额改用 `formatByCurrency(amount, sample.currency)` —— **CNY 行绝不乘汇率** |
| `views/admin/video/VideoTasksView.vue:195-199` | CNY 行保持 ¥ 原值；USD 行（mock 等）×rate 显示 ¥，可加灰字标注原币种 |
| `views/admin/DashboardView.vue` + `components/charts/` 6 个图表组件 + `components/admin/usage/UsageStatsCards.vue`、`UsageTable.vue`、`components/admin/account/AccountStatsModal.vue` | 各自的 `formatCost`/内联 `$` 统一走共享 formatter 显示 ¥ |
| `api/admin/settings.ts` | `SystemSettings` 补 `usd_cny_rate: number` 类型 |
| `views/admin/SettingsView.vue` | 新增「USD/CNY 汇率」设置项（读写 `usd_cny_rate`，管理员可改） |
| 相关 `__tests__` | 更新 `$` 断言为 ¥ |

**明确不改（本期范围外）**：`users.balance` 相关页面（AppHeader 顶栏余额、UserBalanceModal、充值/兑换页、KeysView 的 quota_used）保持 USD —— 余额是账户币种，混改会误导充值。在总览页金额卡下方加一行灰色小字：「账户余额与开卡额度以美元计，汇率 7.2 可在设置中调整」。

**验收**：demo 模式浏览器走查截图级确认（如果无法起 dev server，用组件测试断言覆盖）；npx eslint / vue-tsc / 相关 vitest 全绿。

## Phase S3-5 · 收口

1. 全量门禁重跑（后端 2 条 + 前端 3 条），结果写入审查包。
2. 审查包：`docs/superpowers/codex-handoff/deliverables/2026-07-06-S3-LOOP-review.md`，用 DELIVERABLE_TEMPLATE.md，必含：每 Phase 验收表（pass/fail/blocked + 证据）、双重换算防护说明、给老板的验收步骤（3 步内在浏览器确认 ¥ 生效）。
3. Commit（允许多个逻辑 commit）：
   - `fix(billing): release balance claim on deduct failure for worker retry`
   - `fix(video): enforce production_authorized on admin create path`
   - `feat(console): unify display currency to CNY with usd_cny_rate`
   - `docs(s3): add S3 loop review package`
4. **不 push**。最后向老板汇报一句话 + 审查包路径。

---

## 停止条件（仅以下情况允许中断循环）

① S3-0 发现工作区不干净或 HEAD 不符 → 记录现场，停止。
② 需要真实付费调用才能继续（本任务书**不含**任何真实付费调用，遇到即绕开）。
③ 同一门禁失败连续 3 轮修复无效 → 该项标 `blocked`，跳过继续，审查包写清。
④ 发现必须改 `users.balance` 存储语义才能完成 → 停止并写明（这是老板决策）。

## 验收总表（审查包必须逐条回答）

| # | 验收项 | 判定标准 |
|---|--------|---------|
| 1 | dashboard stats 返回 usd_cny_rate | 集成测试断言 |
| 2 | 样本 API 带 currency | 测试断言 CNY/USD 两种样本 |
| 3 | 扣费失败可重试且不双扣 | 两个新测试通过 |
| 4 | 管理员路径 production gate | 集成测试拦截/放行各一 |
| 5 | 控制台全 ¥、CNY 行不双换算 | eslint+tsc+vitest 绿 + 走查/组件断言 |
| 6 | 余额/额度 USD 语义有清晰标注 | 截图或代码证据 |
| 7 | 全部门禁全绿 | 命令输出摘要 |
