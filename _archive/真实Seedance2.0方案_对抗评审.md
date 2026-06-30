<!-- 生成: 只读勘察 workflow seedance-realchain-recon / 2026-06-15 / 零真实调用零改码 -->

# 红队对抗性评审：真实 Seedance 2.0 全链路接入方案（阶段0 单条冒烟）

## 0. 评审有效性声明（必读限制）

**本次评审存在重大可信度限制，结论须打折看待：**

1. **跨家族（GPT/codex）复核未执行。** codex CLI 未安装，无法做真正的异源交叉验证。本评审为单模型（Claude）自我对抗，存在与方案作者同源的盲区。
2. **方案引用的"A 侧"文件在本仓库根本不存在。** 我实测确认：`apps/hono-api/`、`sub2api.routes.ts`、`sub2api.video-mock-gateway.service.ts`、`studioV2RealTaskAdapter.ts`、`studioV2BusinessBridge.ts`、`day0CandidateStore.ts`、`studioToWujieAdapter.ts` —— **全部不在 `D:\...\02_source\sub2api` 内**（`find` 仅命中一个 git tag 名）。本仓库 frontend 是 Vue admin（`VideoCreateTaskView.vue` 等），不是方案描述的 QCanvas React 栈。
   - 后果：**§1 链路图、§2.b、§4-B、§5（wujie 边界）、§6（vendor auto 隐患）、§3.3 的 A 侧落点全部无法在此环境验证。** 这些断言的可信度 = 三份 recon 报告的可信度，而 recon 报告我也无法核对。方案自称"recon 自洽、引用了所有 file:line 落点"——但落点指向一个**不在场的代码库**。这是评审的最大风险。
3. 后端（Go）侧落点我已逐条核对，**基本属实**（详见下文）。所以本评审对后端的判断置信度高，对 A 侧的判断置信度低。

**我已实测核对并确认属实的后端事实：** smoke gate 五条件（`video_gateway_adapter.go:358-376`，逐字符匹配方案 §2.a-S3）、key 注入与解密（`video_key_encryptor.go`、`worker.go:141`）、daily 1/user/day 限制（`video_gateway_service.go:444-449`）、worker 串行循环（`worker.go:73-162`）、**无预算/并发熔断**（grep 确认 service 层无 budget/in-flight 校验）。

---

## A. 架构风险

| 严重度 | 弱点 | 落点 | 修复 |
|---|---|---|---|
| **BLOCKER** | **创建请求从不向 Ark 发送 `duration`。** smoke gate 校验 `task.Duration ∈ [1,5]`（`adapter.go:372`），但 `CreateTask` 构造的 payload（`adapter.go:163-169`）只含 `model`、`content`、`negative_prompt`——**没有 duration、没有 resolution**。意味着：闸门校验的 5 秒上限是**纸面约束**，真实生成时长完全由 Ark 默认值决定。若 Ark 默认 10s/更长，单条成本直接翻倍以上，且与 §3 全部成本估算脱钩。 | `adapter.go:163-169` vs `:372` | 在 create payload 显式注入 `"duration": task.Duration`（及 resolution/aspect_ratio）；并在收到 Ark 响应后**回读实际计费时长**写入 task。冒烟前必须修。 |
| **HIGH** | **taskId 契约链路在本仓库不可验证，且方案 §8.5 自承存在 int64↔string、`id` vs `task_id`、丢弃 `upstream_task_id` 三处映射差异。** 后端 `CreateTask` 返回 `parsed.ID`（string，`adapter.go:202,217`），但 Ark 真实响应字段是否为 `id`（而非 `task_id`/`data.id`）**未经真实响应核对**——adapter 是"skeleton/未验证真实响应结构"。若字段名不符，`UpstreamTaskID` 为空，poll URL 变成 `/tasks/`（`adapter.go:251`），轮询永远拿不到结果→任务挂到超时失败。 | `adapter.go:201-217,251` | 冒烟时第一件事是抓取并核对 Ark create 的真实 JSON 结构，确认 `id`/`status`/`content.video_url` 字段名。这是冒烟的**主要目的**之一，不能假设已对。 |
| **HIGH** | **轮询竞态：worker tick 与 timeout 判定基于 `task.CreatedAt`，但 submit 与首次 poll 之间无最小间隔保证。** worker 每 tick 用 `interval` 作为 ctx 超时（`worker.go:80`），若 `poll_interval_seconds` 设得过小，可能在 Ark 任务尚未 ready 时高频轮询；更危险的是 `processTask` 对 `Submitted/Running` 都会 poll（`worker.go:154-157`），而 submit 本身也在同一 worker——首 tick submit、次 tick poll，间隔 = 整个 tick interval，**poll 间隔无独立下限控制**，可能对 Ark 造成高频 GET。 | `worker.go:80,149-160` | 冒烟阶段把 `poll_interval_seconds` 设保守（≥10s），并确认 worker tick interval ≥ poll interval。 |
| **MED** | **dry-run→real 切换泄漏（无法在此验证，但设计上脆弱）。** 方案承认唯一离开 dry-run 的开关是**浏览器本地** `studioV2RealChainReady`（query param 或 localStorage），无 env、无服务端二次确认。query param 可被分享/缓存/书签固化，localStorage 在共享机器上持久。后端 smoke gate 是真正的安全网——但 A 侧"开关"本身不是闸门级控制。 | `studioV2RealTaskAdapter.ts:359-363`（**不在本仓库**） | 不要依赖前端开关作为成本/真实调用的边界；真正的边界必须是后端五条件 gate + daily limit。前端开关仅作 UX。 |
| **MED** | **wujie 边界"可强制执行"无法证实。** 方案称所有路径都带 mock/dry-run 标记故进不了 wujie——但这是基于不在场代码的断言。一旦切到 real 路径，`isMock/dryRun` 标记理应变 false，那么 `studioToWujieAdapter` 的 `not-persistable` 判定还成立吗？方案未说明 real 成功路径下 wujie gate 凭什么仍 OFF。 | `studioToWujieAdapter.ts:102-150`（**不在本仓库**） | 冒烟验收必须实测 `wujieWriteCount:0` 在 **real-success** 路径下（非 mock 路径），否则 §5 的边界主张未被证明。 |

