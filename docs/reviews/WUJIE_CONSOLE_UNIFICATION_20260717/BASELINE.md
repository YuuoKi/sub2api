# 三套前端合一基线

日期：2026-07-17
集成工作树：`D:\sub2api-trunk\.worktrees\console-unification`
集成分支：`codex/wujie-console-unification-20260717`
集成 HEAD：`ab96e5228f5dce70ecc094c6fcdd8a2c1d08ab47`
状态：已冻结

## 工作树快照

| 来源 | 工作树 | 分支与 HEAD | 脏文件 / 约束 | 当前候选完成度 |
| --- | --- | --- | --- | --- |
| main | `D:\Codex创业任务\企业 API 管理后台项目\02_source\sub2api` | `main@ab96e5228f5dce70ecc094c6fcdd8a2c1d08ab47` | 用户修改：`docs/00_START_HERE.md`、`docs/legal/admin-compliance.zh.md`；未触碰。 | 整合底座已确认；三套合一尚未开始，待复核。 |
| K3 | `D:\sub2api-trunk` | `codex/k3-apple-ui-experiment-20260717@16351e1a37dbc094264702f2a997d066b9bc801d` | 用户修改：`frontend/src/views/admin/video/VideoDashboardView.vue`、`frontend/src/views/admin/video/videoUtils.ts`；未跟踪：`frontend/scan-out.txt`、`frontend/scripts-scan-zero-ref.cjs`。四项均只读保留，不纳入本任务。 | K3 视觉层候选，未并入，待复核。 |
| Kling Real / Console v2 | `D:\sub2api-trunk\.worktrees\kling-real` | `feature/kling-real-integration@b918b91e66b14375cdbc767e390f7a4a923b5c81` | 工作树干净；不整体合并旧后端或旧迁移。 | Console v2 业务表面候选，未并入，待复核。 |
| 集成 | `D:\sub2api-trunk\.worktrees\console-unification` | `codex/wujie-console-unification-20260717@ab96e5228f5dce70ecc094c6fcdd8a2c1d08ab47` | 本任务只允许目标、计划、基线文档；`.superpowers/**`、缓存、`dist`、`node_modules` 排除。 | Task 1 已冻结；整体合一进度为 0/7，待后续实施验证。 |

## 已确认的验证基线

| 检查 | 命令 / 环境 | 结果 |
| --- | --- | --- |
| 前端依赖 | `corepack pnpm install --frozen-lockfile`，pnpm `9.15.9` | 退出 0。系统默认 pnpm `11.1.2` 因 `overrides` / lockfile 配置差异被拒绝，不作为本基线安装方式。 |
| 前端门禁 | lint、typecheck、全量 Vitest、生产 build | 全部退出 0；其中全量 Vitest 为 159 files / 978 tests，0 failure（63.58s）。 |
| 后端静态检查 | `go vet ./...` | 退出 0。 |
| 后端全量测试 | 固定 `TMP`、`TEMP`、`GOCACHE` 至工作树 `.cache/**` 后执行 `go test ./... -count=1` | 退出 0。 |

## 已知环境约束

未固定临时目录时，Windows 中文用户临时目录路径解析会导致仅 `TestResolvePageImagePath` 失败。将 `TMP` / `TEMP` 固定到工作树 `.cache/test-tmp` 后，同一测试连续 3 次通过，且 Go 全量测试通过；为环境路径问题，业务代码未改。

## 非授权动作与后续口径

本基线不授权真实/付费 Provider、真实支付、生产数据、公网部署、push 或不可逆删除。Kling 真实调用保持关闭；任何后续“可用”结论必须补充对应的自动化测试、受控浏览器路径和审查证据。
