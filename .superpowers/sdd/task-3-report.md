# Task 3 report - canonical video administrator surface

Status: 内部可用；canonical runtime 待复核

## Scope

- Seedance provider administration: encrypted secret input, masked responses, group binding, enable/disable, and a one-time `tiny_real` authorization record.
- Administrator task list/detail evidence and read-only system check.
- Complete standard-mode routes/sidebar entries; no employee administrator entry and no demo/simple hiding.
- Additive development migration 178 records authorization state only and performs no dispatch.

## Safety

- Real upstream calls executed: 0.
- No secret or `.env` is committed or printed.
- `docs/legal/admin-compliance.zh.md` remains the pre-existing dirty baseline and is excluded.
- No push or production migration.

## Verification

- Focused Go service/repository/handler/route tests: pass.
- Frontend typecheck: pass.
- Focused video, brand, route and sidebar Vitest: 20/20 pass.
- `go test ./migrations -count=1`: pass.
- `go vet ./...`: pass.
- `go build ./...`: pass.
- Frontend production build: pass (existing chunk-size/dynamic-import warnings only).

## Follow-up cold review closure

- Simple mode now keeps the complete video administrator entry visible and has a runtime filtering test.
- Provider create/update uses a backend-owned canonical model and allowlisted Ark endpoint; the UI renders this contract read-only.
- A paid dispatch requires the process kill switch and an unconsumed database provider grant. Provider grant, global gate, and task dispatch claim are committed atomically before any upstream request.
- Dispatch-time checks revalidate the task group, active standard group, canonical model and endpoint. A denied process gate explicitly fails and releases an already reserved task.
- Administrator contract route has HTTP employee 403 / administrator 200 coverage. Task detail exposes upstream cost, provider error code and last-frame evidence; load failures are explicit.
- Real upstream calls executed: 0. Status remains 待复核 until the approved paid smoke produces complete evidence.
- The PostgreSQL integration assertions for unauthorized zero mutation and concurrent single consumption are committed, but local execution is 待复核 because testcontainers rejects the active rootless Docker context on Windows. Ordinary Go tests, vet, build, frontend typecheck, focused Vitest and production build pass.

## Final important-item closure

- Migration 179 is additive-only: it documents the canonical contract without rewriting historical providers or adding a uniqueness constraint that could reject existing rows.
- Repository create/update validation rejects new duplicate canonical providers per standard employee group.
- One-time authorization revalidates the active standard group, canonical model and allowlisted endpoint before recording the grant; missing providers remain 404 and unavailable grants remain 409.
- The administrator UI filters the selector to standard employee groups and states that restriction explicitly.
- Process authorization has concurrent-consumption coverage: exactly one caller succeeds and `Allowed()` becomes false after consumption.
