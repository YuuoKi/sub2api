# Task 2 report - HC-ATOM async image provider

Status: PARTIAL / blocked from completion by the existing batch-result persistence contract.

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
