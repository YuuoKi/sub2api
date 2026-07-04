# Round 14 — Frontend Auth Redirect Re-verify

**Trigger:** duplicate rounds 12–13 wakes skipped; round 14 executed

## Re-verification

| ID | Finding | Status |
|----|---------|--------|
| LBA-P1-009 | `LoginView.vue:493-494,527-528` uses raw `query.redirect` without `sanitizeRedirectPath` | **CONFIRMED** |
| LBA-P1-010 | `EmailVerifyView.vue:554` `router.push(pendingRedirect)` unsanitized | **CONFIRMED** |
| LBA-P1-011 | OAuth callbacks define `sanitizeRedirectPath` (`OidcCallbackView.vue:393-399`) but legacy hash tokens still parsed (`:369-390`) | **CONFIRMED** |

Contrast: OAuth callback views **do** sanitize redirects; login/email-verify **do not**.

## Round 14 outcome

0 new IDs; 3 P1 auth findings re-confirmed; open ~166+
