# Round 13 — Router Guard Matrix

**Files:** `router/index.ts`, `guards.spec.ts`

## Findings

- **MLA-P1-008** — backend_mode Stripe routes not in allowlist; redirect lost
- requiresPayment / requiresRiskControl depend on async public settings — flash risk (P3)
- Video demo mode allowlist — tested in guards.spec

## Tests

`guards.spec.ts` — PASS
