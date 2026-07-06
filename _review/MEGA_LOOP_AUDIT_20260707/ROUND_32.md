# Round 32 — Blue Team Defenses

## For R31 Chains

| Defense | Location |
|---------|----------|
| API key middleware stack | api_key_auth.go |
| Billing fail-closed RPM | billing_cache_service.go:715+ |
| Payment RECHARGING lock | payment_fulfillment.go |
| Video SSRF | video_gateway_ssrf.go |
| Admin role checks | jwt_auth + router |

## Gaps

- MLA-P1-001 stale fulfillment → 500 (defense fails open to provider retries)
- MLA-P1-005 dual refresh (session integrity)
