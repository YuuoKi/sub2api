# Sub2API moat status (2026-07-02)

## Conclusion

Sub2API already has a real moat foundation around generation-content capture: capture table, collector, response tee, Anthropic main-path wiring, redaction, retention NULL-OUT, and read-only dashboard are present in Git history and current code. The capture path remains dark by default: no live provider call or flag arming was performed in this task.

Skill Engine is not yet a standalone backend engine. Current `skill` hits are mainly drama/video skill cards, dry-run exports, and UI labels, not a reusable skill execution/learning engine with repository/contracts/flags.

## Evidence By Commit

| Commit | Evidence | Status |
| --- | --- | --- |
| `6478237a` | Adds `ai_generation_content` migration, collector, capped sink, redaction, repo, config structs, and tests. | M1-B.1 foundation exists; flag default off by zero-value/no default. |
| `b919650f` | Adds response capture tee and response capture tests. | Response-side sampling exists and is bounded. |
| `eca1b65c` | Adds `CollectGenerationContent` on Anthropic `/v1/messages` main path. | Main Anthropic path is wired when flag/collector are enabled. |
| `e078749a` | Adds admin generation-content handler/routes/frontend dashboard. | Read-only value-output dashboard exists; admin-only route. |
| `38df1bcd` | Adds structured redaction and retention NULL-OUT service/repo/index/tests. | D3 redaction/retention mechanism exists; dark/default-safe. |

## Current Code Anchors

- Capture call sites: `backend/internal/handler/gateway_handler.go:535`, `backend/internal/handler/gateway_handler.go:935`, `backend/internal/handler/gateway_handler_chat_completions.go:282`.
- Capture flag: `backend/internal/config/config.go:640-744` defines `Gateway.ContentCapture`; `backend/internal/service/generation_content.go:176` gates on `cfg.Gateway.ContentCapture.Enabled`.
- Capture storage: `backend/migrations/140_ai_generation_content.sql`; repo methods in `backend/internal/repository/generation_content_repo.go`.
- Retention NULL-OUT: `backend/internal/service/generation_content_retention_service.go:23`, `backend/internal/repository/generation_content_repo.go:120-159`, `backend/migrations/141_ai_generation_content_retention_index.sql`.
- Dashboard: `backend/internal/server/routes/admin.go:253-259`, `backend/internal/handler/admin/generation_content_handler.go:20-53`, `frontend/src/router/index.ts:633-635`, `frontend/src/views/admin/GenerationContentView.vue`.

## Skill Engine Gap

Observed `skill` code is scoped to video/drama reporting:

- `backend/internal/service/drama_gateway_service.go` builds `DramaSkillCard` / `DramaSkillAnalysisExport` and creates `skill_event` task events.
- `backend/internal/handler/admin/video_handler.go` exposes `SkillCards` and `SkillAnalysisExport`.
- `frontend/src/api/admin/video.ts` and `frontend/src/views/admin/video/VideoDashboardView.vue` surface those cards/exports.

Missing as of this pass:

- No `backend/internal/service/skill_engine.go`.
- No generic `SkillEngine` interface.
- No flag-off runtime skeleton for skill extraction/execution.
- No repository boundary for durable skill artifacts beyond existing video task events/exports.

## Boundary

- No push, merge, remote/config change, provider call, credential read, or AUTH_CONTRACT change.
- This is a code/history evidence report only. It does not claim production capture has been armed or verified on live traffic.
