# Adversarial Pass AV9 — Community Standards

| Pattern | Industry | This project |
|---------|----------|--------------|
| Webhook idempotency 2xx | Stripe docs recommend 2xx for unrecoverable | Partial — stale RECHARGING gaps |
| Single refresh mutex | OAuth2 BCP / common SPA pattern | Violated — dual path |
| Pagination total count | REST best practice | Drama list violates |
| UPDATE claim billing | Idempotent charge pattern | S3 fix aligns with best practice |

**MLA-P1-001, MLA-P1-005, MLA-P2-001** align with known anti-patterns in community guidance.
