# Sub2API night moat commit groups log (2026-07-02)

## Baseline

- Repo: `D:\sub2api-trunk`
- Branch start: `wujie/trunk`
- HEAD start: `38df1bcd`
- Work branch: `wujie/night-moat-20260702`
- Red lines: no push, no merge, no delete, no real provider call, no credential reads, no AUTH_CONTRACT changes.
- Extra untracked outside groups: `MORNING_RESULT_2026_06_28.md` left untouched unless later explicitly classified.

## Group 1 - archive root docs

- Status: committed.
- Gate: status/path review only; no runtime safety gate needed for pure archive/docs move.
- Commit: `a3e6d935` `chore(archive): reorganize legacy root docs into _archive`.

## Group 2 - backend, migrations, worker hardening

- Status: committed.
- Planned gate: migration read-only/destructive-term review, `go build ./...`, `go test ./...`.
- Migration review: 142/144 add concurrent indexes; 143 adds nullable worker claim columns; 145 creates daily reservation table. `ON DELETE SET NULL` appears only as FK behavior, not a destructive command.
- Gate result: `go build ./...` exit 0; `go test ./...` exit 0.
- Commit: `0d0221c3` `fix(video-gateway): timezone startup, S3 streaming, worker concurrency, daily-trial reservation fixes`.

## Group 3 - frontend

- Status: committed.
- Planned gate: `pnpm install --frozen-lockfile`, `pnpm test:run`, `pnpm typecheck`, `pnpm lint:check`.
- Gate result:
  - `pnpm install --frozen-lockfile` exit 0 (`Already up to date`).
  - `pnpm test:run` exit 0 (`97 passed`, `581 passed`).
  - `pnpm typecheck` exit 0.
  - `pnpm lint:check` exit 0.
- Commit: `4fddab3e` `fix(frontend): stabilize admin usage views and table preferences`.

## Group 4 - tooling and editor rules

- Status: committed.
- Planned gate: `python tools/secret_scan.py --include-untracked`; if hits appear, exclude hit files only and commit the rest.
- Explicit exclusion: `.impeccable/hook.cache.json`.
- Note: system `python` is a WindowsApps shim and exits 1 without output; used bundled Python at `C:\Users\浩臣移动工作站\.cache\codex-runtimes\codex-primary-runtime\dependencies\python\python.exe`.
- Gate result: `tools/secret_scan.py --self-test` exit 0; `tools/secret_scan.py --include-untracked` exit 0 (`no high-confidence tracked-plus-untracked findings`).
- Commit: `8280ce57` `chore(tooling): add local quality gates and secret scanner`.

## Group 5 - review evidence

- Status: committed.
- Planned gate: `tools/secret_scan.py --include-untracked`; if clean, add `_review/`.
- Gate result: bundled Python `tools/secret_scan.py --include-untracked` exit 0 (`no high-confidence tracked-plus-untracked findings`).
- Boundary: current scanner intentionally skips `_review`; historical evidence was committed without whitespace cleanup to avoid rewriting evidence.
- Commit: `3fa81e9a` `docs(review): track historical Sub2API review evidence (M0A-M1B, D3 redaction, dashboard C)`.

## S1 closeout

- Five requested commit groups completed locally on `wujie/night-moat-20260702`.
- No push, no merge, no real provider call, no credential read.

## S2 - moat status

- Status: committed.
- Output: `_review/night-moat-20260702/moat-status.md`.
- Evidence inputs: `git show --stat` for `6478237a`, `b919650f`, `eca1b65c`, `e078749a`, `38df1bcd`; current-code `rg` anchors for content capture, retention, dashboard, and skill gaps.
- Commit: `60da4f5c` `docs(review): record Sub2API moat status`.

## S3 - Skill Engine v0 skeleton

- Status: committed.
- Files: `_review/night-moat-20260702/skill-engine-design.md`, `backend/internal/service/skill_engine.go`, `backend/internal/service/skill_engine_test.go`.
- Safety posture: flag-off by default; enabled path returns explicit not-implemented; no provider call, no DB write, no route wiring.
- Gate result:
  - `go test ./internal/service -run TestSkillEngine -count=1` exit 0.
  - `go build ./...` exit 0.
  - `go test ./...` exit 0.
  - bundled Python `tools/secret_scan.py --include-untracked` exit 0.
- Commit: `cbf7af25` `feat(moat): add skill engine v0 skeleton (flag off)`.

## Optional Stage C - dependency audit

- Status: committed.
- Command: `pnpm audit --json` from `frontend`.
- Result: exit 1, vulnerabilities reported; raw JSON saved to `_review/night-moat-20260702/frontend-pnpm-audit-raw.json`.
- Summary: 1 critical / 14 high / 33 moderate / 3 low, 51 advisories total.
- Boundary: report only; no upgrade/fix/approve-builds command executed.
- Commit: `55040eef` `docs(review): add optional dependency audit report`.

## Final closeout

- Status: in progress.
- Current branch: `wujie/night-moat-20260702`.
- Final secret scan: bundled Python `tools/secret_scan.py --include-untracked` exit 0.
- Remaining untracked, intentionally not committed: `.impeccable/`, `MORNING_RESULT_2026_06_28.md`.
