# R1 Backend Review Patch Report

Date: 2026-07-05
Scope: R1-A to R1-F only. No frontend files were intentionally changed in this pass; pre-existing frontend dirty files were left untouched.

## Result Matrix

| Item | Status | Evidence |
|---|---|---|
| R1-A video balance deduction | PASS | `TestSeedanceSucceededTaskDeductsUserBalanceInUSDAndQueuesCache`, `TestSeedanceBalanceChargeIsIdempotentOnWorkerRetry`, `TestSeedanceFailedTaskDoesNotDeductBalance`; full `go test ./...` passed |
| R1-B Seedance production vs smoke gate | PASS | `TestSeedanceProductionAuthorizedSkipsSmokeDurationLimit`, `TestSeedanceUnapprovedProviderKeepsTinySmokeDurationLimit`; `docs/api/video-gateway-contract.md` updated with trial vs production behavior |
| R1-C video_usage_logs idempotency | PASS | `TestVideoGatewayRepositoryInsertUsageLogIsIdempotent`, `TestVideoGatewayRepositoryClaimVideoBalanceCharge`; migrations `153` and `154` added |
| R1-D Ark golden sample | PASS | `backend/internal/service/testdata/ark_poll_succeeded.json`; `TestSeedancePollExtractsUsageActualsAndLastFrame`, `TestSeedanceCreatePayloadSnapshotMatchesArkContract` |
| R1-E media URL allowlist | PASS | `TestValidateExternalVideoURLMediaAllowlistOverridesLegacyVideoAllowlist`, `TestValidateExternalVideoURLAllowlist`; image/video contracts updated |
| R1-F repo hygiene | PARTIAL | `.gitignore` exceptions added for `docs/api/**` and `docs/superpowers/codex-handoff/**`; `backend/internal/service/user_member_type.go` remains untracked because staging/commit was disallowed |

## Changed Files

- Billing/idempotency: `backend/internal/service/video_gateway_billing.go`, `backend/internal/service/video_gateway_service.go`, `backend/internal/service/video_gateway_types.go`, `backend/internal/repository/video_gateway_repo.go`, `backend/internal/service/setting_service.go`, `backend/internal/service/wire.go`, `backend/cmd/server/wire_gen.go`
- Tests/fixtures: `backend/internal/service/video_gateway_billing_test.go`, `backend/internal/service/video_gateway_worker_test.go`, `backend/internal/repository/video_gateway_repo_test.go`, `backend/internal/service/video_gateway_single_smoke_authorized_test.go`, `backend/internal/service/video_gateway_content_contract_test.go`, `backend/internal/service/video_gateway_poll_response_contract_test.go`, `backend/internal/service/video_gateway_security_test.go`, `backend/internal/server/routes/api_key_video_gateway_test.go`, `backend/internal/service/testdata/ark_poll_succeeded.json`
- Migrations: `backend/migrations/153_video_tasks_balance_charged_at.sql`, `backend/migrations/154_video_usage_logs_unique_task_notx.sql`, plus schema integration assertion in `backend/internal/repository/migrations_schema_integration_test.go`
- Gate/docs: `backend/internal/service/video_gateway_adapter.go`, `backend/internal/service/video_gateway_ssrf.go`, `docs/api/video-gateway-contract.md`, `docs/api/image-gateway-contract.md`, `.gitignore`
- Existing dirty/untracked backend files from V-1 to A-4 remain present and were not staged.

## Implementation Notes

- Successful terminal video tasks now claim `video_tasks.balance_charged_at` atomically before user balance deduction. A repeated worker closeout sees claim=false and does not deduct again.
- Seedance provider-usage cost remains stored as CNY; real user balance deduction converts to USD via `settings.usd_cny_rate`, defaulting to `7.20`.
- Real deduction path uses `UserRepository.DeductBalance`, followed by `BillingCacheService.QueueDeductBalance` only after DB deduction succeeds.
- `StaticBudgetGuard.Charge` remains a no-op/legacy hook; real billing is not hidden inside the budget guard.
- `video_usage_logs` now has `UNIQUE(video_task_id)` and `InsertUsageLog` uses `ON CONFLICT (video_task_id) DO NOTHING`.
- `provider_account.metadata.production_authorized=true` counts as Seedance real-call authorization and skips only the 1-5s smoke duration cap. Env gate, redacted event log, explicit Seedance model, and URL allowlist remain required.
- Media URL allowlist now reads `SUB2API_MEDIA_URL_ALLOWLIST` first and falls back to `SUB2API_VIDEO_URL_ALLOWLIST`.

## Claude Frontend Notes

- No frontend change is required for R1-A/C/D/E.
- For Seedance production tests from QCanvas `/v1/video/tasks`, current backend still requires `trial_mode:"tiny_real"` when `provider:"seedance"` is used with API-key auth. With a `production_authorized` provider account, duration may use the contract range `-1` or `4..15` instead of the tiny smoke 1..5s cap.
- Frontend should keep sending `content[]` with `type`, `role`, `url/text`; backend forwards Seedance payload as `content[]` plus top-level `duration`, `resolution`, `ratio`, and bool options.
- New ops-facing env name is `SUB2API_MEDIA_URL_ALLOWLIST`; old `SUB2API_VIDEO_URL_ALLOWLIST` remains compatible fallback.

## Boss Decisions Needed

- Whether API-key `/v1/video/tasks` may call formal Seedance without `trial_mode:"tiny_real"` once the provider account is `production_authorized`.
- Final production media allowlist domains for QCanvas/TapCanvas assets and Ark result CDN.
- Whether to authorize one paid Seedance confirmation for `ratio:9:16` portrait output and video-to-video `video_url` content behavior.

## Verification

- `go test ./...` in `D:\sub2api-trunk\backend`: PASS.
- `golangci-lint run ./...`: PASS, `0 issues`; tool emitted Windows cache write permission warnings under `AppData\Local\golangci-lint`, not code findings.
- `git diff --check`: PASS; only line-ending warnings for existing files.
- `tools/secret_scan.py --include-untracked`: PASS, no high-confidence tracked-plus-untracked findings; git emitted global ignore permission warning.

## Not Done

- No frontend R1-G work.
- No real provider call, no `.env` read, no key inspection.
- No staging/commit/push/deploy.
- `backend/internal/service/user_member_type.go` tracking remains pending because staging is outside this task boundary.

## Rollback

- Code rollback: revert the video billing/service/repository/adapter/ssrf changes and related tests.
- DB rollback if migration already applied: drop `uq_video_usage_logs_video_task_id`; drop `video_tasks.balance_charged_at`.
- Ops rollback: unset `SUB2API_MEDIA_URL_ALLOWLIST` to fall back to legacy `SUB2API_VIDEO_URL_ALLOWLIST`, or unset both to fail closed.
