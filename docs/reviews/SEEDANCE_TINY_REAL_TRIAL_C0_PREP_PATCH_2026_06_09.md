# Sub2API C0 Seedance Tiny Real Trial Prep Patch Review

**Date:** 2026-06-09
**Round:** C0 Prep Patch (code only — no real calls, no production writes)

---

## 1. Repository Baseline & Status

- **Path:** `/mnt/d/Codex创业任务/企业 API 管理后台项目/02_source/sub2api`
- **Branch:** `phase-3.8.2-overnight-readiness`
- **HEAD before:** `3351338d feat: add api-key video mock gateway for qcanvas`
- **HEAD after:** `3351338d` (uncommitted changes)
- **Status:** 3 modified files in `backend/`; pre-existing dirty files in `deploy/`/`docs/` untouched

---

## 2. Files Changed This Round

| # | File | Change |
|---|------|--------|
| 1 | `backend/internal/handler/video_handler.go` | Added `trial_mode` to request; route mock vs seedance-trial; dynamic response fields (`MockOnly`, `ProviderBoundary`, `RealProviderDispatchCount`, `TrialMode`, `BlockedReason`, `TrialGateResult`) |
| 2 | `backend/internal/service/video_gateway_service.go` | Added `ListAPIKeyTrialProviders` (returns seedance readiness with blocked reasons); added `CreateAPIKeySeedanceTinyTrialTask` (full gate chain: env + account metadata + model + duration + daily limit) |
| 3 | `backend/internal/server/routes/api_key_video_gateway_test.go` | Added 8 new tests covering all blocked paths + success path; kept existing mock tests passing |

---

## 3. What Was NOT Done (Explicitly Out of Scope)

- ❌ **No real Seedance calls.** All paths to the real adapter are gated and currently blocked.
- ❌ **No reading of secrets.** No `.env`, key, token, or secret was read or printed.
- ❌ **No writing to production asset store.** All results remain in-memory or test-repo only.
- ❌ **No deploy.** No Caddyfile, Dockerfile, or deploy script modified.
- ❌ **No push.** Changes remain local.
- ❌ **No git add .** Only whitelisted modified files were staged (if committed).
- ❌ **No production configuration changes.** Env gates remain unset.

---

## 4. Four-Layer Block Handling Results

| Layer | Before | After | Status |
|-------|--------|-------|--------|
| Sub2API `/v1/video/*` hardcoded mock-only | `provider != mock` → 403 | **Conditional gate:** mock unchanged; seedance allowed only with `trial_mode=tiny_real` + all env gates | **Retained as gate** (default still blocked) |
| Seedance smoke gate (env + account metadata) | Existed in adapter but unreachable from API-key layer | **Surfaced:** `ListAPIKeyTrialProviders` returns readiness; `CreateAPIKeySeedanceTinyTrialTask` runs full gate chain | **Retained as gate** (all conditions must be true) |
| Demo account key_status / RouteAvailable | No seedance account configured | Still not configured; synthetic provider returned with `route_available=false` | **Still blocked** — requires user to configure account + metadata |
| QCanvas Hono proxy + frontend adapter hardcoded `provider: "mock"` | Unconditional mock | **Conditional:** default mock; seedance requires env gates + trial mode (see QCanvas review doc) | **Retained as gate** (default still blocked) |

---

## 5. New Gate List

All gates must be simultaneously true for a real Seedance call to proceed:

| # | Gate | Controlled By | Current Value |
|---|------|---------------|---------------|
| 1 | `SUB2API_VIDEO_REAL_SMOKE_ENABLED=1` | Environment variable | ❌ Not set |
| 2 | Provider account exists and enabled | Database / `ListProviderAccounts` | ❌ Not configured |
| 3 | `single_smoke_authorized=true` in account metadata | Database metadata | ❌ Not set |
| 4 | `SUB2API_VIDEO_REDACTED_EVENT_LOG` non-empty | Environment variable | ❌ Not set |
| 5 | Model name contains "seedance" | Request body | ✅ Validated |
| 6 | Duration 1–5 seconds | Request body | ✅ Validated |
| 7 | `trial_mode="tiny_real"` in request | Request body | ✅ Validated |
| 8 | Max 1 real call per user per day | Service logic (`ListTasks` count) | ✅ Enforced |
| 9 | No batch, no auto-retry | Architecture (single task endpoint) | ✅ By design |
| 10 | No fallback to mock | Explicit error return on block | ✅ Enforced |

---

## 6. New Diagnostics / Response Fields

### API-Key Video Task Response (`apiKeyVideoTaskResponse`)

| Field | Type | Meaning |
|-------|------|---------|
| `mock_only` | `bool` | True for mock tasks, false for seedance trial |
| `provider_boundary` | `string` | `"api-key-video-mock-only"` or `"api-key-video-seedance-tiny-trial"` |
| `real_provider_dispatch_count` | `int` | 0 for mock, 1 for seedance trial |
| `trial_mode` | `string` | `"tiny_real"` when applicable |
| `blocked_reason` | `string` | Why trial was blocked (if applicable) |
| `trial_gate_result` | `string` | `"passed"` or `"blocked"` |

### Provider List Response (`videoProviderAccountResponse` metadata)

