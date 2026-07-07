# MLA Dbug Progress — 2026-07-07

> 执行者：Codex
> 范围：`CODEX_TASK_MLA_DBUG.md`
> 当前状态：DBug-6 已完成；下一 Phase 为 DBug-7（VideoTasksView 轮询）
> 红线：不 push、不 deploy、不读 `.env`、不触发真实 provider、每个修复 Phase 单独 commit

| Phase | Status | Commit | 审查包 |
|-------|--------|--------|--------|
| DBug-0 | done | — | `docs/superpowers/codex-handoff/deliverables/2026-07-07-DBug-0-baseline.md` |
| DBug-1 | done | `414ff2e9` | `docs/superpowers/codex-handoff/deliverables/2026-07-07-DBug-1-webhook-stale-recharging.md` |
| DBug-2 | done | `704c8f4c` | `docs/superpowers/codex-handoff/deliverables/2026-07-07-DBug-2-token-refresh-unify.md` |
| DBug-3 | done | `0b8eee6c` | `docs/superpowers/codex-handoff/deliverables/2026-07-07-DBug-3-payment-jsapi-orphan-order.md` |
| DBug-4 | done | `2ac31bf5` | `docs/superpowers/codex-handoff/deliverables/2026-07-07-DBug-4-backend-mode-stripe-return.md` |
| DBug-5 | done | `35d49031` | `docs/superpowers/codex-handoff/deliverables/2026-07-07-DBug-5-drama-list-pagination.md` |
| DBug-6 | done | `b5350c38` | `docs/superpowers/codex-handoff/deliverables/2026-07-07-DBug-6-keysview-error-display.md` |
| DBug-7 | pending | | |
| DBug-8 | pending | | |

## DBug-0 Notes

- 当前仓库：`D:/sub2api-trunk`
- 当前分支：`wujie/video-capture-moat-20260702`
- 当前 HEAD：`f2c6a61d`
- 后端快速门禁：`go test ./internal/service -run Payment -count=1` PASS
- 前端快速门禁：`npx.cmd vitest run src/router/__tests__/guards.spec.ts --reporter=basic` PASS
- 当前脏树：文档/证据/交付物输入项；未见 `backend/**`、`frontend/**` 或 `.cursor/hooks/**` 生产代码脏改
- DBug-0 不 commit；DBug-1 起恢复每 Phase 单独 commit
