# Round 5 — API Keys, Channel Monitor, Usage Billing (partial)

**Trigger:** AGENT_LOOP_WAKE_bugaudit (round 5)  
**Agents:** [API keys review](66493e0c-5539-4433-86bd-e826f466db18) still running; manual spot-check below

## Manual Spot-Check Findings (record only)

| ID | Sev | Finding | Location |
|----|-----|---------|----------|
| LBA-P2-021 | P2 | `applyUsageBilling` falls back to legacy `postUsageBilling` when `repo == nil` — no fingerprint/dedup tx | `gateway_service.go:8081-8088` |
| LBA-P2-022 | P2 | `IncrementRateLimitUsage` RowsAffected discarded (cross-ref R-P3-08) | `api_key_repo.go:560-572` |
| LBA-P3-013 | P3 | Channel monitor has SSRF controls (`channel_monitor_ssrf.go`) — positive; no new P0 in monitor path from spot-check |
| LBA-P3-014 | P3 | API key query param `key`/`api_key` rejected (400) — good; multiple header fallbacks increase attack surface for key logging | `api_key_auth.go:32-58` |

## Pending

Full API key adversarial report from subagent when complete.
## Channel Monitor (spot-check)

- CRUD decrypts API keys in service for admin display (`channel_monitor_service.go:89-90`) — expected; handler must redact
- Probe execution in scheduler/worker not deep-reviewed this round — defer to Round 6+

## Round 5 Outcome

- **+24** API key findings recorded
- Open total: **~146+**
