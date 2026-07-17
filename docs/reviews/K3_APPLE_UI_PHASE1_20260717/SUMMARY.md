# K3 Apple-like UI Phase 1 — 审查检查点（Review Checkpoint）

> 状态：**可演示**（视觉检查点，非“内部可用”，非生产 READY）
> 生成时间：2026-07-17（UTC+8）｜生成方式：Task 5 自动门禁 + 全 mock 截图 + 人工逐图核对
> 增补：Phase 1.5 抛光（动效/融入方案/中文化）已收口，见文末「附：Phase 1.5 增补」

---

## 1. 目标与设计原则

**目标**：在不改动任何业务契约与付费 Provider 路径的前提下，交付 Sub2API 第一个可评审的 Apple-like 前端检查点，供用户决定是否批准全站迁移方向。

**设计原则**（源自 `docs/superpowers/specs/2026-07-17-k3-apple-ui-experiment-design.md`）：

- 中性画布与分层表面（`--ui-canvas/--ui-surface/--ui-surface-raised`），移除装饰性渐变（原 `bg-mesh-gradient` 薄荷底色）；
- 保留 Wujie 青（teal）品牌强调色，状态色只用于真实语义（失败红、警告黄、成功绿）；
- 结果先行（outcome-first）的信息层级：结论 → 主 KPI（≤4）→ 次要指标 → 关注项 → 趋势与排行；
- 表格数字（tabular numerals）、克制标签、单一主行动按钮、显式空状态；
- 明暗双主题同源 token；尊重 `prefers-reduced-motion`；
-  Router → View → Store/Composable → API 流程不变，仅表现层重组。

## 2. 执行分支、baseline HEAD、review HEAD

| 项 | 值 |
|---|---|
| 执行分支 | `codex/k3-apple-ui-experiment-20260717` |
| 对照基线分支 | `wujie/video-capture-moat-20260702` |
| Baseline HEAD（merge-base） | `deb9ff5b8876a5bca1a96f78d94f602a3fd5adb4` |
| Review HEAD（产品代码 tip） | `6b66dbf06d8465c288534c0b882f589e48a109cb` |
| 提交链 | `f9c6e881` 设计 → `b7e42b34` 计划 → `ea6da06b` tokens → `8ea98019` shell → `e7e6ede1` dashboard → `6b66dbf0` video →（本审查包文档提交） |

## 3. 精确变更清单

`git diff --name-only wujie/video-capture-moat-20260702...HEAD`（20 个文件，+1394/−639）：

**产品代码（全部位于 `frontend/**`，18 个文件）**

| 文件 | 归属任务 | 变更性质 |
|---|---|---|
| `frontend/tailwind.config.js` | Task 1 | 间距/圆角/阴影/颜色/动效 token |
| `frontend/src/style.css` | Task 1 | `--ui-*` 变量 + `.ui-*` 语义类 + reduced-motion |
| `frontend/src/__tests__/visualTokens.spec.ts` | Task 1 | token 契约测试（14 例） |
| `frontend/src/components/layout/AppLayout.vue` | Task 2 | 页面画布、256/72px 侧栏、粘性头部、landmark |
| `frontend/src/components/layout/AppSidebar.vue` | Task 2 | 导航呈现重组（计算逻辑未动） |
| `frontend/src/components/layout/AppHeader.vue` | Task 2 | 页首层级 |
| `frontend/src/components/layout/TablePageLayout.vue` | Task 2 | 表格页组合 |
| `frontend/src/components/layout/__tests__/AppLayoutVisualContract.spec.ts` | Task 2 | shell landmark 契约测试 |
| `frontend/src/components/layout/__tests__/AppSidebar.spec.ts` | Task 2 | 导航构建器回归断言 |
| `frontend/src/views/admin/DashboardView.vue` | Task 3 | 老板仪表盘 outcome-first 重排 |
| `frontend/src/views/admin/__tests__/DashboardView.spec.ts` | Task 3 | 层级顺序断言 |
| `frontend/src/i18n/locales/en/admin/overview.ts` | Task 3 | 新文案键（英文） |
| `frontend/src/i18n/locales/zh/admin/overview.ts` | Task 3 | 新文案键（中文） |
| `frontend/src/views/admin/video/VideoDashboardView.vue` | Task 4 | 视频总览：状态 → 推荐下一步 → 入口 |
| `frontend/src/views/admin/video/VideoCreateTaskView.vue` | Task 4 | 创建页主流程层级 |
| `frontend/src/views/admin/video/VideoTasksView.vue` | Task 4 | 筛选与结果分离、状态显式 |
| `frontend/src/views/admin/video/__tests__/VideoCreateTaskExecutionMode.spec.ts` | Task 4 | 表现层断言 |
| `frontend/src/views/admin/video/__tests__/VideoTasksView.spec.ts` | Task 4 | 表现层断言 |

