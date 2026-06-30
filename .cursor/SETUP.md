# Cursor quality workflow — one-time setup

**Automation is on at two levels:**
- **All projects (this machine):** `~/.cursor/hooks.json` — Agent 结束时自动 quality gate 提醒
- **Sub2API only:** `.cursor/hooks/project-quality-gate.ps1` — 精确的 make test-backend / test-frontend 提示

New project template: `~/.cursor/templates/project-quality/` (copy `.cursor/` into repo root).

Complete these once (≈5 minutes):

1. **Cursor Team Kit** — In Agent chat: `/add-plugin cursor-team-kit`
   (Also cached locally at `~/.cursor/plugins/local/cursor-team-kit`.)

2. **Superpowers** — Confirm enabled in Customize → Plugins (`/add-plugin superpowers` if missing).

3. **Bugbot** — https://cursor.com/dashboard
   - Integrations → Connect GitHub
   - Bugbot tab → Enable `Wei-Shaw/sub2api`
   - Review rules are in `.cursor/BUGBOT.md` (already in repo).

4. **Snyk (optional)** — Marketplace → search **Snyk** → Install → set `SNYK_TOKEN` from https://app.snyk.io/account

## Daily workflow (copy to Agent)

```
用 Superpowers：brainstorm → writing-plans → subagent-driven-development，
每个 task 后 verification-before-completion。
改完跑 check-compiler-errors / sub2api-check-compiler-errors。
push 前 /review-bugbot（payment 加 /review-security）。
push 后 ci-watcher，红了就 fix-ci。
```
