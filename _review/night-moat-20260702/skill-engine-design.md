# Skill Engine v0 design (flag off)

## Goal

Create the smallest backend contract for a future Skill Engine without arming it in production or calling any provider. v0 is a typed, explicit-failure skeleton: it gives the rest of the codebase a stable place to depend on later, while keeping runtime behavior dark by default.

## Non Goals

- No provider calls.
- No automatic extraction from live traffic.
- No database writes.
- No AUTH_CONTRACT changes.
- No fallback skill generation, mock skill cards, or silent success.

## Contract

`backend/internal/service/skill_engine.go` defines:

- `SkillEngineConfig`: currently only `Enabled` plus an input-byte limit.
- `SkillEngineInput`: redacted prompt/response text and source metadata.
- `SkillEngineResult`: explicit status, reason, byte count, and timestamp.
- `SkillEngine.Run(ctx, input)`: one entry point.

Behavior:

| Case | Result | Error |
| --- | --- | --- |
| nil engine or `Enabled=false` | `status=disabled` | `ErrSkillEngineDisabled` |
| `Enabled=true`, empty redacted input | `status=invalid_input` | `ErrSkillEngineEmptyInput` |
| `Enabled=true`, valid input | `status=not_implemented` | `ErrSkillEngineNotImplemented` |
| context canceled | `status=invalid_input` | context error |

## Future Integration Path

1. Feed only redacted `ai_generation_content` rows into the engine.
2. Add durable storage for skill artifacts after schema review.
3. Keep provider-backed summarization behind a separate explicit flag and budget gate.
4. Surface generated skill artifacts to admin dashboards only after provenance and redaction evidence are present.

## Safety

The v0 skeleton is intentionally not wired into server routes or workers. Running it cannot dispatch to real providers, mutate persistence, or claim a skill was produced.
