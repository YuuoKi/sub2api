# K3 Apple-like 前端实验设计

更新时间：2026-07-17
状态：设计已批准，待书面规格复核
执行分支：`codex/k3-apple-ui-experiment-20260717`
基线分支：`wujie/video-capture-moat-20260702`
基线 HEAD：`deb9ff5b8876a5bca1a96f78d94f602a3fd5adb4`

## 1. 目标

在独立实验分支上重塑 Sub2API 全站前端，使老板、技术管理员和员工三类用户获得更简洁、克制、高端且高效率的操作体验。

本轮采用 Apple-like 交互原则，而不是复制 Apple 商标、素材或专有界面。核心目标是减少视觉噪音、建立清楚的信息层级、提升任务聚焦度，并保持桌面端、移动端、亮色主题和暗色主题的一致性。

## 2. 产品状态与事实边界

- 当前产品状态仍为 `READY_FOR_USER_REAL_TEST / 待复核`，不是生产 READY。
- 当前前端为 Vue 3、TypeScript、Vite、Vue Router、Pinia、Tailwind CSS、Vue I18n。
- 基线审查包为 `docs/reviews/LATEST_REVIEW_PACKAGE.html`。
- 基线视觉证据位于 `docs/reviews/assets/real-product-readiness-20260715/`。
- 本轮仅使用 mock 或无付费数据验证，不调用真实付费 Provider。

## 3. 设计原则

### 3.1 克制

- 保留无界青绿色作为唯一品牌强调色。
- 减少彩色底块、渐变、发光阴影和装饰性玻璃效果。
- 大面积使用中性背景、分层表面和轻边界，不把每个信息块都包装成强卡片。
- 危险、成功、警告颜色只表达真实状态，不作为装饰。

### 3.2 层级

- 每个页面只保留一个明确主任务和一个主操作。
- 页面标题、摘要、关键指标、主要内容和次要操作形成稳定层级。
- 把低频操作收入菜单、展开区或详情面板，减少首屏按钮竞争。
- 空状态必须告诉用户当前事实和下一步动作。

### 3.3 空间与排版

- 继续使用系统字体栈，包括 `-apple-system` 和 `BlinkMacSystemFont`。
- 使用一致的间距、圆角、描边、阴影、字号和内容宽度 tokens。
- 桌面端保持高信息效率，但避免密集表格贴边；移动端优先卡片化与任务动作可达性。
- 数字、金额、状态、时间和任务编号应有稳定的对齐与视觉语义。

### 3.4 动效与交互

- 动效用于反馈层级变化和操作结果，建议时长 150–240ms。
- 支持 `prefers-reduced-motion`。
- 所有可交互元素必须具备 hover、active、focus-visible、disabled 和 loading 状态。
- 不使用只能靠颜色识别的状态，不牺牲键盘操作和基础可访问性。

## 4. 改造范围

K3 可修改 `frontend/**`，包括：

- 全局设计 tokens 与 Tailwind 配置。
- `AppLayout`、`AppSidebar`、`AppHeader`、`TablePageLayout` 等共享壳层。
- 通用按钮、输入框、卡片、对话框、表格、筛选器、状态徽标和空状态组件。
- 老板、技术管理员和员工可见页面。
- 页面文案的层级、长度和一致性，但不得改变业务事实或计费语义。
- 与前端改造直接相关的测试。

除 `frontend/**` 外，只允许更新本任务的设计、计划、验证日志、截图索引和审查包；这些文档改动不得夹带产品代码或运行时配置变化。

允许新增小型、职责单一的展示组件和 tokens 文件。应优先复用现有组件和数据流，不得用单个超大文件替代现有结构。

## 5. 冻结范围

以下内容不得修改：

- `backend/**`、数据库、迁移、部署和运行时配置。
- API 路径、请求字段、响应字段和公开契约。
- 登录、角色、管理员权限、feature flag、simple mode 和 backend mode 判断。
- 计费、余额、币种、额度、Provider 对账与真实调用确认语义。
- 视频和图片任务创建、Idempotency-Key、轮询、终态、归档、预览、下载和再次引用逻辑。
- `mock | review_real | internal_real` 执行模式和真实调用防护。
- 真实 Provider、生产数据、密钥、token、cookie、`.env`、公网部署与外发。

如视觉方案需要改变上述任一业务逻辑，必须停止并记录为提案，不得自行实现。

## 6. 关键业务不变量

- 员工、管理员和老板必须继续看到各自允许的导航与页面。
- `/admin/video/create`、`/admin/video/tasks` 的员工访问边界保持现状；providers 与 system-check 仍为管理员专属。
- 视频创建默认仍为 mock；真实模式仍需能力门控和二次确认。
- 任务 created、processing、succeeded、failed 等状态不得在 UI 中被合并或伪造。
- `任务 succeeded`、`result_url 存在`、`本地资产已交付` 是三个不同事实。
- 图片与视频资产的预览、下载、再次引用必须继续指向服务端真实资产语义。
- Provider 账单导入和差异展示不得自动修改余额。
- 已知 `$` 与 CNY 文案债不得通过猜测业务规则来“统一”。
- 390px 移动端任务路径不得退化成横向溢出的桌面表格。

## 7. 信息架构与组件策略

