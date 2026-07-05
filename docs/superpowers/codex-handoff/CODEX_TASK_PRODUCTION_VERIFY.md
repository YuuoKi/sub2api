# Codex 任务书 — 生产验证与收尾（R2）

> **前置**：PR [#3726](https://github.com/Wei-Shaw/sub2api/pull/3726) 已含 V-1→R1 后端 + 控制台 v2 前端。  
> **R2-A 状态**：`done`（2026-07-05，见 [R2-A 审查包](./deliverables/2026-07-05-R2-A-production-smoke-review.md)）

---

## R2-A — 正式 Seedance 端到端 ✅ DONE

真实任务 **#4** 已通过 API-key 正式路径（无 `trial_mode`）跑通：`succeeded`，108900 tokens，CNY 5.0094 扣费，余额 $20.00 → $19.30。

---

## R2-B / R2-C — 可选后续

见下方任务书章节；R2-C 已有 1 条真实视频样本（task #4）。

---

## 第一步：读这些

1. [CODEX_START_HERE.md](./CODEX_START_HERE.md)
2. [docs/api/video-gateway-contract.md](../../api/video-gateway-contract.md)
3. [deliverables/2026-07-05-R2-A-production-smoke-review.md](./deliverables/2026-07-05-R2-A-production-smoke-review.md)

---

## R2-B — 图片网关生产验证（A-3 回归）

对 `gemini-3.1-flash-image-preview`（NB2 四档）跑一条 API-key 或 JWT 作图：

- `imageConfig` / `responseModalities` 透传
- `usage_logs.media_type=image`、多图 count、512 档计价

交付：`deliverables/YYYY-MM-DD-R2-B-image-smoke-review.md`

---

## R2-C — 计费对账抽检（P0-2 收尾）

抽 3 条已完成视频任务 + 3 条图片 usage_log：

- 控制台 `unified_total_actual_cost` 与明细之和一致
- Seedance CNY 与供应商账单 ±1%
- 文档化 `usd_cny_rate` 当前值与调整方式

交付：`deliverables/2026-07-05-R2-C-billing-reconciliation-review.md`（partial，含 task #4）

---

## R2-D — 运维小项（可选，P1）

| ID | 任务 |
|----|------|
| R2-D1 | 卡额度 80%/100% 告警（P1-1 后端，复用 `balance_notify_*`） |
| R2-D2 | 任务成功后资产归档到 `/app/data/assets/`（P1-3） |

---

## 明确禁止

- 不要提交 `.env`、真实 API Key、密码
- 不要 force-push main
