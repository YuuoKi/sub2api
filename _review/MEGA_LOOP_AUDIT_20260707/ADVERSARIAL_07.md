# Adversarial Pass AV7 — FE/BE Contract

| Field/Flow | BE | FE | Drift |
|------------|----|----|-------|
| Token refresh response | `{access_token, refresh_token}` | client + store both consume | Dual path drift MLA-P1-006 |
| Payment order status polling | status enum | PaymentStatusPanel | OK |
| Drama list total | `int64(len(out))` | expects DB total | **DRIFT** MLA-P2-001 |
| usd_cny_rate | admin APIs | composables | OK post-S3 |
