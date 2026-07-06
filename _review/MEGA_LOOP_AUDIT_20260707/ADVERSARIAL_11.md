# Adversarial Pass AV11 — Test Gap Mapping

| Finding | Suggested test pattern |
|---------|------------------------|
| MLA-P1-001 | Table-driven handler test like existing ErrPaymentRejected |
| MLA-P1-005 | Vitest fake timers + concurrent 401 mock |
| MLA-P1-007 | PaymentView spec: mock JSAPI fail, assert createOrder called once |
| MLA-P1-008 | guards.spec backend_mode stripe path |
| MLA-P2-001 | Go unit test ListDramaTasks with filter + pagination |
| MLA-P2-007 | KeysView handleSubmit with interceptor-shaped error |

Follow: `payment_fulfillment_p0_regression_test.go`, `guards.spec.ts`
