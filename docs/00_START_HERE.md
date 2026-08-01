# 无界 AI 管理中台 · START HERE

更新时间：2026-07-18 Asia/Shanghai

## 2026-08-02 生产收口事实（最新）

> 当前状态：**待复核 / 已阻塞（BLOCKED_AUTH）**。本分支只证明代码与离线门禁；服务器活动栈、备份、canary、切换、线上回归和真实供应商闭环尚未证明。

- 发布分支：`codex/sub2api-production-readiness-20260802`
- 当前实现提交：`2c109008ef4d3ea9732b7fd903d916483cbeb828`；本轮 release-hardening 提交将在本文件之后生成，以 `git rev-parse HEAD` 为准。
- `GET /v1/key-context` 只返回 `object`、稳定非敏感 `subject_id`、`group_id`、`model_kinds`；不返回原始 Key、余额、供应商凭据或敏感账号信息。
- embedded frontend 已将 `/sub2api/` 纳入 API bypass；无 Key 或无效 Key 必须返回 JSON 401，不得返回 SPA HTML。
- PostgreSQL migration advisory lock 使用同一个专用 `sql.Conn` 获取与释放，并有并发 session 集成测试。
- delivery preflight 要求显式 `RepoRoot`、`ReleaseRoot`、`ExpectedBranch`、`RequiredProductCommit`；构建身份由 `VERSION/COMMIT/DATE` 写入 manifest。
- 已通过：key-context/route/auth/unit 定向测试、migration 定向测试、delivery preflight 离线契约测试、`go test ./... -count=1`。`-tags embed` 编译因该 worktree 缺少生成的 `internal/web/dist` 被阻塞；前端全量门禁、Docker canary 尚未在当前环境重新证明。
- 服务器严格 host key 校验通过，但 `ubuntu@114.132.50.149` 返回 `Permission denied (publickey,password)`；在 SSH 恢复前禁止部署、备份、密码操作或付费任务。
- 公网 HTTP 管理入口是已接受的长期安全例外；本轮密码操作只允许用户自行通过 SSH 隧道完成，Agent 不读取、保存、打印或输入秘密。

## 一句话状态

三套前端合一分支 `codex/wujie-console-unification-20260717` 当前产品代码 HEAD 为 **`7ef02f628`**；导航功能基线为 `7f4c15ca1`，文档收口为 `2898b8419`，首轮 gap-fill 为 `4c6111502`。2026-07-18 已在隔离本地栈完成当前产品代码镜像构建、loopback `:8080`、真实 disposable PostgreSQL migrations 182–186、管理员/员工浏览器与零费用 mock 任务预览下载。技术管理员独立账号、1440/390 双视口、Linux dirfd、真实 Seedance仍未验证。整体为**待复核 / 部分门禁通过**。**禁止**写成生产 READY、商业交付完成或已 push。

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
| 开卡 P0 修复 | `7ef02f628b3a6d6e35b43d81d523ed56e24d7615` | 新卡绑定员工可用组；覆盖资格过滤与异步加载竞态；独立复核无 Critical/Important |
| 后端全量测试 | `go test ./... -count=1`（TMP/TEMP/GOCACHE→worktree `.cache`） | exit **0** |
| 后端 vet | `go vet ./...` | exit **0** |
| 前端门禁 | pnpm 9.15.9 lint / typecheck / test:run（190/1124）/ build | 全部 exit **0** |
| git whitespace | `git diff --check` | exit **0** |
| Docker build（历史 Task 6 / `7f4c15ca1`） | `wujie-console-unification:task6`（移出 `.cache` 后） | exit **0**；~124MB |
| Docker smoke（历史 Task 6 / `7f4c15ca1`） | `GET /` → **200**；`GET /health` → **200**；host `:3000` 监听数 **0** | **历史已验证** |
| Fresh Docker（产品代码 `7ef02f628`） | image `sha256:f1a51cf…c7876e4`；隔离容器 healthy；`GET /health` → **200** | **PASSED** |
| 品牌 dist | `Sub2API`×3 internal；Wei-Shaw/weishaw/GitHub×0；lowercase allowlisted | **PASSED with notes** |
| 浏览器角色 | disposable admin + employee：合规 remap、管理员五项、员工五项无“更多”、开卡/首次改密/mock 任务 | **PARTIAL PASS**；技术管理员独立账号与 1440/390 未验证 |
| 真实 PG migrations 182–186 | fresh disposable PostgreSQL 18；`schema_migrations` 记录 182–186 | **PASSED（隔离本地）** |
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
- 功能实现祖先含 `7cf7404f` 等；产品代码证据截止 `7ef02f628`，仓库最新文档提交以 `git rev-parse HEAD` 为准。

## 验收纪律

必须同时看到以下事实才可把 canonical 本机入口升级为可演示：

1. `http://127.0.0.1:8080` HTTP 200；
2. raw title / 产品名命中无界品牌；
3. 运行镜像为约定的无界本地构建（非上游 latest）；
4. host `:3000` 无监听；
5. 完整管理面、管理员开卡、员工 Key / usage / 仿真任务路径可见；
6. 截图和日志来自同一真实运行实例。

**当前：**隔离本地镜像已满足 (1)(2)(3)(5)，并有同实例员工任务详情截图与 SVG 下载证据；技术管理员独立账号、1440/390 双视口仍缺，故 (6) 仅部分满足。不得把 mock 闭环外推成真实 Provider 或生产验收。

## 安全与回滚

- 禁止 push、生产库 migration 和删除备份目录。
- 本轮未授权真实 Provider；隔离 mock usage ≠ 真实调用。
- `.env` 与上游密钥保持未跟踪、未提交。
- 回滚：对产品提交使用 `git revert <sha>`（见审查包）；禁止 reset/clean/rebase。

## 下一步

1. 交付前保留当前隔离栈或按 SOP 重新生成本地凭据；不得打印或复用验收凭据。
2. 补技术管理员独立账号和 1440/390 双视口浏览器证据；如运行宿主变化，重新做同镜像 `/health` 与 mock 任务冒烟。
3. 状态保持「待复核 / 部分门禁通过」；仍不得写生产 READY，除非另授并取得真实 Seedance / Linux dirfd 证据。
4. 禁止再次触发真实付费；历史 `realCallExecuted=2` 封口仍有效。
5. 禁止 push。
