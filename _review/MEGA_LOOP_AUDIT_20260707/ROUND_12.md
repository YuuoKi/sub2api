# Round 12 — Payment Flow + Recovery

**Files:** `paymentFlow.ts`, `PaymentView.vue`, `PaymentStatusPanel.vue`

## Findings

- **MLA-P1-007** — JSAPI fallback duplicate order
- **MLA-P2-010** — strict payAmount in recovery snapshot
- clientSecret correctly in sessionStorage (LBA-P1-013 FIXED)

## Tests

`paymentFlow.spec.ts`, `PaymentView.spec.ts` — PASS; gap: no JSAPI fallback duplicate test
