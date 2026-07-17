# 无界企业 AI 中台三套前端合一实施计划

日期：2026-07-17
执行分支：`codex/wujie-console-unification-20260717`
起始提交：`ab96e5228f5dce70ecc094c6fcdd8a2c1d08ab47`

## Global Constraints

- 只在 `D:\sub2api-trunk\.worktrees\console-unification` 写入；不得覆盖 `main`、K3 或 Kling Real 工作树的既有改动。
- 最终保持一个 Vue SPA、一个 Go 内嵌前端和一个 `:8080` 入口；不引入 iframe、微前端、第二套服务或公网入口。
- 最终用户可见品牌统一为“无界 · 企业 AI 中台”；不得在渲染界面或生产 bundle 中出现用户可见的 `Sub2API`、`Wei-Shaw`、`weishaw/sub2api` 或上游 GitHub 跳转。
- 不盲改内部兼容标识：WebSocket 协议、导入格式、本地存储键、数据库/镜像/服务名、模块路径和 LICENSE 归属保持兼容，除非有独立测试证明可以修改。
- 真实/付费 Provider、生产数据、真实支付、公网部署、push、不可逆删除均不在授权范围；Kling 真实调用保持关闭。
- 后端以当前 `main` 架构、迁移序列和事务边界为准；不得整体合并旧 Kling 后端，不重放其 136–156 迁移。只有当前 schema 确实缺字段时才添加新的向前迁移。
- 行为变更必须 TDD：先运行并记录失败测试，再做最小实现；每个任务结束前运行聚焦测试、`git diff --check` 并提交独立 commit。
- 状态只使用：内部可用、可演示、待复核、已阻塞、已冻结。mock 证据不得写成生产 READY。

## Task 1: 冻结三套基线与当前目标

在集成工作树中建立可审计的起点，不修改业务代码。

- 更新 `docs/goals/03_CURRENT_GOAL.md`，写明三套合一目标、单一入口、禁止真实付费、公网与破坏性清理的边界。
- 新增 `docs/reviews/WUJIE_CONSOLE_UNIFICATION_20260717/BASELINE.md`，记录 main、K3、Kling Real 的工作树、分支、HEAD、脏文件和当前候选完成度。
- 记录基线验证：pnpm 9 frozen install、lint、typecheck、全量 Vitest、build、Go vet、Go 全量测试；注明 Windows 中文用户临时目录导致 `TestResolvePageImagePath` 失败，固定 `TMP/TEMP` 到工作树 `.cache/test-tmp` 后全量通过，业务代码未改。
- 记录 K3 两个未提交 Vue 改动和两个临时扫描文件只读保留，不纳入本任务提交。
- 提交信息：`chore(integration): freeze frontend baselines`。

## Task 2: 补齐当前 main 的内部管理契约

以 Console v2 需要的能力为验收面，先补后端再接页面。

- 建立路由注册测试，覆盖 generation-content stats/samples/weekly-report/adoption、monthly-budget、管理员为成员创建密钥及密钥更新/删除、视频本地资产读取。
- 对照 `feature/kling-real-integration` 只移植缺失的 handler/service/repository 行为，并适配当前 main 架构；复用现有 schema，禁止复制旧迁移。
- 视频任务列表/详情响应补齐 `currency`、`pricing_source`、`pricing_version`，保持 QCanvas 既有公开字段兼容。
- 管理员路由继续经过现有鉴权与合规确认；普通员工不得访问 `/api/v1/admin/*`。
- 不接入或启用 Kling 真实分发。
- 聚焦 Go 测试全绿后提交：`fix(console): align admin and media contracts`。

## Task 3: 移植 Console v2 业务骨架

从 `feature/kling-real-integration@b918b91e6` 选择性移植前端业务视图，使用 Task 2 的当前接口。

