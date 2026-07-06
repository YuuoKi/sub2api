# Round 22 — auth_handler / payment_service Untested Cores

**Files:** `auth_handler.go`, `payment_service.go`

## Coverage Gap

- No dedicated `auth_handler_test.go` — OAuth tests scattered
- No `payment_service_test.go` — indirect via fulfillment/webhook

## Findings

No new logic bug isolated; coverage gap P2 for login/register edge paths