**非产品代码（仅任务文档，2 个文件）**：`docs/superpowers/plans/2026-07-17-k3-apple-ui-phase1.md`、`docs/superpowers/specs/2026-07-17-k3-apple-ui-experiment-design.md`。

**本次 Task 5 新增（仅 `docs/reviews/**`）**：本 SUMMARY、`validation.log`、`capture_k3.mjs`、`capture-evidence.json`、`screenshots/*.png`（8 张）、`assets/mock-remote-preview.svg`、`docs/reviews/LATEST_REVIEW_PACKAGE.html`（更新）。

## 4. 前后对比截图

基线：`docs/reviews/assets/real-product-readiness-20260715/`（2026-07-15 G6 本地栈真实渲染）；新图：`screenshots/`（本次全 mock 渲染）。

| 视图 | 前（基线） | 后（本阶段） | 对照结论 |
|---|---|---|---|
| 老板仪表盘 1440 浅色 | `01-boss-dashboard.png` | `boss-dashboard-light-1440.png` | 薄荷渐变底 → 中性画布；8 张平级 KPI 卡 → 结论横幅 + 4 主 KPI + 次要条；新增“需要关注”区 |
| 老板仪表盘 1440 深色 | （基线无深色） | `boss-dashboard-dark-1440.png` | 同源 token 暗色表面，层级一致 |
| 管理员视频总览 1440 浅色 | `02-admin-video-dashboard.png` | `admin-video-dashboard-light-1440.png` | 大 hero + 2×2 统计格 → 紧凑 4 状态条 + 单一推荐下一步卡片 |
| 员工创建任务 1440 浅色 | `06-employee-video-create.png` | `employee-video-create-light-1440.png` | 结构保持（黄金路径不动），薄荷底 → 中性底；执行模式紧邻主行动 |
| 员工任务记录 1440 浅色 | `07-employee-video-tasks.png` | `employee-video-tasks-light-1440.png` | 筛选与结果明确分区；状态/失败原因/交付语义显式 |
| 员工任务记录 1440 深色 | （基线无深色） | `employee-video-tasks-dark-1440.png` | 暗色 badge 对比度正常 |
| 员工任务记录 390 浅色 | `09-employee-tasks-mobile.png` | `employee-video-tasks-light-390.png` | 卡片列表保留，无横向溢出 |
| 员工任务记录 390 深色 | （基线无深色） | `employee-video-tasks-dark-390.png` | 卡片列表暗色正常，无横向溢出 |

逐图人工核对结论：8 张均无横向溢出；无装饰渐变残留（页面背景为中性 token 色）；无任何真实付费触发痕迹（执行模式为免费试跑，任务全部【Mock】标记）。

## 5. 三角色与四种视图矩阵

| 角色 \ 视图 | 老板仪表盘 | 管理员视频总览 | 员工创建任务 | 员工任务记录 |
|---|---|---|---|---|
| Boss（admin 账号） | ✅ 浅色+深色 1440 | ✅ 浅色 1440 | — | — |
| Admin（管理员） | （同 Boss 视图） | ✅ 浅色 1440 | — | — |
| Employee（员工账号） | — | — | ✅ 浅色 1440 | ✅ 浅色/深色 1440 + 浅色/深色 390 |

