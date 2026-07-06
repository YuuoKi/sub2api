# Adversarial Pass AV12 — Maintainability Review of AV10 Fixes

| Fix | LOC est | Risk | Thermo verdict |
|-----|---------|------|----------------|
| P1-001 sentinel | ~8 lines | Low | APPROVE |
| P1-005/006 unify refresh | ~40-60 lines | Med | APPROVE — avoid refactor beyond auth module |
| P1-007 no duplicate order | ~15 lines | Low | APPROVE |
| P1-008 router allowlist | ~5 lines | Low | APPROVE |
| P2-001 drama SQL | ~80-120 lines | Med | APPROVE — repo layer only |

No gateway_service or SettingsView refactor recommended.
