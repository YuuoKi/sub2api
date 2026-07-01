# Sub2API video capture moat summary

## Conclusion

Status: 内部可用 / 待阶段 3 授权复核.

Stages 0-2 are complete on branch `wujie/video-capture-moat-20260702`. Successful mock video tasks now flow into `ai_generation_content` through a video-shaped capture path when `content_capture.enabled=true`. The implementation stays fail-open, idempotent by `task_id`, and leaves `api_key_id`, `group_id`, and `account_id` empty for video rows.

Stage 3 was not executed.

## Changes

- Added `backend/migrations/146_ai_generation_content_task_id_unique_notx.sql` with a concurrent partial unique index on `ai_generation_content(task_id)`.
- Added `TaskID` to `GenerationContent` and a video-specific `VideoGenerationContentCaptureArgs` / `CollectVideoTask` path.
- Added repository method `CreateVideoTaskContent` using `ON CONFLICT (task_id) WHERE task_id IS NOT NULL DO NOTHING`.
- Added optional `GenerationContentCollector` injection to `VideoGatewayService`.
- Wired `pollTask` succeeded terminal branch to capture only after usage logging and video billing deduction.
- Wired production construction in `backend/cmd/server/wire_gen.go`.
- Added tests for video capture success, flag-off zero capture, fail-open behavior, task-id SQL, and duplicate no-op.

## Verification

All required gates passed:

- `cd backend; go build ./...`
- `cd backend; go test ./...`
- `cd frontend; pnpm test:run` -> 97 files / 581 tests passed
- `cd frontend; pnpm typecheck`
- `cd frontend; pnpm lint:check`
- bundled Python `tools/secret_scan.py --include-untracked` -> no high-confidence findings

See `capture-evidence.md` for RED/GREEN and final gate details.

## Boundaries

- No push, merge, rebase, reset, clean, delete, or deploy.
- No `.env`, key, token, cookie, or provider credential read.
- No real provider call.
- `AUTH_CONTRACT_SPLIT` untouched.
- Remaining pre-existing untracked files left untouched: `.impeccable/hook.cache.json`, `MORNING_RESULT_2026_06_28.md`.

## Rollback

Local rollback is to revert the final implementation commit after `d11a13c0`, or revert the touched files listed in this summary. No remote rollback is needed because nothing was pushed.

## Next Step

Stop here. Stage 3 requires separate human authorization because it involves controlled flag verification and retention activation.
