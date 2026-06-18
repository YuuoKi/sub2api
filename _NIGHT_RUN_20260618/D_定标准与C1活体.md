# 阶段 0+1 审查包 · 定标准 → C1 进程内活体（Sub2API · ¥0 · keyless · 未开真实门）

> 2026-06-18 · 分支 `night-run/20260618-D-c1-alive`（off 阶段C tip `47cf1146`）· Claude Code（opus-4-8）
> 原则：只读定标准 + 进程内 route 活体（真 gin 路由+handler+service+worker，内存 repo，stub 鉴权）。**不起 Docker/PG、不铸/碰任何 key、不开真实门、不真实付费、不 push、不碰 :8080 在跑实例。**
> 用户裁定（本轮范围）：**只走进程内 route harness（不起 Docker/PG）+ 不铸任何 key**；跨进程 §5 curl 与浏览器活体推迟阶段2（老板在场+环境就绪）。

---

## 0. 一句话结论

- **阶段0（定标准）完成**：视频网关 = **100% fork 自加 bespoke**（上游 sub2api 是 LLM/图片聚合代理，**无任何视频网关**，故「切回沿用上游」无对象、亦无轮子可造）。Seedance 字段契约对照官方/第三方文档**逐项定标**：顶层 `ratio`/`duration`/`resolution` + `content[]` + 轮询 `content.video_url` **✅ 对齐**（首枪真验 + 文档佐证）；唯一**❌-likely**：v2v 视频参考字段（我方 `video_url` vs 第三方提示 `type:"video"`+`role:"reference_video"`）——**未改码**，升级为「明确待坐实+具体假设」入待授权。
- **阶段1（C1 活体）完成（keyless 等价物）**：新增 route 级活体测试，把 QCanvas/§5 的真实请求形态**跑过真 gin 路由→handler→service→worker**，mock 任务 `queued→submitted→running→succeeded`，回 `result_url`，**竖屏 `9:16` 全程贯通到候选**，**真实门全程关**（`mock_only=true`/`real_dispatch=0`/`boundary=api-key-video-mock-only`）。这是 §5 跨进程 curl 的 ¥0、无 key、无 Docker/PG 等价物。
- **真实门一键就绪**：单一主开关 `SUB2API_VIDEO_REAL_SMOKE_ENABLED=1`（+6 道 fail-closed 门）+ 预算门仅在 `per_call_budget>0` 时上臂——**两道默认皆关**，已有测试证明门关、开门命令文档化（**未翻开**）。
- 验证：`go build`/`go vet` clean；routes（含新 C1 活体）/handler C1/service 契约+预算 全绿。**未真实调用、未碰密钥、未部署、未 push。**

---

## 1. 阶段0 · 上游 vs 自加 判定（坐实）

| 问 | 答 | 证据 |
|---|---|---|
| 上游 sub2api 有视频网关吗？ | **无**。上游是 LLM/图片 API 聚合代理（`/v1/messages`、`/chat/completions`、`/v1/responses`、`/images/generations`）。 | `git log origin/main` 无任何视频提交；上游 README/路由无 video。 |
| `video_gateway_*` 是沿用上游还是自加？ | **100% fork 自加 bespoke**。`VideoAdapter` 接口本地定义（`video_gateway_adapter.go` 顶部），三实现 mock/seedance/kling 内联同文件；首现于本仓 `4c5de849`（P0 mock）。 | 接口/实现/路由（`server/routes/gateway.go:115-124`）全在 fork 内，无上游基类/契约。 |
| 「切回用现成」是否可行？ | **不适用**——**不存在上游视频适配可切回**。结论本身即阶段0 合格项之一：不是「另造轮子」，是「上游无此轮、本就只能自建」。 | 同上。 |

> 即任务书 0.1 的判定 = 「沿用上游 + N 处补丁」**不成立于视频线**；正确表述：**视频线全为自建，无上游可沿用**。LLM/图片线才有上游 channel 抽象（本轮不碰）。

---

## 2. 阶段0 · Seedance 字段对照表（定标准）

文档源（best-effort 联网，只读）：火山方舟官方 Seedance 2.0 API 参考（`volcengine.com/docs/82379/1520757` 等，**SPA 动态页 WebFetch 仅得导航壳**，记此渲染卡点）；以 **WebSearch 官方摘要 + 第三方镜像文档**（apidog/laozhang）佐证；create/poll 主链已被 **2026-06-17 真打首枪**坐实。

