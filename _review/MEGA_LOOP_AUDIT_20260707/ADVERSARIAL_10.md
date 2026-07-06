# Adversarial Pass AV10 — Minimal Fix Sketches

## MLA-P1-001

```go
// payment_webhook_handler.go after ErrPaymentRejected block
if errors.Is(err, service.ErrPaymentFulfillmentStale) {
    slog.Error("...", ...)
    writeSuccessResponse(c, resolvedProviderKey) // + metric/alert
    return
}
```

## MLA-P1-005/006

```ts
// Single module: refreshMutex + export refreshAccessToken()
// client.ts interceptor AND auth.ts timer call same function
// On success: update localStorage AND authStore.$patch({ token, refreshTokenValue })
```

## MLA-P1-007

```ts
// PaymentView: on JSAPI failure, resume existing order polling — do NOT createOrder again
```

## MLA-P1-008

```ts
// router: add '/payment/stripe', '/payment/stripe-popup' to BACKEND_MODE_ALLOWED_PATHS
// backend-mode login redirect: query.redirect = to.fullPath
```

## MLA-P2-001

```go
// Push drama filters to repo SQL; return SQL COUNT(*) for total
```
