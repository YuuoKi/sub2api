# CODEX_TASK_MLA_DBUG_CLOSEOUT — Dbug 收尾：全量门禁 + 台账同步（2026-07-08）

> **执行者：Codex（Sub2API 仓，单会话）。预计 60–90 分钟，纯收尾，无新功能。**
> 背景：DBug-1~8 全部完成（8 个 fix commit + 补强 `42b94ca2`，各 Phase 窄测全绿），但任务书 §4 规定的全量门禁未跑、CLOSEOUT 缺失、审计台账未同步。本任务补齐这三件事。
>
> **禁止**：push、deploy、读/写 `.env`/密钥、真实 provider 调用、reset/clean/rebase、任何生产代码改动（除非全量门禁红了才允许最小修复）。

## Phase C1 · 全量门禁

```powershell
cd d:\sub2api-trunk\backend
go test ./...
golangci-lint run ./...
cd ..\frontend
pnpm run lint:check
pnpm run typecheck
pnpm exec vitest run
```

- 全绿 → 进 C2。
- 有红 → 最多修 3 轮（最小修复，单独 commit `fix(dbug-closeout): <原因>`）；仍红 → 标 BLOCKED 写明原因并停止。

## Phase C2 · CLOSEOUT 文档

新建 `docs/superpowers/codex-handoff/deliverables/2026-07-08-MLA-DBUG-CLOSEOUT.md`：

- `## Final Gate`：C1 全量门禁完整计数（tests / lint issues）
- `## Commits`：DBug-1~8 fix commit hash 表 + 补强 `42b94ca2`（补记其内容：DBug-3 JSAPI bridge timeout、DBug-5 Postgres integration test）
- `## Bug 映射`：MLA-P1-001/005/006/007/008、MLA-P2-001/007/013、MLA-REV-SUP1 → Fixed + commit
- `## Still Open`：如无则写「无」
- `## Verdict`：一句话（状态词只用 内部可用/可演示/待复核/已阻塞/已冻结）

## Phase C3 · 审计台账与任务表同步

1. `_review/MEGA_LOOP_AUDIT_20260707/FINDINGS.md`：8 条 Confirmed 改 `Fixed (commit <hash>)`
2. `_review/MEGA_LOOP_AUDIT_20260707/TRUTH_MATRIX.md`：追加 MLA 修复行（锚定 HEAD）
3. `_review/MEGA_LOOP_AUDIT_20260707/FINAL_REPORT.md`：执行摘要「7 confirmed bugs remain」处加更新注记（不删原文，追加 2026-07-08 状态行）
4. `CODEX_TASK_MLA_DBUG.md` §6 进度表：全部标 done
5. `2026-07-07-MLA-DBUG-PROGRESS.md`：补记 `42b94ca2`
6. `CODEX_START_HERE.md`：MLA DBug 标记 **done**，指向本 CLOSEOUT

## Phase C4 · 收工 commit

```
docs(dbug): close out MLA dbug loop with full gates and ledger sync
```

单 commit 收全部文档改动；commit 前 `git diff --check`；确认 `git status` 干净后结束。
