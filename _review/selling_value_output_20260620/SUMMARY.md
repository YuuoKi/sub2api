# 采集口数据 · 第一个价值出口 · 方案设计 SUMMARY

> 任务编号:`G_SELLING_VALUE_OUTPUT_DESIGN` ｜ 生成:2026-06-20 ｜ 执行:Claude Code(ultracode 对抗 workflow)
> 性质:**纯只读静态分析 + 方案设计**。零代码、零服务、零数据库、零 git 改动。
> 并行安全:与"修房子"session(`G_M1B_REALRUN_ENV_AND_VERIFY`)同时跑;全程只读、未起任何服务/连库/占端口、未改任何代码/分支/worktree。本文件写在独立目录 `_review/selling_value_output_20260620/`,与对方 `_review/M1B_realrun_verify_*` 文件级不碰撞。
> 方法:官方 Dynamic Workflow,3 候选 × 4 维独立打分(12 评估器)+ 每候选 1 个对抗 skeptic + 1 个综合裁判 + 1 个 winner demo 设计器,共 17 个**只读** subagent(`agentType: Explore`,工具级无 Edit/Write)。

---

## 0. 一句话结论(先读这条)

**首推第一个价值出口 = 候选 C「护城河快照卡 + 样本墙」(Moat Snapshot + Sample Spotlight)** —— 一张卡:顶部几个真实计数(已采 N 条真实生产内容 · M 名员工 × K 个团队 × P 个模型 · 近 7 日 +Y/日 · 累计字节量),底部一面 **6–8 条真实脱敏 prompt→response 样本墙**作为肉眼可见的护城河证据。

**但本任务最重要的发现不是"做哪个 UI",而是一条把三个候选全部击中的致命反驳:**

> **采集口默认是黑的(DARK)。** `gateway.content_capture.enabled` 默认 `false`,目前没有任何真实环境把它打开过 → **采集表 `ai_generation_content` 现在是空的**。无论做哪个出口,day-one 打开都是"0 条、0 字节、空样本墙",对老板反而是"这系统还没跑起来"的反效果。

所以**第一个价值出口设计的第一要务,是"day-one 怎么有真数据"**,UI 形态是第二位的。我对 workflow 的关键修正:裁判团倾向"灌一批合成假数据(显示 ~5000 条/150MB)当门面",**我不推荐把伪造的聚合数字当真实证据展示给老板**(任务明令"真实、可演示",造假是反面)。正解是更诚实也更便宜的一条:**把 flag 在受控环境点亮(fail-open、与计费隔离)→ 跑 30–60 分钟代表性真流量 → 采集表落进真行 → 快照卡展示真聚合 + 真脱敏样本。** 这才是"第一个真实价值出口"。

**C 赢的硬道理:**②"3 秒冲击"对三者一视同仁地低(都因空表)→ 不是区分项;真正区分 winner 的是 ①技术可行性 ③最小工作量 ④与现采数据匹配,**C 这三项全胜**:它只吃 M1 此刻真写的字段,零预留列、零跨仓回流、零外部依赖,~1 天即可做出真 demo。

---

## 1. 采集数据现状(只读核实,作为所有候选的地基)

### 1.1 表 `ai_generation_content`(migration `140_ai_generation_content.sql`)
append-only 旁路表,按 `(api_key_id, request_id)` 与 `usage_logs` 1:1。列分两类:

| 类别 | 字段 | 说明 |
|---|---|---|
| **M1 此刻真写** | `id`,`request_id`,`api_key_id`,`user_id`,`group_id`,`account_id`,`model` | 三级归因(员工/团队/上游账号)+ 模型 |
| | `prompt_redacted` TEXT(≤256KiB) | 脱敏后输入 |
| | `response_redacted` TEXT(≤64KiB) | 脱敏后产出(超限截断) |
| | `prompt_bytes` / `response_bytes` / `response_truncated` | 体量与截断标记 |
| | `redaction_version`,`request_payload_hash`,`created_at` | 审计/去重/时间 |
| **预留 · M1 不写** | `task_id`,`adoption_status`,`quality_score` | 留给 M5/QCanvas 跨仓回流——**任何价值出口都不该依赖它们** |

索引齐备:`created_at DESC`、`(user_id, created_at)`、`(group_id, created_at)`、`task_id`、唯一`(api_key_id, request_id)`。→ 聚合 COUNT/SUM 与"取最近 N 条样本"两类查询都走得动现成索引。

