# Round 18 — FINDINGS Duplicate Sweep

**Trigger:** duplicate round 17 wake skipped; round 18 executed

## Merged duplicates (single canonical ID)

| Canonical | Duplicate / alias | Action |
|-----------|-------------------|--------|
| LBA-P0-001 | Thermo P1-7 markCompleted | Keep P0 only |
| LBA-P1-022 | R-P1-01 AccrueQuota | Same finding |
| LBA-P1-023 | R-P1-02 concurrent accrual cap | Same finding |
| LBA-P2-022 | R-P3-08 IncrementRateLimitUsage | Same finding |
| LBA-P2-026 | Video R-P3-4 RoutingEvents limit | Same finding |
| LBA-P1-015 | R-P3-07 refresh token orphan ops | Cross-layer chain |

## Count reconciliation

| Severity | Unique IDs | Notes |
|----------|------------|-------|
| P0 | 7 | stable |
| P1 | 32 | 001-014 + 015-017 + 018-021 + 022-026 + 027-032 |
| P2 | 27+ | see round files |
| P3 | 26+ | incl. test gaps + env hygiene |

**Open unique findings: ~167** (not 259 if aliases counted twice)

## Round 18 outcome

0 new bugs; ledger deduped; ready for rounds 19–25 re-verify passes
