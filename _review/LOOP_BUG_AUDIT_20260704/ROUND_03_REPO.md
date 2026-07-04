# Round 3 — Repository Layer (supplement)

**Agent:** [Repository layer adversarial review](93c2350c-d38a-4197-9f47-c3dd5c07728c)  
**Scope:** payment-adjacent, auth, video_gateway repos (no payment_order repo — lives in service/ent)

## Summary

| Severity | Count | Top themes |
|----------|-------|------------|
| P0 | 0 | No SQL injection in scoped repos |
| P1 | 5 | Affiliate double-accrual; idempotency state regression; promo max_uses race |
| P2 | 8 | Auth identity two-phase create; bind TOCTOU; video trial ambiguity; UpdateTask no version guard |
| P3 | 12 | nil-receiver silent no-ops; Redis refresh token orphan ops; RowsAffected discarded |

## P1 IDs (mapped to LBA)

- **LBA-P1-022** — `affiliate_repo.go:117-163` — AccrueQuota no unique/ON CONFLICT on ledger
- **LBA-P1-023** — `affiliate_repo.go:165-181` — concurrent accrual can exceed per-invitee cap
- **LBA-P1-024** — `idempotency_repo.go:175-214` — MarkSucceeded/Failed no status guard
- **LBA-P1-025** — `idempotency_repo.go:217-236` — DeleteExpired removes processing rows
- **LBA-P1-026** — `promo_code_repo.go:242-247` — IncrementUsedCount no max_uses CAS

## Note

Payment order persistence is **not** in repository/ — aligns with prior P0 findings in `payment_fulfillment.go` (service layer).
