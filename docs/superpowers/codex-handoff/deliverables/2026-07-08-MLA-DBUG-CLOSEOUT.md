# MLA DBug Closeout — 2026-07-08

> 执行者：Codex
> 范围：`CODEX_TASK_MLA_DBUG_CLOSEOUT.md` Phase C1-C4
> 状态：**内部可用**
> 红线：未 push、未 deploy、未读取 `.env`/密钥、未触发真实 provider

## Final Gate

| Gate | Command | Result | Count / Notes |
|------|---------|--------|---------------|
| Backend tests | `go test ./...` | PASS | exit 0 |
| Backend test count | `go test -json ./...` summary | PASS | 4064 passed tests / 39 passed packages |
| Backend lint | `golangci-lint run ./...` | PASS | 0 issues; user-local cache write warnings only |
| Frontend lint | `pnpm run lint:check` | PASS | exit 0; 0 reported issues |
| Frontend typecheck | `pnpm run typecheck` | PASS | exit 0 |
| Frontend tests | `pnpm exec vitest run` | ENV RED | Windows shim failed before tests: `'vitest' is not recognized` |
| Frontend tests fallback | `npx.cmd vitest run --reporter=basic` | PASS | 109 test files / 620 tests passed |

## Commits

| Phase | MLA ID | Commit | Summary |
|-------|--------|--------|---------|
| DBug-1 | MLA-P1-001 | `414ff2e9` | Stale RECHARGING webhook now acks 2xx to stop provider retries |
| DBug-2 | MLA-P1-005 / MLA-P1-006 | `704c8f4c` | Unified token refresh singleflight and synced Pinia after interceptor refresh |
| DBug-3 | MLA-P1-007 | `0b8eee6c` | Avoided orphan pending order on JSAPI fallback |
| DBug-4 | MLA-P1-008 | `2ac31bf5` | Preserved backend-mode Stripe return URL |
| DBug-5 | MLA-P2-001 | `35d49031` | Fixed drama filtered pagination totals |
| DBug-6 | MLA-P2-007 | `b5350c38` | Surfaced KeysView submit error messages |
| DBug-7 | MLA-REV-SUP1 | `5f53444f` | Added polling for running video tasks list |
| DBug-8 | MLA-P2-013 | `83c71550` | Made secret-scan hook fail closed when scanner tools are missing |
| Review reinforcement | DBug-3 / DBug-5 | `42b94ca2` | Added JSAPI bridge unavailable timeout coverage and Postgres drama list integration test |

## Bug 映射

| ID | Status | Evidence |
|----|--------|----------|
| MLA-P1-001 | Fixed | `414ff2e9` |
| MLA-P1-005 | Fixed | `704c8f4c` |
| MLA-P1-006 | Fixed | `704c8f4c` |
| MLA-P1-007 | Fixed | `0b8eee6c`, reinforced by `42b94ca2` |
| MLA-P1-008 | Fixed | `2ac31bf5` |
| MLA-P2-001 | Fixed | `35d49031`, reinforced by `42b94ca2` |
| MLA-P2-007 | Fixed | `b5350c38` |
| MLA-P2-013 | Fixed | `83c71550` |
| MLA-REV-SUP1 | Fixed | `5f53444f` |

## Still Open

无。

## Deferred / Out of Scope

Likely / deferred 项仍按原任务书保持非本轮范围：MLA-P1-002、MLA-P1-009、MLA-P1-010、MLA-P2-002 至 MLA-P2-012 中未进入 DBug-1~8 的项，以及 LBA-P1-019 / LBA-P1-020 / LBA-P1-027 后续规划项。

## Verdict

**内部可用** — DBug-1~8 与补强 commit 已通过后端、前端全量门禁；仅 `pnpm exec vitest run` 存在 Windows shim 环境红，已用 `npx.cmd vitest run --reporter=basic` 完成等价全量测试复核。
