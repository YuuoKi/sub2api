# Sub2API Video Capture Evidence

## Stage 1 Baseline Gates

Status: pass.

| Area | Command | Result |
| --- | --- | --- |
| Backend | `cd backend; go build ./...` | First sandbox run failed with `Access is denied` against `AppData\Local\go-build`; rerun with approved elevated filesystem access exited 0. |
| Backend | `cd backend; go test ./...` | exit 0. |
| Frontend | `cd frontend; pnpm test:run` | exit 0, 97 files / 581 tests passed. Existing test stderr/warnings were observed but did not fail the suite. |
| Frontend | `cd frontend; pnpm typecheck` | exit 0. |
| Frontend | `cd frontend; pnpm lint:check` | exit 0. |
| Secret scan | `python tools/secret_scan.py --include-untracked` | PATH had no `python`; command could not start. |
| Secret scan | bundled Python `tools/secret_scan.py --include-untracked` | exit 0, `secret-scan: no high-confidence tracked-plus-untracked findings`. |

## Stage 1 Notes

- No real provider call was triggered.
- No secret, `.env`, token, cookie, or provider credential file was read.
- Remaining untracked files from the prior boundary remain untouched: `.impeccable/hook.cache.json`, `MORNING_RESULT_2026_06_28.md`.
- `_review/video-capture-moat-20260702/` is new evidence output for this task.

## Stage 2 TDD Evidence

### RED

Command:

```powershell
cd D:\sub2api-trunk\backend
go test ./internal/service -run "TestCollectorStoresVideoTaskMetadataWithoutAccountAttribution|TestVideoGatewayCapturesSucceededTaskContent|TestVideoGatewayCaptureDisabledByFlag|TestVideoGatewayCaptureFailOpenKeepsSucceededAndUsage" -count=1
```

Result:

```text
FAIL github.com/Wei-Shaw/sub2api/internal/service [build failed]
internal\service\generation_content_collector_test.go:116:12: collector.CollectVideoTask undefined
internal\service\generation_content_collector_test.go:116:51: undefined: VideoGenerationContentCaptureArgs
internal\service\generation_content_collector_test.go:132:9: row.TaskID undefined
internal\service\video_gateway_worker_test.go:299:6: svc.SetGenerationContentCollector undefined
```

The first sandbox run hit Go build-cache ACL denial; the elevated rerun above reached the intended red failure.

### GREEN

Command:

```powershell
cd D:\sub2api-trunk\backend
go test ./internal/service ./internal/repository -run "TestCollectorStoresVideoTaskMetadataWithoutAccountAttribution|TestVideoGatewayCapturesSucceededTaskContent|TestVideoGatewayCaptureDisabledByFlag|TestVideoGatewayCaptureFailOpenKeepsSucceededAndUsage|TestGenerationContentRepositoryCreateVideoTaskContent" -count=1
```

Result:

```text
ok github.com/Wei-Shaw/sub2api/internal/service    4.224s
ok github.com/Wei-Shaw/sub2api/internal/repository 4.109s
```

## Stage 2 Final Gates

Status: pass.

| Area | Command | Result |
| --- | --- | --- |
| Backend | `cd backend; go build ./...` | exit 0 |
| Backend | `cd backend; go test ./...` | exit 0 |
| Frontend | `cd frontend; pnpm test:run` | exit 0, 97 files / 581 tests passed; existing stderr/warnings observed but suite passed |
| Frontend | `cd frontend; pnpm typecheck` | exit 0 |
| Frontend | `cd frontend; pnpm lint:check` | exit 0 |
| Secret scan | bundled Python `tools/secret_scan.py --include-untracked` | exit 0, `secret-scan: no high-confidence tracked-plus-untracked findings`; git global ignore warning observed |

## Stage 2 Safety Notes

- No provider call was triggered; all video execution proof uses the existing mock adapter and in-memory repository tests.
- `content_capture.enabled` remains the gate. Injecting a collector into `VideoGatewayService` is not enough to collect when the flag is off.
- Video rows leave `api_key_id`, `group_id`, and `account_id` empty. `task_id` is the durable idempotency/join key.
- `provider_account_id` is not written to `ai_generation_content.account_id`.
- Capture runs after terminal usage logging and video billing deduction; collector repo errors/panics are fail-open.