### 1.2 数据形态与脱敏
- 入库**前**脱敏:prompt = `RedactJSON → PII(email/电话) → 密钥/token 模式`;response = `RedactText → PII → 密钥`(`generation_content_redact.go`)。→ 存的是**真实但去敏**的自然语言文本;视频/图片产出是短 HTTPS URL。
- 已知残留(M1A 自陈):全字母凭据、12–19 字符不透明 token、正则外 PII 仍可能漏 → **样本墙对外展示前需人工过一眼**(见 §5 治理)。

### 1.3 采集状态 —— 关键
- **DARK by default**:`content_capture.enabled` 默认 `false`(热路径零开销、不采)。三条路(主 Anthropic `/v1/messages`、gemini、openai chat-completions)已接线,但**任何环境在 flag 翻开前,表是空的**。翻 flag 是 fail-open、与计费隔离的。
- 目前**无记录显示该 flag 在任何真实环境开过** → 设计必须假设 day-one 表 0 行。

### 1.4 消费面 precedent(决定工作量——非常强)
| 层 | 现成可镜像的东西 | 证据 |
|---|---|---|
| 路由/鉴权 | Gin,`/api/v1/admin` + `adminAuth` 中间件(role==admin) | `routes/admin.go`、`middleware/admin_auth.go:190` |
| 聚合读端点 | `GET /admin/dashboard/stats|models|groups|trend|users-ranking`(全读 `usage_logs`) | `handler/admin/dashboard_handler.go:69` `parseTimeRange→service→response.Success(gin.H{...})` |
| 分页列表 | `GET /admin/usage` 已是**带搜索的分页列表** | `routes/admin.go:545` |
| 仓库读法 | 新表是非 ent 裸 SQL;List/Count/聚合可镜像 `usage_log_repo.go`(`GetUserStats` COUNT/SUM 套路) | 现 `generation_content_repo.go` 只有 `Create()` |
| 前端 | Vue3+Vite+Pinia+Tailwind+Chart.js;`UsageView.vue` 已由可复用组件拼成 | `AppLayout / UsageStatsCards / DateRangePicker / ModelDistributionChart / GroupDistributionChart / UsageTable / Pagination / 导出` |
| 导出 | CSV 导出 precedent | `handler/admin/redeem_handler.go` Export |
| 人读归因 | `users(email,username)`、`groups(name=团队/部门,description)`、`api_keys(name)`、`accounts(name)` | migration 001 等 |

> 含义:**"采集内容的读出口"几乎是把现有 usage 看板/快照那套零件,改指向一张新表。** 唯一真·净新工作是新表的裸 SQL 读方法(目前只有 `Create`)。

---

## 2. 三个候选 + 四维评估

> 四维:①技术可行性(对真 schema/消费面)②老板 3 秒冲击 ③最小工作量(最快出 demo)④与现采数据匹配。分值 1–5,越高越好。

### 候选定义
- **A · 采集内容看板(全量仪表盘)**:计数卡 + 按模型/按团队分布图 + 采集量时间趋势 + 可搜索分页大表(逐行 员工/团队/模型/时间/prompt 预览/response 预览/字节,点开看全文)+ CSV 导出。基本 1:1 镜像 `UsageView.vue` 指向新表。
- **B · 第一个 skill 蒸馏闭环(数据→降本飞轮 demo)**:从采集 prompt 里聚类一类高频任务 → 手蒸馏成一个可复用 skill → 演示它替人做这类活 →"数据→降本"飞轮第一圈。多为离线/人工分析 + 一个可跑 demo,**依赖足量同类采集行**。
- **C · 护城河快照 + 样本墙(一眼最小punch)**:一张卡:一行/一组英雄计数 +(6–8 条真实脱敏 prompt→response 样本墙)。一条聚合查询 + 一条"最近 N 条样本"查询,最薄前端。

### 评分表(已对 workflow 噪声去噪,见 §3 脚注)

| 候选 | ①可行性 | ②3秒冲击 | ③最小工作量 | ④数据匹配 | 均分 | 名次 |
|---|:--:|:--:|:--:|:--:|:--:|:--:|
| **C 快照+样本墙** | **4** | 2 | **4** | **5** | **3.75** | **🥇 1** |
| A 全量看板 | 2 | 2 | 2 | 4 | 2.5 | 🥈 2 |
| B skill 蒸馏 | 2 | 2 | 2 | 2 | 2.0 | 🥉 3 |

