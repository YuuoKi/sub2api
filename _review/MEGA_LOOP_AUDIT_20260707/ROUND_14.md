# Round 14 — Keys + Open Redirect

**Files:** `KeysView.vue`, `sanitizeRedirectPath.ts`, OAuth callbacks

## Findings

- **MLA-P2-006** — key status toggle semantics
- **MLA-P2-007** — error shape mismatch
- **MLA-P3-005** — no unmount abort
- Open redirect: LoginView/EmailVerify sanitized — LBA fixes hold

## Coverage Gap

KeysView.vue — no dedicated spec (P0 surface)
