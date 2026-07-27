# Task 2 report - HC-ATOM async image provider

Status: LOCAL CODE/MOCK GREEN; real provider and production deployment remain unverified.

## RED evidence

1. `go test ./internal/service -run '^TestHCAtomBatchProvider_SubmitUsesFixedAsyncContract$' -count=1`
   initially failed because `NewHCAtomBatchImageProvider`, `NewHCAtomBatchHTTPClient`, and `PlatformHCAtom` did not exist.
2. `go test ./internal/service -run '^TestHCAtomBatchProvider_UsesDedicatedPlatformAndMediaGroupGate$' -count=1`
   initially failed with `expected hc_atom, actual gemini`; HC was still sent to the Gemini platform pool.

## GREEN evidence

`GOCACHE=D:\sub2api-trunk\.gocache-task2 go test ./internal/service -run '^TestHCAtomBatchProvider_(SubmitUsesFixedAsyncContract|UsesDedicatedPlatformAndMediaGroupGate)$' -count=1` - PASS.

`git diff --check` - no diagnostics before the focused commit.

## Landed code

Commit `287befa1e` adds:

- distinct `hc_atom` platform and batch provider registration;
- fixed HTTPS origin and POST/GET/DELETE path construction; no configurable relay or provider fallback;
- Bearer auth, one stable `hc-image:<batch-id>` upstream idempotency key, exact-one-item enforcement, enabled-model gate, strict status mapping, sanitized HTTP/business errors, and deterministic URL deduplication helper;
- explicit HC account selection and HC media-group batch gate;
- fake-transport tests for the fixed create contract and group/platform selection.

## Completion blockers / remaining risks

- `OpenResult` intentionally fails closed with `HC_ATOM_ARCHIVE_REQUIRED`. The current shared `BatchImageProvider.OpenResult` contract exposes provider JSONL directly and has no Sub2-owned asset store/index reference. Implementing controlled download, redirect/DNS revalidation, MIME/signature/byte/dimension checks, archive/index persistence, and durable download reads requires a new result-asset persistence boundary. It must be added before HC can be considered an end-to-end enabled provider.
- The admin account persistence/cache path has not yet acquired a dedicated HC encrypted credential domain; no plaintext/admin/cache sentinel scan has been added.
- No full batch service/package regression was run after the focused provider test; no real provider call, real key read, push, deploy, merge, reset, clean, or rebase occurred.

## Files

- `backend/internal/service/batch_image_provider_hc_atom.go`
- `backend/internal/service/batch_image_provider_hc_atom_test.go`
- `backend/internal/service/batch_image_public.go`
- `backend/internal/service/batch_image_provider.go`
- `backend/internal/service/batch_image.go`
- `backend/internal/{domain,service}/constants.go`

## Secret-domain completion (Task2 focused follow-up)

Status: GREEN for the dedicated HC account credential boundary.

### RED evidence

1. `go test -tags unit ./internal/service -run '^TestHCAtomSecretBoundary_CreateEncryptsSentinelBeforeRepository$' -count=1`
   failed because the repository received the transient sentinel in `credentials.api_key`.
2. `go test -tags unit ./internal/service -run '^Test(ProtectHCAtomAccountCredentials|ResolveHCAtomAPIKey)_' -count=1`
   failed because the HC cipher/protection/resolver API did not exist.
3. `go test ./internal/config -run '^TestLoadHCAtomImageCredentialDomain' -count=1`
   failed because the independent HC enable/key config did not exist.
4. `go test ./internal/handler/dto -run '^TestAccountFromServiceShallow_RedactsSensitiveCredentials$' -count=1`
   failed because the authenticated ciphertext was returned in the admin DTO.
5. `go test ./internal/repository -run '^TestSchedulerCacheHCAtomSnapshotNeverSerializesPlaintextSentinel$' -count=1`
   failed because no HC-safe scheduler snapshot serializer existed.

### GREEN evidence

