# Codex 任务书 · R1 定向复查与补齐（2026-07-05）

> **性质**：复查 + 修补，不是重做 V-1→A-4。  
> **前置**：先读完 `deliverables/2026-07-05-*.md`（9 份）+ `docs/api/*-contract.md` + 本文件。  
> **分工**：本文件列的 **R1-A～R1-F 由 Codex 执行**；**R1-G 前端由 Claude 执行**（等 R1 审查包交付后再做）。  
> **交付**：`deliverables/2026-07-05-R1-backend-review.md`（一份总审查包，内含子项 pass/fail 表）。

---

## 背景：Claude 第二轮复查结论（2026-07-05）

V-1→A-4 **代码与单测大体可信**（`go test ./...` 全绿），但存在 **6 类「审查包写得比生产乐观」的缺口**。本轮只补这些，不重写已完成的契约/价表/归总逻辑。

| ID | 严重度 | 问题 | 现状 |
|----|--------|------|------|
| R1-A | 🔴 P0 | 视频成功后的**用户余额扣减**仍是空操作 | `StaticBudgetGuard.Charge()` 明确 no-op；`chargeForVideo` 仅在 `budget!=nil` 时调用它 |
| R1-B | 🔴 P0 | **真实 Seedance 生产路径**与契约 4–15s / content[] 不一致 | smoke gate 仍限 1–5s；API-key 需 `trial_mode=tiny_real`；kling 403 |
| R1-C | 🟡 P1 | `video_usage_logs` **可重复插入** | worker 重试可能 duplicate；无 `video_task_id` 唯一约束 |
| R1-D | 🟡 P1 | 适配器仍有 `UNVERIFIED` 注释 | 虽有 fixture 单测，缺官方 Ark 金样本归档 |
| R1-E | 🟡 P1 | `SUB2API_VIDEO_URL_ALLOWLIST` 管视频+Gemini 图片 URL | 命名误导；生产未配 allowlist 时全拦 |
| R1-F | 🟢 P2 | 文档/git/历史数据 | `docs/*` 被 ignore；历史 log 默认 USD/fallback；Sonnet5 全年按 9 月价 |

---

## R1-A：视频真实扣费（V-2B）【最高优先】

### 目标

Seedance 任务 terminal 成功后，除写入 `video_usage_logs.cost_estimate`（CNY）外，**必须扣减 `users.balance`**（系统余额单位为 USD，按 `usd_cny_rate` 折算）。

### 设计要求

1. **与 LLM 扣费同路径语义**：参考 `gateway_service.go` 的 `userRepo.DeductBalance` + `billingCacheService.QueueDeductBalance`。
2. **幂等**：同一 `video_task_id` 只扣一次。建议：
   - migration：`video_tasks.balance_charged_at TIMESTAMPTZ` 或 `video_usage_logs` 上 `UNIQUE(video_task_id)` + `ON CONFLICT DO NOTHING`；
   - `ClaimVideoBalanceCharge(ctx, taskID) (bool, error)` 原子 claim。
3. **金额**：
   - 扣费基数 = `chargeableVideoCost(task)`（Seedance 用 token 真实价，CNY）；
   - 扣用户余额 = `ConvertBillingAmount(cost, task.Currency, USD, usd_cny_rate)`；
   - `usd_cny_rate` 从 `SettingService` 读取，默认 7.20。
4. **与 StaticBudgetGuard 解耦**：
   - `CheckBudget` 仍可走 per-call cap（可选）；
   - **余额扣减不要放在 `StaticBudgetGuard.Charge` 里**（保持其为 no-op 或仅做 cap 记账）；
   - 在 `chargeForVideo` 或独立 `deductUserBalanceForVideo` 中实现。
5. **wire**：`ProvideVideoGatewayService` 注入 `UserRepository`、`SettingService`、`BillingCacheService`（与 `GatewayService` 一致）。
6. **失败任务**：费用 0，不扣余额（已有逻辑，勿破坏）。

### 验收

| 验收项 | 证据 |
|--------|------|
| mock Seedance 成功任务扣减用户余额 | 单测：`DeductBalance` 被调用且金额 = CNY/汇率 |
| 同一 task worker 重试不双扣 | 单测：第二次 charge 不调用 DeductBalance |
| 失败任务不扣 | 已有 `TestSeedanceFailedTaskCostsZero` 扩展 |
| `go test ./...` + lint 全绿 | 命令输出 |

### 给 Claude（R1-G 前置）

- 前端展示视频花费时：Seedance 行 `currency=CNY`，总览用 `unified_total_actual_cost`（已折算 USD）。

---

## R1-B：Seedance 正式通道策略（P0-1 复查）

### 目标

智能画布（QCanvas）走 `/v1/video/tasks` 时，**契约层（4–15s、content[]、全能参考）与生产 gate 对齐**，并写清文档。

### 必须厘清并落地