### 各候选最强点 / 最大风险(取自评估器原文)
- **C**:最强 = schema/索引生产就绪,只吃 M1 现写字段(④=5),全栈每层都有直接 precedent,~6–9h 一个窄竖切;风险 = day-one 空表(②=2)、脱敏样本可读性需人工curation、对内容表展示的治理/保留期未定。
- **A**:最强 = 代码复用最大化、`UsageView.vue` 几乎照搬;风险 = **不最小**(只为演示价值却要做计数卡+多图+大表+导出)、`generation_content_repo` 读层从零写裸 SQL、截断/脱敏行的展示策略未articulate、day-one 同样空。
- **B**:最强 = 唯一直指"数据潜在价值"(降本飞轮),正面回应"没人消费";风险 = **致命依赖不存在的数据量**、`request_payload_hash` 是去重哈希不是聚类键、相似度未预计算(需 ~500–1000 LOC)、脱敏会破坏任务语义、实为 1–2 周后端工程而非"纯设计/离线"。

---

## 3. 对抗收敛过程

### 3.1 三个 skeptic 的判决:**同一条致命反驳,全员 `fatal`**
| 候选 | 最强反驳 | 判定 |
|---|---|---|
| A | 表 DARK 默认 → 老板看到全 0 看板 = 信号"没在运行",不是价值 | **fatal** |
| B | 表空 + 无量 → 从空表手蒸馏 skill 是"白板草图",且若灌志愿/合成数据就断了"采集数据→降本"的因果链 | **fatal** |
| C | 英雄行与样本墙 day-one 全空,1–2 天爬坡才有数据 → 最弱证据点 | **fatal** |

→ **关键洞察:致命的不是某个候选,是"空表"这个共同前置。它把竞争从"哪个 UI"挪到了"day-one 怎么有真数据"。**

### 3.2 裁判收敛
- 致命反驳对三者一致,**C 是用最小额外工作即可存活的那个**:A/B 要先砸进昂贵基建(全看板 / 聚类管线)才显示一个 0;C 只需一张卡 + 2 条查询,就能在"有真数据"时立刻成立。
- **winner = C**;**runner-up = A**;**从 A 嫁接**:把 A 的"多计数卡 + 迷你时间趋势(sparkline)"布局移植进 C 的英雄区(多个真计数比单行更有冲击),复用 `DashboardView.vue` 既有 CounterCard/MiniTrendChart。
- **关键权衡**:最小化 vs 仪表化。C 主动舍弃 A 的日期筛选/分布图/导出,换"day-one 可加载、非零、可信"的单一证据点。**A 的全看板是未来态,不是第一个价值出口的 MVP。**

### 3.3 我(Claude Code)对裁判结论的一处修正 —— ⚠️ 重要
裁判团把"灌合成 fixture(英雄行写死 `~5000 条=150MB / 12 团队 / 45 员工`)、加 `Demo Data` 角标"当 day-one 主方案。**我不推荐把伪造聚合数字当真实证据展示给老板**:

- 任务明令"**真实、可演示**"、禁"空泛/theater";伪造的聚合数字一旦被当真,是比空表更大的信誉地雷(护城河证据变成造假表演)。
- 更诚实也更便宜的路就在手边,且 skeptic 自己也提到了:**受控环境翻 flag(fail-open、与计费隔离)→ 跑 30–60 分钟代表性真流量 → 表落真行 → 快照卡展示真聚合 + 真脱敏样本。** 成本极小(网关本来就在转 LLM 调用,只是把采集开关打开一会儿)。
- 因此 **day-one 数据路径排序**:① **受控真流量点亮(推荐主路径,真实可信)** > ② 仅做"明确标注为示例/illustrative"的样本占位(用于真流量尚未跑时的空态,**绝不展示成真实聚合数字**)> ❌ 把合成聚合数字当真实证据。
- 英雄指标措辞也要诚实:领头用**真实可数的事实**(已采 N 条 · M 员工 × K 团队 × P 模型 · 近 7 日 +Y/日),"字节量"叙述为"已沉淀内容体量"而非过度宣称"X MB 可复用资产价值"(字节≠价值,这是叙述选择不是数据保证)。

