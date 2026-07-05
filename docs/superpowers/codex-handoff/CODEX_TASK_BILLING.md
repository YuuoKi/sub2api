# Codex 任务书 · 计费补齐专项（P0-2R）

> 生成时间：2026-07-05  
> 前置：读完 [CODEX_START_HERE.md](./CODEX_START_HERE.md) 再读本文件。本文件是 P0-2 的**展开版**，官方价格数据已由 Claude 于 2026-07-05 调查核实并写在下方，**你不需要再上网查价**。  
> 交付：每个子任务完成后按 [DELIVERABLE_TEMPLATE.md](./DELIVERABLE_TEMPLATE.md) 在 `deliverables/` 写审查包。

---

## 背景：审计结论（2026-07-05，Claude 核实）

产品定位：N 个上游密钥聚合成一条出口，调用数据回收归总分析。**聚合调度已完备**，缺口全部在计费：

| 领域 | 现状 | 结论 |
|------|------|------|
| LLM token 计费 | LiteLLM JSON + 渠道覆盖 + 硬编码兜底，三层解析 | ✅ 基本完备，小修 |
| 图片计费（nanobanana2） | 按图计费框架存在，但**无 NB2 价格条目**，兜底 $0.134 是 NB Pro 价 | ❌ NB2 会被按 2 倍价记账 |
| 视频计费（Seedance） | `cost_estimate` 恒为 0，`Charge()` 是空操作，无价格表 | ❌ 完全没有真实计费 |
| 视频+LLM 归总 | `usage_logs` 与 `video_usage_logs` 各自独立，总览不含视频 | ❌ 老板看到的总花费缺视频 |
| 对账字段 | 无 `pricing_source` / `pricing_version` / `currency` | ❌ 无法对官方账单 |

关键代码位置：

- LLM/图片计费核心：`backend/internal/service/billing_service.go`（`computeTokenBreakdown`、`CalculateImageCost`、`initFallbackPricing`）
- 价格解析链：`backend/internal/service/model_pricing_resolver.go`（channel → litellm → fallback）
- LiteLLM 打包价表：`backend/resources/model-pricing/model_prices_and_context_window.json`
- 视频计费（stub）：`backend/internal/service/video_gateway_billing.go`、`video_gateway_budget_guard.go`（`Charge` 为 no-op）
- 视频记录：`backend/internal/repository/video_gateway_repo.go`（`video_tasks` / `video_usage_logs`，migration 136）
- 归总：`backend/internal/repository/dashboard_aggregation_repo.go`（只算 `usage_logs`）

---

## 官方价格数据（2026-07-05 核实，写死可用）

### A. Nano Banana 2 = `gemini-3.1-flash-image-preview`（Google 官方，USD）

来源：https://ai.google.dev/gemini-api/docs/pricing

- 文本/图片输入：$0.50 / 1M tokens；文本输出：$3.00 / 1M tokens
- **图片输出：$60.00 / 1M image tokens**，按分辨率折合每图：

| 分辨率 | image tokens | 每图单价（标准） | 每图单价（Batch，5 折） |
|--------|--------------|------------------|--------------------------|
| 0.5K (512px) | 747 | **$0.045** | $0.022 |
| 1K (1024px) | 1120 | **$0.067** | $0.034 |
| 2K (2048px) | 1680 | **$0.101** | $0.050 |
| 4K (4096px) | 2520 | **$0.151** | $0.076 |

注意：现有 `CalculateImageCost` 的 2K=1.5×、4K=2× 倍率是按 NB Pro 设计的，NB2 的实际比率是 2K≈1.507×、4K≈2.254×，**不要用统一倍率，直接按上表落价**。

对照（已在打包 JSON 中，勿改错）：`gemini-3-pro-image-preview`（NB Pro）1K/2K = $0.134，4K = $0.24。

### B. Seedance（火山方舟官方，人民币 CNY）

来源：火山方舟计费文档（2026-03 公布）。按输出视频 token 量计费（元/百万 token），**仅对成功生成的视频计费**：

| 模型 | 在线推理（元/M tokens） |
|------|--------------------------|
| doubao-seedance-2.0 | 不含视频输入 46.00；含视频输入 28.00 |
| doubao-seedance-2.0-fast | 不含视频输入 37.00；含视频输入 22.00 |
| doubao-seedance-1.5-pro | 有声 16.00；无声 8.00（离线 8.00/4.00） |
| doubao-seedance-1.0-pro | 15.00（离线 7.50） |
| doubao-seedance-1.0-pro-fast | 4.20（离线 2.10） |
| doubao-seedance-1-0-lite | 10.00（离线 5.00） |

每秒折算参考（Seedance 2.0，不带参考视频）：480p ≈ ¥0.5/s，720p ≈ ¥1.0/s，1080p ≈ ¥2.5/s；Fast：480p ¥0.4/s，720p ¥0.8/s。15 秒纯生成约消耗 30.888 万 tokens ≈ ¥15。

**首选实现**：Ark 轮询响应里带 `usage.total_tokens`（任务完成时返回实际消耗 token）。**用真实 usage × 单价计费**，每秒折算表只做创建前预算 gate 的预估。

**币种**：Seedance 是 CNY，系统现有计费默认 USD。必须落 `currency` 字段，汇率换算方案见任务 V-3。

### C. Claude LLM（Anthropic 官方，USD / MTok）

来源：https://platform.claude.com/docs/en/about-claude/pricing（2026-07 核实）