覆盖说明：四种视图全部有图；老板仪表盘与员工任务记录覆盖了明暗双主题；员工任务记录覆盖 1440 与 390 双视口。管理员视频总览深色、员工创建页深色与移动视口未纳入本阶段矩阵（见第 7 节）。

## 6. 验证命令、退出码和结果

全部命令在 `frontend/` 下执行（Windows git-bash，Node v24.15.0），详见 `validation.log`：

| 门禁 | 命令 | 退出码 | 结果 |
|---|---|---|---|
| ESLint | `npx.cmd eslint . --ext .ts,.vue --max-warnings=0` | 0 | PASS（0 错误 0 警告） |
| 类型检查 | `npx.cmd vue-tsc --noEmit` | 0 | PASS |
| 全量测试 | `npx.cmd vitest run --reporter=basic` | 0 | PASS（152 文件 / 964 测试全过，0 失败 0 跳过） |
| 生产构建 | `"$HOME/AppData/Roaming/npm/pnpm.cmd" run build`（= `vue-tsc -b && vite build`） | 0 | PASS（33.83s；产物写入 gitignored 的 `backend/internal/web/dist/`） |

仓库卫生：`git diff --check` 退出 0；`git status --short` 干净；与基线分支 diff 中产品代码改动全部位于 `frontend/**`，非 frontend 改动仅 2 个任务文档。

截图证据（`capture-evidence.json`）：8/8 成功；API 调用 73 次全部被本地 mock 拦截；控制台错误 0；密钥模式命中 0；最终 URL 与目标路由一致（无登录重定向）。2 个未预置 fixture 的端点（`GET /keys`、`GET /admin/system/check-updates`）由兜底空响应处理并已记录。

## 7. 未验证或跳过项

1. **截图为全 mock 渲染**：vite preview + 路由拦截，未连真实后端/数据库；真实栈下的接口形态差异（字段缺省、长文本、极端值）未在本阶段验证。
2. **深色覆盖不全**：管理员视频总览深色、员工创建页深色/移动视口未截图。
3. **未做像素级 diff**：前后对照为人工逐图比对，未引入视觉回归基线工具。
4. **真实付费路径未触发**（设计上禁止）：未调用真实 Provider、未上传真实账单、未执行真实 create。
5. **后端门禁未运行**：本阶段无 `backend/**` 改动，Go 侧测试/构建未在本任务执行。
6. **构建警告**：`AccountsView` chunk >500 kB 为既有体积警告（非本次引入），未处理。
7. **i18n 运行时警告**：测试输出存在既有 vue-i18n runtime-build 警告（payment 组件），非本次引入。
8. **基线无深色/新 KPI 布局参照**：深色图为首次产出，无历史对照。

## 8. 业务不变量复核

| 不变量 | 复核方式 | 结论 |
|---|---|---|
| 路由守卫/角色边界不变 | `guards.spec.ts`、`feature-access.spec.ts`、`video-route-access.spec.ts` 全绿；diff 未触碰 router | ✅ |
| 执行模式（mock/review_real/internal_real）语义不变 | `VideoCreateTaskExecutionMode.spec.ts` 全绿；diff 未改能力门逻辑 | ✅ |
| Idempotency-Key 行为不变 | 视频测试套件全绿；diff 未改提交链路 | ✅ |
| 任务生命周期轮询条件不变 | `useVideoTaskLifecycle` 相关测试全绿；轮询条件未在 diff 中 | ✅ |
| 本地归档资产 vs 远端可交付物区分 | 截图 #5001/#5003（本地已归档）与 #5002（远端可交付 + 过期时间）并陈；`hasUsableRemoteAsset` 等逻辑未改 | ✅ |
| 货币语义（$ 不擅自转 CNY） | DashboardView 仍以 `$` 展示；`formatByCurrency` 未改；diff 无货币逻辑 | ✅ |
| 对账结论诚实待处理 | boss-conclusions fixture 返回 `not_uploaded`，UI 如实显示“未上传/对账待处理”，未伪造已对齐 | ✅ |
| 功能开关/菜单计算不变 | `AppSidebar.spec.ts` 断言 `adminNavItems/userNavItems/applyFeatureFlags` 保留，全绿 | ✅ |
| API 契约不变 | 无 `src/api/**` diff；无后端 diff | ✅ |

