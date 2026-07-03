# Sub2API Chat Strategy Context

Updated: 2026-07-04 Asia/Shanghai

This file gives new chats and follow-up agents the current strategy context for Sub2API.

## One-line Judgment

Sub2API can continue as Wujie internal AI API and production-scheduling evidence infrastructure. B2 fixed a payment webhook P1 issue and connected the QCanvas mock/dev adoption feedback bridge, but real paid provider operation remains `已冻结` unless separately authorized.

## What Sub2API Is

- Internal API management and model scheduling substrate.
- Central evidence surface for usage, cost, tasks, generation-content ledger, weekly reports, and Admin review.
- Server-side owner of provider credentials and admin ledger APIs.
- The target that QCanvas can proxy to through hono-api for controlled adoption feedback.

## What Sub2API Is Not

- Not a public commercial platform ready for external users.
- Not a blanket authorization for real Seedance/Kling paid provider production.
- Not a place for QCanvas web clients to hold admin keys.

## Verified Capabilities

- Payment webhook P1: commit `628ddd10` rejects provider lookup failures with `400 verify failed`; unknown order ack remains 2xx.
- B2 final-audit log: commit `4b1cf24b` appended B2 records to tracked `_review/MEGA_LOOP_FINAL_AUDIT_20260703/VERIFY.log`.
- Backend gates recorded in B2: PaymentWebhook unit test, full `go test ./...`, and `golangci-lint run` exit 0.
- Frontend gates recorded in B2: lint, typecheck, test:run exit 0.
- Secret scan recorded in B2: bundled Python run exit 0.
- QCanvas B2 proxy: `POST /sub2api/v1/generation-content/:task_id/adoption` is reached through QCanvas hono-api, not from web to Sub2API admin directly.

## QCanvas Bridge Contract

QCanvas web sends user-authenticated adoption feedback to QCanvas hono-api:

`POST /sub2api/v1/generation-content/:task_id/adoption`

QCanvas hono-api validates JWT/user context, then uses server-side Sub2API credentials to call:

`POST /api/v1/admin/generation-content/:task_id/adoption`

Sub2API `saved:false` responses remain meaningful and must be shown by QCanvas rather than converted to generic success. Required reasons covered by B2 are:

- `content_capture_disabled`
- `adoption_feedback_unavailable`
- `task_not_found`

## Pending / Blocked

- Real paid provider calls: `已冻结`.
- Public deployment / external access: `已冻结`.
- AUTH redesign: outside B2.
- QCanvas root pnpm: `已阻塞`.

## Recommended Next Chat Path

1. Read this file and `00_START_HERE.md`.
2. Read `_review/B2_webhook_fix_20260704/SUMMARY.md` and `VERIFY.md`.
3. If validating the cross-repo loop, read QCanvas `docs/reviews/B2_adoption_20260704/`.
4. Stay in mock/dev unless the operator explicitly authorizes real provider use with budget and stop conditions.
