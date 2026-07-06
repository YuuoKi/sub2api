# Adversarial Pass AV8 — Concurrency Timelines

## Timeline: Dual 401 Refresh

```
T0: Request A 401
T1: Interceptor sets isRefreshing=true, starts refresh POST
T2: scheduleTokenRefresh fires (store), second refresh POST
T3: Server rotates refresh_token on first response
T4: Second refresh fails → logout cascade
```

**MLA-P1-005 CONFIRMED**

## Timeline: Quota race (MLA-P1-009)

```
T0: Key quota 1.0 left, cache shows 0.9 used
T1: Request A passes fresh check (1.0)
T2: Request B passes fresh check (1.0) before A increments
T3: Both proceed — over-quota possible
```

**LIKELY** — mitigated by post-hoc UpdateQuotaUsed; race window small

## Timeline: Double webhook

Recent RECHARGING → 2xx; Stale → 500 (MLA-P1-001)
