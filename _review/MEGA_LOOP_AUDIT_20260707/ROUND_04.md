# Round 4 — Auth / OAuth / JWT

**Agents:** Security-review, Frontend-explore
**Files:** `auth_service.go`, `auth_oauth_pending_flow.go`, `jwt_auth.go`, `routes/auth.go`

## Findings

- JWT middleware reloads user + TokenVersion revocation — sound
- OAuth pending flow large surface — table tests exist
- **LBA-P1-009/010/011** — redirect/fragment fixes verified in TRUTH_MATRIX
- No new P0 auth bypass found in static trace

## Residual

- Pending OAuth session in localStorage — sanitized on consume (P3)
- Rate limit Redis fail-close tested in middleware tests