## 9. 风险与回滚

**风险**

- R1 视觉验证基于 mock 数据，真实数据密度下的折行/溢出未完全暴露（缓解：第 7 节已声明；建议 Phase 2 用本地 G6 栈复截）。
- R2 `frontend/src/i18n/locales/{en,zh}/admin/overview.ts` 新增文案键，若有第三方语言包缺失将回退到键名（影响面：仅新 UI 文案）。
- R3 仪表盘模型分布表在 1440 宽度下末列轻微截断（mock 长模型名下观察到），不影响功能（记录在案，本任务不修产品代码）。
- R4 趋势图 X 轴标签直接渲染后端日期串，长 ISO 串下可读性一般（mock fixture 放大此现象；真实后端返回格式未变）。

**回滚**：本阶段纯前端、无数据库迁移、无后端改动、无配置变更。回滚 = 在分支上 `git revert ea6da06b^..6b66dbf0`（或直接以 `deb9ff5b` 为基线废弃本分支），即可回到基线外观；审查包文档可独立删除，不影响产品。

## 10. 当前状态

**可演示**：四视图在明暗/双视口下可稳定复现演示（mock 模式），门禁全绿，审查证据齐全。
明确声明：本检查点**不构成**“内部可用”判定，**不构成**生产 READY；真实数据验收与真实栈复截完成前，状态不得上调。

## 11. Phase 2 建议与明确停止点

**建议（按优先级）**

1. 用本地 G6 栈（docker 可用环境）对真实后端复截同矩阵，替换 mock 证据；
2. 引入像素级视觉回归（如 Playwright snapshot）固化本阶段外观为基线；
3. 将 token 层推广到剩余高频视图（用户管理、渠道、密钥、使用记录），每次一个切片、同一门禁节奏；
4. 补齐深色矩阵（管理员视频总览、创建页）与 390 视口其余页面；
5. 可访问性走查（对比度、焦点序、reduced-motion 实测）；
6. 处理 R3/R4 两个表现层小瑕疵。

**明确停止点**：本次执行已在 Phase 1 审查包处停止。未经用户明确批准，不得开始 Phase 2；不得 push、合并、部署；不得将状态上调为“内部可用”。

## 12. 文件索引

| 路径 | 说明 |
|---|---|
| `docs/reviews/K3_APPLE_UI_PHASE1_20260717/SUMMARY.md` | 本文件 |
| `docs/reviews/K3_APPLE_UI_PHASE1_20260717/validation.log` | 四条门禁与卫生检查原始记录 |
| `docs/reviews/K3_APPLE_UI_PHASE1_20260717/capture_k3.mjs` | 全 mock 截图脚本（可复跑） |
| `docs/reviews/K3_APPLE_UI_PHASE1_20260717/capture-evidence.json` | 截图证据（逐张角色/路由/视口/主题/URL + API 拦截清单 + 密钥扫描） |
| `docs/reviews/K3_APPLE_UI_PHASE1_20260717/screenshots/` | 8 张新截图（命名即矩阵坐标） |
| `docs/reviews/K3_APPLE_UI_PHASE1_20260717/assets/mock-remote-preview.svg` | 远端结果预览占位图（截图 fixture） |
| `docs/reviews/LATEST_REVIEW_PACKAGE.html` | 自包含审查包（本阶段内容，已更新） |
| `docs/reviews/assets/real-product-readiness-20260715/` | 前后对比基线（2026-07-15） |
| `docs/superpowers/plans/2026-07-17-k3-apple-ui-phase1.md` | 执行计划 |
| `docs/superpowers/specs/2026-07-17-k3-apple-ui-experiment-design.md` | 设计规格 |

## 13. 可复制后续提示词