---

## 4. 首推方案 C 的最小 demo 设计

> 名称:**护城河快照卡 + 样本墙**(workflow demo 设计器误标为"Collected-Content Dashboard",其内容即 C 的最小版,已正名)。设计 only,不实现。

### 4.1 读哪些字段(全部 M1 此刻真写,零预留列)
`created_at, user_id, group_id, model, prompt_redacted, response_redacted, prompt_bytes, response_bytes, response_truncated, request_id`
+ 人读归因 LEFT JOIN:`users.email/username`、`groups.name`、`api_keys.name`。

### 4.2 净新后端(2 个只读端点,均挂 `adminAuth`)
| 端点 | 读什么 | 返回 | 镜像谁 |
|---|---|---|---|
| `GET /api/v1/admin/generation-content/stats` | `ai_generation_content` 聚合:`COUNT(*)`、`COUNT(DISTINCT user_id/group_id/model)`、`SUM(prompt_bytes+response_bytes)`,今日 + 近 7 日 | `{captured_today, captured_week, distinct_employees, distinct_teams, distinct_models, total_bytes, daily_avg_rate}` | `dashboard_handler.go GetStats` + `parseTimeRange` |
| `GET /api/v1/admin/generation-content/samples` | `ORDER BY created_at DESC LIMIT 6–8` + JOIN 归因 | `{samples:[{employee_name, team_name, model, created_at, prompt_preview(120字), response_preview(80字), total_bytes, truncated}], is_live:boolean}` | `usage_handler.go List` 的 JOIN/分页套路 |

> 仓库层:`generation_content_repo.go` 加 2 个裸 SQL 方法 `GetCaptureStats(...)` / `GetRecent(ctx, limit)`,镜像 `usage_log_repo.go` 的 COUNT/SUM/LIMIT 套路。这是**唯一真·净新代码**。
> 路由:`routes/admin.go` 的 `registerDashboardRoutes` 加 2 行(GET)。

### 4.3 净新前端(1 个视图,复用现成组件)
- 文件:`frontend/src/views/admin/GenerationContentView.vue`,镜像 `UsageView.vue` 结构。
- 复用:`AppLayout`、`CounterCard`(×4–6:今日/本周采集数、员工数、团队数、模型数、累计体量)、`MiniTrendChart`(7 日 sparkline)、新增轻量 `ContentWall`(6–8 条样本卡:员工/团队/模型/时间/prompt 摘要/response 摘要/字节/截断标)。
- API client:`frontend/src/api/admin/generation_content.ts`,镜像 `dashboard.ts`。

### 4.4 老板看到的屏幕
```
┌─────────────────────────────────────────────────────────────┐
│  护城河计量器 · 采集口已沉淀的真实生产内容                          │
│  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐    7日趋势 ▁▂▃▅▆▇   │
│  │今日 N│ │本周 N│ │员工 M│ │团队 K│ │模型 P│    +Y/日           │
│  └─────┘ └─────┘ └─────┘ └─────┘ └─────┘                     │
├─────────────────────────────────────────────────────────────┤
│  真实脱敏样本墙(6–8 条)                          [● 实时数据]    │
│  ▸ [研发组·张三·claude-opus-4-8·10:21]                         │
│     prompt: "帮我审查这段 Go 并发代码…[已脱敏]"                   │
│     response:"这里有个 use-after-return…[已脱敏] (省 3.2KB)"     │
│  ▸ [产品组·李四·gemini-…] …                                    │
└─────────────────────────────────────────────────────────────┘
```
右上角徽标二态:`● 实时数据`(真流量已点亮)/ `○ 示例(未开启采集)`(空态明确标注,绝不冒充真聚合)。

### 4.5 最小步骤(~1 天)
| # | 步骤 | 工时 |
|---|---|---|
| 1 | 后端 repo:`GetCaptureStats` + `GetRecent` 两个裸 SQL(镜像 usage_log_repo) | ~2h |
| 2 | 后端 handler:`generation_content_handler.go`(镜像 dashboard/usage handler,继承 adminAuth) | ~1h |
| 3 | 后端路由:dashboard 组加 2 个 GET | ~0.5h |
| 4 | 前端 API client(镜像 dashboard.ts) | ~1h |
| 5 | 前端视图:复用 CounterCard/MiniTrendChart + 新 ContentWall | ~3h |
| 6 | **day-one 真数据(推荐主路径)**:本地/staging 翻 `content_capture.enabled=true` → 跑 30–60 分钟代表性真流量(三条路各几条)→ 表落真行 → 卡展示真聚合 + 真样本;截图给老板 | ~1h |
| 7 | 验证:翻 flag 真/空两态、样本墙人工过一眼脱敏、`is_live` 切换 | ~1h |
| | **合计** | **~9.5h(≈1 个工作日)** |