| 模型 | Input | Output | Cache 写(5m) | Cache 读 |
|------|-------|--------|--------------|----------|
| Claude Fable 5 / Mythos 5 | $10 | $50 | $12.50 | $1.00 |
| Claude Opus 4.5–4.8 | $5 | $25 | $6.25 | $0.50 |
| Claude Sonnet 5（至 2026-08-31） | $2 | $10 | $2.50 | $0.20 |
| Claude Sonnet 5（2026-09-01 起） | $3 | $15 | $3.75 | $0.30 |
| Claude Sonnet 4.5 / 4.6 | $3 | $15 | $3.75 | $0.30 |
| Claude Haiku 4.5 | $1 | $5 | $1.25 | $0.10 |
| Claude Haiku 3.5 | $0.80 | $4 | $1.00 | $0.08 |

现有 fallback 核对结论：Opus 4.5/4.6/4.7 ✅ 正确；`claude-3-5-haiku` 写的 $1/$5 实为 Haiku 4.5 价（3.5 官方 $0.80/$4）❗；**缺** Fable 5 / Mythos 5 / Opus 4.8 / Sonnet 5 条目。

---

## 子任务清单（按顺序执行，每个出独立审查包）

### V-1 图片计费修正：NB2 价格落地【最高优先】

1. 在打包价表 `model_prices_and_context_window.json` 增加 `gemini-3.1-flash-image-preview` 条目（含 `output_cost_per_image` 分辨率梯度或按 `output_cost_per_image_token: 6e-05`）。
2. `CalculateImageCost` 支持 **按模型 × 分辨率查表**（0.5K/1K/2K/4K 四档），查不到才退统一倍率逻辑；新增 0.5K 档。
3. 兜底价 $0.134 保留但仅限 NB Pro 系模型；NB2 系兜底按上表 A。
4. 计费时把 `media_type='image'` 落进 `usage_logs`（列已存在但从未写入）。
5. 单测：NB2 四档价格断言、NB Pro 不回归、未知模型走兜底。

验收：模拟 NB2 1K 出图一次，`usage_logs.total_cost == 0.067`（rate=1 时 actual 同）。

### V-2 视频计费真实化：Seedance 计价引擎

1. 新建视频价格表（**代码内配置结构 + 可被 DB/config 覆盖**，不强制迁移）：`provider × model × (是否含视频输入/有声无声) → 元/M tokens`，数据用上表 B。
2. Seedance adapter 轮询完成时读取 Ark 响应 `usage.total_tokens`，写入 `VideoAdapterResult`；`cost = tokens/1e6 × 单价`。
3. `video_tasks.cost_estimate` 语义拆分：新增 `actual_cost`（或复用并注释），任务成功回收时写真实费用；失败任务费用为 0（官方规则：失败不计费）。
4. 创建前预算 gate 改用每秒折算表（B 中参考价）做预估，替代现在恒 0 的 `cost_per_second` 默认值。
5. `chargeForVideo` 从 no-op 改为真实记账：扣用户余额或记账到 usage 体系（与现有 `actual_cost` 语义一致，乘 rate multiplier）。
6. 单测：token→费用换算、失败不计费、预算 gate 拦截。

验收：一条 720p 5s Seedance 任务（真实或 mock usage=103k tokens），`video_usage_logs` 记录费用 ≈ ¥4.74（46 × 0.10296），与火山账单口径一致（±1%）。

### V-3 币种与对账字段

1. `usage_logs` 与 `video_usage_logs` 增加迁移：`currency`（默认 'USD'，视频 Seedance 记 'CNY'）、`pricing_source`（channel/litellm/fallback/provider_usage）、`pricing_version`（价表日期字符串）。
2. `ResolvedPricing.Source` 现在只在内存里，落库。
3. 汇率：新增系统设置 `usd_cny_rate`（管理员可改，默认 7.20），归总展示时统一折算成一种币种；**不要**自动拉汇率 API。
4. 单测：字段落库、归总换算。

### V-4 归总统一：总览含视频花费

1. 后端给 `/admin/dashboard/stats`（或新端点 `/admin/dashboard/unified-spend`）增加视频花费汇总：`video_total_cost`（按 V-3 汇率折算后）+ 分渠道明细。
2. `users-ranking` 保持现有行为，另出 `video_spend_by_user`（video_tasks 有 user_id）或直接在 ranking 里加 `video_cost` 列——选实现成本低的，写进审查包告诉 Claude。
3. 单测：LLM+视频合计正确。

### V-5 LLM 兜底价小修（顺手，低风险）

1. `initFallbackPricing`：修正 `claude-3-5-haiku` 为 $0.80/$4；新增 `claude-opus-4.8`（同 4.5）、`claude-fable-5`/`claude-mythos-5`（$10/$50，cache 写 $12.50、读 $1）、`claude-sonnet-5`（用 9 月后标准价 $3/$15，注释注明 8/31 前官方是 $2/$10）。
2. 模糊匹配确保 `fable`/`mythos` 命中新条目。
3. 单测断言各条目价格。

---

## 硬性约束

- 分层 handler → service → repository；depguard 过
- 每个子任务：`go test ./...` + `golangci-lint run ./...` 全绿
- 涉及迁移：新增 `backend/migrations/1xx_*.sql`，向后兼容，默认值安全
- **不改** `frontend/**`；前端展示（币种符号、视频费用列、统一总览卡）写进审查包「给 Claude 的说明」
- 不提交任何真实 Key；测试用 mock/sqlmock
- 价格数字以本文件表 A/B/C 为准；若实现中发现官方口径与本表矛盾，在审查包「风险」里说明，不要自行改价

## 交付顺序建议

```text
会话 1：V-1（图片 NB2）→ 审查包
会话 2：V-2（视频计价引擎）→ 审查包
会话 3：V-3 + V-4（币种/对账 + 归总）→ 审查包
会话 4：V-5（LLM 兜底小修）→ 审查包（可并入会话 3）
```
