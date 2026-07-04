# Round 7 — Gateway Routing + OpenAI WS (partial)

**Trigger:** loop wake (round 6 duplicate skipped; round 7 started)

## Findings (record only)

| ID | Sev | Finding | Location |
|----|-----|---------|----------|
| LBA-P2-023 | P2 | Beta policy unknown scope → `match all` (fail-open) | `gateway_service.go:6476-6477` |
| LBA-P2-024 | P2 | Beta rule fallback empty → `BetaPolicyActionPass` (fail-open) | `gateway_service.go:6502-6505` |
| LBA-P2-025 | P2 | Waiting queue explicitly fail-open on Redis error (tested) | `gateway_waiting_queue_test.go:86-99` |
| LBA-P3-015 | P3 | OpenAI WS v2 passthrough path — billing/auth delegated to forwarder ingress; no new P0 from entry skim (`openai_ws_v2/entry.go`) |

## Round 7 outcome

+4 records; P0 still 7 verified; open ~154+