| 字段 | 上游现成 | 我方当前（file:line） | 官方/第三方标准（出处） | 判定 | 处理 |
|---|---|---|---|---|---|
| 路线一致（base_url↔model 同线） | 无 | `ark.cn-beijing.volces.com/api/v3`（adapter.go:158）+ 默认 `doubao-seedance-2-0-260128`（:163） | 火山 Ark 上 `doubao-seedance-2-0-260128` 同线（官方 search 摘要） | ✅ | 无 |
| 传参形态（顶层 JSON vs content 内嵌 `--rt`） | 无 | **顶层** `model`+`content[]`+顶层 `ratio`/`duration`/`resolution`（adapter.go:190-216） | 顶层 `model`/`content`/`ratio`/`resolution`/`duration`（laozhang 镜像） | ✅ 顶层形态，非 `--rt` 内嵌 | 无 |
| `ratio` 竖屏 | 无 | adapter.go:214-216 顶层 `ratio`，`normalizeSeedanceRatio` 竖屏→`9:16`（B1 已修，曾误发 `aspect_ratio`→被忽略→默认16:9） | 顶层 `ratio`（"16:9"…1:1~21:9，官方 search + laozhang） | ✅ 字段名+取值对 | **真实竖屏出图**待 1 次付费坐实（待授权①） |
| 时长 `duration` | 无 | adapter.go:208-210 顶层 `duration`（秒, int） | `duration` integer seconds（laozhang）；时长 2-12s（官方 search） | ✅ 字段名对（请求侧仅响应回显+第三方坐实） | 低风险，可随①一枪坐实 |
| `resolution` | 无 | adapter.go:211-213 顶层 `resolution`（720p/1080p） | `resolution` 480p/720p/1080p（官方 search + laozhang） | ✅ | 无（B2 分辨率分层 poll 已配套） |
| 图生视频 `image_url` | 无 | adapter.go:174 `type:"image_url", image_url:{url}`（**无 role**） | `type:"image_url", image_url:{url}` + `role` first_frame/last_frame/reference_image（laozhang/官方 search） | ⚠️ 字段名对，**缺 role**（首帧/尾帧/参考图区分） | i2v role 细化待坐实；本轮 t2v 为主线，i2v role 非本轮范围 |
| **v2v 视频参考** | 无 | adapter.go:187 `type:"video_url", video_url:{url}`（镜像 image，**自标 UNVERIFIED** :181-185） | 官方页未渲染坐实；**第三方提示视频参考应为 `type:"video"`+`role:"reference_video"`**（laozhang，inferred） | **❌-likely 形态可能不符** | **不改码**（第三方推断≠官方直引，臆改破坏 B3 草案风险）→ 升级待授权②：需**官方 v2v 文档或 1 次真实 v2v 调用**坐实确切 type/字段名 |
| 轮询路径 + 状态机 | 无 | GET `/contents/generations/tasks/{id}`；解析 `status`+`content.video_url`+`error.message`；`NormalizeStatus` queued/submitted/running/succeeded/failed/cancelled | 异步 submit POST→poll GET→succeeded，输出 `content.video_url`，状态 queued/running/succeeded/failed/expired/cancelled（laozhang+官方 search） | ✅（首枪真验 `content.video_url`） | 无 |
| Files API / `@Video` 引用 | 无 | 未用 Files API（直接传 URL） | 官方页未渲染坐实是否需先上传 + `@Video` 语法 | ❓ 未坐实 | 与 v2v② 同批待坐实 |

**无 ❌ 悬空**：除 v2v（❌-likely，已列具体假设+待授权②）外全 ✅/⚠️；⚠️/❓ 均落「待真实坐实」并指明坐实方式。**阶段0 允许零改码**——本轮据此**未改任何 adapter 字段**（B 阶段已修的 `ratio` 仍对；v2v 按铁律不臆改）。

---

## 3. 阶段1 · C1 进程内活体（核心交付）

### 3.1 新增测试（唯一改动）
`backend/internal/server/routes/api_key_video_gateway_c1_alive_test.go`（新增，package `routes`，复用既有 harness 脚手架）：
- `newAPIKeyVideoGatewayC1AliveRouter`：镜像既有 `newAPIKeyVideoGatewayTestRouter`，但**额外返回 `*VideoGatewayService`**，以便**同步驱动 worker**（`svc.ProcessRunnableTasks`，非后台 goroutine→确定性、无 flake）。handler 与 tick **共享同一 service+repo**，tick 推进的任务对下一次 HTTP GET 可见。
- `TestAPIKeyVideoGatewayC1MockAliveReachesSucceeded`：发 QCanvas/§5 真实 body（`provider=mock`,`task_type=text_to_video`,`prompt`,`model`,`metadata{nodeId,nodeLabel}`,**`aspect_ratio:"9:16"` 竖屏**）→ 真路由受理 → tick 推进 → 轮询到 `succeeded`。