### 4.6 day-one 空表方案(我修正后的排序)
1. **首选·受控真流量点亮**(步骤 6):fail-open、与计费隔离,最真最可信。
2. **次选·明确标注的示例空态**:真流量未跑时显"○ 示例"占位 + 几条**标注为 illustrative** 的样本,**不展示伪造聚合数字**。
3. **❌ 不做**:把合成聚合(5000 条/150MB)当真实证据展示。

### 4.7 治理与保留期 note(展示真实脱敏内容前必看)
- 仅 `adminAuth`(admin 角色)可见;内容已去敏但**残留已知**(全字母凭据、长 token、正则外 PII)→ 样本墙对外展示前**人工过一眼**。
- **内容表清理/保留期任务尚未建**(M1A 决策#2"独立较短保留期+清理任务"仍是 deferred 项,grep 全仓无内容表 cleanup)→ 若价值出口要长期对外,**建议先补内容/PII 保留期清理**,与计费保留期解耦。
- 不新增导出路径;未来若加 CSV 导出,按 redeem Export precedent 并审计日志。

### 4.8 给老板的一句话(护城河落地)
> "每一次真实生产调用的输入与产出,都已被脱敏沉淀、可归因到人/团队/模型——采集口不是基建,是正在变厚的护城河;这面墙上每一条,都是别人拿不走的资产。"

---

## 5. 落地步骤建议(若学者拍板做 C)

1. **先定 day-one 数据路径**(§4.6):授权在受控环境短暂点亮 flag + 跑真流量(推荐),而非灌假数据。
2. 立"实现任务书"(本任务只设计):范围严格限定 C 的 2 端点 + 1 视图 + repo 2 方法,**不顺手做 A 的全看板**(留作后续演进)。
3. 实现前补内容表保留期/清理(若决定长期对外展示),与计费解耦。
4. 实现后:Codex + Claude 双家族复核(沿用既有 review-loop 习惯),全栈真跑(需对方 session 的 PG 环境就绪后,或另起隔离环境)。
5. 验收 = 真流量点亮下,老板 3 秒看懂"系统在沉淀可归因的真实生产资产"。

---

## 6. 给学者的拍板清单

| # | 决策点 | 选项 | 我的推荐 |
|---|---|---|---|
| **D1** | **day-one 数据路径**(最重要) | ①受控真流量点亮 / ②明确标注示例空态 / ③合成假聚合当真 | **①(辅以②做空态);❌③** |
| **D2** | 第一个价值出口选哪个 | **C 快照+样本墙** / A 全看板 / B skill 蒸馏 | **C**(A 留作后续演进,B 待有数据量后再议) |
| **D3** | 样本墙是否对外/给老板展示真实脱敏内容 | 是(先补保留期清理)/ 先内部只读 | **先补内容表保留期清理,再对外** |
| **D4** | 是否现在开实现任务书 | 开 / 暂缓 | 由你拍(本任务只产设计) |

---

## 7. 红线遵守声明
全程只读:未起任何服务、未连任何数据库、未占任何端口、未改任何代码、未 `git add/commit/push`、未动任何分支/worktree/main、未提取任何凭据、未实现任何方案、未碰 B.3。`git status` 仅本目录 `_review/selling_value_output_20260620/` 新增(审查包不入 git)。与"修房子"session 物理隔离,零干扰。

## 附:workflow 透明度(诚实记账)
17 个只读 subagent,~907k tokens,~9.4 分钟。**已知噪声**:12 个打分单元中 2 个返回退化占位(`parallel[6]` schema 校验失败;另一格输出 "Risk one"/"point"/`DimensionMinimal` 占位)。§2 评分表是我对评估器**定性原文**的去噪重判(与裁判排名 C>A=B 一致),非机械搬运退化数字。§3.3 是我对裁判"灌假数据"倾向的修正。
