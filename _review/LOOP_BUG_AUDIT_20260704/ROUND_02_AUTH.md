# Round 2 — Auth Layer (partial)

**Agent:** [Auth layer adversarial review](3fedf822-3852-4e91-a5cf-1b6e0676e520)  
**Scope:** JWT, OAuth, sessions, admin middleware (`handler/auth_*`, `service/auth*`, `server/middleware`)

## Summary

| Severity | Count | Top themes |
|----------|-------|------------|
| P0 | 0 | No unconditional bypass found |
| P1 | 3 | Refresh reuse no family revoke; OAuth identity reassignment on inactive owner; admin JWT in WebSocket subprotocol |
| P2 | 9 | Logout leaves access JWT valid; email/code enumeration; admin API key → first admin; bind cookie HMAC uses JWT secret; X-Forwarded-Proto cookie Secure; OIDC id_token skip; TOTP secret prefix in logs; bind-login race |
| P3 | 6 | min password 6; OAuth GET no rate limit; stale JWT claims; token-pair fallback; temp_token log prefix; middleware path note |

## P1 IDs (record only)

- **LBA-P1-015** — `auth_service.go:1444-1449` — refresh reuse logged but family not revoked; `IsTokenInFamily` unused
- **LBA-P1-016** — `auth_oauth_pending_flow.go:767-778` — OAuth identity reassigned when prior owner inactive/deleted
- **LBA-P1-017** — `admin_auth.go:33-44,95-115` — admin JWT via `Sec-WebSocket-Protocol` (log/proxy exposure)

## P2 IDs (record only)

- **LBA-P2-001** — Logout does not invalidate access JWT (`auth_handler.go:697-714`)
- **LBA-P2-002** — Registration verify enables email enumeration (`auth_service.go:336-338`)
- **LBA-P2-003** — Promo/invitation validation leaks code validity (`auth_handler.go:465-560`)
- **LBA-P2-004** — Admin API key impersonates first admin (`admin_auth.go:137-149`)
- **LBA-P2-005** — OAuth bind cookie HMAC keyed with JWT secret (`auth_linuxdo_oauth.go:1125-1141`)
- **LBA-P2-006** — `X-Forwarded-Proto` trusted for cookie Secure (`auth_linuxdo_oauth.go:907-912`)
- **LBA-P2-007** — OIDC may skip id_token validation when config false (`auth_oidc_oauth.go:296-308`)
- **LBA-P2-008** — TOTP verify logs secret prefix (`totp_service.go:374-390`)
- **LBA-P2-009** — OAuth bind-login binds before session consume (`auth_oauth_pending_flow.go:1591-1606`)

## Positive controls noted

JWT alg restriction, token version checks, constant-time API key compare, OAuth state/PKCE cookies, refresh rotation, enumeration-resistant password reset.
