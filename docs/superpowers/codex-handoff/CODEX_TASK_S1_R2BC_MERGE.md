# Codex 任务书 · S1 收尾 — R2-B / R2-C 扩样 / PR #3726 合并准备

> **性质**：生产验证 + 对账扩样 + 合并门禁，**不是**重做 V-1→R1 后端。  
> **规划真相源**：北极星 V5.0 `#roadmap` S1 · `#current-state` 商用签字分层  
> **前置**：R2-A `done`（task #4）；R2-C 基线 `partial`（1 条视频）  
> **分支**：`wujie/video-capture-moat-20260702` · PR [#3726](https://github.com/Wei-Shaw/sub2api/pull/3726)  
> **交付**：`deliverables/YYYY-MM-DD-S1-R2BC-review.md`（一份总审查包，子项 pass/fail/skip 表）

---

## 老板判定（2026-07-05 · 本任务书依据）

| 维度 | 判定 | 说明 |
|------|------|------|
| **内部测试** | ✅ 可以开 | 控制台 v2 + 视频正式链路 + 真实扣费已证（R2-A）；按 `production_authorized` 逐 Key 授权 |
| **商用签字** | ✅ 视频链路可签 | R2-A 三证齐全；R2-B/C 为**加强项**，不挡内部测试 |
| **对外 SaaS** | ❌ 仍禁止 | 多租户、登录产品化、未授权账号日常真调用 — 均不在本阶段 |

**本任务目标**：把 S1 剩余三项做完或标清 `blocked`，让 PR #3726 可放心合并。

---

## 第一步：读这些（顺序固定）

1. [CODEX_START_HERE.md](./CODEX_START_HERE.md)
2. [CODEX_TASK_PRODUCTION_VERIFY.md](./CODEX_TASK_PRODUCTION_VERIFY.md)（R2-A 上下文）
3. [deliverables/2026-07-05-R2-A-production-smoke-review.md](./deliverables/2026-07-05-R2-A-production-smoke-review.md)
4. [deliverables/2026-07-05-R2-C-billing-reconciliation-review.md](./deliverables/2026-07-05-R2-C-billing-reconciliation-review.md)
5. 契约：`docs/api/video-gateway-contract.md`、`docs/api/image-gateway-contract.md`
6. 工具（可选）：`tools/r2a-bootstrap.ps1`、`tools/r2a-smoke-probe.ps1`、`tools/r2a-probe-body.json`

---

## 执行顺序（单 Codex 会话，禁止跳步）

```text
S1-0  合并门禁自检（go test + lint + 审查包索引）
S1-1  R2-B  图片 NB2 真实冒烟（1 条起步，目标 3 条四档样本）
S1-2  R2-C  对账扩样（补 2 视频 + 3 图片，与 R2-A #4 合计 3+3）
S1-3  R2-M  PR #3726 合并准备清单（CI / 冲突 / START_HERE 一致性）
```

---

## S1-0 — 合并门禁自检

### 目标

确认 PR #3726 合入前仓库健康，不因「只跑冒烟不测编译」引入回归。

### 必做

| 项 | 命令 / 动作 |
|----|-------------|
| 后端测试 | `cd backend && go test ./...` |
| 后端 lint | `cd backend && golangci-lint run ./...` |
| 审查包齐全 | `deliverables/2026-07-05-*.md` 与 R2 收口摘要可读 |
| git 债务 | 确认 `user_member_type.go`、`docs/superpowers/` 已 track（R1-F 应已解） |
| 临时 Key | 审查包再次提醒老板废弃 R2-A 临时 Key（若尚未轮换） |

### 验收

| 验收项 | 证据 |
|--------|------|
| go test 全绿 | 命令输出摘要 |
| golangci-lint 无新增 error | 命令输出摘要 |
| deliverables 索引表 | 审查包 §1 |

---

## S1-1 — R2-B 图片网关真实冒烟

### 目标

对 **Nano Banana 2**（`gemini-3.1-flash-image-preview`）跑至少 **1 条** API-key 正式作图（老板授权 Key 后扩至 **3 条**，覆盖 512 / 1K / 2K 或 4K 中不同档）。

### 路径

- `POST /v1/images/generations`（Bearer 员工 API Key）
- 或 JWT 管理路径（仅 dev 自测，审查包注明）

### 必须验证

| 项 | 期望 |
|----|------|
| `imageConfig` / `responseModalities` | 透传至上游，响应含图片 URL |
| `usage_logs.media_type` | `image` |
| 多图 count | `n>1` 时 count 与计价一致 |
| 512 档计价 | `cost_estimate` 与价表一致（见 `image-gateway-contract.md`） |
| 余额扣减 | CNY→USD 折算与 `users.balance` Δ 一致（同 R2-A 语义） |
| 幂等 | 同一任务/请求不双扣 |

### 请求示例（脱敏，按契约调整）

```json
{
  "model": "gemini-3.1-flash-image-preview",
  "prompt": "S1 R2-B smoke: minimal product still life",
  "n": 1,
  "imageConfig": { "aspectRatio": "1:1", "imageSize": "1K" }
}
```

### 阻塞处理

- 无 NB2 Key / 未 `production_authorized` → 审查包 `status=blocked`，列明缺什么
- allowlist 拦截 → 记录 URL 域名，建议老板补 `SUB2API_MEDIA_URL_ALLOWLIST`

### 交付证据

- task / usage_log id
- `cost_estimate`、tokens、余额前后
- 可选：控制台 `AiRecordsView` 或 API 明细截图路径

---

## S1-2 — R2-C 对账扩样（3 视频 + 3 图片）

### 背景

R2-C 已有 **1 条视频**（task #4，`partial` pass）。本步补齐：

- **视频**：再跑 **2 条**（建议：不同 duration 或 resolution，均 `production_authorized`）
- **图片**：R2-B 产生的 **3 条** usage 纳入对账（若 R2-B blocked 则图片项标 skip）

### 必须验证

| 项 | 期望 |
|----|------|
| 控制台 `unified_total_actual_cost` | 与所选样本明细之和一致（USD 口径） |
| 每条 CNY→USD | `cost / usd_cny_rate` = 余额 Δ（±0.0001 USD 浮点容差） |
| `usd_cny_rate` | 记录 dev 当前值（默认 7.20）与调整入口 |
| 火山账单 ±1% | 列出 `upstream_task_id`，**待老板**在方舟控制台核对（Codex 不登录供应商） |
| usage_log 幂等 | 每 `video_task_id` 仅 1 行 |

### 输出表（审查包必填）

| # | 类型 | id | cost (CNY) | balance Δ (USD) | 一致？ |
|---|------|-----|------------|-----------------|--------|
| 1 | video | 4 | 5.0094 | -0.69575 | pass（已有） |
| 2 | video | … | | | |
| 3 | video | … | | | |
| 4 | image | … | | | |
| 5 | image | … | | | |
| 6 | image | … | | | |

---

## S1-3 — PR #3726 合并准备

### 目标

产出「可合并」清单，**Codex 不 force-push、不自行 merge**（除非老板明确要求）。

### 必查

| 项 | 动作 |
|----|------|
| PR 状态 | `gh pr view 3726` 或 GitHub UI：CI、review、conflict |
| 与 main 差异 | 无意外大文件、无 `.env`/密钥 |
| `00_START_HERE.md` | 与 V5.0 `#current-state` 一致（2026-07-05 已同步，合并前再 spot-check） |
| R2 收口 | 更新 [R2-closeout-summary](./deliverables/2026-07-05-R2-closeout-summary.md) 中 R2-B/C 行 |
| 合并后 | 建议老板：打 tag / 通知 QCanvas S2 可引用 Sub2API v3 契约 |

### 验收

审查包 §「合并门禁」表：每项 pass / fail / N/A + 链接或命令输出。

---

## 明确禁止

- 不要提交 `.env`、真实 API Key、员工密码
- 不要 force-push `main` / `master`
- 不要改 `frontend/src/views/admin/console/`（除非 roadmap 明确后端配套且无可避免）
- 不要以 QCanvas 画布 UI 作 Sub2API 验收证据（两仓分离）

---

## 完成后汇报（对老板一句话）

> 「S1 R2-B/C 已完成（或 R2-B blocked：缺 NB2 Key），审查包：`docs/superpowers/codex-handoff/deliverables/YYYY-MM-DD-S1-R2BC-review.md`；PR #3726 合并门禁：pass/fail。」

---

## 与 QCanvas 的边界

Sub2API S1 完成后 **不等于** QCanvas 可真实出片。画布 S2（content[] + 去 mock 壳层）见 QCanvas 仓：

`D:\Codex创业任务\QCanvas（无界版）\QCanvas\docs\codex-handoff\CODEX_TASK_S2.md`

**现在从 S1-0 合并门禁自检开始。**
