# Sub2API START HERE

Updated: 2026-07-05 Asia/Shanghai

## North Star (唯一总规划真相源)

跨仓总规划请先读：`D:\Codex创业任务\QCanvas（无界版）\北极星\北极星V5.0_无界智能画布_无界AI管理中台_总规划_最终版_20260705.html`  
分工：Fable 5 规划 · Codex 后端执行 · Composer 2.5 前端/走查。本仓任务书：`docs/superpowers/codex-handoff/CODEX_START_HERE.md`。

## Current Position

Sub2API is the Wujie internal AI API / model scheduling / ledger middle layer. It is useful for internal decision evidence and controlled demos, not a public production platform.

## Current Status

- Overall status: `内部可用 / 待复核` for controlled internal mock/dev ledger work.
- Phase A' tiny real: `内部可用 / 可演示` for one controlled single-call evidence path only.
- B1 generation-content ledger: `内部可用 / 待复核`; adoption feedback, weekly report, and Admin ContentWall sample review path exist.
- B2 webhook P1: fixed in commit `628ddd10`; provider lookup failure no longer returns 2xx success, while unknown order ack semantics are preserved.
- B2 VERIFY.log: appended and tracked in commit `4b1cf24b`; `_review/MEGA_LOOP_FINAL_AUDIT_20260703/VERIFY.log` remains the final-audit log.
- LOOP bug audit + fix wave (2026-07-04): 25-round adversarial audit recorded in `_review/LOOP_BUG_AUDIT_20260704/`; fixes committed as 10 grouped commits (`17112f8b`..`f176c256`) covering all 7 P0 and ~24 of 32 P1 findings. Remaining known gaps: P1-019/020 (video billing brake, blocked before real Seedance), P1-027 (atomic quota reservation). Verification: `_review/BUG_FIX_STATUS_20260704/VERIFY.md`.
- QCanvas B2 bridge: QCanvas hono-api commit `5e87e3f` proxies `POST /sub2api/v1/generation-content/:task_id/adoption`; QCanvas web commit `c0f1deb` adds Studio V2 session History and adoption UI.

## Current Truth Sources

- Bug audit + fix package (latest): `_review/LOOP_BUG_AUDIT_20260704/FINDINGS.md`, `_review/BUG_FIX_STATUS_20260704/VERIFY.md`
- B2 webhook package: `_review/B2_webhook_fix_20260704/VERIFY.md`, `_review/B2_webhook_fix_20260704/SUMMARY.md`
- Final audit package: `_review/MEGA_LOOP_FINAL_AUDIT_20260703/`
- Chat context: `docs/CHAT_STRATEGY_CONTEXT.md`
- Phase A' success evidence: `_review/phase-a-prime-tiny-real_20260702/success_result.json`
- QCanvas B2 package: `D:\Codex创业任务\QCanvas（无界版）\QCanvas\docs\reviews\B2_adoption_20260704\`

## Do Not Claim

- Do not claim public production readiness.
- Do not claim real paid provider daily calls are generally authorized.
- Do not claim QCanvas/Auth/root pnpm gates are solved.
- Do not read or print `.env`, keys, tokens, cookies, or provider credentials.

## Boundaries

- No push, deploy, reset, clean, or rebase.
- No real provider call without explicit authorization, budget, stop condition, and redaction plan.
- Keep Sub2API admin credentials server-side. QCanvas web must call QCanvas hono-api, not Sub2API admin APIs directly.