---

## B. 安全漏洞

| 严重度 | 弱点 | 落点 | 修复 |
|---|---|---|---|
| **HIGH** | **A↔B 鉴权用共享员工 key（`SUB2API_API_KEY`），用户 JWT 不传递到上游。** 这是"auth-contract split"的设计核心——意味着**任何能到达 Hono `/sub2api` 路由的请求都以同一员工身份打 Go 后端**。daily 1/user/day 限制按 `p.CreatedBy` 计（`service.go:430`），但 `CreatedBy` 由谁填？若 Hono 用员工 key 调 Go，Go 看到的 user 是员工身份还是透传的前端 user？若是员工身份，则 **1/day 限制变成全公司 1/day 或形同虚设**（取决于 CreatedBy 来源）。无法在此验证 CreatedBy 注入链。 | `service.go:430`；A 侧 `gateway.service.ts`（不在场） | 冒烟时必须确认 `video_tasks.created_by` 填的是真实终端用户标识，且 daily limit 据此生效。否则限速基础被 auth split 架空。 |
| **HIGH** | **Hono 视频代理的 SSRF/open-proxy 风险无法评估（代码不在场）。** 方案 Q4 描述 upstream URL = `${SUB2API_BASE_URL}/v1/video/tasks`，base 来自 env，看似固定。但 create 请求体含 `ReferenceImageURL`（`adapter.go:159-161`）会被**后端原样转发给 Ark 作为 image_url**，且 poll 返回的 `result_url`（`content.video_url`）被前端直接当可播 URL 写入 Day0。**这是两个外部可控 URL 的信任边界**。 | `adapter.go:159-161`（image_url 转发）；`adapter.go:292`（result_url 信任） | (1) 对 `ReferenceImageURL` 做 allowlist/scheme 校验，禁止 `file://`、内网 IP、`localhost`。(2) `result_url` 在前端播放/Day0 写入前校验为 https + 预期 CDN 域名，不信任 Ark 返回的任意 URL。 |
| **HIGH** | **result_url 完全被信任。** poll 直接把 `parsed.Content.VideoURL` 塞进 `task.ResultURL`（`adapter.go:292`），前端据 `resultSource='sub2api-real-trial'` 写 Day0 并播放。若 Ark 响应被篡改（MITM）或返回非预期 URL，前端会加载它。 | `adapter.go:292` | result_url 域名 allowlist 校验，放在 poll 解析后、写 DB 前。 |
| **MED** | **encryption key 的 dev fallback 到 totp key（已知，但仍是生产风险）。** `video_key_encryptor.go:16-20` 确认：`video_gateway.encryption_key` 为空时静默 fallback 到 `totp.encryption_key`，仅打 Warn。生产若漏配，真实 Seedance 密钥用 totp 密钥加密——**密钥域混淆**，且 Warn 易被忽略。 | `video_key_encryptor.go:16-20` | 生产环境应让 fallback **硬失败**（return error）而非 Warn。冒烟前确认 `video_gateway.encryption_key` 已显式配置。 |
| **MED** | **smoke gate 的 `single_smoke_authorized` 接受字符串 "1"/"true"/"yes"（`metadataBool`，`adapter.go:386-391`）。** metadata 是 DB 字段，若 admin 建账号接口对 metadata 无严格校验，授权位可能被意外置真。 | `adapter.go:363,378-393` | 确认建账号路径对 `metadata_json` 写入有审计；冒烟后立即把该位清 false 收口。 |

