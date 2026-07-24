# Task 2 Report — Sub2 员工复用开卡

**Worktree:** `D:\Codex创业任务\QCanvas（无界版）\QCanvas\.worktrees\sub2-guangzhou-hotfix-20260725-grok`  
**Branch:** `codex/grok-guangzhou-sub2-20260725`  
**Base:** `cc6a150c1644915c1576ca8e1263071a5a54e16f`  
**Date:** 2026-07-25

## Summary

Phase 2 only: Staff console now lists non-admin `human` + `tool` employees with visible member-type badges; create-user conflicts use the flat axios reject shape (`status` / `reason`) and reuse active owners without converting identity. Admin / disabled / not-found conflicts fail with explicit Chinese errors. Create-then-key-fail keeps `wizardUser` so retry does not recreate.

## TDD evidence

### RED (tests first, before implementation)

Command:

```bash
pnpm exec vitest run src/views/admin/console/__tests__/StaffView.keyOnce.spec.ts src/__tests__/gate2-remaining-ux-contract.spec.ts
```

Result: **7 failed | 18 passed (25)**

Failed (intentionally false-green rewritten / new contract):

1. lists human+tool + member type badges
2. flat `{ status: 409, reason: 'EMAIL_EXISTS' }` tool reuse
3. active human reuse (no identity conversion)
4. admin reject (Chinese)
5. disabled reject (Chinese)
6. not-found reject (Chinese)
7. gate2 source contract for human/tool list

Root cause of prior false-green: tests mocked `{ response: { status: 409 } }` while the real interceptor rejects a flat object, matching the buggy production check.

### GREEN (after implementation)

Same command: **25 passed (25)**  
`StaffView.keyOnce.spec.ts`: 16 passed  
`gate2-remaining-ux-contract.spec.ts`: 9 passed

## Files changed

| File | Change |
|------|--------|
| `frontend/src/views/admin/console/StaffView.vue` | List human+tool; member-type badge; flat 409/EMAIL_EXISTS; conflict resolve; keep owner on key fail |
| `frontend/src/views/admin/console/__tests__/StaffView.keyOnce.spec.ts` | Rewrite false-green 409 mocks; add human/admin/disabled/not-found/list/retry tests |
| `frontend/src/__tests__/gate2-remaining-ux-contract.spec.ts` | Contract updated for human/tool list + EMAIL_EXISTS (no tool-only hard filter) |
| `.superpowers/sdd/task-2-sub2-card-report.md` | This report |

## Behavior contract (implemented)

1. **List:** non-admin accounts where `member_type` is `human` or `tool`; badge `员工账号` / `工具账号`.
2. **Conflict detect:** `status === 409` OR `reason === 'EMAIL_EXISTS'` on flat reject object (not `error.response.status`).
3. **Exact email** (`trim` + lowercase):
   - active human/tool → reuse owner → dual-key + recharge
   - admin → `该邮箱属于管理员账号，不能用于员工开卡`
   - disabled → `该邮箱对应账号已停用，无法开卡，请先启用后再试`
   - not found → `该邮箱已被占用，但找不到精确匹配的账号…`
   - never auto-convert human↔tool
4. **Retry:** `wizardUser` retained after create success + later key failure; second submit skips `users.create`.

## Out of scope (not touched)

- Dialog close behavior (Phase 3)
- VersionBadge (Phase 4)
- No `.env` reads, no push, no provider calls

## Self-review

- [x] Flat interceptor error shape handled
- [x] Tool-only hard filter removed from `loadStaff` / `filteredUsers` / email lookup
- [x] Explicit Chinese failures; no silent fallback / identity conversion
- [x] Create-then-key-fail retry safe
- [x] Unit tests cover reuse + rejects + list
- [x] Commit message exact; Phase 3/4 excluded
