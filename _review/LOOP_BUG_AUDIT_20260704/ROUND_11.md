# Round 11 — Payment Webhook Test Gap Deep-Dive

**Trigger:** duplicate round 9 wake skipped; round 11 executed

## Covered well

- Provider lookup failure → 400 (B2 fix): `TestPaymentWebhookProviderLookupFailureRejects*`
- `extractOutTradeNo`, `verifyNotificationWithProviders` unit tests
- `ErrOrderNotFound` sentinel + handler ack helpers
- `parseLegacyPaymentOrderID`, redeem action, provider metadata validation

## Missing integration / negative tests (record only)

| ID | Gap | Maps to |
|----|-----|---------|
| LBA-P3-019 | No test: `confirmPayment` Get fails → handler returns 2xx | LBA-P0-002 |
| LBA-P3-020 | No test: `markCompleted` zero-row UPDATE still writes audit | LBA-P0-001 |
| LBA-P3-021 | `TestUnknownOrderWebhookAcksWithSuccess` tests sentinel/response only, **not** full `handleNotify` E2E | handler comment at :123 |
| LBA-P3-022 | No test: legacy `sub2_{id}` credits wrong order when out_trade_no not found | LBA-P0-003 |
| LBA-P3-023 | No test: expired-beyond-grace returns nil → 2xx ack | LBA-P0-004 |
| LBA-P3-024 | No test: concurrent PAID webhook → 500 retry storm | LBA-P1-001 |

## Round 11 outcome

+6 test-gap records; no new runtime bugs; open ~165+