---

## C. 成本失控点

| 严重度 | 弱点 | 落点 | 修复 |
|---|---|---|---|
| **BLOCKER** | **"单条冒烟"并非真正 bounded 到 1 次 Ark 计费调用。** 实测：create 调一次 Ark（`adapter.go:185`），但**每次 worker tick 对 Submitted/Running 任务都会调一次 Ark poll**（`adapter.go:259`，`worker.go:155`）。poll 也是真实 HTTP 调用。若任务在 Ark 侧 running 15 分钟、poll interval 30s，则 = **1 次 create + ~30 次 poll = 31 次真实上游调用**。方案 §7 称"单条 ≤ ¥10"只算了 create 计费——poll 是否计费未知。"单条"是 1 次**生成**，不是 1 次**调用**。 | `adapter.go:259`；`worker.go:155` | (1) 确认 Ark poll(GET) 不计费（多数厂商不计，但 §3 是 ASSUMPTION，未读真实定价——poll 计费同样未知）。(2) 加 poll 次数上限 + 退避。 |
| **BLOCKER** | **方案承认的"三大熔断缺失"——并发上限、预算硬上限、circuit breaker——我已实测确认全部不存在于热路径。** grep `service/` 目录：无任何 budget/spend 校验逻辑，`cost_estimate` 仅 `result.CostEstimate > 0` 时记录（`worker.go:195`），而 **seedance adapter 从不填 CostEstimate**，所以连记录都是空的。`ProcessRunnableTasks`（`worker.go:98-112`）只是串行 for 循环，无 in-flight ceiling。**唯一的成本边界是 daily 1/user/day + smoke gate**。 | `worker.go:98-112,195`；`service.go:444` | 阶段0 单条冒烟**可接受**（daily=1 + 人工盯 + smoke gate 是足够边界）。但方案 §7-阶段1 把熔断列为"阶段1 前补"——**正确**。绝不可在熔断落地前进入任何批量。 |
| **HIGH** | **duration 不下发导致成本估算与真实脱钩**（同 A-BLOCKER）。§3 全部基于"5 秒上限"，但 Ark 实际按其默认时长计费。¥10/条 硬预算是建立在一个**不会被发送的参数**上。 | `adapter.go:163-169` | 同上：下发 duration，并以冒烟实账单校准 §3.1 的 ASSUMPTION。 |
| **MED** | **daily limit 查询 `PageSize:100`（`service.go:432`），且统计的是当天所有 seedance 任务（含失败/取消）。** 对单条冒烟无害（反而更保守：失败也占额度）。但要注意：若冒烟失败想重试，**当天额度已耗尽**，需手动改库或换 user。 | `service.go:428-444` | 知悉即可。冒烟失败重试需换 `CreatedBy` 或清当天记录。 |

---

## D. 凭据泄露风险

