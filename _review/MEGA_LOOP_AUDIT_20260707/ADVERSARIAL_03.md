# Adversarial Pass AV3 — Test / Repro Design

## MLA-P1-001 Repro

1. Create order → PAID → RECHARGING
2. Simulate fulfillment hang > 2min (mock clock or sleep in integration test)
3. Send duplicate success webhook
4. Expect: currently 500; desired 2xx + alert

**Existing test gap:** no `TestWebhook_StaleRecharging_Acks2xx`

## MLA-P1-007 Repro

1. Mobile WeChat pay → trigger JSAPI error path
2. Observe second `createOrder` in network tab
3. Verify two PENDING orders in DB

## MLA-P2-001 Repro

1. Seed 20 drama tasks, filter to 3 matches
2. Request page=1 pageSize=10
3. Expect total=3 but may return <10 rows with wrong total metadata

**Verify-this:** MLA-P2-001 → **VERIFIED** by code inspection `ListDramaTasks:429`
