# Round 5 — API Keys (supplement)

**Agent:** [API keys adversarial review](66493e0c-5539-4433-86bd-e826f466db18)

## Summary

| Severity | Count | Top themes |
|----------|-------|------------|
| P0 | 0 | — |
| P1 | 6 | Stale quota cache bypass; rate-limit fail-open; SupportedModelScopes never enforced; inactive group not gated |
| P2 | 9 | PUT clears IP rules; negative quota = unlimited; SimpleMode skips billing; TOCTOU on rate limits |
| P3 | 9 | No handler tests; full key in list JSON; invalidation no-ops on errors |

## P1 IDs (mapped to LBA)

- **LBA-P1-027** — Stale `quota_used` in auth cache bypass (`api_key_auth_cache_impl.go:217-218`, `api_key_auth.go:170-172`)
- **LBA-P1-028** — USD rate limits fail-open on DB error (`billing_cache_service.go:538-553`)
- **LBA-P1-029** — USD rate limits skipped when cache nil (`billing_cache_service.go:532-535`)
- **LBA-P1-030** — RPM fail-open on Redis errors (`billing_cache_service.go:739-776`)
- **LBA-P1-031** — `SupportedModelScopes` cached but not enforced at request time (P1-5)
- **LBA-P1-032** — Inactive group status not enforced on gateway (`api_key_auth_cache_impl.go:252-253`)

## Cross-ref Round 5 manual

- LBA-P2-021 (usage billing legacy fallback) — distinct from API key cache issues
- LBA-P2-022 — same as agent P2 rate-limit path
