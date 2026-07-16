# 无界 AI 管理中台 · START HERE

更新时间：2026-07-16 Asia/Shanghai

## 一句话状态

canonical `http://127.0.0.1:8080`、无界 embedded 品牌、本地镜像与完整管理后台五段路径已有同实例运行证据；Gemini 图片与 Seedance 2.0 视频各一次真实 tiny_real 也已成功且无重试。`realCallExecuted=2`、`open_p0_count=0`、`blocked=[]`，整体为**待复核**；不得外推为商业交付完成、生产上线或已 push。

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
- canonical runtime：`http://127.0.0.1:8080` HTTP 200、title 为 `无界 · 企业 AI 管理中台`、镜像为 `wujie-sub2api:local`、host `:3000` 无监听；完整管理面、管理员开卡、员工 Key 与 usage 五段路径已留证。
- 视频网关实现与付费安全收口：当前 HEAD `d6687e89`（视频能力落点祖先含 `15880e23`）；`/v1/video/tasks`、管理员视频控制面、原子一次授权/dispatch claim、usage/settlement 与 process-only gate 均已验证。
- Gemini 图片真实证据：request `client:995b4b75-be49-4ecd-a8ff-c7bc283fda69`，终态成功，本地 PNG 1336210 bytes，实扣 `$0.0585`；实账 `billedImageSize=2K`（勿再写成已验证 512）。
- Seedance 2.0 真实证据：local task `1`、upstream task `cgt-20260716174852-8p5hj`、终态 `succeeded`、87300 tokens，本地 MP4/尾帧齐全，实扣 `$0.3395`。
- 两次合计 `$0.3980` / 约 `¥2.8656`，员工余额 `$3.0000 → $2.6020`，低于授权上限 `¥15`；process-only 单次门已恢复关闭。

## 验收纪律

必须同时看到以下事实才可把 canonical 本机入口升级为可演示：

1. `http://127.0.0.1:8080` HTTP 200；
2. raw title 命中无界品牌；
3. 运行镜像为 `wujie-sub2api:local`；
4. host `:3000` 无监听；
5. 完整管理面、管理员开卡、员工 Key 调用与 usage 可见；
6. 截图和日志来自同一真实运行实例。

G1 已满足以上口径。不得用源码字符串、单元测试、隔离 smoke、one-off 容器或空 UI 截图冒充后续真实付费验收。

## 安全与回滚

- 禁止 push、生产库 migration 和删除备份目录。
- 本 CYCLE 已消费完明确授权的两次真实 tiny_real，`realCallExecuted=2`；禁止再次真实调用、自动重试或模型降级。隔离 loopback mock usage 仍不等于真实 provider 调用。
- `.env` 与上游密钥保持未跟踪、未提交；文档只记录脱敏标识、task id、资产 hash 与费用差分。

## Canonical video control plane (G2.R1)

- Administrator UI: `/admin/video/providers` -> `/admin/video/tasks` -> `/admin/video/system-check`.
- Provider creation is Seedance-only and requires an active standard employee group. Backend contract fixes model `doubao-seedance-2-0-260128` and endpoint `https://ark.cn-beijing.volces.com/api/v3`; the UI does not ask the operator to guess them. Stored secrets are encrypted; API responses return only `api_key_configured` and `masked_key`.
- Enabling a provider does not authorize a paid call. `POST /api/v1/admin/video/providers/:id/tiny-real-authorization` records one explicit `tiny_real` authorization and does not dispatch a task.
- Task evidence exposes local/upstream task ids, terminal status, provider error, asset URLs, settled cost and `real_dispatch_count`. Failed tasks are not replaced by mock output.
- These pages remain in the complete standard-mode administrator surface. Ordinary employees have no administrator video route; their runtime entry remains the API-key protected `/v1/video/*` gateway.
- 如需回滚，只按 SOP 停止本轮应用容器并恢复旧应用容器；不删除 volume、数据库或备份。

## 下一步

G3 文档同步与独立冷审**已完成**；交叉验证 IV 文档消毒后，等老板确认再 commit（本仓仅允许既有 `docs/legal/admin-compliance.zh.md` dirty 不纳入提交，除非老板另批）。

当前真实下一步：

1. 真相源对齐：以本文件、QCanvas 总规划 `#qcanvas-state`（整体**待复核**）、`20260716_FINAL_FINDINGS.md` 顶部权威段为准；不得再把 `GROK45_COMMERCIAL_DELIVERY_LOOP_STATE_20260715.json` 当当前口径。
2. 不得再次触发真实付费；`realCallExecuted=2` 已封口。
3. 可选补证：老板提供 QCanvas `projectId/flowId` 后，仅通过 `tapcanvas-api` 复用本地 PNG/MP4/尾帧做非付费画布回填；可选 pilot 浏览器链补证。
4. UX P1 排期与密钥轮换（会话曾出现明文 key 粘贴）按老板授权单独处理；禁止 push。
