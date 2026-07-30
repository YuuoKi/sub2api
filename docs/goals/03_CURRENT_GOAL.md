# 当前目标：Sub2API 本地分支收口

日期：2026-07-30
整合分支：`codex/consolidate-sub2api-20260730`
整合目录：`D:\sub2api-trunk\.worktrees\consolidate-sub2api-20260730`
整合代码 HEAD：`74daac1e7408ae24db46885e3c9280a3917c2603`
状态：**本地合流与验证通过，待 main 快进和冗余 refs 清理**

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

## 完成条件

1. `main` 以 `--ff-only` 前移到已验证整合结果。
2. 显式移除四个旧关联 worktree。
3. 删除八个冗余本地分支和临时整合分支。
4. 最终仅剩 `main` 与 `D:\sub2api-trunk` 主工作树。
5. 两个 archive 标签可解析到原始 SHA，主工作树干净。