- `go test -tags unit ./internal/service -count=1` - PASS.
- `go test ./internal/service ./internal/config ./internal/handler/dto ./internal/repository -count=1` - PASS.
- `go build ./cmd/server` - PASS.
- `git diff --check` - PASS.

### Implemented boundary

- Dedicated AES-256-GCM HC image credential domain with domain AAD and `hc1:` envelope.
- Default-denied `batch_image.hc_atom_enabled`; enabling requires
  `BATCH_IMAGE_HC_ATOM_ENCRYPTION_KEY` as a 32-byte hex key and rejects reuse of
  the video or JWT key.
- Admin Create/Update accepts `credentials.api_key` only transiently, persists
  ciphertext plus masked/configured metadata, and preserves the prior
  ciphertext when Update omits a new key.
- Admin DTO strips ciphertext and plaintext; scheduler full/metadata snapshots
  strip plaintext while retaining only ciphertext plus masked/configured/model
  metadata.
- HC provider account checks do not decrypt. Submit/Get/Delete/OpenResult decrypt
  on demand after a compatible `platform=hc_atom,type=apikey` account is selected.
- Config-backed provider registry does not register HC unless the dedicated
  feature and key are enabled. Default registry remains HC-disabled.

### Remaining risks

- No migration was added for hypothetical pre-existing plaintext HC account
  rows; HC image accounts are new and fail closed without the ciphertext field.
- The broader Task2 controlled result-archival and end-to-end product status are
  governed by the provider/archive commits and their separate acceptance.

No real API call, no real secret read, no frontend/video/docs change, no push,
deploy, merge, reset, clean, or rebase.

## Pipeline closeout (Task2 integration)

Status: GREEN for the fake HC public-batch path; no production provider call was made.

### RED evidence

1. `TestBatchImagePublicSubmit_HCAtomRejectsGeminiOnlyGroupBeforeAccountSelection`
   initially received `nil`: a Gemini-platform group could submit an explicit
   `hc_atom` request because the group gate accepted either platform without
   matching the requested provider.
2. `TestHCAtomBatchHTTPClient_DeleteRejectsBusinessFailure` initially received
   `nil` for HTTP 200 with `code:40001`: DELETE treated an HC business failure
   as confirmed cancellation.

### GREEN evidence

- Focused HC pipeline suite (group, DELETE, cancel hold, disabled Dola,
  result URL/usage, fake E2E, and uncertain-submit no replay): PASS.
- `go test -tags unit ./internal/service -count=1`: PASS (105.968s).
- `go test ./internal/config ./internal/repository ./internal/handler/dto -count=1`: PASS.
- `git diff --check`: PASS before commit `95fcbf5`.

### Landed pipeline changes

- Commit `95fcbf5` binds an explicit provider to its media-group platform;
  HC cannot enter a Gemini group, and account selection is never reached.
- HC DELETE now accepts 204/empty success but validates a non-empty 2xx
  business envelope. A rejected DELETE remains submitted, emits no cancel
  event, and does not release the hold.
- Fake end-to-end coverage uses the real HC provider with fake HTTP transports:
  public create, HC account selection, hold, one Submit, GET SUCCESS,
  validated owned JSONL/base64 archival, index, settle, and repeated settlement
  with one capture only.
- Additional coverage verifies Dola stays disabled, one-item enforcement,
  deterministic URL deduplication, missing `usage.imageCount` does not invent
  a count, and uncertain Create is not replayed for the same local intent.

### Remaining risk

The tests prove the local fake pipeline only. Real HC credentials, paid calls,
deployment, push, merge, frontend, and video were not used or performed.

## Review remediation (non-secret hardening)

Status: GREEN for the requested local non-secret hardening scope.

- `039401208 fix(images): persist HC owned batch results`: HC archives a
  successful remote result once into a configured, deterministic owned JSONL
  path using restrictive permissions and atomic rename. Later provider instances
  read the opaque owned ref without upstream GET; output cleanup removes it.
