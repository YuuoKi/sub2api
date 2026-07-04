# Round 4 — P0 Verification + Dedupe

**Trigger:** AGENT_LOOP_WAKE_bugaudit (round 4)

## P0 Verification (code-path confirmed)

| ID | Verdict | Evidence |
|----|---------|----------|
| LBA-P0-001 | **CONFIRMED** | `markCompleted` Save result unused; audit always written (`payment_fulfillment.go:302-313`) |
| LBA-P0-002 | **CONFIRMED** | `confirmPayment` Get err → `return nil` (`:71-74`) |
| LBA-P0-003 | **CONFIRMED** | `parseLegacyPaymentOrderID` only on NotFound; numeric suffix → PK (`:38-48`, `:51-67`); test covers `sub2_42` |
| LBA-P0-004 | **CONFIRMED** | `alreadyProcessed` EXPIRED case `return nil` after audit (`:193-204`) |
| LBA-P0-005 | **CONFIRMED** | JWT `CreateTask` handler calls `video.CreateTask` directly; no `CreateDailyTrialTask` (`video_handler.go:191-203` vs trial at `video_gateway_service.go:451-461`) |
| LBA-P0-006 | **CONFIRMED** | `CreateDramaTask` → `CreateTask` with `ProviderAccountID` unset (0) → `resolveVideoRoute(ctx, 0)` (`drama_gateway_service.go:313-324`, `video_gateway_service.go:592`) |
| LBA-P0-007 | **CONFIRMED** | JWT request binds `ProviderAccountID` from client (`video_handler.go:192`) |

**All 7 P0 candidates verified in source. None downgraded.**

## Dedupe Notes

| Keep | Merge / cross-ref |
|------|-------------------|
| LBA-P0-001..007 | Distinct paths; no merge |
| LBA-P1-007 (handler fail-open) | Cross-ref LBA-P2-018/019 (repo nil masking) — same chain, different layer |
| LBA-P1-015 (refresh family) | Cross-ref R-P3-07 (Redis orphan tokens) |
| LBA-P1-014 (affiliate in fulfillment) | Distinct from LBA-P1-022 (repo AccrueQuota) — service vs repo |
| Thermo P1-7 markCompleted | Same as LBA-P0-001 — duplicate ID removed from agent raw output only |

## Round 4 Outcome

- P0 set stable at **7 verified**
- No new findings
- Open count unchanged: **122+**