- 移植并适配 BossOverview、KeyVault、Staff、AiRecords 及必要小组件/API 类型，不复制旧后端或旧迁移。
- 管理员顶层导航固定为：总览、密钥库、成员与开卡、任务记录、系统。
- 员工导航固定为：我的工作台、创建任务、任务记录、我的密钥、我的花费。
- 增加 `/admin/console/ai-records` 等缺失路由，确保 VideoTasks 内链不再进入 404。
- 保留完整 Key 只展示一次的既有安全语义；测试角色导航、路由守卫、页面 API 调用及错误/空状态。
- 提交：`feat(console): port console v2 business surfaces`。

## Task 4: 应用 K3 设计系统与无界品牌

从 `codex/k3-apple-ui-experiment-20260717@16351e1a3` 移植已验证的视觉层，不搬入其临时扫描文件。

- 移植 `--ui-*` tokens、Tailwind 桥接、壳层、通用空状态/进度/骨架组件和关键页面视觉结构；默认深色青绿，浅色可切换。
- 移植并验证 K3 当前未提交的视频文案精简和分钟精度日期改动。
- 统一浏览器标题、登录/欢迎/设置 fallback 和用户可见产品名为“无界 · 企业 AI 中台”。
- 清除用户可见上游品牌与 GitHub 入口，同时保留内部兼容标识和第三方 LICENSE。
- 覆盖 reduced-motion、键盘焦点、1440/390 响应式和明暗主题契约测试。
- 提交：`feat(ui): apply k3 design system and wujie branding`。

## Task 5: 收纳旧功能并闭合用户路径

- “系统”仅管理员可见、默认折叠，分为“运行与配置”和“高级与历史”。
- 渠道、供应商、IP、公告、账单、审计等进入“运行与配置”；订阅、兑换码、优惠券、订单、返佣等进入“高级与历史”。
- 旧功能路由和 feature flags 保留，不删除；普通员工手工输入管理员路由仍被守卫拒绝。
- 闭合创建成员/开卡、密钥一次复制、禁用/启用/删除、mock 图片/视频任务、轮询、失败原因、预览/下载/复用、AI 采用与周报、币种/计价来源、对账状态。
- 提交：`feat(navigation): simplify role-based console`。

## Task 6: 全量验证、Docker 与浏览器证据

- 运行 `go test ./... -count=1`、`go vet ./...`、前端 lint/typecheck/全量 Vitest/build、pnpm 9 frozen lock、`git diff --check`。
- 使用根 `Dockerfile` 构建统一镜像，确认 Go 内嵌新的 dist；只启动 loopback/受控本地栈，确认 `:8080` HTTP 200 且不依赖 `:3000`。
- 用 Playwright/浏览器验证老板、设计师、技术管理员三角色，以及 1440/390、深/浅主题；截图和日志不得包含密钥。
- 生产 bundle 与渲染页面扫描用户可见旧品牌为零；内部兼容标识命中单独列出，不误报。
- 不执行真实 Provider；状态最多为“内部可用 / 真实供应商待复核”。
- 提交：`test(review): verify unified console`。

## Task 7: 更新真相源与最终审查包

- 更新 `00_START_HERE.md`、`01_PROJECT_BASELINE.md`、`02_CURRENT_REALITY_STATUS.md`、`docs/goals/03_CURRENT_GOAL.md`，只写本轮有新鲜证据的状态。
- 重建 `docs/reviews/LATEST_REVIEW_PACKAGE.html`，包含目标、执行路径、三套来源、变更/API/权限矩阵、验证命令与退出码、截图、风险、回滚、状态、文件索引和可复制后续提示词。
- 记录旧镜像/数据库回滚前提；不得在本任务删除 K3/Kling 工作树或部署到公网。
- 提交：`docs(review): package unified console signoff`。

## Final Acceptance

- 一个代码分支、一个 Vue SPA、一个 Go 内嵌 dist、一个 `:8080` 入口。
- 老板、设计师、技术管理员各自只看到需要的导航与数据；管理员高级历史功能仍可访问。
- AI 记录不再 404，管理契约、币种和资产交付字段有自动测试。
- 全量门禁、Docker 冒烟和三角色浏览器证据齐全；旧品牌在用户可见表面为零。
- 真实 Provider 未授权时明确保留“待真实复核”，不声称生产 READY。