### 7.1 全局壳层

- 侧栏按角色和业务任务分组，减少同级菜单竞争。
- 顶栏弱化低频工具，突出当前页面、关键环境信息和用户入口。
- 主内容区使用稳定最大宽度、响应式边距和统一页面标题区域。
- 折叠、移动抽屉、深色模式、语言和用户菜单行为保持可用。

### 7.2 页面模板

建立少量稳定页面模板：

- 概览页：关键结论、指标、异常和下一步动作。
- 表格页：标题、主要操作、搜索筛选、批量操作、数据区、分页。
- 创建页：目标说明、核心表单、风险确认、提交反馈。
- 详情页：状态摘要、关键事实、时间线、结果资产、失败原因和后续动作。
- 设置页：分组导航、局部保存、危险操作隔离。

### 7.3 通用组件

共享组件应只负责展示和交互状态，不重新实现业务请求。页面继续通过现有 stores、composables 和 API 模块取数与提交。

## 8. 数据流与错误处理

- 保持现有 Router → View → Store/Composable → API 数据流。
- 不在展示组件内新增隐藏网络请求或业务状态副本。
- loading、empty、partial、error、permission denied 和 stale 状态必须可区分。
- 后端返回的真实失败原因应安全、可读地展示；不得回显密钥、绝对路径或敏感 URL query。
- 提交按钮必须防止重复触发，并沿用现有幂等和任务创建逻辑。

## 9. 分阶段产出

### 阶段 1：视觉基础与黄金路径

- 全局 tokens、背景、排版、按钮、输入框、卡片和状态样式。
- `AppLayout`、`AppSidebar`、`AppHeader`、`TablePageLayout`。
- 老板仪表盘。
- 管理员视频总览。
- 员工视频创建页和任务列表。
- 1440px 与 390px、亮色与暗色对比截图。

阶段 1 完成后必须先审查方向，再继续全站扩展。

### 阶段 2：全站扩展

- 用户、账号、分组、渠道、使用记录、Provider 对账、运维、订单与个人页面。
- 优先迁移到阶段 1 确立的模板和组件，不为每个页面重新发明样式。
- `SettingsView.vue`、`GroupsView.vue` 等大型业务页面最后处理，并保持业务逻辑最小改动。

### 阶段 3：一致性与收口

- 处理亮/暗主题、桌面/移动、空状态、错误态、长文本和极端数据。
- 清理重复样式和未使用的视觉分支。
- 完成全量验证和单一审查包。

## 10. 验证与验收

每阶段至少执行：

```text
frontend: npx.cmd eslint . --ext .ts,.vue --max-warnings=0
frontend: npx.cmd vue-tsc --noEmit
frontend: npx.cmd vitest run --reporter=basic
frontend: pnpm run build
repo: git diff --check
```

必须重点保留：

- `frontend/src/components/layout/__tests__/AppSidebar.spec.ts`
- `frontend/src/__tests__/integration/navigation.spec.ts`
- `frontend/src/router/__tests__/guards.spec.ts`
- `frontend/src/router/__tests__/feature-access.spec.ts`
- `frontend/src/router/__tests__/video-route-access.spec.ts`
- `frontend/src/views/admin/__tests__/DashboardView.spec.ts`
- `frontend/src/views/admin/video/__tests__/`
- `frontend/src/views/user/__tests__/BatchImageGuideViewAssets.spec.ts`

视觉验收必须包含：

- 老板、技术管理员、员工三种角色。
- 1440px 桌面和 390px 移动视口。
- 亮色和暗色主题。
- 正常、加载、空、失败、禁用和权限受限状态。
- 与基线截图的并排对比。
- 不触发真实 Provider，截图和日志不得包含敏感信息。

## 11. 提交与审查规则

- 分支：`codex/k3-apple-ui-experiment-20260717`。
- 不 push、不部署、不合并、不删除基线资产。
- 每个阶段使用窄范围提交，提交信息说明视觉范围与验证结果。
- 阶段 1 未通过视觉和业务复核前，不进入阶段 2。
- 最终更新单一审查包 `docs/reviews/LATEST_REVIEW_PACKAGE.html`，内容包含目标、变更、截图、命令、结果、风险、回滚、状态和文件索引。

## 12. 验收标准

只有同时满足以下条件，实验分支才可标记为“可演示”：

1. 三类用户的主路径更清楚，首屏主任务和主操作明确。
2. 全站视觉语言一致，亮色、暗色、桌面和移动均无明显断层。
3. 所有冻结业务边界保持不变，关键路由和任务测试通过。
4. 真实状态、计费、币种、失败原因和资产交付语义没有被美化或合并。
5. 全量前端 lint、typecheck、Vitest、build 与 `git diff --check` 通过。
6. 审查包包含可核查的前后对比截图和验证日志。

状态不得因视觉改造直接提升为“内部可用”或生产 READY。

## 13. 风险与回滚

主要风险是全局 tokens 造成大面积回归、巨型业务页被误改、权限入口隐藏、真实状态被视觉抽象抹平，以及移动端表格退化。

回滚以阶段提交为单位使用 `git revert`。出现业务逻辑漂移、权限错误、重复任务、状态误导、敏感信息暴露或无法通过关键测试时，立即停止该阶段，不继续扩散修改。
