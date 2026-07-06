# Round 18 — payment_order_expiry Background

**Files:** `payment_order_expiry_service.go`

## Findings

- **MLA-P2-011** — no unit tests for expiry ticker + goroutine
- Logic: marks PENDING orders expired after timeout — appears sound
- Risk: untested concurrent expiry vs webhook race (P2 theoretical)
