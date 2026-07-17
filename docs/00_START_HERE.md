# 无界 AI 管理中台 · START HERE

更新时间：2026-07-18 Asia/Shanghai

## 一句话状态

三套前端合一分支 `codex/wujie-console-unification-20260717` 当前产品 HEAD 为 **`4c6111502`**（查缺补漏）；导航功能基线为 `7f4c15ca1`，文档收口为 `2898b8419`。当前 HEAD 的前端全量门禁已通过；Docker loopback `:8080` HTTP 200 仅有 Task 6 对旧 HEAD 的历史证据，`4c6111502` fresh Docker build 因 WSL `CreateVm/E_INVALIDARG` **NOT VERIFIED**。三角色浏览器、真实 PostgreSQL migrations 182–186、Linux dirfd、真实 Seedance也均未验证。整体为**待复核 / 部分门禁通过**。**禁止**写成生产 READY、商业交付完成或已 push。

审查包：[`docs/reviews/LATEST_REVIEW_PACKAGE.html`](reviews/LATEST_REVIEW_PACKAGE.html)。

## 唯一前端定义

- 唯一前端是“完整管理后台 + 无界品牌”。用户可见产品名为 **`无界 · 企业 AI 中台`**（Task 4 起；勿再写回「管理中台」旧 title 口径）。
- 禁止启用 `video_gateway_demo`、simple/demo 或 video-only 全隐藏模式。
- 管理员顶层五入口：总览、密钥库、成员与开卡、任务记录、系统（嵌套「运行与配置」「高级与历史」）。
- 员工顶层精确五入口：我的工作台、创建任务、任务记录、我的密钥、我的花费；生产侧栏无「更多」。
- 员工仿真视频走 JWT `/api/v1/user/video/simulation/*` + 页面 `/video/*`；不走真实 Seedance。
- 当前后端角色仅 `admin` / `user`（员工）；`admin` 兼任管理者开卡，不扩第三角色。

源码证据：

- `frontend/src/components/layout/roleAwareNavigation.ts` 锁定顶层路径契约。
- `frontend/src/components/layout/AppSidebar.vue` 生产接线不含 `includeMoreGroup`。
- `frontend/src/__tests__/brand-scan.spec.ts` / Task 6 dist 扫描：用户可见 Wei-Shaw/GitHub 为零；`Sub2API` 仅内部 upstream 常量。

## 唯一本机入口

只使用：

- 构建验收镜像（Task 6）：`wujie-console-unification:task6`（loopback 冒烟用）
- 日常 SOP 镜像名仍可按 [`../deploy/WUJIE_SINGLE_ENTRY_SOP.md`](../deploy/WUJIE_SINGLE_ENTRY_SOP.md) 的 `wujie-sub2api:local`
- 地址：`http://127.0.0.1:8080`
- host 映射：`127.0.0.1:8080:8080`
- 容器端口：`SERVER_PORT=8080`
- 运行模式：`RUN_MODE=standard`

禁止：

- 用 Vite `http://127.0.0.1:3000` 验收；
- 用 `weishaw/sub2api:latest` 冒充无界构建；
- 用公网绑定、其他 host 端口、one-off 诊断容器冒充唯一入口；
- 打印本地环境文件、密钥、token、cookie 或连接串。

## 本轮合一证据（Tasks 2E–6，2026-07-18）

