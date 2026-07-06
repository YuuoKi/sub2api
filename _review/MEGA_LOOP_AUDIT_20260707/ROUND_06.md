# Round 6 — S3 CNY Display Regression

**Focus:** No double conversion; usd_cny_rate propagation
**Files:** `display_currency.go`, `useAdminDisplayCurrencyRate.ts`, ContentWall, GenerationContentView specs

## Verification

- `formatByCurrency`: CNY native no multiply — specs PASS
- Admin stats APIs return `usd_cny_rate` — handler tests PASS
- GenerationContentView preserves valid rate when weekly lacks field — spec PASS

## Findings

**No new bugs.** S3 fixes hold. Residual: dashboard fallback 7.2 when settings unreachable (documented).