```text
你是 K3_Phase2_Executor。工作目录 D:\sub2api-trunk（分支 codex/k3-apple-ui-experiment-20260717）。
先读 docs/reviews/K3_APPLE_UI_PHASE1_20260717/SUMMARY.md 与 docs/superpowers/plans/2026-07-17-k3-apple-ui-phase1.md。
用户已批准 Phase 2：<在此粘贴用户批准原文>。
按 SUMMARY 第 11 节优先级执行：先在可用 docker 环境用本地 G6 栈复截真实后端同矩阵并替换 mock 证据；
再引入 Playwright 像素基线；然后每次只迁移一个高频视图到 token 层，迁移后跑
npx.cmd eslint . --ext .ts,.vue --max-warnings=0 / npx.cmd vue-tsc --noEmit / npx.cmd vitest run --reporter=basic / 构建四门禁。
约束不变：只改 frontend/** 与 docs/**；不读密钥；不调真实付费 Provider；不 push/部署/合并；
若用户未批准某子项，保持停止。
```

---

# 附：Phase 1.5 抛光增补（2026-07-17，WP-D 收口记录）

> 状态：仍为**可演示**（视觉检查点，非“内部可用”，非生产 READY）。本节为 Phase 1 审查包的增量补充；上文 Phase 1 全部结论继续有效。

## A. 三个工作包与变更摘要

| 工作包 | 提交 | 内容 | 规模 |
|---|---|---|---|
| WP-A 动效与进度可视化 | `c05fca349` `feat(ui): make svg motion intuitive and show task progress` | 新增 `AnimatedEmptyState`（场景化空状态插画 + 微动画循环）、`TaskProgressRing`（不定态进度环 + 阶段文案，不伪造百分比）、`Skeleton` shimmer 扫光；视频区 6 视图文案“说人话”微调（业务事实不变）；全部动效 `motion-reduce` 降级；新增契约测试 | 18 文件，+797/−73 |
| WP-C 主项目融入排版方案 | `556285dea` `docs: plan k3 frontend integration into northstar project` | 新增 `docs/main-project/k3-frontend-integration-plan.md`：北极星 V5 定位对齐（Sub2API = 管理中台本体）、IA 映射、视觉 token 映射、命名红线、排版规范、分阶段融入路线、开放决策清单（纯文档，无产品代码） | 1 文件，+203 |
| WP-B 中文化收口 | `a2bad5ceb` `feat(i18n): finish chinese copy across shell and billing` | 404 页中文化、供应商账单页中文表头与表单、视频路由 titleKey 中文化（浏览器标签）、残留英文硬编码组件收口（`CreateAccountModal` / `PricingEntryCard` / `OAuthAuthorizationFlow`）；新增 key 双语对称 | 17 文件，+295/−70 |

**K3 产出合计（Phase 1 + Phase 1.5，基线 `wujie/video-capture-moat-20260702` → `a2bad5ceb`）：65 文件，+5157/−856，全部位于 `frontend/**` 与 `docs/**`。**

## B. 四条门禁结果

全部命令在 `frontend/` 下执行（Windows git-bash，Node v24.15.0），**跑在 HEAD `a2bad5ce`**（2026-07-17 03:23–03:27 UTC+8），原始记录见 `validation-phase1.5.log`：

| 门禁 | 命令 | 退出码 | 结果 |
|---|---|---|---|
| ESLint | `npx.cmd eslint . --ext .ts,.vue --max-warnings=0` | 0 | PASS（0 错误 0 警告） |
| 类型检查 | `npx.cmd vue-tsc --noEmit` | 0 | PASS |
| 全量测试 | `npx.cmd vitest run --reporter=basic` | 0 | PASS（156 文件 / 988 测试全过，0 失败 0 跳过） |
| 生产构建 | `pnpm.cmd run build`（= `vue-tsc -b && vite build`） | 0 | PASS（23.73s；产物写入 gitignored 的 `backend/internal/web/dist/`） |

仓库卫生：`git diff --check` 退出 0；跑门禁时与基线的 diff 全部位于 `frontend/**` 与 `docs/**`（非 frontend/docs 改动为零）。

## C. Phase 1.5 截图清单与 WP-D 自查结论

