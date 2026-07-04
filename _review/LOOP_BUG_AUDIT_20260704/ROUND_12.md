# Round 12 — Affiliate Accrual Re-check

**Trigger:** duplicate round 10 wake skipped; round 12 executed

## Re-verification

| Layer | Idempotency | Status |
|-------|-------------|--------|
| Service `tryClaimAffiliateRebateAudit` | `INSERT ... ON CONFLICT` on payment_audit_logs | **Present** (`payment_fulfillment.go:452+`) |
| Service `applyAffiliateRebateForOrder` | Gated by claim; calls repo `AccrueQuota` | Claim protects normal path |
| Repo `AccrueQuota` | Ledger INSERT **no UNIQUE** on `(source_order_id, action, user_id)` | **CONFIRMED** LBA-P1-022 |
| Repo `GetAccruedRebateFromInvitee` + `AccrueQuota` | Separate read/write, no lock | **CONFIRMED** LBA-P1-023 |

## New note

- **LBA-P3-025:** If `tryClaimAffiliateRebateAudit` succeeds but `AccrueQuota` fails mid-tx rollback, claim row may block retry depending on audit finalization path — needs trace of `updateClaimedAffiliateRebateAudit` on failure (not expanded this round).

## Round 12 outcome

0 new P1; prior affiliate findings **re-confirmed**; +1 hygiene note; open ~166+
