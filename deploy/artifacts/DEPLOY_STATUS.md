# Staff console hotfix — deploy status (2026-07-26)

## Status: LIVE

| Item | Value |
|------|-------|
| Branch | `fix/staff-console-hotfix-20260726` |
| Commit | `f8cc438f6` |
| Version | `广州内部版 2026.07.26-r1` |
| Public | `http://114.132.50.149/` → `build_commit=f8cc438f6`, `/health` ok |
| Active compose dir | `~/wujie/wujie-tencent-guangzhou-dualkey-d6e54a8-35d5f77` |
| Bookkeeping dir | `~/wujie/wujie-tencent-guangzhou-dualkey-d6e54a8-f8cc438f6` |
| Image | retagged to `wujie-production-sub2api:latest` |
| Archive | older dualkey trees tar’d under `~/wujie/archive/` |

## Notes

- Hot-updated with `docker compose up -d --no-deps --force-recreate --no-build sub2api` (volumes preserved).
- Remembered path `…-0561ed5-239ec7e` does **not** exist on server.
- No local SSH private key on the home PC; password auth used for this cutover (password not stored in repo/vault).
