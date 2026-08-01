# 当前目标：Sub2API 本地分支收口

日期：2026-07-30
主分支：`main`
执行目录：`D:\sub2api-trunk`
整合代码 HEAD：`74daac1e7408ae24db46885e3c9280a3917c2603`
门禁审查提交：`8023abdd16b1d9ba3e163344fb848288f41a133d`
状态：**本地历史收口已被本轮生产收口目标覆盖；当前待复核 / 已阻塞（BLOCKED_AUTH）**

## 2026-08-02 生产收口目标（最新覆盖）

- 发布分支：`codex/sub2api-production-readiness-20260802`，已推送到 `fork`；未推送 Sub2API 上游 `origin`，未触碰任何 `main`。
- 代码实现提交：`66fe36c16`；真相源随后有一笔仅文档提交，发布前必须以 `git rev-parse HEAD` 作为精确 SHA。
- 目标：完成可追溯构建、部署前验证和真实一图一视频闭环；任一门禁失败即停止并保留回滚证据。
- 已落地：Sub2 key-context 合约、`/sub2api/*` embed bypass、专用连接 advisory lock、显式 delivery roots/commit、构建身份 manifest。
- 已验证：key-context/route/auth/unit 定向测试、migration 定向测试、delivery preflight 离线契约测试、`go test ./... -count=1`；`-tags embed` 因缺少生成的 `internal/web/dist` 阻塞。
- 未完成：服务器只读盘点、完整备份、Sub2 canary/正式切换、QCanvas green 切换、浏览器闭环、用户 SSH 隧道输入密码、真实图片/视频付费验收。
- 阻塞证据：严格 host key 校验可通过；`ssh ... ubuntu@114.132.50.149` 返回 `Permission denied (publickey,password)`。禁止绕过验证或读取秘密。
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
