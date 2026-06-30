# Sub2API night moat SUMMARY (2026-07-02)

## Conclusion

Local-only branch `wujie/night-moat-20260702` completed S1, S2, S3, and optional C. No push, merge, real provider call, credential read, or AUTH_CONTRACT change was performed.

## Commits On This Branch

| Commit | Message |
| --- | --- |
| `a3e6d935` | `chore(archive): reorganize legacy root docs into _archive` |
| `0d0221c3` | `fix(video-gateway): timezone startup, S3 streaming, worker concurrency, daily-trial reservation fixes` |
| `4fddab3e` | `fix(frontend): stabilize admin usage views and table preferences` |
| `8280ce57` | `chore(tooling): add local quality gates and secret scanner` |
| `3fa81e9a` | `docs(review): track historical Sub2API review evidence (M0A-M1B, D3 redaction, dashboard C)` |
| `60da4f5c` | `docs(review): record Sub2API moat status` |
| `cbf7af25` | `feat(moat): add skill engine v0 skeleton (flag off)` |
| `55040eef` | `docs(review): add optional dependency audit report` |

## Gates

| Area | Command | Result |
| --- | --- | --- |
| Backend group | `go build ./...` | exit 0 |
| Backend group | `go test ./...` | exit 0 |
| Frontend group | `pnpm install --frozen-lockfile` | exit 0 |
| Frontend group | `pnpm test:run` | exit 0, 97 files / 581 tests passed |
| Frontend group | `pnpm typecheck` | exit 0 |
| Frontend group | `pnpm lint:check` | exit 0 |
| Tooling/review/S3 | bundled Python `tools/secret_scan.py --include-untracked` | exit 0, no high-confidence findings |
| Skill engine | `go test ./internal/service -run TestSkillEngine -count=1` | exit 0 |
| Skill engine | `go build ./...` | exit 0 |
| Skill engine | `go test ./...` | exit 0 |
| Optional dependency audit | `pnpm audit --json` | exit 1, report only |

## Outputs

- `_review/night-moat-20260702/00_commit-groups-log.md`
- `_review/night-moat-20260702/moat-status.md`
- `_review/night-moat-20260702/skill-engine-design.md`
- `_review/night-moat-20260702/dependency-audit.md`
- `_review/night-moat-20260702/frontend-pnpm-audit-raw.json`
- `backend/internal/service/skill_engine.go`
- `backend/internal/service/skill_engine_test.go`

## Current Boundaries

- Skill Engine v0 is a flag-off skeleton. Default disabled returns `ErrSkillEngineDisabled`; enabled valid input returns `ErrSkillEngineNotImplemented`.
- Existing moat capture path remains dark by default; this task did not arm capture flags or verify live traffic.
- Optional dependency audit found 1 critical / 14 high / 33 moderate / 3 low frontend advisories; no upgrades were executed.
- Remaining untracked files intentionally left uncommitted: `.impeccable/`, `MORNING_RESULT_2026_06_28.md`.

## Rollback

Local rollback is by reverting the branch commits above in reverse order. No remote rollback is needed because nothing was pushed.
