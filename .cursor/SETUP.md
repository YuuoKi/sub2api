# Cursor quality workflow — one-time setup

**Automation is on at two levels:**
- **All projects (this machine):** `~/.cursor/hooks.json` — Agent 结束时自动 quality gate 提醒
- **Sub2API only:** `.cursor/hooks/project-quality-gate.ps1` — 精确的 make test-backend / test-frontend 提示

New project template: `~/.cursor/templates/project-quality/` (copy `.cursor/` into repo root).

## Trigger 设计说明(2026-07 全量改造)

对齐 [官方 Hooks 文档](https://cursor.com/docs/hooks) 与 [社区论坛共识](https://forum.cursor.com/t/guide-thin-self-review-hooks-for-cursor-the-model-is-the-auditor-windows-linux/162875)（afterFileEdit 记编辑 → stop 单次提醒）:

| 事件 | 脚本 | 行为 |
|------|------|------|
| `afterFileEdit` (matcher: `Write`) | `~/.cursor/hooks/after-file-edit-marker.ps1` | 本轮对话编辑了源码才写 session 标记 |
| `stop` | `~/.cursor/hooks/agent-stop-quality.ps1` | 有 session 标记 + 有未提交源码 diff → 发一次 `followup_message` |
| `beforeShellExecution` (`git push`) | 项目级 `secret-scan.ps1` 硬拦截；用户级 reminder 自动让位 | push 前 secret 扫描 |

**stop 钩子的刹车逻辑（多层，与论坛推荐一致）:**

1. `status=aborted` 或 `loop_count > 0` → 直接 `{}`（不检查、不提醒）
2. 本轮对话没有 `afterFileEdit` 标记 → `{}`（纯讨论 / 只读，即使工作区有旧脏文件也不 nag）
3. diff 为空（已提交/还原）→ 清除标记和状态 → `{}`
4. 同一会话 + 同一份未处理 diff 签名 → `{}`（去重，不每轮重复）
5. 以上都通过 → 发一次提醒，清除 session 标记（one-shot brake）

状态文件在 `~/.cursor/hooks/.state/`（按 `conversation_id` 隔离，不进 git）。

`.cursor/hooks/project-quality-gate.ps1` 只匹配真实源码扩展名(`.go` / `.vue` / `.ts(x)` / `.js(x)`)。

**Windows 注意:** stop 的 `followup_message` 在部分版本有 stdout 捕获问题（[论坛已知 bug](https://forum.cursor.com/t/stop-hook-followup-message-not-captured-on-windows-execution-log-shows-despite-valid-json-on-stdout/155078)）；脚本已加 `[Console]::Out.Flush()`。若 Execution Log 显示 `{}` 但手动测试有输出，属 Cursor 侧问题，可在 Settings → Hooks → Execution Log 排查。

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
