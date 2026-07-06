# Round 16 — Repository Transaction Boundaries

**Focus:** Ent in service layer, payment_fulfillment direct ent access

## Findings

- `payment_fulfillment.go` uses ent client in service — architectural smell; tests compensate
- Affiliate rebate uses tx but audit log on outer ctx (MLA-P3-002)
- Repository integration tests PASS

## No new P1

Transaction boundaries appear consistent for payment fulfillment lock pattern.
