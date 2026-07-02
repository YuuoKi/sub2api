# A2 mock/dev 复核阻塞包

状态：已阻塞

## 结论

A2 本轮不能安全重跑。A2-Sub2API 的新鲜 G3 复核需要启动 dev compose 环境；用户明确要求需要 Docker 时必须使用 WSL Ubuntu-24.04，不使用 Docker Desktop 桌面端。当前 `wsl.exe -l -v` 返回失败，未确认可用 Ubuntu-24.04。现有 G3 harness 还会读取 deploy 配置中的管理员登录凭据，本轮红线禁止读取敏感配置内容。

A2-QCanvas 也未执行写操作，因为 ABC progress 仍显示 RUNNING/阻塞记录。按超级循环规则，本 phase 记为已阻塞并跳到下一独立 phase：B1-Sub2API。

## 执行目录

- Sub2API：`D:/sub2api-trunk`
- QCanvas：只读 `D:/Codex创业任务/QCanvas（无界版）/QCanvas`
- 北极星 progress：`D:/Codex创业任务/QCanvas（无界版）/北极星/codex任务书/超级循环_progress_20260703.md`

## 已重读真相源

- 北极星锚文件 `#current-state/#roadmap/#guardrails`
- Sub2API `00_START_HERE.md` 与 `docs/reviews/LATEST_REVIEW_PACKAGE.html`
- QCanvas `docs/reviews/LATEST_REVIEW_PACKAGE.html` 与 `docs/LATEST_REVIEW_PACKAGE.md`
- 现有 G3 包：`_review/capture-arming-D2-20260702-G3/SUMMARY.md`
- progress log 最新行：A1 已生成跨仓契约快照并 commit `cce0d5b4`

## Gate 证据

### A2-Sub2API

触发条件：任务书要求“重跑 G3 受控 dev（chat + mock video + SQL + Admin probe）”。

阻塞原因：

- `wsl.exe -l -v` 退出码为 1，输出为 Windows/WSL 错误文本，未确认存在可用 `Ubuntu-24.04` distro。
- 用户红线要求 A2 需要 Docker 时使用 WSL Ubuntu-24.04；因此不能退回 Docker Desktop 路径。
- 现有 `_review/capture-arming-D2-20260702-G3/run_g3_verify.ps1` 会读取 deploy 配置中的管理员登录凭据；本轮不能读取敏感配置内容，也不能读取本地临时凭据文件。

未执行：

- 未启动 Docker/compose。
- 未读取敏感配置内容。
- 未执行真实 provider。
- 未 push。

### A2-QCanvas

触发条件：A2-QCanvas 需重跑 D2 web/api/agents 包内测试并可能写审查包。

阻塞原因：

- QCanvas ABC progress 仍有 RUNNING/阻塞记录。
- 超级循环规则要求 QCanvas 写操作遇 ABC RUNNING 时跳过，不双写。

未执行：

- 未写 QCanvas。
- 未跑 QCanvas 包内测试。
- 未启动浏览器验证。

## 现有历史证据

历史 G3 包仍记录一轮受控 dev 成功证据：

- `g3_http_result.json`：chat ok、mock video succeeded、result URL present。
- `g3_sql_recent_count.txt`：`2`。
- `g3_sql_suspicious_rows.txt`：`0`。
- `g3_admin_result.json`：`is_live=true`、`captured_today=2`、`sample_count=2`。

这些证据可作为历史依据，但不能替代 A2 本轮“新鲜复核”。

## 后续解阻条件

任选其一：

- 提供可用 WSL Ubuntu-24.04，并允许在该 distro 内执行 compose/dev harness。
- 提供不读取敏感配置内容的 A2 mock/dev harness，例如全部使用一次性测试账号与一次性环境变量注入，并保证日志不输出敏感认证串。
- 明确授权使用另一条本地 Docker 路径；授权必须单独写明边界、清理和禁止输出内容。

## 回滚方案

本阻塞包只新增文档，不改业务代码。若后续 A2 重新执行，可保留本包作为历史阻塞证据，或在新 commit 中追加成功复核包。

## 可复制后续提示词

```text
继续 A2-Sub2API 解阻：先确认 WSL Ubuntu-24.04 可用，且不要读取/打印任何敏感配置内容。目标是在 D:\sub2api-trunk 重跑 G3 受控 dev/mock：chat + mock video + SQL + Admin probe，输出 _review/G3复核_A2_20260703 成功证据；若仍需要读取敏感配置或 WSL 不可用，保持“已阻塞”并跳到下一独立 phase。
```
