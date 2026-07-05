# Sub2API START HERE

Updated: 2026-07-05 Asia/Shanghai

## North Star (唯一总规划真相源)

跨仓总规划请先读：`D:\Codex创业任务\QCanvas（无界版）\北极星\北极星V5.0_无界智能画布_无界AI管理中台_总规划_最终版_20260705.html`  
分工：Fable 5 规划 · Codex 后端执行 · Composer 2.5 前端/走查。本仓任务书：`docs/superpowers/codex-handoff/CODEX_START_HERE.md`。

## Current Position

Sub2API is the Wujie internal AI API / model scheduling / ledger middle layer. It is useful for internal decision evidence and controlled demos, not a public production platform.

## Current Status (对齐北极星 V5.0 `#current-state` · 2026-07-05)

- Branch: `wujie/video-capture-moat-20260702` · PR [#3726](https://github.com/Wei-Shaw/sub2api/pull/3726)
- Admin console v2: `内部可用 / 可演示` — 总览 / 密钥库 / 成员与开卡 / 任务记录 / 系统（六导航）
- Real Seedance + balance deduct: `内部可用` — R2-A（task #4, ¥5.0094, 108900 tokens）；审查包 `docs/superpowers/codex-handoff/deliverables/2026-07-05-R2-A-production-smoke-review.md`
- Billing (LLM + image + video unify): `内部可用` — migrations 149–154；R2-B 图片冒烟、R2-C 3+3 扩样仍属 S1
- Generation-content ledger: `内部可用` — `ai_generation_content` 在真实链路运行
- Real paid calls: 按 `production_authorized` 账号级 gate + 老板逐 Key 授权；**未授权账号不得日常真实调用**
- Codex 执行入口: `docs/superpowers/codex-handoff/CODEX_START_HERE.md` + `deliverables/`

## Historical Evidence (追溯用 · 非当前规划入口)

- Phase A' tiny real (已被 R2-A 超越): `_review/phase-a-prime-tiny-real_20260702/success_result.json`
- G3 受控 dev mock: `_review/capture-arming-D2-20260702-G3/SUMMARY.md`
- LOOP bug audit + fix (2026-07-04): `_review/LOOP_BUG_AUDIT_20260704/`, `_review/BUG_FIX_STATUS_20260704/VERIFY.md`
- B2 webhook fix: `_review/B2_webhook_fix_20260704/`
- MEGA final audit: `_review/MEGA_LOOP_FINAL_AUDIT_20260703/`
- QCanvas B2 adoption bridge: QCanvas `docs/reviews/B2_adoption_20260704/`

## Do Not Claim

- Do not claim public production readiness or multi-tenant SaaS.
- Do not claim real paid provider daily calls are generally authorized (only `production_authorized` accounts after boss Key approval).
- Do not claim QCanvas S2 / root pnpm gates are solved.
- Do not read or print `.env`, keys, tokens, cookies, or provider credentials.

## Boundaries

- No push, deploy, reset, clean, or rebase.
- No real provider call without explicit authorization, budget, stop condition, and redaction plan.
- Keep Sub2API admin credentials server-side. QCanvas web must call QCanvas hono-api, not Sub2API admin APIs directly.
