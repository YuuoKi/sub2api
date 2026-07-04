# Round 17 — Migrations + Config Secrets

**Trigger:** duplicate round 16 wake skipped; round 17 executed

## Secret scan

| Gate | Result |
|------|--------|
| `make secret-scan` / `python tools/secret_scan.py` | **Not run** — `python`/`py` unavailable on Windows host |

**LBA-P3-026:** Local Windows cannot run secret-scan without Python on PATH (CI/Linux unaffected).

## Config spot-check

- OAuth/client secrets default to `""` in viper (`config.go`) — no hardcoded production secrets found in defaults skim.

## Migration cross-check (affiliate idempotency)

- `user_affiliate_ledger` indexes on `source_order_id` are **non-unique** — supports LBA-P1-022 (round 3 repo review).

## Round 17 outcome

+1 env hygiene; no new P0/P1; open ~167+
