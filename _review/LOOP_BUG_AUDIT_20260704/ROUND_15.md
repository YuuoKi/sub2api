# Round 15 — Video Worker Lifecycle Re-verify

**Trigger:** duplicate round 14 wake skipped; round 15 executed

## Re-verification

| ID | Finding | Status |
|----|---------|--------|
| LBA-P1-018 | `submitTask`: upstream `CreateTask` before `repo.UpdateTask` (`video_gateway_worker.go:331-349`) — crash → duplicate billed create | **CONFIRMED** |
| LBA-P2-003 | `processTask`: `APIKeyDecryptFailed` returns error without terminalizing (`:277-279`) | **CONFIRMED** |
| LBA-P2-004 | Reference URLs validated at adapter submit, not create | unchanged (round 2) |

Code comment at `:337-342` explicitly documents double-submit risk if status regresses to queued.

## Round 15 outcome

0 new findings; 2 video worker issues re-confirmed; open ~166+
