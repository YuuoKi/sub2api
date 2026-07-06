# Round 11 — Dual Token Refresh Race

**Agents:** Frontend-explore R11-R15
**Files:** `client.ts`, `stores/auth.ts`

## Findings

- **MLA-P1-005** CONFIRMED — parallel refresh via interceptor + `scheduleTokenRefresh`
- **MLA-P1-006** CONFIRMED — localStorage updated, Pinia stale
- **MLA-P2-005** — login 401 clears tokens
- **MLA-P2-009** — NaN token_expires_at

## Existing Tests

`auth.spec.ts`, `client` interceptor tests — do not cover concurrent dual-path refresh
