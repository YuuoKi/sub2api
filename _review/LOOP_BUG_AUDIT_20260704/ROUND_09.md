# Round 9 — Frontend Critical Vitest + Payment Re-scan

**Trigger:** loop wake (round 8 duplicate skipped)

## Test gate

| Suite | Result |
|-------|--------|
| Makefile `FRONTEND_CRITICAL_VITEST` (6 files) | **pass** — 6 files / **79 tests** |
| `paymentFlow.spec.ts` + `PaymentView` + `OidcCallback` spot | **38 passed** |

PowerShell may report exit 1 on vitest stderr (browserslist warning); tests green.

## Test coverage gaps vs recorded bugs

| ID | Bug | Test gap |
|----|-----|----------|
| LBA-P1-009 | Login open redirect | **No `LoginView.spec.ts`** in critical list; redirect unsanitized untested |
| LBA-P1-011 | OAuth fragment tokens | Oidc/Wechat tests **assert legacy fragment flow works** — documents risk, not a guard |
| LBA-P1-012/013 | client_secret URL / localStorage | `AirwallexPaymentView` tests recovery from localStorage; Stripe URL path less covered |
| LBA-P3-018 | **new** | `FRONTEND_CRITICAL_VITEST` omits `paymentFlow.spec.ts`, `PaymentStatusPanel`, `EmailVerifyView` |

## Payment re-scan conclusion

No new P0/P1 beyond Round 1 frontend findings; critical tests pass but **do not exercise Login redirect sanitization**.

## Round 9 outcome

+1 test-gap hygiene; open ~159+