| Field | Type | Meaning |
|-------|------|---------|
| `route_available` | `bool` | Always `false` for seedance in API-key gateway |
| `route_skip_reason` | `string` | Human-readable readiness status |
| `tiny_real_trial_supported` | `bool` | `true` if seedance adapter exists |
| `requires_real_smoke_gate` | `bool` | `true` for seedance |
| `blocked_reasons` | `[]string` | List of unsatisfied gates (in metadata) |

---

## 7. Test Commands & Results

```bash
cd backend
go test ./internal/server/routes -run "TestAPIKeyVideoGateway" -v
```
**Result:** 11 passed, 0 failed

```bash
cd backend
go test ./internal/service -run "TestVideoGateway|TestVideoProvider|TestVideoAdapterContractSafeProviderBehavior" -v
```
**Result:** 7 passed, 0 failed

### New Tests Added

| Test | Purpose |
|------|---------|
| `TestAPIKeyVideoGatewayBlocksSeedanceWithoutTrialMode` | Seedance without `trial_mode` → 403 |
| `TestAPIKeyVideoGatewayBlocksKling` | Kling always blocked |
| `TestAPIKeyVideoGatewaySeedanceTrialBlockedWithoutGate` | All env/account gates missing → 403 |
| `TestAPIKeyVideoGatewaySeedanceTrialBlockedDurationTooLong` | Duration=10 → blocked |
| `TestAPIKeyVideoGatewaySeedanceTrialBlockedMissingAuthorization` | Missing `single_smoke_authorized` → blocked |
| `TestAPIKeyVideoGatewaySeedanceTrialBlockedMissingEventLog` | Missing `SUB2API_VIDEO_REDACTED_EVENT_LOG` → blocked |
| `TestAPIKeyVideoGatewaySeedanceTrialSuccess` | All gates satisfied → task created (in-memory) |
| `TestAPIKeyVideoGatewayProvidersReturnSeedanceReadiness` | `/providers` returns seedance with readiness info |

---

## 8. Five Items User Must Confirm Before Next Round

Before executing a **single real Seedance tiny real trial**, the user must explicitly confirm:

1. **Provider + Model**
   - Provider: `seedance`
   - Model: must contain `"seedance"` (e.g., `seedance-2`, `seedance-2-fast`)
   - Confirm this is the intended model and not a different provider.

2. **Single Budget & Max Call Count**
   - Max 1 call per user per day enforced in code.
   - Duration capped at 1–5 seconds.
   - Confirm the expected cost range for a 1–5s Seedance video and that spend is acceptable.

3. **Credential Injection Method**
   - Sub2API seedance provider account must be created with valid API key.
   - Account metadata must include `"single_smoke_authorized": true`.
   - Env vars: `SUB2API_VIDEO_REAL_SMOKE_ENABLED=1` and `SUB2API_VIDEO_REDACTED_EVENT_LOG=<path>`.
   - Confirm credentials will be injected securely (encrypted at rest, never in logs).

4. **Write Boundary**
   - Even if the trial succeeds, the result MUST NOT write to the production asset store.
   - Result is stored as a task row only; no formal asset persistence.
   - Confirm this boundary is acceptable.

5. **Success / Failure Stop Condition**
   - On success: STOP. Do not proceed to a second call without explicit re-authorization.
   - On failure: STOP. Do not retry automatically. Review diagnostics, fix root cause, then re-authorize.
   - Confirm the stop-after-one rule is understood.

---

## 9. Next Round Task Draft

**Task Name:** Sub2API C1 Seedance Tiny Real Trial — Single Controlled Call

**Prerequisites (ALL must be true):**
- [ ] User explicitly confirms the 5 items above in writing.
- [ ] Sub2API seedance provider account created with valid encrypted API key.
- [ ] Sub2API account metadata: `"single_smoke_authorized": true`.
- [ ] Sub2API env: `SUB2API_VIDEO_REAL_SMOKE_ENABLED=1`.
- [ ] Sub2API env: `SUB2API_VIDEO_REDACTED_EVENT_LOG=<non-empty-path>`.
- [ ] Sub2API tests pass at HEAD.

**Execution Steps:**
1. Create a single task via `POST /v1/video/tasks` with:
   - `provider: "seedance"`
   - `trial_mode: "tiny_real"`
   - `model: "seedance-2-fast"` (or agreed model)
   - `duration: 3`
   - `prompt: <safe test prompt>`
2. Worker picks up the task and dispatches to Seedance adapter.
3. Poll task until terminal state.
4. Capture and record:
   - Upstream task ID from Seedance
   - Result URL (if succeeded)
   - Full event log
   - Redacted event log entries
5. Verify:
   - `real_provider_dispatch_count === 1`
   - No production asset writes occurred
6. STOP. Report results. Await user authorization for next step.

**Rollback Plan:**
- Unset `SUB2API_VIDEO_REAL_SMOKE_ENABLED` to immediately block all real calls.

---

## 10. Blocked Status Markers

| Item | Status | Notes |
|------|--------|-------|
| Code paths prepared | ✅ Ready | Handler + service have gated code |
| Mock tests passing | ✅ Ready | All existing + new tests pass |
| Real provider account | ❌ BLOCKED | Not configured; user must create |
| Env smoke gate | ❌ BLOCKED | Not set; user must set |
| Event log path | ❌ BLOCKED | Not set; user must set |
| Real trial executed | ❌ BLOCKED | Requires all above |

**Verdict:** C0 prep patch is complete. The system is ready to enter C1 single trial **only after** the user explicitly configures the blocked items and confirms the 5 prerequisites.
