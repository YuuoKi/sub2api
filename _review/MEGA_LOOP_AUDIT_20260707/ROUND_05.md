# Round 5 — API Key + Billing Fail-Open

**Agents:** Backend-explore, Thermo-nuclear (billing_cache)
**Files:** `api_key_auth.go`, `billing_cache_service.go`, `api_key_auth_cache_impl.go`

## LBA ROUND_05 Re-verify

| ID | Verdict |
|----|---------|
| P1-027 | PARTIAL — `CheckAPIKeyQuotaAndExpiryFresh` at request time |
| P1-028..030 | FIXED — fail-closed when limits configured |
| P1-032 | FIXED — inactive group blocked `api_key_auth.go:112-115` |

## New Findings

- **MLA-P1-009** — no atomic quota reservation before handler runs
- **MLA-P1-010** — model scope enforcement gap (needs gateway trace)

## Note

RPM increment errors return `ErrBillingServiceUnavailable` (fail-closed) per comment P1-028 fix.
