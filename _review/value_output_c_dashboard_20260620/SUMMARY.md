# 价值出口 C 看板工程 · 全栈真跑收口 · SUMMARY

> 任务:G_VALUE_OUTPUT_C_DASHBOARD ｜ 日期:2026-06-20 ｜ 分支:`wujie/trunk` @ `eca1b65c`(未提交)
> 工作区:`D:\sub2api-trunk` ｜ 全本地 throwaway、¥0、未连真库、未真调上游、未用真 key、未 push、未碰 main/旧路径/worktree、**未改任何已提交产品码的运行逻辑**
> 执行:Claude Code(Opus 4.8, ultracode)｜ Dynamic Workflow 对抗复查 + 全栈真跑

---

## 结论(红线放最前)

**看板工程 = 技术就绪(壳子能跑)= PASS。** 2 个只读 admin 端点(stats/samples)+ 2 个裸 SQL 读方法 + 1 个前端视图(快照卡 + 7 日 sparkline + 样本墙)全部实现并**全栈真跑实证**:本地原生 PG + mock 采的测试数据下,聚合正确、样本脱敏正确、rune-safe 截断正确、`is_live` 二态正确、adminAuth 鉴权正确。

**红线全过**:未碰主路由 `/v1/messages`、未碰写客户端响应语句、未碰采集热路径、未改已提交补线 `eca1b65c`(含 `wire_gen.go:188` 采集器注入**逐字保留**)。新端点全部挂 adminAuth(未鉴权 → 401 实测)。

**对抗复查抓出并修掉 1 个 HIGH 缺陷**(时区错位的当日边界,见 §3),修后真跑复验通过。

**⚠️ 这≠"能给老板看真护城河"**:还差两步单独决策——**D1 真数据来源**(受控环境跑真流量采真数据)+ **D3 保留期清理 + 样本人工过一眼**(脱敏有残留)。本任务**不做**(见 §6)。

无停止条件触发。

---

## 1. 改动清单

### 新增产品文件(4)
| 文件 | 作用 |
|---|---|
| `backend/internal/handler/admin/generation_content_handler.go` | 只读 handler:GetStats / GetSamples(rune-safe 截断、二态、7 日零填充) |
| `frontend/src/api/admin/generation_content.ts` | 前端 API client(2 端点 + TS 类型) |
| `frontend/src/components/admin/generation-content/CaptureSparkline.vue` | 轻量内联 SVG sparkline(零依赖) |
| `frontend/src/components/admin/generation-content/ContentWall.vue` | 样本墙 + 诚实空态(不伪造样本) |
| `frontend/src/views/admin/GenerationContentView.vue` | 看板视图(快照卡 + sparkline + 样本墙 + 二态徽标) |

### 修改产品文件(净增式,无逻辑改写)
- `backend/internal/service/generation_content.go`:接口加 `GetCaptureStats`/`GetRecent` + 3 个结果类型(**Collect/SetGenerationContentCollector/CollectGenerationContent 运行逻辑未动**)。
- `backend/internal/repository/generation_content_repo.go`:加两裸 SQL 读方法(纯 SELECT;唯一写入仍是原有 Create)。
- `backend/internal/server/routes/admin.go`:`registerGenerationContentRoutes`(admin 组下,自动继承 adminAuth)。
- DI 接线:`handler/handler.go`(+字段)、`handler/wire.go`(+参/字面量/provider)、`repository/wire.go`(+provider)、`cmd/server/wire_gen.go`(**手改**:+repo/handler 构造 2 行 + ProvideAdminHandlers 实参;**未跑 wire codegen,line 188 逐字保留**)。
- 测试 mock 修:`generation_content_collector_test.go` / `_panic_test.go`(接口扩了 → 给两 mock 补空实现)。
- 前端接线:`api/admin/index.ts`(注册)、`router/index.ts`(路由)、`AppSidebar.vue`(导航项)、`i18n/locales/{en,zh}.ts`(en+zh 键块)。

### diff 概览
`git diff --stat` = 14 改 + 4 新,**仅落在上述文件**;`git diff --name-only | grep gateway` = 空(主路径零触碰)。

---

## 2. 本地验证证据(全栈真跑)

环境:本地原生 PG17.5@5433(zonky,SQL_ASCII)+ redis@6380 + mock@9099 + 本任务新建 server.exe@8090(`content_capture.enabled=true`,session TZ=Asia/Shanghai)。证据在 `evidence/`。

| # | 项 | 实测 | 判定 |
|---|---|---|---|
| ENV | 环境起栈 | PG/redis/mock/server 全起;DB 自动迁移建 `ai_generation_content` | ✅ |
| 采集链路仍通 | 接口扩展未破坏采集 | 真 gateway 请求(走 mock)→ 落 1 行(`mock-req-1`,全归因 user/group/account,prompt 178/resp 300) | ✅ |
| stats 聚合 | 数字对 | `captured_today=3, captured_week=8, distinct_employees=4, distinct_teams=3, distinct_models=3, total_bytes=77084, daily_rate=8/7`;与 sqlrunner 直查**逐字一致** | ✅ |
| 7 日序列 | 零填充 + UTC 对齐 | `06-14..06-20` 共 7 点,`06-16=0`(零填充),序列和=8=本周 | ✅ |
| samples 脱敏 | 显脱敏非原文 | gateway 行客户端响应是原始电话 `+86-10-87654321`,但样本墙 prompt/response 显 `[PHONE]`/`[已脱敏]`(读 `*_redacted`,非原文) | ✅ |
| samples 截断 | rune-safe + 标记 | 长中文样本 120/80 rune 处**整字截断无乱码**`truncated=true`;上游 `response_truncated` 行 `truncated=true` | ✅ |
| samples 归因/排序 | JOIN + DESC | employee/team 名经 LEFT JOIN 正确解析;`created_at DESC` 正确 | ✅ |
| adminAuth | 仅 admin | 未带 key → **401**;带 `x-api-key: <admin_api_key>` → 200 | ✅ |
| 空态 | 二态 + 不造假 | TRUNCATE 后 `/stats` 全 0 + 7 个 0 点 + `is_live=false`;`/samples` 空数组 + `is_live=false`,**无伪造聚合** | ✅ |