| 项 | 证据 | 状态 |
|---|---|---|
| HEAD | `7f4c15ca1be3eed730cec91188edb2ccdc77ccac` | 已记录 |
| Gap-fill HEAD | `4c6111502eb59e83e2c5d750a2a724aaf1f70b55` | `.dockerignore`、管理员登录落地、合规回归；前端 lint/typecheck/test/build exit 0 |
| 后端全量测试 | `go test ./... -count=1`（TMP/TEMP/GOCACHE→worktree `.cache`） | exit **0** |
| 后端 vet | `go vet ./...` | exit **0** |
| 前端门禁 | pnpm 9.15.9 lint / typecheck / test:run（190/1124）/ build | 全部 exit **0** |
| git whitespace | `git diff --check` | exit **0** |
| Docker build（历史 Task 6 / `7f4c15ca1`） | `wujie-console-unification:task6`（移出 `.cache` 后） | exit **0**；~124MB |
| Docker smoke（历史 Task 6 / `7f4c15ca1`） | `GET /` → **200**；`GET /health` → **200**；host `:3000` 监听数 **0** | **历史已验证** |
| Fresh Docker（当前 `4c6111502`） | WSL `CreateVm/E_INVALIDARG`，未进入 build context | **NOT VERIFIED** |
| 品牌 dist | `Sub2API`×3 internal；Wei-Shaw/weishaw/GitHub×0；lowercase allowlisted | **PASSED with notes** |
| 三角色浏览器 / 1440 / 390 | 无安全 fixture 凭据 | **SKIPPED** |
| 真实 PG migrations 182–186 | 未对受控库执行 | **NOT VERIFIED** |
| Linux dirfd | Windows host，无内核证据 | **NOT VERIFIED** |
| 真实 Seedance | 禁止 / 未尝试 | **NOT VERIFIED** |

Task 2E 交付（代码层）：内部 mock 视频仿真（migration 186、JWT 员工路由、隔离 worker、SVG 结果、零计费）；对抗 harden 提交 `07ffeff05`。单测通过 ≠ 生产可用。

## 三套来源（冻结口径）

| 来源 | 用途 | 冻结引用 |
|---|---|---|
| main | 后端 / 迁移 / 8080 底座 | `ab96e5228` |
| Console v2 / Kling Real | 业务骨架（Task 3） | `feature/kling-real-integration@b918b91e6` |
| Kimi K3 | 视觉壳层（Task 4） | `codex/k3-apple-ui-experiment-20260717@16351e1a3` |

详细基线见 [`reviews/WUJIE_CONSOLE_UNIFICATION_20260717/BASELINE.md`](reviews/WUJIE_CONSOLE_UNIFICATION_20260717/BASELINE.md)。

## 历史 canonical 证据（2026-07-16 及更早，勿与本轮 Task 6 混写）

以下为**更早**主线/本机入口证据，**不是** Task 6 合一镜像的三角色浏览器证明：

- 品牌实现与部署契约、完整管理面五段路径、隔离开卡/usage smoke 等见当时运行记录。
- Gemini 图片 / Seedance 2.0 各一次 tiny_real（`realCallExecuted=2`）属历史授权封口；**本轮合一未再触发真实付费**，且不得自动重试。
- 功能实现祖先含 `7cf7404f` 等；仓库当前合一 HEAD 以 `git rev-parse HEAD` 为准（现为 `4c6111502`）。

## 验收纪律

必须同时看到以下事实才可把 canonical 本机入口升级为可演示：

1. `http://127.0.0.1:8080` HTTP 200；
2. raw title / 产品名命中无界品牌；
3. 运行镜像为约定的无界本地构建（非上游 latest）；
4. host `:3000` 无监听；
5. 完整管理面、管理员开卡、员工 Key / usage / 仿真任务路径可见；
6. 截图和日志来自同一真实运行实例。

**当前：**(1)(4) 在 Task 6 镜像上已满足；(2) HTML shell 无 Sub2API；(5)(6) 浏览器三角色仍 **SKIPPED**。不得用源码字符串或单元测试冒充浏览器验收。

## 安全与回滚

- 禁止 push、生产库 migration 和删除备份目录。
- 本轮未授权真实 Provider；隔离 mock usage ≠ 真实调用。
- `.env` 与上游密钥保持未跟踪、未提交。
- 回滚：对产品提交使用 `git revert <sha>`（见审查包）；禁止 reset/clean/rebase。

## 下一步

1. 重启/修复 WSL 后先运行 `deploy/wujie-delivery-preflight.ps1 Check`，再运行 `Build`；任一步失败都不得启动旧镜像冒充当前 HEAD。
2. 在具备受控凭据与 PostgreSQL 时，完成浏览器三角色冒烟 + migrations 182–186（提示词见审查包第 10 节）。
3. 状态至多升到「内部可用 / 真实供应商待复核」——仍不得写生产 READY，除非另授真实 Seedance / dirfd 证据。
4. 禁止再次触发真实付费；历史 `realCallExecuted=2` 封口仍有效。
5. 禁止 push。
