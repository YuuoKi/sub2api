# 无界 AI 管理中台 · START HERE

更新时间：2026-07-16 Asia/Shanghai

## 一句话状态

当前 W4 收口基线 `00f98a95` 的代码、无界 embedded 品牌、本地镜像、唯一 loopback `:8080` 部署契约和隔离开卡/usage smoke 为**内部可用**；canonical compose 当前 exit 1、无 HTTP，既有 `BLOCKED:totp-encryption-key` 未解除，所以整体为**已阻塞**，不得标记为**可演示**。

## 唯一前端定义

- 唯一前端是“完整管理后台 + 无界品牌”。页面 title 为 `无界 · 企业 AI 管理中台`。
- 禁止启用 `video_gateway_demo`、simple/demo 或 video-only 全隐藏模式。
- 禁止藏管理页。管理员必须保留用户、账号、分组、兑换码、设置等管理入口；员工保留个人 API Key 与 usage 入口。
- 2C 销售导航只隐藏订阅购买、订单支付、分销返利、促销等消费商业入口，不删除后台路由，也不等于把完整管理后台裁成 demo。
- 当前后端角色仅 `admin` / `user`，其中 `user` 对应员工；`admin` 兼任管理者开卡，不扩第三角色。

源码证据：

- `frontend/src/router/index.ts` 保留完整管理路由。
- `frontend/src/components/layout/AppSidebar.vue` 保留核心管理入口，并只从实际侧栏移除 2C 销售导航。
- `frontend/src/__tests__/integration/brand-admin-surface.spec.ts` 锁定无界品牌、管理路由与禁止 demo 全隐藏。
- `frontend/src/components/layout/__tests__/AppSidebar.spec.ts` 锁定完整管理面可见、2C 销售导航隐藏。

## 唯一本机入口

只使用：

- 镜像：`wujie-sub2api:local`
- 地址：`http://127.0.0.1:8080`
- host 映射：`127.0.0.1:8080:8080`
- 容器端口：`SERVER_PORT=8080`
- 运行模式：`RUN_MODE=standard`

禁止：

- 用 Vite `http://127.0.0.1:3000` 验收；
- 用 `weishaw/sub2api:latest` 冒充无界构建；
- 用公网绑定、其他 host 端口、one-off 诊断容器或历史审查容器冒充唯一入口；
- 打印本地环境文件、展开后的 compose 配置、密钥、token、cookie 或连接串。

实际操作只遵循 [`../deploy/WUJIE_SINGLE_ENTRY_SOP.md`](../deploy/WUJIE_SINGLE_ENTRY_SOP.md)。仓内 compose 已将 host 入口、容器端口与 `RUN_MODE` 硬定；环境文件不能改写 host 端口或切换隐藏模式。

## 当前证据

- 品牌实现：`4dc96e38`。缺失、空白和精确上游站点名统一到无界品牌，自定义站点名保留；embedded raw HTML title 与服务端站点名同源。
- 部署契约：`00f98a95`。唯一 host 发布为 `127.0.0.1:8080:8080`，独立终审 Critical=0、Important=0。
- 本地镜像：`sha256:bc7202509217701388f6877d07222a94b5fbf9bf10d7435b57b8d84f20f1d01b`；embedded binary 无界 title=1、精确上游 title=0。
- 隔离开卡/usage smoke：只显示掩码 key，`usage_delta=1`、`balance_delta=0.00045000`；临时 PostgreSQL/Redis 已终止，无真实 provider 或生产数据。
- canonical runtime：精确镜像下 app 仍 exit 1、HTTP 不可用；未读取 env 值或容器日志，应用已停止，无浏览器五段截图。

## 验收纪律

必须同时看到以下事实才可把 canonical 本机入口升级为可演示：

1. `http://127.0.0.1:8080` HTTP 200；
2. raw title 命中无界品牌；
3. 运行镜像为 `wujie-sub2api:local`；
4. host `:3000` 无监听；
5. 完整管理面、管理员开卡、员工 Key 调用与 usage 可见；
6. 截图和日志来自同一真实运行实例。

当前第 1 项未满足，因此保持 `BLOCKED:totp-encryption-key`。不得用源码字符串、单元测试、隔离 smoke、one-off 容器或空 UI 截图冒充 canonical 浏览器验收。

## 安全与回滚

- 禁止 push、生产库 migration、真实付费出图/视频和删除备份目录。
- 本 CYCLE 未真实付费生成，`realCallExecuted=0`；隔离 loopback mock usage 请求不等于真实 provider 调用。

## Canonical video control plane (G2.R1)

- Administrator UI: `/admin/video/providers` -> `/admin/video/tasks` -> `/admin/video/system-check`.
- Provider creation is Seedance-only and requires an explicit employee group binding. Stored secrets are encrypted; API responses return only `api_key_configured` and `masked_key`.
- Enabling a provider does not authorize a paid call. `POST /api/v1/admin/video/providers/:id/tiny-real-authorization` records one explicit `tiny_real` authorization and does not dispatch a task.
- Task evidence exposes local/upstream task ids, terminal status, provider error, asset URLs, settled cost and `real_dispatch_count`. Failed tasks are not replaced by mock output.
- These pages remain in the complete standard-mode administrator surface. Ordinary employees have no administrator video route; their runtime entry remains the API-key protected `/v1/video/*` gateway.
- 如需回滚，只按 SOP 停止本轮应用容器并恢复旧应用容器；不删除 volume、数据库或备份。

## 下一步

人工提供有效 TOTP encryption key 后，才允许按 SOP 重跑 canonical `:8080` 与五段浏览器验收；代理不得读取或回显该值。真实付费证据另受 `BLOCKED:real-paid-generation` 控制，必须单独授权。
