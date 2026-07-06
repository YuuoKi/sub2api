# Round 15 — Admin Settings Monolith

**Files:** `SettingsView.vue` (~8600 lines)

## Findings

- **MLA-P2-008** — load/save race without request versioning
- **MLA-P3-007** — OAuth redirect URL fields not client-validated
- Partial test coverage in `SettingsView.spec.ts` (critical CI)

## Thermo Note

File size exceeds maintainability threshold — not a runtime bug; regression risk P2
