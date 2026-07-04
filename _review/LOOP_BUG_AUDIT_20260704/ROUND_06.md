# Round 6 — Test Gaps + P0 Sample Re-verify

**Trigger:** loop wake (post round 5 duplicate skip)

## P0 test coverage gaps (new hygiene findings)

| ID | P0 | Test gap |
|----|-----|----------|
| LBA-P0-001 | markCompleted zero rows | No test asserts audit not written when UPDATE matches 0 rows |
| LBA-P0-002 | confirmPayment Get → nil | No test for webhook ack when order Get fails after legacy path |
| LBA-P0-005 | JWT bypass trial | No handler test comparing JWT CreateTask vs API-key CreateDailyTrialTask |
| LBA-P0-006 | Drama auto-route | No test that safe-demo does not hit live provider when smoke armed |

`parseLegacyPaymentOrderID` has unit tests (`payment_fulfillment_test.go:315+`) but not end-to-end wrong-order credit.

## P0 sample re-verify (LBA-P0-002)

Re-read `confirmPayment` lines 71-74 — still returns `nil` on Get error. **Still CONFIRMED.**

## Round 6 outcome

- 4 test-gap records (P3 hygiene under test coverage theme)
- P0 count unchanged at 7 verified
- Open ~146+
