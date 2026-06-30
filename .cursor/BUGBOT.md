# Bugbot review rules — Sub2API (project-wide)

Bugbot runs on PRs for this repository. Enable the repo at https://cursor.com/dashboard (Integrations → GitHub → Bugbot → `Wei-Shaw/sub2api`).

## Architecture

- **handler** and **service** must NOT import `internal/repository`, `gorm.io/gorm`, or `github.com/redis/go-redis/v9` (see `backend/.golangci.yml` depguard).
- Business logic belongs in **service**; HTTP parsing in **handler**; DB/Redis in **repository**.

## Security (blocking)

- Flag hardcoded API keys, webhook secrets, JWT secrets, or Stripe/Alipay/WeChat credentials.
- Flag missing validation on payment webhook signatures.
- Flag `eval()`, unsafe SQL string concatenation, or logging of request bodies that may contain secrets.
- Video gateway and API key paths must redact sensitive fields in logs/responses.

## Tests

- Backend behavior changes should include or extend `_test.go` coverage when feasible.
- Frontend payment/auth views should keep critical vitest paths green (`make test-frontend-critical`).

## Dependencies

- Frontend: use **pnpm**; lockfile changes must accompany `package.json` edits.
- Do not introduce npm-only lockfiles.

## Ignore / context

- `_review/` and `_archive/` are historical; do not treat as production-ready state.
- Current status truth source: `00_START_HERE.md` and `docs/reviews/LATEST_REVIEW_PACKAGE.html`.
