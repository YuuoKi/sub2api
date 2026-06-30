# Empirical full-stack re-run recipe (BLOCKED tonight on Docker; run when daemon is up)

Tonight Docker Desktop's Linux engine would not start unattended (`com.docker.service` Stopped — needs
elevation; WSL `docker-desktop` distro Stopped). The fresh real-PG golden/after byte-diff + SELECT could not
be re-run. This recipe lets you (or Claude) fire it in ~10 min once `docker version` responds.

Pre-built binaries are ready in this folder:
- `server_baseline.exe` — built from clean `b919650f` (PRE-patch) → use for `golden/`
- `server_patched.exe`  — built from `b919650f` + the 12-line patch (POST-patch) → use for `after/`

## Stack (mirror of _review/M1B_smoke_test_20260619/SUMMARY.md §3)
1. `docker run --rm -d --name smoke-pg   -e POSTGRES_PASSWORD=postgres -p 5433:5432 postgres:18-alpine`
2. `docker run --rm -d --name smoke-redis -p 6380:6379 redis:7-alpine`
3. config.yaml in the binary's workdir — see `config.empirical.yaml` in this folder (flag ON, db@5433, redis@6380, port 8090).
4. Seed (raw SQL): user(active,balance=100) + group(anthropic,active) + account(**type=`apikey`**,
   credentials.base_url=`http://127.0.0.1:9099`) + account_group + api_key(`sk-smoke-localtest-0001`).
   ⚠ `account.type` MUST be `apikey` (no underscore) — wrong value makes GetBaseURL fall back to REAL Anthropic.
5. Mock upstream @127.0.0.1:9099 returning a FIXED Anthropic JSON (with `usage`) for non-stream, and a FIXED
   SSE sequence for stream. Determinism (no timestamps/random ids) is required so golden==after is byte-comparable.

## Procedure
- P0: after start, confirm upstream is mock@9099 (one request; verify it did NOT hit real Anthropic).
- golden: run `server_baseline.exe`; curl non-stream + stream (fixed requests) → save raw client bytes to `golden/`.
  Also `SELECT ... FROM ai_generation_content` → confirm main path produces NO row (bug repro).
- after:  run `server_patched.exe` (same stack/seed); same requests → save to `after/`.
  `SELECT id,left(prompt_redacted,40),left(response_redacted,40),response_bytes FROM ai_generation_content ORDER BY id DESC`
  → confirm a NEW row with prompt+response on the same row, redacted (DB pass).
- BC: `diff golden/ after/` (non-stream byte-for-byte; stream compare SSE event order + each data + [DONE]) → MUST be identical.
- BC-OFF: set content_capture.enabled=false, re-run `server_patched.exe`, confirm bytes == baseline-flag-off and 0 new rows.

## Prior empirical proof of THIS identical change (smoke, 2026-06-19/20)
The smoke test already executed this exact end-to-end (one-line equivalent of this patch) and recorded:
- client got mock response unchanged, **Content-Length 294** (transparency held under real run);
- after the line was added, `ai_generation_content` id=2 landed: prompt+response on one row, both redacted,
  `response_bytes=294` (== client Content-Length), full attribution. See M1B_smoke_test_20260619/SUMMARY.md §3.
