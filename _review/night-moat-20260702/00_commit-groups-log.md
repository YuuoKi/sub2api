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

- Status: in progress.
- Planned gate: `tools/secret_scan.py --include-untracked`; if clean, add `_review/`.
