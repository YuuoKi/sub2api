# AGENTS.md

This repository is the Sub2API-based enterprise AI video API dispatch console. It is a white-label and extension project, not a from-scratch gateway rewrite.

## Working Directories

- Source repository: `02_source/sub2api`
- Review packages: `03_审查包`, outside this Git repository
- Frozen Phase 3.8 snapshot: `03_审查包/05_Phase_3_8_封存快照`

## Long-Term Rules

- Do not push.
- Do not deploy to production.
- Do not read, print, copy, export, or package real API keys, cookies, Authorization values, tokens, JWTs, or account passwords.
- Do not document reverse engineering, packet capture, bypass, or risk-control evasion steps.
- Do not implement automatic cookie, token, or login-state collection.
- Do not modify the Phase 3.8 frozen snapshot.
- Do not mix customer-shareable packages with internal review packages.
- Do not rewrite Sub2API auth, users, permissions, or the LLM Gateway main path unless explicitly approved for a later phase.
- Keep video-provider work behind thin adapters and safe review packages.

## Validation Commands

- `git status --short`
- `git diff --check`
- Backend source changes: `cd backend && go test ./...`
- Backend source changes: `cd backend && go build ./cmd/server`
- Frontend source changes: `cd frontend && pnpm build`

## Completion Standard

- Scope matches the approved phase.
- Review package is self-contained and `MANIFEST.json` parses.
- No real credentials appear in frontend responses, logs, screenshots, review packages, or ZIP files.
- Production remains `NOT_READY` until deployment, monitoring, backup, security, and authorized real-provider checks are complete.
