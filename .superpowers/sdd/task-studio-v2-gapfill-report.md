# Studio V2 最小入口查缺补漏报告

**工作树：** `D:\sub2api-trunk\.worktrees\console-unification`  
**分支：** `codex/wujie-console-unification-20260717`  
**实现提交：** `323aa74f0`（新增入口）  
**审查修复提交：** `3773e59fc`（原生链接与 origin 校验）  
**日期：** 2026-07-18  
**状态：** **待复核 / 部分门禁通过**

## 结论

QCanvas 现有运行实例已经支持“只填写项目名称 → 创建空 Flow → 携带真实 `projectId`、`ownerId`、`flowId` 进入 Studio V2”。本仓原缺口不是项目创建实现损坏，而是员工前端没有任何可达入口。

本次最小修复没有复制 QCanvas 项目模型、没有新增权限或 feature flag，也没有改动冻结的员工五项顶栏；仅在员工仪表盘快捷操作中增加始终可见的 Studio V2 原生链接。默认目标为当前实测 QCanvas `http://127.0.0.1:5174/projects`，可由 `VITE_QCANVAS_BASE_URL` 覆盖。

## 变更

| 文件 | 变更 |
|---|---|
| `frontend/src/utils/qcanvas.ts` | 生成项目页 URL；空配置使用本机默认值；只接受无凭据、无路径/查询/片段的 HTTP(S) origin。 |
| `frontend/src/components/user/dashboard/UserDashboardQuickActions.vue` | 增加无额外权限前置的 Studio V2 原生链接；使用 `_blank` 与 `noopener noreferrer`。 |
| `frontend/src/components/user/dashboard/__tests__/UserDashboardQuickActions.studio.spec.ts` | 覆盖默认/覆盖地址、非法地址、无批量生图权限时仍显示入口、链接安全属性。 |
| `frontend/src/i18n/locales/{zh,en}/dashboard.ts` | 增加入口标题和说明。 |

员工/管理员导航合同未改；管理员资产交接 `/asset-handoff` 及其一次性 ticket 安全边界未改。

## 验证证据

| 证据 | 结果 |
|---|---|
| TDD RED | 定向测试先因 `@/utils/qcanvas` 不存在而失败。 |
| Studio + 导航定向回归 | **PASS：3 files / 20 tests**。 |
| ESLint | **PASS，exit 0**。 |
| `vue-tsc --noEmit` | **PASS，exit 0**。 |
| 全量 Vitest | **PASS：191 files / 1134 tests，exit 0**。 |
| 生产 build | **PASS，exit 0**；仅有既存 chunk/Browserslist 警告。 |
| 独立只读审查 | 员工五项顶栏未改、未引入权限或项目模型；URL 子路径静默丢弃和脚本弹窗静默失败已在 `3773e59fc` 修复。 |
| Docker build | **PASS**：`sub2api:wujie-delivery-3773e59fc`；使用 `git archive` + WSL ext4 临时上下文，仓库 Dockerfile 未改。 |
| Docker 8080 | **PASS**：应用、PostgreSQL 18、Redis 均 healthy；应用绑定 `127.0.0.1:8080`。 |
| 中台 Chrome | 登录落地 `/admin/console/overview`；`/dashboard` 真实渲染 Studio V2 链接，目标为 `http://127.0.0.1:5174/projects`。 |
| QCanvas Chrome | 只填项目名“Studio V2 交付验收”后进入 Studio V2；URL 含真实 `projectId`、`ownerId`、`flowId`，页面显示“Studio V2 画布”“创建第一个节点”“已同步”，控制台无错误。 |

## Docker 昨夜启动问题

服务代码没有持续崩溃。阻塞发生在构建阶段：Dockerfile 顶部 syntax frontend 元数据拉取/镜像源受限，加上 Windows 挂载目录作为原生构建上下文较慢。改用不修改仓库的临时 Dockerfile（仅在 WSL 临时目录移除 syntax 行）和 ext4 构建上下文后，两次镜像均成功。2026-07-18 中午最终验收栈已 healthy。

## 明确非声明

- 浏览器三角色与 1440/390 视口矩阵：**NOT VERIFIED**。
- 生产 PostgreSQL 或生产迁移：**NOT VERIFIED**；本次仅为新建的本地隔离 PostgreSQL 18。
- Linux dirfd 竞态：**NOT VERIFIED**。
- 真实 Seedance/任何真实或付费 Provider：**NOT VERIFIED，未调用**。
- 跨应用单点登录：未实现；中台入口负责打开 QCanvas 项目页，QCanvas 身份会话仍由 QCanvas 自己管理。
- 产品生产 READY：**不声明**。当前状态仅为 **待复核 / 部分门禁通过**。

## 风险与回滚

- 默认 `5174` 来自本机真实运行证据；其他环境必须提供纯 origin 的 `VITE_QCANVAS_BASE_URL`。
- QCanvas 未运行或用户未登录时，会由目标应用展示其连接/登录状态；本仓不复制其鉴权。
- 代码回滚：依次 `git revert 3773e59fc`、`git revert 323aa74f0`，不得使用 reset/clean/rebase。
- 本地验收栈可停止并恢复旧 `wujie-acceptance-7ef-sub2api` 容器；未 push、未部署、未调用 Provider。