8 张新截图（`screenshots/phase1.5-*.png`，全 mock 渲染；脚本 `capture_k3_phase1.5.mjs`，证据 `capture-evidence-phase1.5.json`）：

| 文件 | 内容 | 角色/视口 |
|---|---|---|
| `phase1.5-video-tasks-progress-ring-light-1440.png` | 任务记录：排队中/生成中不定态进度环 + 阶段文案 | employee / 1440 浅色 |
| `phase1.5-video-tasks-progress-ring-light-390.png` | 同上，移动卡片布局 | employee / 390 浅色 |
| `phase1.5-video-tasks-empty-light-1440.png` | 任务记录空状态（AnimatedEmptyState） | employee / 1440 浅色 |
| `phase1.5-video-dashboard-empty-light-1440.png` | 视频总览空状态 | admin / 1440 浅色 |
| `phase1.5-boss-dashboard-empty-light-1440.png` | 老板仪表盘空状态（mock 500 故意触发） | admin / 1440 浅色 |
| `phase1.5-skeleton-shimmer-light-1440.png` | Skeleton shimmer 扫光（演示节点注入，仅 CSS 验证） | admin / 1440 浅色 |
| `phase1.5-notfound-zh-light-1440.png` | 404 中文页 | admin / 1440 浅色 |
| `phase1.5-provider-billing-zh-light-1440.png` | 供应商账单中文表头与导入表单 | admin / 1440 浅色 |

**WP-D 逐图自查结论**（抽查 5 张：进度环 ×2、404、账单、老板仪表盘空态；其余由证据 JSON 覆盖）：

- ✅ 无横向溢出（含 390 视口进度环卡片布局）；
- ✅ 主要路径无英文残留：404 全中文；账单页表头/表单全中文；进度环为不定态 + 阶段文案，无伪造百分比；空状态插画与文案正常渲染；
- ✅ 无密钥/真实凭据：证据 JSON `secretHits = 0`；截图中 SHA-256、账号名均为 mock 值（`mock-acct-01`、`mocksha25600000…`），任务全部 Mock 数据；
- ✅ 控制台错误仅 2 条，均为老板仪表盘空态故意触发的 mock 500（by design，用于渲染空状态）；
- ⚠️ 两处轻微英文残留/显示瑕疵，记录在案（见 E 节 R7/R8），本任务不修产品代码。

截图证据汇总：8 captures / 86 次 API 调用全部被本地 mock 拦截 / 2 个 unknownEndpoints（与 Phase 1 相同的兜底端点 `GET /keys`、`GET /admin/system/check-updates`）/ 3 runs。

## D. 外部后端提交的如实说明（重要）

四条门禁跑完后、本审查包收尾前，**6 个外部后端修复提交**（非 K3 会话产出）落到了本分支：

| 提交 | 摘要 |
|---|---|
| `4d0a55992` | fix(service): redact percent-encoded proxy credentials in upstream errors |
| `2471ee1c8` | fix(routes): seed real-access policy for Seedance trial success test |
| `1be6b3c28` | fix(repository): hold migration advisory lock on a dedicated conn |
| `b80ae135e` | fix(middleware): fail closed when Docker is unavailable in CI |
| `5231d97d5` | fix(deploy): generate VIDEO_GATEWAY_ENCRYPTION_KEY in docker-deploy.sh |
| `bf5b14890` | fix(repository): finalize billable tasks with review_required reservations |

- 触碰范围：`backend/**`（12 文件）+ `deploy/docker-deploy.sh`（1 文件），共 13 文件，+420/−47（`git diff --stat a2bad5ceb..HEAD` 核实）。
- 前端代码自 `a2bad5ce` 起**零变化**（`git diff a2bad5ceb..HEAD -- frontend/` 为空）→ B 节前端门禁结果对当前 HEAD 仍然有效。
- **相对基线的 diff 统计分列**：K3 产出（基线..`a2bad5ceb`）= 65 文件 +5157/−856，全部在 `frontend/**` 与 `docs/**`；外部提交（`a2bad5ceb..HEAD`）= 13 文件 +420/−47，全部在 `backend/**` 与 `deploy/**`；合并全量（基线..HEAD）= 78 文件 +5577/−903。对基线 diff 中 `frontend/**` 与 `docs/**` 之外的文件（13 个）已逐一核实全部来自上述 6 个外部提交。
- 这些提交**不属于 K3 审查范围**：K3 未运行 Go 侧门禁，不对其正确性背书；本审查包的状态声明不覆盖它们。

