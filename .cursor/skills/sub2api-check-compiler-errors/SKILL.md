---
name: sub2api-check-compiler-errors
description: Run Sub2API compile, lint, and type-check commands. Use after backend or frontend edits before claiming work complete.
---

# Sub2API Check Compiler Errors

## Trigger

Backend or frontend files were edited and local validation is needed.

## Commands

Run from repository root:

```bash
# Backend (Go tests + golangci-lint)
make test-backend

# Frontend (eslint + vue-tsc + critical vitest)
make test-frontend

# Secret scan before push
make secret-scan
```

## Workflow

1. Detect which areas changed (`backend/` vs `frontend/`).
2. Run the matching commands above.
3. Fix highest-confidence errors first.
4. Re-run until clean or blocked.

## Output

- Pass/fail per command
- Errors grouped by file
- Remaining blockers