| 严重度 | 弱点 | 落点 | 修复 |
|---|---|---|---|
| **HIGH** | **Ark 上游错误响应被原样截断 500 字符写入 DB 错误字段并向下游传播。** `CreateTask`/`PollTask` 在非 2xx 时返回 `truncate(string(respBody), 500)`（`adapter.go:198,272`），该 error 经 `failTask` 写入 `task.ErrorMessage` + `video_task_events.Message`（`worker.go:152,156` → `failTask` `:226-237`），并随 poll 响应回到前端。**若 Ark 401/403 响应体回显了 Authorization 头、key 前缀、或请求 echo——密钥片段直接落 DB 且进 API 响应。** 这是真实存在的泄露路径，不是 dry-run 假设。 | `adapter.go:198,272`；`worker.go:226-237` | 上游错误体在写库/返回前必须经脱敏过滤（剔除 `Bearer`、key 模式、Authorization）。**冒烟前必须修**——这是密钥泄露最可能的实际通道。 |
| **HIGH** | **`SUB2API_VIDEO_REDACTED_EVENT_LOG` 写的是文件路径。** smoke gate 要求该 env 非空（`adapter.go:366`）即开启脱敏事件日志。但我**未在本仓库看到该日志的写入实现与脱敏逻辑**（adapter 只检查 env 存在，payload 里 `redacted_event:true` 只是个标志位，`adapter.go:225,297`）。"redacted"是否真的脱敏、写到哪个文件、文件权限如何、是否会被 git 跟踪——**全未验证**。一个名为"redacted"但未证实脱敏的日志文件是经典泄露源。 | `adapter.go:225,297,366` | 冒烟前定位该日志的真实写入代码，确认：(1) 确实脱敏 Authorization/key；(2) 路径在 `.gitignore` 内；(3) 文件权限受限。**未确认前不要冒烟。** |
| **MED** | **`PlainAPIKey` 为 transient 字段，但 `account` 结构体被整体传入 adapter（`worker.go:165,183`）。** 任何对 `account` 的 `slog`/`fmt.Sprintf("%+v")`/panic dump 都会打印明文 key。实测 `worker.go:108` 的 Warn 只打 `task.ID/provider/error`（安全），但需确认全代码路径无 `%+v` 打 account。 | `worker.go:165,183`；`account.PlainAPIKey` | 给 `VideoProviderAccount` 加 `String()`/`LogValue()` 方法屏蔽 PlainAPIKey，防御性兜底。 |
| **MED** | **方案 §4 的"绝不落盘明文"原则正确，但 admin 建账号时明文 key 经 HTTP body 传入（`VideoProviderCreateParams.APIKey`）。** 该请求体可能进 Hono/Go 的 access log、反代日志、APM。 | `video_gateway_types.go:175`（建账号入参） | 确认建账号接口路径的 request body 不被任何中间件记录；用 admin UI 单次注入后即停。 |

---

## GO / NO-GO 裁决

### 阶段0 单条冒烟：**CONDITIONAL NO-GO**（修完下方 3 项后转 GO）

**理由：** 成本边界（daily=1 + smoke gate + 人工盯）对单条而言**架构上足够**，方案的分阶段纪律（熔断留到阶段1前）是正确的。**但有三个 blocker 级缺陷会让"单条冒烟"本身失去意义或造成泄露**：duration 不下发使成本/时长失控且无法校准、Ark 错误体可能回显密钥、redacted 日志的脱敏未证实。在这三项修复并验证前，冒烟既不安全也无法达成其校准目的。

此外叠加评审限制：**A 侧全部代码不在本仓库**，方案有约一半（QCanvas/Hono/wujie/Day0）我无法验证。在异源复核缺失 + 半数落点不在场的双重盲区下，谨慎默认值是 NO-GO。

### 冒烟前必须修复的 Top 3

1. **【密钥泄露，最高优先】证实 redacted 事件日志真的脱敏 + 封堵 Ark 错误体回显。** 定位 `SUB2API_VIDEO_REDACTED_EVENT_LOG` 的写入实现，确认脱敏 Authorization/key 且路径在 `.gitignore`；同时给 `adapter.go:198,272` 的上游错误体在写库/回传前加脱敏过滤（`worker.go:226-237` 落 DB 之前）。**未确认前禁止发真实请求**——这是密钥进日志/DB/响应最现实的通道。

2. **【成本失控 + 契约】在 create payload 下发 `duration`，并以冒烟实响应核对 Ark 字段契约。** 修 `adapter.go:163-169` 加入 duration（否则 5 秒上限是纸面约束，¥10 预算建立在不发送的参数上）；冒烟第一步抓取 Ark create/poll 真实 JSON，核对 `id`/`status`/`content.video_url` 字段名是否与 `adapter.go:201-217,275-284` 假设一致（这决定轮询是否能工作）。

3. **【鉴权基础】确认 `video_tasks.created_by` 填真实终端用户、daily 1/day 据此生效，且 result_url/reference_image_url 经域名 allowlist 校验。** 验证 auth-contract split 下员工 key 不会把 daily limit 架空（`service.go:430`），并对两个外部可控 URL（`adapter.go:159-161` 转发的 image_url、`:292` 信任的 result_url）加 SSRF/域名校验。

> 补充强制项（非 Top 3 但冒烟验收必查）：实测 wujie 边界在 **real-success** 路径下 `wujieWriteCount:0`（而非仅 mock 路径）；确认 poll 不计费或加 poll 次数上限；冒烟后立即把 `single_smoke_authorized` 与 `SUB2API_VIDEO_REAL_SMOKE_ENABLED` 复位收口。