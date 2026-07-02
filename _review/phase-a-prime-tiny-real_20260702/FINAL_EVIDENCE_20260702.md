# Phase A' Tiny Real Final Evidence

Status: 内部可用 / 可演示

## Scope

- Sub2API ran in WSL Docker under compose project `sub2api_phasea_prime`.
- QCanvas `/studio-v2` ran against Sub2API at `http://127.0.0.1:8080`.
- No secrets, API keys, JWTs, cookies, database passwords, or full signed result URLs are included in this evidence file.

## Evidence Summary

- Seedance preflight: `ready`
- QCanvas task id: `1`
- Final status: `succeeded`
- Result URL present: `true`
- QCanvas node `realChainReady`: `true`
- SQL capture rows for task `1`: `1`
- Admin stats `is_live`: `true`
- Admin stats `captured_today`: `1`

## Evidence Files

- QCanvas review package: `D:\Codex创业任务\QCanvas（无界版）\QCanvas\docs\reviews\PhaseA-prime-tiny-real_20260702\REVIEW_PACKAGE.html`
- QCanvas latest review package: `D:\Codex创业任务\QCanvas（无界版）\QCanvas\docs\reviews\LATEST_REVIEW_PACKAGE.html`
- QCanvas three-proof screenshot copy: `_review\phase-a-prime-tiny-real_20260702\qcanvas_three_proofs_masked.png`
- QCanvas node screenshot copy: `_review\phase-a-prime-tiny-real_20260702\qcanvas_studio_v2_node_masked.png`

## Notes

- An earlier direct Sub2API attempt failed because the previous worker poll cap was too short for Seedance's normal completion latency. That failed task was documented in `BLOCKED_TINY_REAL_20260702.md` and was not repeatedly retried.
- The successful proof path is the QCanvas `/studio-v2` path through Hono to Sub2API and Seedance.
- The review package masks result URL query parameters and does not serialize keys or tokens.

## Cleanup Checklist

- QCanvas Hono/web dev processes stopped.
- Sub2API `docker compose down -v` completed for project `sub2api_phasea_prime`.
- `deploy\.env` removed.
- Temporary local token/key files removed.
