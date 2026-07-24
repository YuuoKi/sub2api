# Task 4 Report — Sub2 不可变构建身份与自更新移除

**Worktree:** `D:\Codex创业任务\QCanvas（无界版）\QCanvas\.worktrees\sub2-guangzhou-hotfix-20260725-grok`  
**Branch:** `codex/grok-guangzhou-sub2-20260725`  
**Base tip:** `ecc10b66828c154c68eb37ee9cf62cf756aadec0`  
**Commit:** `59d5b17f555c8e105bf9acbb1510f86a8b4ed6c7`  
**Date:** 2026-07-25

## Summary

Phase 4 only: VersionBadge is read-only “当前部署版本”; self-update / online rollback UI and HTTP mounts are hard-deleted. Deploy identity is unified via `internal/buildinfo.Info` (`广州内部版 YYYY.MM.DD-rN` + `build_commit` + `build_date`) across HTML injection, public settings, admin `GET /version`, and CLI `--version`. App store never calls `check-updates`.

## TDD evidence

### RED → GREEN (frontend)

```bash
pnpm exec vitest run src/components/common/__tests__/VersionBadge.immutable.spec.ts src/stores/__tests__/app.version.immutable.spec.ts src/stores/__tests__/app.spec.ts
```

Result: **31 passed (31)** (plus guards suite **41 passed** after mock update)

- `VersionBadge.immutable.spec.ts`: no update/rollback/refresh/restart controls; SHA without forced `v`
- `app.version.immutable.spec.ts`: `fetchVersion` uses injected public identity; no update network path
- Conflicting tests deleted: `VersionBadge.restart.spec.ts`, `admin.system.rollback.spec.ts`

### GREEN (backend)

```bash
go test ./internal/buildinfo/ ./internal/server/routes/ -count=1
go test -tags unit ./internal/handler/ ./internal/handler/admin/ -count=1
go test -tags unit ./internal/handler/dto/ ./cmd/server/ ./internal/service/ -count=1 -run "Injection|ProvideServiceBuildInfo|PublicSettings|BuildInfo|SetVersion"
```

Result: all **ok**

- Routes: `check-updates` / `update` / `rollback-versions` / `rollback` not mounted; `GET /version` kept
- Four identity sources share one `buildinfo.Info`
- DTO injection schema includes `build_commit` / `build_date`

## Files changed (high signal)

| Area | Change |
|------|--------|
| `backend/internal/buildinfo/` | Single BuildInfo source + label formatting |
| `backend/internal/server/routes/admin.go` | Unmount self-update routes |
| `backend/internal/handler/admin/system_handler.go` | `GetVersion` from BuildInfo only |
| `backend/internal/handler/setting_handler.go` + injection/DTO | Public contract adds `build_commit`/`build_date` |
| `frontend/src/components/common/VersionBadge.vue` | Read-only deploy identity UI |
| `frontend/src/stores/app.ts` + `api/admin/system.ts` | No `check-updates`; immutable identity |
| Tests | New immutable contracts; delete self-update specs |

## Behavior contract (implemented)

1. UI shows only **当前部署版本** (`广州内部版 YYYY.MM.DD-rN`); details = full SHA + build time; never force-prefix SHA with `v`.
2. Backend mounts only `GET /admin/system/version` (+ ops `POST /restart` kept off VersionBadge).
3. One Go object (`buildinfo.Info`) feeds HTML injection, public settings, admin version, CLI `--version`.
4. Public JSON adds `build_commit`, `build_date`.
5. Zero `check-updates` network after load (store uses injected/public settings).

## Out of scope

- Phase 2/3 staff/dialog work
- Real deploy / push / Provider calls
- Reading `.env`

## Self-review

- [x] Hard delete self-update UI (not lan_admin hide)
- [x] Routes unmounted
- [x] Four identity sources consistent
- [x] Conflicting tests removed / replaced
- [x] Exact commit message; no push
