# Sub2 adversarial P0/P1 fix report

**Worktree:** `.worktrees/sub2-guangzhou-hotfix-20260725-grok`  
**Branch:** `codex/grok-guangzhou-sub2-20260725`  
**Base tip:** `e4abbf474`  
**Date:** 2026-07-25  
**Process:** RED tests first → GREEN fixes → commits (no push, no `.env`)

## Verdict

| Item | Severity | Status |
|------|----------|--------|
| Blank idempotent replay / double-submit → false success | P0 | **FIXED** |
| Bare gateway 409 treated as EMAIL_EXISTS | P0/P1 | **FIXED** |
| `page_size=100` truncation on staff list / conflict resolve | P1 | **FIXED** |
| CreateQCanvasKeyPair allows admin/disabled targets | P1 | **FIXED** |
| `idempotency.observe_only` default true weakens RequireKey | P1 | **FIXED** |
| Multi-admin race: concurrent dual-key issuance per user | P2 | **RESIDUAL** (documented, no redesign) |

## Commits

| SHA | Message |
|-----|---------|
| `a195b0831b06598f930dc06090485d20b528f249` | `fix(console): reject blank idempotent key replay and paginate staff list` |
| `aa2c2f8e0707b260044fc87cab199ee705cefb09` | `fix(api): reject admin/disabled qcanvas key targets; default observe_only false` |

## Changes

### Console (`StaffView.vue` + tests)

1. **Single-flight:** `submitStaff` starts with `if (submitting) return`.
2. **Blank replay hard-fail:** After pair response, both keys empty → Chinese `showError`, never assign `issuedQCanvasPair`, no「我已安全保存，完成」success path; `wizardUser` kept for retry without re-create.
3. **409 detector:** `isEmailConflictError` requires `reason === 'EMAIL_EXISTS'` only (bare status 409 is not email conflict).
4. **Pagination:** `listStaffPages` loops `page_size=100` until exhausted; used by `loadStaff` and `findAccountByExactEmail`.

### API (`api_key_service.go`, idempotency defaults)

1. **CreateQCanvasKeyPair:** Reject `role=admin` / `status=disabled` with Chinese BadRequest **before** minting keys.
2. **observe_only default false:** `DefaultIdempotencyConfig()`, `viper.SetDefault("idempotency.observe_only", false)`, `deploy/config.example.yaml`. StaffView still always sends UUID Idempotency-Key.

## Tests (GREEN)

```text
# frontend
pnpm exec vitest run src/views/admin/console/__tests__/StaffView.keyOnce.spec.ts \
  src/__tests__/gate2-remaining-ux-contract.spec.ts
→ 2 files / 37 tests passed

# backend
go test ./internal/service/ -run "TestAPIKeyService_CreateQCanvasKeyPair|TestIdempotencyCoordinator_RequireKey|TestIdempotencyCoordinator_ExecuteNilExecutor"
go test ./internal/config/ -run "TestLoadDefaultIdempotencyConfig|TestLoadIdempotencyConfigFromEnv"
go test ./internal/handler/admin/ -run "QCanvas|Idempotency|APIKeyCreate"
→ all ok
```

New/updated RED→GREEN coverage:

- Double-submit single-flight
- Blank pair keys → no `wizard-done` / hard failure / wizardUser retained
- Bare 409 without EMAIL_EXISTS → no reuse
- Staff list multi-page merge
- Conflict search multi-page exact owner
- CreateQCanvasKeyPair rejects admin/disabled
- Default `observe_only=false`; RequireKey rejects missing key; env can still force observe mode

## Residual risks

1. **P2 multi-admin race:** Two admins (or parallel tabs with different Idempotency-Keys) can still mint multiple active video/media pairs for the same user. A clean fix needs a server-side uniqueness / “one active dual-key issuance per user” constraint (DB partial unique index or claim row) — not done here to avoid redesign scope.
2. **Blank replay recovery UX:** User keeps the form and can retry with the same session Idempotency-Key; if the first issuance already succeeded server-side, retries keep returning blank secrets until a new modal session (new UUID). Error copy tells them not to treat it as success.
3. **observe_only deploy drift:** Example/default is now `false`. Existing production YAML that still sets `observe_only: true` will keep observe mode until ops updates config — intentional override path.
4. **Pagination cost:** Full staff list / conflict search may issue multiple `/admin/users` pages when totals exceed 100; acceptable for console scale, not an exact-email endpoint.

## Explicit non-claims

- No push.
- No `.env` / secret reads.
- No READY / boss verification claim.
- P2 multi-admin issuance race remains open.
