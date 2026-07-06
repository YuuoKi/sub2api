# Phase 2 Triage — 2026-07-07

## Dedup Rules Applied

- Merged webhook 500 + stale RECHARGING → single root MLA-P1-001
- Merged dual refresh + Pinia desync → related pair MLA-P1-005/006 (fix together)
- LBA-P0-004 excluded from open bugs → Intentional

## Confirmed Candidate Set (enters AV1-15)

| ID | Sev | Confidence pre-AV |
|----|-----|-------------------|
| MLA-P1-001 | P1 | 92% |
| MLA-P1-005 | P1 | 88% |
| MLA-P1-006 | P1 | 90% |
| MLA-P1-007 | P1 | 85% |
| MLA-P1-008 | P1 | 87% |
| MLA-P2-001 | P2 | 95% |
| MLA-P2-007 | P2 | 82% |

## Likely (monitor, lower AV depth)

MLA-P1-002, MLA-P1-009, MLA-P1-010, MLA-P2-002..010

## FalsePositive / Intentional

| ID | Reason |
|----|--------|
| LBA-P0-004 | Documented manual recovery |
| LBA-P1-020 | Phase-2B deliberate no-op |
| LBA-P1-019 | Phase-2A budget not wired by design |

## Count after triage

**7 Confirmed** for full adversarial pass · **12 Likely** · **9 P3**
