# 当前目标：Sub2API 本地分支收口

日期：2026-07-30
主分支：`main`
执行目录：`D:\sub2api-trunk`
整合代码 HEAD：`74daac1e7408ae24db46885e3c9280a3917c2603`
门禁审查提交：`8023abdd16b1d9ba3e163344fb848288f41a133d`
状态：**生产程序已切换 / REAL_E2E_PENDING；双 Key、价格和真实一图一视频闭环待复核**

## 2026-08-02 23:08 CST 生产切换事实（最新）

- 完整备份已完成：`/home/ubuntu/wujie/backups/production-readiness-20260802-20260802T141843Z`，包含两套 PostgreSQL dump、11 个数据卷、compose/Caddyfile、镜像归档与恢复脚本；SHA256/gzip 检查通过。
- Sub2API 已先在 `127.0.0.1:18081` canary 验证，再替换正式服务。线上运行 SHA 为 `97d29909a4313a31c56aabdd3b2a6242f63ed53d`；无 Key 的 `/v1/key-context`、`/v1/*` 与 `/sub2api/*` 均返回 JSON 401，正式 video worker 已恢复启用。
- QCanvas 已完成仅加列迁移、green 启动和 Caddy 公网切流；线上前后端 SHA 为 `b08fb288015d0a732b6d34fd1d94abd29aba941a`，真实 Chromium 资源请求全部 200、控制台 0 error。旧容器与全部回滚证据保留。
- 新付费调用仍为 0。剩余目标：用户经 SSH 隧道登录 Sub2 管理端，核验同员工双 Key、模型可用性与准确价格；仅当一图一视频总预估不超过 ¥30 时，按图片后视频顺序各提交一次并证明 taskId、upstream ID、终态、资产、刷新恢复、History/Library、复用、usage/cost、结算和归档。
- 任一 401/403/409/5xx、空目录、无 taskId、成功无资产、刷新丢失、scope 缺失或结算不一致均判失败；超时或失败不重试付费任务，不伪造 READY。

## 2026-08-02 生产收口目标（最新覆盖）

- 发布分支：`codex/sub2api-production-readiness-20260802`，已推送到 `fork`；未推送 Sub2API 上游 `origin`，未触碰任何 `main`。
- 代码实现提交：`66fe36c16`；真相源随后有一笔仅文档提交，发布前必须以 `git rev-parse HEAD` 作为精确 SHA。
- 目标：完成可追溯构建、部署前验证和真实一图一视频闭环；任一门禁失败即停止并保留回滚证据。
- 已落地：Sub2 key-context 合约、`/sub2api/*` embed bypass、专用连接 advisory lock、显式 delivery roots/commit、构建身份 manifest。
- 已验证：key-context/route/auth/unit 定向测试、migration 定向测试、delivery preflight 离线契约测试、`go test ./... -count=1`、前端 test/typecheck/build 与 `-tags embed` Docker 构建；并发 advisory-lock 集成测试因本地 harness 判断 Docker 不可用而明确跳过，未伪报通过。
- 已完成：服务器盘点、完整备份、Sub2 canary/正式切换、QCanvas green/公网切换和未登录浏览器门禁。
- 旧只读核验中的 404、SPA HTML 与旧 build identity 已由正式发布修复；部署后的运行证据见上方最新事实与发布证据目录。
- SSH 写通道已通过用户完成密码认证后的 ControlMaster 恢复；Agent 不保存密码。Sub2 管理密码和员工 Key 仍仅由用户在本机 SSH 隧道页面输入。
- 回滚原则：旧容器、compose/Caddy、镜像、卷、dump 和发布目录在用户另行授权清理前全部保留；线上失败只恢复旧镜像/compose。

## 目标

- 以 `codex/hc-key-vault-ux-20260729@d605aa51d7` 为完整功能源。
- 保留原 `main@8c2d2a7ed7` 的审计与本地制品卫生提交。
- 保留 `fix/staff-console-hotfix-20260726@4e0290fb3` 的部署 checkpoint。
- 将本地 9 个分支、5 个 worktree 收口为仅保留 `main` 和主工作树。
- 不 fetch、push、部署，不修改远端分支，不调用真实付费供应商。

## 已确认事实

- `fd2aad9bc` 已将 Key Vault 完整功能线合入整合分支。
- `74daac1e7` 已将 staff 部署 checkpoint 合入整合分支。
- `fix/runtime-billing-blockers-20260725@cadc1b0d8` 不直接 merge。
- 现有 `b2378efa1` 已语义覆盖余额原子更新、API Key quota、独立 reset 与 nil billing cache 防护；定向测试和独立复核均未发现缺口。
- 两个恢复标签已经创建：
  - `archive/pre-consolidation-main-20260730`
  - `archive/runtime-billing-blockers-20260725`

## 验收门禁

- 后端计费定向测试：PASS。
- `go test ./... -count=1`：PASS。
- `go build ./...`：PASS。
- 前端全量 Vitest：PASS。
- `vue-tsc --noEmit`：PASS。
- ESLint 只读检查：PASS。
- `vue-tsc -b` 与 Vite production build：PASS。
- `git diff --check`：PASS。

## 硬边界

- 不读取、打印、提交或截图任何 key、token、cookie、生产连接串。
- 不触发真实图片或视频任务。
- 不执行 fetch、push、部署、reset、clean 或 rebase。
- 任一最终 Git 验收失败时停止删除 refs，并在审查包中标记“已阻塞”。

## 完成结果

1. `main` 已以 `--ff-only` 前移到已验证整合结果。
2. 五个旧/临时关联 worktree 已显式移除。
3. 八个冗余本地分支和临时整合分支已删除。
4. 最终仅剩 `main` 与 `D:\sub2api-trunk` 主工作树。
5. 两个 archive 标签均可解析到原始 SHA，主工作树干净。
