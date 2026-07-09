# Sub2API START HERE

Updated: 2026-07-09 Asia/Shanghai

## North Star (唯一总规划真相源)

跨仓总规划请先读：`D:\Codex创业任务\QCanvas（无界版）\北极星\北极星V5.0_无界智能画布_无界AI管理中台_总规划_最终版_20260705.html`  
分工：Fable 5 规划 · Codex 后端执行 · Composer 2.5 前端/走查。本仓任务书：`docs/superpowers/codex-handoff/CODEX_START_HERE.md`。

## Current Position

Sub2API is the Wujie internal AI API / model scheduling / ledger middle layer. It is useful for internal decision evidence and controlled demos, not a public production platform.

## Current Status (对齐北极星 V5.0 · Unified Sweep 2026-07-09)

- Branch: `wujie/video-capture-moat-20260702` · PR [#3726](https://github.com/Wei-Shaw/sub2api/pull/3726)
- Admin console v2: `内部可用 / 可演示` — 总览下钻、月预算进度、额度告警、备份超期提示
- Real Seedance + balance deduct: `内部可用` — R2-A（task #4，16:9）；**G6 Form A 9:16 done**（≈¥5）；**R2-B NB2 产品链 done**（usage_log#1，$0.045）；v2v skip
- Billing: `内部可用` — migrations 149–155（含 `local_asset_*`）
- Phase B 运营：卡额度告警 + 月预算 + 成品本地归档 + 采纳/周报（console 提示词 tab）已落地
- 当前执行入口：`docs/superpowers/codex-handoff/CODEX_TASK_UNIFIED_SWEEP_20260709.md` · 审查包 `deliverables/2026-07-09-UNIFIED-SWEEP-review.md`
- Real paid calls: 按 `production_authorized` + 老板逐 Key 授权；**临时 Key 用后请立即废弃**；未授权不得日常真实调用

## Historical Evidence (追溯用 · 非当前规划入口)

- Night Phase B: `deliverables/2026-07-09-NIGHT-PHASEB-review.md`
- R2-A: `deliverables/2026-07-05-R2-A-production-smoke-review.md`
- MLA DBug closeout: `deliverables/2026-07-08-MLA-DBUG-CLOSEOUT.md`

## Do Not Claim

- Do not claim public production readiness or multi-tenant SaaS.
- Do not claim real paid provider daily calls are generally authorized.
- Do not claim QCanvas S2 / root pnpm gates are solved.
- Do not read or print `.env`, keys, tokens, cookies, or provider credentials.

## Boundaries

- No push, deploy, reset, clean, or rebase.
- No real provider call without explicit authorization, budget, stop condition, and redaction plan.
- Keep Sub2API admin credentials server-side. QCanvas web must call QCanvas hono-api, not Sub2API admin APIs directly.
