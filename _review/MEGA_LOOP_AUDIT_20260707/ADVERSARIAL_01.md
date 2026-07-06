# Adversarial Pass AV1 — Static Trace (Top P1)

## MLA-P1-001 Call Graph

```
POST /api/v1/payment/webhook/{provider}
  → payment_webhook_handler.HandleWebhook
  → paymentService.HandlePaymentNotification
  → confirmPayment → alreadyProcessed
  → if RECHARGING > 2min → ErrPaymentFulfillmentStale
  → handler: NOT in sentinel list → 500
```

**Fail-open/closed:** Provider sees failure → retries (fail-open for attacker replay; fail-closed for ops alerting)

## MLA-P1-005 Trace

```
Request A 401 → client interceptor starts refresh (isRefreshing=true)
Timer fires → auth.performTokenRefresh (parallel if race window)
Both POST /auth/refresh with same refresh_token
```

**Verdict:** CONFIRMED trace
