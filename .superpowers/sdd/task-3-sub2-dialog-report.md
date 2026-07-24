# Task 3 Report — Sub2 无损开卡弹窗与稳定幂等键

**Worktree:** `D:\Codex创业任务\QCanvas（无界版）\QCanvas\.worktrees\sub2-guangzhou-hotfix-20260725-grok`  
**Branch:** `codex/grok-guangzhou-sub2-20260725`  
**Base:** `82ec1cc346f9a9c6527e06f0f2f2058ea43316d7` (Phase 2)  
**Date:** 2026-07-25

## Summary

Phase 3 only: Staff issuance and recharge modals are lossless. Backdrop / Escape no longer close or clear forms. After dual keys are shown, only 「我已安全保存，完成」 closes. Cancel with filled form confirms in Chinese before discard. One stable `crypto.randomUUID()` is minted on formal open and reused for all submit retries until cancel/complete.

## TDD evidence

### RED (tests first, before implementation)

Command:

```bash
pnpm exec vitest run src/views/admin/console/__tests__/StaffView.keyOnce.spec.ts
```

Result: **5 failed | 18 passed (23)**

Failed (new contracts):

1. staff backdrop does not close/clear
2. recharge backdrop does not close/clear
3. after keys: only 「我已安全保存，完成」 closes; Escape/backdrop do not
4. Cancel with filled form requires Chinese confirmation
5. stable idempotency key reused across retries; regenerated only after reopen

Escape-not-close cases already passed (no Escape close binding existed); implementation still adds an explicit capture-phase Escape blocker while either modal is open.

### GREEN (after implementation)

Command:

```bash
pnpm exec vitest run src/views/admin/console/__tests__/StaffView.keyOnce.spec.ts src/__tests__/gate2-remaining-ux-contract.spec.ts
```

Result: **32 passed (32)**  
`StaffView.keyOnce.spec.ts`: 23 passed  
`gate2-remaining-ux-contract.spec.ts`: 9 passed

## Files changed

| File | Change |
|------|--------|
| `frontend/src/views/admin/console/StaffView.vue` | Remove `@click.self` close; Escape blocker; cancel confirm; done-only close after keys; stable `staffIdempotencyKey` |
| `frontend/src/views/admin/console/__tests__/StaffView.keyOnce.spec.ts` | Backdrop / Escape / key-success close / cancel confirm / idempotency tests |
| `.superpowers/sdd/task-3-sub2-dialog-report.md` | This report |

## Behavior contract (implemented)

1. **Backdrop:** staff + recharge backdrops have no close handler; click does not clear forms.
2. **Escape:** window capture listener + dialog `@keydown.escape.prevent.stop` while modal open; does not close/clear.
3. **Key success:** cancel/X absent; only `data-test="wizard-done"` labeled 「我已安全保存，完成」 calls `completeStaffModal`.
4. **Cancel discard:** filled staff form → `requestConfirmation` Chinese prompt; refuse keeps form; accept clears and drops idempotency key.
5. **Idempotency:** `staffIdempotencyKey = crypto.randomUUID()` in `openCreateStaff`; all `createQCanvasKeyPairForUser` retries reuse it; cleared only on cancel/complete; next open generates a new key.
6. **Owner retry (Phase 2 preserved):** create-then-key-fail still keeps `wizardUser` and now also keeps the same idempotency key.

## Out of scope (not touched)

- VersionBadge / update routes (Phase 4)
- No `.env` reads, no push, no provider calls

## Self-review

- [x] `@click.self` removed from both modals
- [x] Escape cannot close either modal
- [x] One-shot keys only closable via 「我已安全保存，完成」
- [x] Filled-form cancel confirms before clear
- [x] Stable idempotency key per open session
- [x] TDD RED then GREEN
- [x] Commit message exact; Phase 4 excluded