证据文件:`stats.json` / `samples.json` / `stats_empty.json` / `samples_empty.json` / `tz_fix_demo.txt` / `seed_extra.sql` / `synthetic_rows.sql`。

---

## 3. 对抗复查(Dynamic Workflow)结论 + 修复

4 维并行复查 + 逐条对抗验证(6 agent):
- **handler-contract / frontend / redline-compliance = CLEAN**(JSON 契约逐键对齐前端 TS;i18n en+zh 全齐;主路由/采集热路径/line 188 全未碰;端点 admin-gated;纯 SELECT)。
- **backend-sql-wire = 1 HIGH(已修)**:`GetCaptureStats` 的"当日/7日窗口"边界用 `date_trunc('day', NOW() AT TIME ZONE 'UTC')` 产出**裸 timestamp**,与 `created_at`(timestamptz)比较时被 Postgres 按**会话 TZ**(本仓库 DSN 默认 `Asia/Shanghai`/UTC+8)强转 → 边界早 8 小时,当日计数虚高、序列窗口与 handler 的 UTC 零填充错位。
  - **修复**:两处边界尾缀 `AT TIME ZONE 'UTC'` 回锚 timestamptz(同仓库既有正确惯例 `ops_repo_preagg.go:33`/`account_repo.go:1888`)。
  - **实证**(`tz_fix_demo.txt`):buggy 有效边界 `2026-06-20 00:00 +0800`(=06-19 16:00 UTC)vs fixed `2026-06-20 08:00 +0800`(=06-20 00:00 UTC),正好差 8h;修后真跑 `captured_today=3` 正确。

INFO(不修):`/samples` 也返回 `is_live` 但视图只用 `/stats` 的 `is_live` 驱动徽标——冗余无害,各端点自描述,保留。

---

## 4. 措辞纪律落地
英雄指标全用真实可数事实(已采 N 条 · M 员工 × K 团队 × P 模型 · 近 7 日 +Y/日);字节叙述为"已沉淀内容体量"(`totalBytes` 文案,**未宣称"X MB 可复用资产价值"**);空态显"○ 示例(未开启采集)"+ 诚实横幅,**绝不伪造聚合**(空态实测全 0)。

---

## 5. 提交建议
- 建议提交**仅产品码**(上述 14 改 + 4 新)。**勿提交 `_review/`**(内含 throwaway 测试夹具:`postgres/postgres@5433`、committed `.exe`、admin 测试 key `admin-localtest-key` 等;非生产凭据,但不应入库/推送)。
- 一条建议提交信息:`feat(dashboard): 价值出口 C 只读看板(护城河快照卡+样本墙,读 ai_generation_content;flag 无关,adminAuth)`。
- **勿 push**(origin = 开源上游)。
- Codex 复查按学者节奏攒到批量来;**灰度/对外前那次 Codex 复查别省**。

---

## 6. 🔴 对外/给老板展示前必做(本任务不做,留学者拍板)

1. **🔴 脱敏有残留**:已知 `RedactJSON` 对全字母凭据、长 token、正则外 PII 有漏网(见 M1A/支线)。样本墙**对外或给老板展示前,必须人工过一眼**。
2. **🔴 D3 保留期清理 + 人工过一眼**:`ai_generation_content` 尚无保留期/清理任务(M1A 决策 2 押后)。长期对外前**必须先补内容/PII 保留期清理**(与计费保留解耦)。
3. **D1 真数据来源**:本任务只用本地 mock 测试数据验证;给老板看真护城河需在**受控环境跑真流量**采真数据(分阶段开 `content_capture.enabled`)。

**验收口径**:本地测试数据下读/聚合/展示/脱敏 + 二态 + adminAuth + 全栈真跑过 = **看板技术就绪**。**这≠"能给老板看真护城河"**(还差 D1 + D3)。

---

## 7. 偏差登记(不影响结论)
1. 支线 §4 写"复用 MiniTrendChart"但该组件不存在 → 新建轻量 `CaptureSparkline.vue`(视图工作范围内,非超范围)。
2. 验证环境复用 M1B 修房子那套(zonky PG/SQL_ASCII、自建 mock/sqlrunner);PG 数据簇重新 initdb,server.exe 新建(含 TZ 修复)。均 throwaway、repo 外、已清理。
3. 为验 adminAuth + 归因 + 去重,种子在 M1B 基础上加了 admin 用户 + settings admin key + 额外 users/groups + 合成采集行(`evidence/*.sql`),全 throwaway。

## 8. 异常 / 停止登记
- 未触发任何停止条件(未需接真实生产数据/碰真库/push/真 key/真付费;支线 demo 与真实 schema 仅 1 处对不上=MiniTrendChart 不存在,在范围内新建解决)。
- teardown 已停 server/PG/redis/mock,删 `D:\valc_pg` 与 server 工作目录;`git status` 仅产品码改动 + `_review/` 未跟踪。
