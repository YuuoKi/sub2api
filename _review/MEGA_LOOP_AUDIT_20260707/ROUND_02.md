# Round 2 — Payment / Webhook Money Path

**Agents:** Security-review, Backend-explore (R2-R3 subagent)
**Files:** `payment_fulfillment.go`, `payment_webhook_handler.go`, `payment_webhook_provider.go`

## LBA P0 Re-verify

| ID | Verdict |
|----|---------|
| P0-001..003 | FIXED — markCompleted, confirmPayment, legacy binding |
| P0-004 | INTENTIONAL — ErrPaymentAfterExpiry ack 2xx |

## New / Residual Findings

- **MLA-P1-001** — `ErrPaymentFulfillmentStale` not in webhook sentinel list → 500
- **MLA-P1-002** — `alreadyProcessed` default nil ack
- **MLA-P2-002** — Stripe out_trade_no extraction gap
- **MLA-P2-003** — fulfillment lock `c==0` silent return

## Tests Run

`go test ./internal/service -run Payment -count=1` — PASS
`payment_fulfillment_p0_regression_test.go` — all green