- `8aa5406fd fix(groups): allow HC atom batch image groups`: Admin Create/Update
  now preserves batch-image enablement for `PlatformHCAtom` when image generation
  is allowed.
- RED/GREEN: restart-owned-result and HC group-create focused tests passed.
- `6d09dec56 fix(images): bound HC atom HTTP protocol`: only explicit business
  `code=0` succeeds; the HC API has a dedicated 30-second overall timeout plus
  bounded dial/TLS/header phases, rejects API redirects, and caps JSON bodies at
  1 MiB. Tests cover 401/403/429/5xx, timeout, body-read failure, oversized JSON,
  and missing business code.
- `707e02cea fix(images): validate HC atom result archives`: every result redirect
  hop repeats the HTTPS/443/no-credentials/no-fragment/public-address policy;
  the production transport still validates every DNS answer before dialing.
  JPEG/WebP containers and dimensions are validated, including a 40 MP pixel
  ceiling. Raw images are capped at 11 MiB, the final owned JSONL line must stay
  below the shared 16 MiB index/download scanner boundary, and WebP requires an
  actual complete VP8/VP8L image chunk.
- `5caeead77 fix(images): align HC atom result contracts`: all camel/snake,
  plural/singular result URL aliases aggregate deterministically into one
  `custom_id` JSONL line with multiple `inlineData` parts. A positive
  `usage.imageCount` must match the distinct URL count. Dola remains explicitly
  disabled and is not returned by `ListModels`. Usage logs select the upstream
  endpoint from `job.Provider`; HC records the fixed
  `https://api-aigc.fzyinghe.com/image/generation/tasks` endpoint, not Vertex.
- `a3d5ee3ef fix(images): persist only newly archived output refs`: the full
  service gate exposed a regression from `039401208`; ordinary pre-existing
  output refs were being mistaken for newly archived refs. The indexer now
  persists an output ref only when `OpenResult` changes it, preserving HC owned
  refs without breaking existing providers.

### Additional RED evidence

1. Missing business `code` returned success.
2. The default HC API client had no overall timeout, and an oversized but
   syntactically valid JSON envelope was accepted.
3. Unsafe redirect targets were followed to a second hop; truncated JPEG/WebP,
   a 10001-pixel JPEG axis, a 48 MP JPEG, and a 12 MiB raw image were accepted.
4. `result_url` was dropped when `resultUrl` was also present, and a positive
   usage count could disagree with distinct result URLs.
5. Dola appeared in `ListModels`, while HC usage was logged as
   `vertex:batchPredictionJobs`.
6. The first full service run failed
   `TestBatchImageResultIndexer_WritesCountsAndReplacesItems` and
   `TestBatchImageResultIndexer_ReconcilesMissingAndUnknownCustomIDs` with
   `BATCH_IMAGE_JOB_NOT_FOUND`; the focused reproduction passed after
   `a3d5ee3ef`.

### Fresh GREEN gates

- `go test ./internal/service -run '^TestHCAtomBatch' -count=1` - PASS.
- Focused unit ListModels, settlement, and result-indexer tests - PASS.
- `go test -tags unit ./internal/service -count=1` - PASS (116.8s).
- `go test ./internal/config ./internal/repository ./internal/handler/... -count=1`
  - PASS (46.6s).
- `go build -o D:\sub2api-trunk\.tmp-hc-task2-server.exe ./cmd/server`
  - PASS (24.4s); the temporary binary was removed.

### Remaining boundaries

- No real HC credential was read, no paid/provider call was made, and no
  production deployment or browser path was exercised. These tests prove local
  fake transports, persistence, indexing, settlement, and build behavior only.
- Dola remains disabled until a vendor endpoint and acceptance evidence exist.
- This remediation did not modify the separately owned secret allowlist or error
  sanitizer work.
- No frontend or video code was changed. No push, deploy, merge, reset, clean,
  rebase, or destructive repository cleanup occurred.
