# Round 31 — Red Team Threat Model

## Attacker Goals

1. Free LLM/video calls without balance
2. Double-credit payment webhooks
3. Admin API access via forged JWT
4. SSRF via video reference URLs

## Attack Chains Identified

| Chain | Feasibility | Blocker |
|-------|-------------|---------|
| Stale quota cache bypass | LOW | CheckAPIKeyQuotaAndExpiryFresh |
| Concurrent requests exceed quota | MEDIUM | No atomic reservation (MLA-P1-009) |
| Webhook replay double credit | LOW | Status locks + audit idempotency |
| JWT forge | LOW | TokenVersion + secret |
| SSRF | LOW | urlvalidator layers |

## New from red team

Reaffirms MLA-P1-009 as highest exploit-adjacent backend item.
