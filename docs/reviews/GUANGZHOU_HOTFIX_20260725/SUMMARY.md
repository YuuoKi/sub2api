# Guangzhou Hotfix 20260725 — Sub2API Review Summary

| Field | Value |
|------|--------|
| Date | 2026-07-25 |
| Repo | Sub2 worktree `.worktrees/sub2-guangzhou-hotfix-20260725-grok` |
| Branch | `codex/grok-guangzhou-sub2-20260725` |
| Tip | `92567101c774dad5fe482df69eecba5a9adb3c7c` |
| **Status** | **待复核** — NOT READY（生产开卡须老板亲自签发；agent 不得捕获一次性密钥） |

## Commits in this hotfix chain

| SHA | Message |
|-----|---------|
| `82ec1cc346f9a9c6527e06f0f2f2058ea43316d7` | `fix(console): reuse existing employee owners when issuing cards` |
| `ecc10b66828c154c68eb37ee9cf62cf756aadec0` | `fix(console): make employee card dialogs lossless` |
| `92567101c774dad5fe482df69eecba5a9adb3c7c` | `fix(console): replace self-update UI with immutable build identity` |

## What changed

1. **409 flat error reuse** — Staff create-user conflicts match the flat axios reject shape (`status` / `reason`); active human/tool owners are reused without identity conversion; admin / disabled / not-found fail with explicit Chinese errors.
2. **Human + tool list** — Staff console lists non-admin `human` and `tool` members with visible type badges (`员工账号` / `工具账号`).
3. **Lossless dialogs** — Backdrop / Escape do not close or clear issuance or recharge forms; after dual keys are shown, only「我已安全保存，完成」closes; filled-form cancel requires Chinese confirmation; stable `crypto.randomUUID()` idempotency key per open session.
4. **Immutable 广州内部版 identity** — VersionBadge is read-only deploy identity (`广州内部版·YYYY.MM.DD-rN` + commit + build date); self-update / online rollback UI and HTTP mounts removed.
5. **Removed self-update routes** — Admin `check-updates` / `update` / `rollback*` unmounted; `GET /version` retained from `buildinfo.Info`.

## Tests (code gate only)

| Gate | Result |
|------|--------|
| StaffView + gate2 UX contracts | **32+ PASS** |
| Version / immutable identity related tests | **72** (suite total covering VersionBadge + app version + related guards/mocks) |
| Go tests (buildinfo / routes / handler unit tags as run in Phase 4) | **ok** |

Production card issuance by boss is **not** done in this package.

## Boss must still do (blocking READY)

- Production employee dual-key issuance on the live Guangzhou console.
- Agent **must not** capture, store, screenshot, or forward one-time keys.
- Until boss confirms issuance UX and identity badge on production, status remains **待复核**.

## Risks / rollback

- Revert tip chain with non-destructive `git revert` of the three commits above if needed; no DB schema migration in this chain for the UI/identity work.
- Self-update removal is intentional hard cut — operators rebuild/redeploy images; in-app update is gone.
- This package contains **no** passwords, API keys, `.env`, or one-time card secrets.

## Full package

Open [`REVIEW_PACKAGE.html`](./REVIEW_PACKAGE.html).