### 3.2 活体输出（`go test -run TestAPIKeyVideoGatewayC1MockAliveReachesSucceeded -v`）
```
POST /v1/video/tasks (mock, 9:16) → 201 id=1 status=queued aspect_ratio=9:16 mock_only=true real_dispatch=0
  GET /v1/video/tasks/1 after tick 1 → status=submitted result_url=""
  GET /v1/video/tasks/1 after tick 2 → status=running   result_url=""
  GET /v1/video/tasks/1 after tick 3 → status=succeeded  result_url="https://mock.sub2api.local/video/1.mp4"
LIVE candidate after 3 tick(s): status=succeeded result_url=https://mock.sub2api.local/video/1.mp4 aspect_ratio=9:16 mock_only=true real_dispatch=0 boundary=api-key-video-mock-only
--- PASS
```
断言：`status=succeeded` + `result_url` 非空（QCanvas `normalizeTaskRecord` 的 resultUrl 源）+ `id`/`provider`/`error_message`/`upstream_task_id` 全在 + **`aspect_ratio=9:16` 贯通** + `mock_only=true`/`real_provider_dispatch_count=0`/`boundary=api-key-video-mock-only` + `repo.realProviderTaskCount()==0`。

> 「骨架从契约证明 → 真活体跑通一次」达成：**真 HTTP 栈跑通到 succeeded**，进程内、¥0、零外部依赖、零 key。这是 §5 两条跨进程 curl 的 keyless 等价物（同一份 router/handler/service/worker 代码路径）。

### 3.3 接缝清理 —— 一处「checked & cleared」，零改码
- **预算门 vs mock 路径（重点核查）**：`CreateAPIKeyMockOnlyTask`→`CreateTask`（service.go:372），`CreateTask` 在 `s.budget!=nil` 时跑预算门（:613）且**无 mock 旁路**。曾疑：生产若上臂 `StaticBudgetGuard(0)` 会 fail-closed 连 mock 也拦。**核实结论：不会**——`ProvideVideoGatewayService` 仅在 `per_call_budget>0` 时注入 guard（wire.go:535），**默认 0 → budget 保持 nil → 预算门跳过 → mock 正常**（专有测试 `TestProvideVideoGatewayServiceUnarmedWhenBudgetZero` 钉死）。故我测试用 nil budget **如实匹配生产默认 mock 行为**，无假绿、无生产接缝。
- 其余接缝（字段序列化/鉴权头/`normalizeTaskRecord` 回填/轮询窗）在进程内 harness 已对齐（既有+C1 契约测试绿），新活体测试**未暴露断裂** → **零改码**（铁律：不为「做点什么」而改）。
- **CORS / 跨进程超时**：进程内 harness 本质跑不到（属浏览器/跨进程）→ 明确列阶段2 浏览器活体项，不在本轮伪造。

---

## 4. 真实门一键开关（已就绪 · 验证 + 文档化 · **未翻开**）

- **主开关（唯一）**：`SUB2API_VIDEO_REAL_SMOKE_ENABLED=1`（`video_gateway_adapter.go:570`）。配套 6 门（`seedanceSmokeGateBlockedReasons` :568）：provider metadata `single_smoke_authorized`/`real_smoke_authorized`、脱敏事件日志 `SUB2API_VIDEO_REDACTED_EVENT_LOG`、媒体 allowlist `SUB2API_VIDEO_URL_ALLOWLIST`、显式 seedance 模型、时长 1-5s、发请求前脱敏自检。**任一缺失即 fail-closed 拒发**。
- **预算门（第二道，独立）**：仅 `video_gateway.per_call_budget>0` 时上臂（`wire.go:535`）；默认 0 → 不上臂（生产零变化），真实门时由其挡超预算暴冲。
- **已证（引用既有测试，无需新写）**：
  - 门关（route 级）：`TestAPIKeyVideoGatewaySeedanceTrialBlockedWithoutGate / …DurationTooLong / …MissingAuthorization / …MissingEventLog`。
  - 门开（凑齐 env，`mock://` 无真实网络）：`TestAPIKeyVideoGatewaySeedanceTrialSuccess`。
  - service 级：`TestFormASeedanceHappyPathInMemory`（满门→succeeded，localhost mock-Ark）。
  - 本轮 C1 测试**独立旁证**：mock 路径无 env → `real_dispatch=0`、`realProviderTaskCount()=0`。
- **开门 runbook（阶段2 老板在场执行，本轮一项未设）**：
  ```
  # 注入真实 seedance provider account(含真 key) + 置 provider metadata single_smoke_authorized=true
  SUB2API_VIDEO_REAL_SMOKE_ENABLED=1
  SUB2API_VIDEO_REDACTED_EVENT_LOG=<脱敏日志路径>
  SUB2API_VIDEO_URL_ALLOWLIST=volces.com
  # 然后发: provider=seedance, trial_mode=tiny_real, 显式 seedance 模型, duration 1-5s
  # 可选: video_gateway.per_call_budget=<¥上限> 上臂预算门
  ```
  > **本轮全程未设 `SUB2API_VIDEO_REAL_SMOKE_ENABLED`、未带 `-tags=realsmoke`、未接真实 BaseURL/key、未发任何 trial。**

