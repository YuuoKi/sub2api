# Real Product Readiness Closeout — 2026-07-15

**Status:** `READY_FOR_USER_REAL_TEST / 待复核`  
**Not status:** 内部可用  
**Branch:** `wujie/video-capture-moat-20260702`  
**无付费代码证据 tip:** `8296c2a6`（镜像 `sub2api:real-readiness-8296c2a6` 基于此）  
**G7 审查包纳入提交:** `a2344431`  

**Workdir:** `D:\sub2api-trunk` (linked worktree)

## What was completed without paid Provider calls

| Gate | Result |
|---|---|
| G0 baseline + protect parallel dirty files | PASS — prerequisite `61e776fd` committed; evidence under `.delivery-tools/real-product-readiness-20260715/` |
| G1 shared RealCreateGuard in product chokepoints | PASS — `d6a35110` + follow-up fixes |
| G2 execution_mode mock/review_real/internal_real | PASS — `fd0cc5f0` + `a9e8e12f` |
| G3 Gemini 0-create product recovery fixture | PASS — `5de80939` |
| G4 Gemini persistent assets preview/download/reuse | PASS — `c9c05084` |
| G5 Provider billing import/reconcile | PASS — `05b40e49` + `9af957d3` |
| G6 unpaid regression + Docker + 3-role browser | PASS (details below) |
| G7 truth docs + review package + user test card | This closeout + `docs/reviews/LATEST_REVIEW_PACKAGE.html` |

## G6 unpaid evidence

- Backend: `go vet` + package tests for config/service/handler/routes/cmd/server PASS
- WSL Ubuntu-24.04 repository integration: PASS after `8a8d1ea2` outbox isolation fix (executed, not skipped)
- Frontend: lint / vue-tsc / vitest 150 files 944 tests / production build PASS
- Image: `sub2api:real-readiness-8296c2a6`  
  ID `sha256:ad432520f38c60fe67e85aaab7878ab47c901e12847fb5b2eacb9d972e1864fb`  
  Bind `127.0.0.1:18080`, `/health` ok, app process UID 1000
- Browser (mock only): 9 screenshots, 79 business API 2xx, secret-pattern hits 0  
  Evidence: `docs/reviews/assets/real-product-readiness-20260715/`
- Console: transient admin 423 before compliance accept; employee CSP inline-script noise (same class as prior review)

## Critical fixes after dual review

- API-key Seedance creates no longer select `review_only` accounts; they force `internal_real` policy reservation (`8296c2a6`)
- Admin policy save no longer hardcodes `allow_member: true`
- Batch-image real confirm shows model/spec/count/cost/quota impact
- Image `internal_real` policy reserve converts USD hold → CNY

## Explicitly left for the user

1. Real Gemini low-spec image create (1) with preview/download/reuse
2. Real Seedance 5s 9:16 video create (1, create=1) with play/download/reuse
3. Upload real Gemini + Seedance billing statements
4. **Human eyeball (hard gate, not automatable):**
   - **Media:** real image opens and matches prompt; real video plays ~5s / ~9:16; download opens locally; reuse still resolves
   - **Copy:** real-mode confirm shows cost/quota impact; list vs detail status agree; currency consistent; no secret/path leaks in failure text
   - **Known mock-package copy debt (log, do not block real create):** EN page titles on some video admin/employee pages; EN labels on provider-billing form; `$` balance vs CNY policy limits; G6 snapshot 0/0 ≠ historical budget 1/4 · 2/4 · ¥20
5. Disable review_only accounts and retire temporary keys
6. Decide whether to enable `internal_real` only after formal non-review_only channels exist

## Session budget continuity

Do not reset counters. Current authorized session budget remains: image 1/4, video 2/4, reserved ¥20 / ¥60. Remaining budget is for user real test only.

## Rollback

Prefer `git revert <commit>` per G-stage. Emergency: keep `SUB2API_REAL_REVIEW_SESSION_ENABLED=0` and leave internal_real policy kill switch on. Billing reconcile never auto-mutates user balances.

## Stop local G6 stack

```powershell
wsl.exe -d Ubuntu-24.04 --exec bash -lc "cd /mnt/d/sub2api-trunk/.delivery-tools/real-product-readiness-20260715 && docker compose -f compose.g6.yml -p sub2api-g6 down"
```
