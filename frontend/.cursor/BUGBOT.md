# Bugbot — frontend-specific rules

## Tooling

- Package manager is **pnpm** only.
- Run `pnpm --dir frontend run lint:check` and `typecheck` expectations match CI.

## Vue / TypeScript

- Prefer composition API patterns consistent with existing views.
- Payment and auth views are critical paths — flag regressions in Stripe/Alipay/WeChat flows.
- Avoid inline imports; keep imports at file top (Team Kit rule).

## Security

- Sanitize user HTML (DOMPurify) where rendering markdown or external content.
- Do not expose admin-only settings in public config injection (`vite.config.ts` public settings flow).