## E. 新增风险与已知问题（Phase 1.5）

- **R5 i18n 死 key（WP-D 静态复核）**：`en/admin/overview.ts` 中 K3 直接造成的无引用 key 共 2 个——`admin.dashboard.summary`（Phase 1 新增但从未被引用）与 `admin.dashboard.quickActions`（基线有引用，outcome-first 重排后引用被移除）；`zh/admin/overview.ts` 镜像同样 2 个。另有 30 个 `admin.dashboard.*` key 经基线复核确认**早于 K3 即无引用**（遗留，非 K3 引入）。注：前一工作会话曾记录“11 个无引用死 key”，本次按字面引用审计未能复现该数字，以本复核结论（K3 造成 2 个/语言，双语对称）为准；`en/dashboard.ts` 的 `errors.*` 段与 `en/common.ts` 的 `nav.*` 段在浅层前缀审计中易误判为死 key，实际经根命名空间引用（`t('errors.*')`、路由 `titleKey: 'nav.*'`），已核实非死 key。影响：无运行时影响，仅语言包冗余。
- **R6 `Skeleton.vue` 无运行时消费者**：shimmer 截图系向页面 DOM 注入演示节点完成（仅 CSS 验证），该组件尚未被任何视图引用（`capture-evidence-phase1.5.json` notes 已记录）。
- **R7 供应商对账“对账结论”列显示原始枚举值** `has_diff`（未映射为本地化标签），见 `phase1.5-provider-billing-zh-light-1440.png`；记录在案，本任务不修产品代码。
- **R8 顶栏角色徽章显示英文 `Admin`/`User`**：既有 shell 元素（Phase 1 前已存在），不在 WP-B 范围；记录在案。
- **R9 动效未在真实数据密度下验证**：进度环/空状态微动画/shimmer 均基于 mock 数据与模拟轮询状态截图；真实数据密度、真实轮询时序与低端设备性能未验证（延续 Phase 1 R1）。
- **R10 外部提交与 K3 审查范围正交**：回滚 K3 时不得连带回滚 `4d0a55992..bf5b14890`；外部提交若引入回归，与 K3 前端无关，需独立追责。

## F. 状态与停止点（Phase 1.5）

**可演示**：Phase 1.5 三个工作包门禁全绿、截图与证据齐全，动效/空状态/中文化在 mock 模式下可稳定复现演示。明确声明：本增补**不构成**“内部可用”判定，**不构成**生产 READY；真实数据验收与真实栈复截完成前状态不得上调。

**停止点**：本次执行已在 Phase 1.5 审查包处停止。不进 Phase 2、不 push、不合并、不部署、不回滚外部提交。

## G. Phase 1.5 文件索引

| 路径 | 说明 |
|---|---|
| `docs/reviews/K3_APPLE_UI_PHASE1_20260717/validation-phase1.5.log` | Phase 1.5 四条门禁与卫生检查原始记录 |
| `docs/reviews/K3_APPLE_UI_PHASE1_20260717/capture_k3_phase1.5.mjs` | Phase 1.5 全 mock 截图脚本（可复跑） |
| `docs/reviews/K3_APPLE_UI_PHASE1_20260717/capture-evidence-phase1.5.json` | Phase 1.5 截图证据（8 captures + API 拦截清单 + 密钥扫描） |
| `docs/reviews/K3_APPLE_UI_PHASE1_20260717/screenshots/phase1.5-*.png` | 8 张 Phase 1.5 截图 |
| `docs/main-project/k3-frontend-integration-plan.md` | WP-C 主项目融入排版方案 |
| `docs/superpowers/plans/2026-07-17-k3-phase1.5-polish.md` | Phase 1.5 执行计划 |