---

## 5. 竖屏 `9:16` 全程贯通证据
- 请求侧（Seedance adapter）：`aspect_ratio→ratio` 翻译 + 竖屏映射，既有 `TestSeedanceCreateSendsRatioWithOrientationMapping`（捕获请求体，断言 `ratio:"9:16"`、不再发 `aspect_ratio`）。
- 端到端（mock 候选链路）：本轮 C1 活体测试 —— `aspect_ratio:"9:16"` 经 POST→service `CreateTask`（service.go:604 持久化 AspectRatio）→记录→GET 响应 `aspect_ratio=9:16`（见 §3.2 输出）。**证「竖屏参数贯通到候选」**；「Ark 真出竖屏」属阶段2 真打（待授权①）。

---

## 6. 验证命令（全绿，未触发真实网络）
```bash
cd backend
go build ./...                                                              # ✅
go vet ./internal/server/routes/ ./internal/handler/ ./internal/service/    # ✅
go test ./internal/server/routes/ -run APIKeyVideoGateway -count=1          # ✅ (含新 C1 活体 + 11 既有)
go test ./internal/handler/ -run C1 -count=1                                # ✅
go test ./internal/service/ -run 'Ratio|Seedance|FormA|VideoPoll|MaxPollAttempts|EffectiveTaskTimeout|ProvideVideoGatewayService|StaticBudgetGuard' -count=1   # ✅
```
**未触发**：`-tags=realsmoke` 未编译；`SUB2API_VIDEO_REAL_SMOKE_ENABLED` 未设；零网络出站。

---

## 7. 阻断 / 推迟清单（阶段2 runbook · 本轮不执行）
| 项 | 为何推迟 | 阶段2 前置 |
|---|---|---|
| 跨进程 §5 两条 curl | 需运行实例 + api-key（铁律禁铸/碰）+ PG（Sub2API Postgres-only，无 SQLite，Docker daemon 未起，无原生 PG） | 起隔离 Sub2API（PG）+ 一把 key |
| 浏览器点击→候选区活体截图 | QCanvas hono-api 亦 Postgres-only（Prisma，`DATABASE_URL` 强制）+ R2 + web 构建 | 起 QCanvas PG + hono-api(:8788) + web(:5173) + 三个 env |
| CORS 跨进程校验 | 进程内 harness 跑不到 | 同浏览器活体 |
| 真实门翻开 + 竖屏真实出图（B1）/ v2v 字段坐实（B3,②） | 真 key + 真实付费 + 老板在场 | 待授权①②（个位数 ¥） |

> 文档站 SPA WebFetch 只得导航壳（渲染卡点）→ 官方 v2v 字段未从官方页直引，用第三方镜像佐证 + 列待坐实，未死磕。

---

## 8. 铁律合规逐条
- ✅ 未 push（origin=开源上游 Wei-Shaw，保持不变；新分支 `night-run/20260618-D-c1-alive` 本地、无 upstream）。
- ✅ 未碰密钥（未铸/读/打印/落库任何 key/token；按用户裁定走 keyless）。
- ✅ 未开真实门、未真实付费（无 env、无 `-tags=realsmoke`、无网络）。
- ✅ 未起 Docker/PG、未碰 :8080 在跑实例、未部署、未 DB 迁移。
- ✅ 未碰 Kling/图片/文本、未美化 UI；阶段0 据「允许零改码」**未改 adapter 字段**（含按铁律不臆改 v2v）。
- ✅ git 卫生：仅显式 `git add` 新增 2 文件（测试 + 本审查包）；**无 `git add .`/`reset`/`clean`/`rebase`**；前序未提交改动（`M .gitignore`、`?? deploy/docker-compose.b1-emptybrake.yml`）全程不动、不提交。

---

## 9. 本阶段产出 & 提交
- 新增 `backend/internal/server/routes/api_key_video_gateway_c1_alive_test.go`（C1 进程内活体，无 DB/无 key/无网络）。
- 本审查包 `_NIGHT_RUN_20260618/D_定标准与C1活体.md`（单一锚点）。
- QCanvas 侧**零改动、未切分支**（解 mock 纯三个 env，已就绪；本轮不起 QCanvas 进程）。
- `git add` 显式逐一。**未真实调用、未碰密钥、未部署、未 push。**

---
*物证以本审查包 + git log + go test/vet 输出为准。局部 READY ≠ 产品 READY。**进程内活体 ≠ 跨进程/浏览器活体**（后者阶段2）。*