1. **smoke gate（1–5s）vs 正式调用（4–15s）**：
   - 建议：`provider_account.metadata.production_authorized=true`（或复用 `real_smoke_authorized` 升级语义）时 **跳过 1–5s smoke 限制**，仍保留 allowlist + 审计日志。
   - 未授权账号：保持现有 smoke 限制（安全）。
2. **API-key 路径**：
   - 文档写清：`provider=seedance` 无 `trial_mode` 时的行为（403 vs 正式）；
   - 若 QCanvas 需要正式 Seedance：增加 `production_mode` 或管理员开关，**不要**让智能画布永远只能 `tiny_real`。
3. **duration=-1**：创建校验通过；adapter 透传；预算 gate 用 5s 兜底（已有，保持）。

### 验收

| 验收项 | 证据 |
|--------|------|
| 授权 provider + 10s 任务可创建（非 smoke 路径） | 单测或 harness |
| 未授权 provider 仍被 1–5s 拦住 | 回归 `seedanceSmokeGate` 测试 |
| `docs/api/video-gateway-contract.md` 更新「生产 vs 试跑」章节 | diff |

---

## R1-C：`video_usage_logs` 幂等

### 目标

worker 重试 terminal 处理时，不重复插入 usage log、不重复扣费（与 R1-A 联动）。

### 做法

- migration：`CREATE UNIQUE INDEX ... ON video_usage_logs (video_task_id);`
- `InsertUsageLog` 改为 `ON CONFLICT (video_task_id) DO NOTHING` 或 upsert
- 单测：`TestVideoGatewayRepositoryInsertUsageLogIsIdempotent` 扩展为唯一约束级别

---

## R1-D：Ark 金样本固化

### 目标

消除 `video_gateway_adapter.go` 中 `UNVERIFIED` 注释；用归档 JSON 作为回归基准。

### 做法

1. 新增 `backend/internal/service/testdata/ark_poll_succeeded.json`（与 `video_gateway_poll_response_contract_test.go` 一致，可扩展官方字段变体）。
2. 新增测试：create payload snapshot 含 `content[].role`、`duration`、`resolution`、`ratio`、`generate_audio`。
3. 审查包列出字段名与 [火山方舟 Seedance API 文档](https://www.volcengine.com/docs/82379/1520757?lang=zh) 的对照表。

---

## R1-E：媒体 URL allowlist 澄清

### 目标

避免 Gemini 图片 `fileData.fileUri` 与视频参考 URL 共用 `SUB2API_VIDEO_URL_ALLOWLIST` 却无文档说明。

### 做法（二选一，审查包说明选型）

- **A（推荐）**：新增 env `SUB2API_MEDIA_URL_ALLOWLIST`，读取时 fallback 到 `SUB2API_VIDEO_URL_ALLOWLIST`；更新 `video_gateway_ssrf.go` 与契约文档。
- **B（最小）**：不改 env 名，在 `docs/api/image-gateway-contract.md` 与 `video-gateway-contract.md` 醒目注明「图片与视频共用 VIDEO allowlist」。

---

## R1-F：仓库卫生（顺手）

1. `.gitignore` 增加例外（若老板同意提交文档）：
   ```
   !docs/api/
   !docs/superpowers/codex-handoff/
   ```
2. 确认 `backend/internal/service/user_member_type.go`（P0-3）纳入版本控制。
3. 更新 `chargeForVideo` 过时注释（仍写 duration-based phase-2）。

---

## R1-G：前端（Codex 不做，Claude 做）

等 `2026-07-05-R1-backend-review.md` 交付且 R1-A/B pass 后：

1. `inactive` → `disabled`（admin 改卡）
2. P0-3：成员与开卡 + `member_type`
3. 总览：`unified_total_actual_cost`、`video_spend_by_provider`
4. 任务记录：图/视频流水、`media_type`、币种展示
5. 移除/改造「外部工具接入」导航

---

## 验证命令（R1 结束前必须跑）

```powershell
cd D:\sub2api-trunk\backend
$env:GOCACHE = (Join-Path (Get-Location) '.gocache')
$env:SUB2API_VIDEO_REAL_SMOKE_ENABLED = '0'
go test ./...
golangci-lint run ./...
git diff --check
# secret scan（本机 python 路径按 V-1 审查包）
```

---

## 审查包模板

输出 `deliverables/2026-07-05-R1-backend-review.md`，必须包含：

1. R1-A～R1-F 逐项 pass/fail 表
2. 改了哪些文件
3. **未解决 / 需老板决策**（如：是否开放 API-key 正式 Seedance）
4. **给 Claude 的前端接口变更**（若有）
5. 回滚方案

---

## 禁止事项

- 不要重做 V-1/V-2/V-3/V-4/V-5/A-1/A-2/A-3/A-4 已有逻辑（除非 R1 子项明确要求修 bug）
- 不要改 `frontend/**`（R1-G 是 Claude）
- 不要读 `.env`、真实 Key、不要 push/deploy/commit（除非老板另行要求）
- 不要开第二个并行 Codex 会话改同一仓库
