<!-- 生成: 只读勘察 workflow seedance-realchain-recon / 2026-06-15 / 零真实调用零改码 -->

# 真实 Seedance 2.0 全链路接入方案

> 状态：READ-ONLY 设计方案（只描述改动，不执行任何改动）。本文档不包含任何真实密钥值，仅引用环境变量 / 配置字段的**名称**与代码落点。
> 适用范围：QCanvas Studio V2 视频生成链路 → Sub2API 视频网关 → 真实 Seedance 2.0（火山引擎 Ark）→ 结果回传 Day0 候选池。
> 关键事实：**provider 实现、route 注册、adapter 注册均已存在**。这是一个**配置 / 闸门（gate）问题**，不是缺代码问题（Recon B §3）。

---

## 1. 完整链路图

下图标注了 **dry-run 开关位置**（默认拦截）与 **/wujie 边界**（默认完全隔离）。

```mermaid
flowchart TD
    subgraph QC["QCanvas（无界版）"]
        direction TB
        WB["StudioV2Workbench.tsx:197 / :~413\n run-mode 分流开关"]
        SW["studioV2RealTaskAdapter.ts:359-363\n resolveStudioV2RequestedRunModeForLocalTest()\n ◆ DRY-RUN 开关（浏览器本地）◆\n studioV2RealChainReady / studioV2MockRealEnabled"]
        PF["runVideoPreflight\n studioV2BusinessBridge.ts:220-253\n 默认: AUTH_CONTRACT_SPLIT_BLOCKED\n generationProvider.ts:472-475"]
        GW_CLIENT["sub2api.video-mock-gateway.service.ts\n :214-246 create/poll\n :155 Authorization: Bearer <SUB2API_API_KEY>"]
        CAND["writeCandidateFromSnapshot\n studioV2BusinessBridge.ts:260-331\n → day0CandidateStore.ts:136-169"]
        DAY0[("Day0 候选池\n localStorage only\n qcanvas.day0CandidatePool.v1")]
        WUJIE_X["/wujie 适配器\n studioToWujieAdapter.ts\n ✖ 只读校验，无 I/O\n studioV2BusinessBridge.ts:14")]
    end

    subgraph HONO["Hono API（apps/hono-api）"]
        ROUTE["sub2api.routes.ts:169-291\n 挂载 /sub2api (app.ts:332)\n authMiddleware 要求 userId"]
    end

    subgraph SUB["Sub2API（Go backend）"]
        direction TB
        GROUTE["routes/gateway.go:115-124\n /v1/video/* (API-key 鉴权)\n api_key_auth.go:40-65"]
        HANDLER["handler/video_handler.go:267,280-290\n provider=seedance & trial_mode=tiny_real\n → CreateAPIKeySeedanceTinyTrialTask"]
        WORKER["video_gateway_worker.go:73-162\n loop → processTask\n 解密 key :141 → adapter :145"]
        ADAPTER["video_gateway_adapter.go:129-356\n seedanceVideoAdapter（真实 HTTP）\n smoke gate :358-376"]
        DB[("video_tasks /\n video_provider_accounts /\n video_task_events /\n video_usage_logs")]
    end

    SEEDANCE["真实 Seedance 2.0\n ark.cn-beijing.volces.com/api/v3\n /contents/generations/tasks\n model: doubao-seedance-2-0-260128"]

    WB --> SW
    SW -->|无开关: preflight| PF
    SW -->|seedance tiny_real| GW_CLIENT
    GW_CLIENT --> ROUTE
    ROUTE --> GROUTE --> HANDLER --> DB
    WORKER --> ADAPTER
    ADAPTER -->|HTTP Bearer PlainAPIKey| SEEDANCE
    SEEDANCE -->|result_url| ADAPTER --> DB
    DB -->|GET /v1/video/tasks/:id 轮询| ROUTE
    ROUTE -.poll.-> GW_CLIENT --> CAND --> DAY0
    CAND -. 仅只读校验 .-> WUJIE_X
    WUJIE_X -.->|persistStatus: not-persistable| DAY0

    classDef switch fill:#fde68a,stroke:#d97706,color:#000;
    classDef boundary fill:#fecaca,stroke:#dc2626,color:#000;
    class SW switch;
    class WUJIE_X boundary;
```

链路要点：

- **dry-run 开关**（黄色）：唯一离开 dry-run/preflight 的用户侧开关在 `studioV2RealTaskAdapter.ts:359-363`，由浏览器本地 `studioV2RealChainReady`（query param 或 localStorage）控制；**无 env、无服务端配置**（Recon A §1）。
- **wujie 边界**（红色）：`/wujie` 在 V2 中只是只读校验透镜（`studioV2BusinessBridge.ts:14` 明确 "No writes to /wujie"），后端 wujie 为 memory-only 排练路由（Recon A §4）。
- **轮询而非回调**：B 不向 A 发 webhook；A 通过 `GET /v1/video/tasks/:id` 轮询（Recon C §2）。

---

## 2. 接真实 Seedance 要改什么

### 2.a Sub2API 侧落点清单（file:line + 改什么 + 为什么）

| # | 落点 file:line | 改什么 | 为什么 |
|---|---|---|---|
| S1 | `backend/internal/service/video_gateway_adapter.go:129-356` | **provider impl 已存在**，无需改代码；`seedanceVideoAdapter` 已是完整真实 HTTP 客户端 | 这是配置问题不是缺码问题（Recon B §3） |
| S2 | `backend/internal/service/video_gateway_adapter.go:134-139` | 满足 `account.APIKeyConfigured && account.PlainAPIKey != ""`（通过建账号 + 注入密钥，见 §4） | 否则返回 `ErrVideoProviderDisabled` |
| S3 | `backend/internal/service/video_gateway_adapter.go:358-376` | smoke gate 五条件全过：① env `SUB2API_VIDEO_REAL_SMOKE_ENABLED="1"`；② account `metadata_json` 含 `single_smoke_authorized` 或 `real_smoke_authorized` 为真；③ env `SUB2API_VIDEO_REDACTED_EVENT_LOG` 非空；④ `task.Model` 含 `"seedance"`；⑤ `task.Duration` 为 1–5 秒 | 任一不满足即拦截，真实调用不触发 |
| S4 | `backend/internal/handler/video_handler.go:267,280-290` | 调用时携带 `provider=seedance` + `trial_mode=tiny_real`，命中 `CreateAPIKeySeedanceTinyTrialTask` 分支 | 否则非 mock provider 被 403 `VIDEO_PROVIDER_DISABLED` |
| S5 | `video_provider_accounts` 行（`repository/video_gateway_repo.go:36-138`） | 新建一行：`provider='seedance'`、`enabled=true`、`encrypted_api_key`=真实密钥密文、`base_url`（默认 fallback `https://ark.cn-beijing.volces.com/api/v3`，`adapter.go:148-151`）、`metadata_json.single_smoke_authorized=true` | route/adapter 通过 `adapterFor`（`service.go:1191-1197`）按 `task.Provider` 解析账号 |
| S6 | `backend/internal/service/video_gateway_service.go:444-449` | 阶段0/1 保持 seedance 每用户每日 **1 次** trial 上限不动；批量阶段如需放宽再议 | 当前硬限 1/user/day |
| S7 | `backend/internal/service/video_gateway_service.go:211-218` | **不要**依赖 admin TestProviderAccount 验证真实链路 | 该路径对非 mock provider 返回 "real network test is disabled in P0"，永不发真实请求 |
| S8 | route 启用：`backend/internal/server/routes/gateway.go:115-124` | **已注册**，无需改 | `/v1/video/*` 四个端点已挂在 API-key 鉴权 + group-assignment 后 |
| S9 | config gate：`backend/internal/config/config.go:177-183,1599-1602,1904-1911`；`deploy/config.example.yaml:856-870` | 确认 `video_gateway.worker_enabled=true`（kill-switch），`task_timeout_minutes`、`poll_interval_seconds`、`worker_batch_size` 取保守值（见 §3） | worker 是真实链路的驱动循环 |
| S10 | 凭据读取：`video_key_encryptor.go:16-28`、`service.go:1173-1189` | 配置 `video_gateway.encryption_key`（32 字节 hex），运行时 `decryptProviderKey→encryptor.Decrypt` 解出 transient `PlainAPIKey` | 真实 Bearer 头由 `PlainAPIKey` 注入（`adapter.go:182,256`） |

### 2.b QCanvas 侧落点清单（file:line + 改什么）

| # | 落点 file:line | 改什么 |
|---|---|---|
| Q1 | `studioV2RealTaskAdapter.ts:359-362`（`STUDIO_V2_REAL_CHAIN_READY_SWITCH`） | **dry-run→real 开关**：开启浏览器本地 `studioV2RealChainReady=true`（query param 或 localStorage），使 `requestedRunMode='real'` |
| Q2 | `studioV2RealTaskAdapter.ts:83-100`（`shouldUseSub2ApiVideoMockGateway` + `assertProviderAllowed`） | 客户端传 `provider:'seedance', trialMode:'tiny_real', allowRealCalls:true`，使 `assertProviderAllowed`（:98）放行 seedance tiny_real |
| Q3 | `apps/hono-api/src/modules/sub2api/sub2api.routes.ts:184-224` | 确认 seedance tiny-real trial 闸门三 env 已置 `1`：`SUB2API_VIDEO_REAL_SMOKE_ENABLED`、`SUB2API_REAL_HUMAN_AUTHORIZED`，且 `trialMode==='tiny_real'`；否则 403 `SUB2API_VIDEO_PROVIDER_BLOCKED` |
| Q4 | **Hono proxy 指向**：`sub2api.video-mock-gateway.service.ts:72-78,143-145` | 设置 env `SUB2API_BASE_URL`（指向 Go backend）、`SUB2API_API_KEY`（员工 API key）、`SUB2API_VIDEO_MOCK_GATEWAY_ENABLED`；upstream URL = `${SUB2API_BASE_URL}/v1/video/tasks` |
| Q5 | 客户端调用：`apps/web/src/api/server.ts:6249-6273`（create/poll/cancel）+ `withAuth` `:35-41` | 无需改；前端 JWT 仅用于认证到 Hono，upstream 用服务端员工 key（auth-contract split） |
| Q6 | **候选池回写**：`studioV2BusinessBridge.ts:260-331` → `day0CandidateStore.ts:136-169` | 无需改；成功快照经 `writeCandidateFromSnapshot` 写入 Day0 localStorage 池（`resultSource='sub2api-real-trial'`） |
| Q7 | 硬契约（**本方案不动**）：`StudioV2Workbench.tsx:200-201`、`studioV2Lifecycle.ts:24`、`studio-v2-mock-real.service.ts:23-24,35-37` | `allowRealCalls` 在多处被钉死为 `z.literal(false)` — 这些只影响**通用全真链路（Seedream/Kling 等）**，与 seedance tiny-real trial 路径无关，**保持不动** |

> 结论：seedance tiny-real trial 是"最小改动即可发真实视频"的路径，**全部为配置/env/账号行 + 客户端参数**，无需改前端硬契约代码（Recon A §2）。

---

## 3. 成本估算

### 3.1 单条成本（显式 ASSUMPTION — 未读到真实定价）

> ⚠️ **ASSUMPTION**：recon 中**未读取任何真实 Seedance 定价**。前端 `generationSafetyGuards.ts:46-58` 的 video base `0.24` USD 是 dry-run 估算系数，**非真实报价**；mock 返回 `CostEstimate:0`（`adapter.go:86`），真实 seedance adapter **不填 cost**。以下区间为外部假设，上线前必须以真实账单校准。

- **假设定价区间（ASSUMPTION）**：按秒计费 **¥0.5 – ¥2.0 / 秒**（1080p Seedance 2.0 类模型的行业估算区间）。
- 阶段0 单条约束：`duration` ≤ 5 秒（smoke gate `adapter.go:372`）。
- **单条成本估算（5 秒上限）**：
  - 低端：5 s × ¥0.5 = **¥2.5 / 条**
  - 高端：5 s × ¥2.0 = **¥10 / 条**
  - 取中位规划值：**约 ¥6 / 条**

### 3.2 200 元能跑几条（算术）

| 场景 | 单条成本 | 200 元可跑条数 |
|---|---|---|
| 低端（¥2.5） | ¥2.5 | 200 / 2.5 = **80 条** |
| 中位（¥6） | ¥6 | 200 / 6 ≈ **33 条** |
| 高端（¥10） | ¥10 | 200 / 10 = **20 条** |

> 规划口径：以**高端 ¥10/条**做硬预算保护，即 **200 元 ≤ 20 条**，避免低估超支。

### 3.3 建议熔断阈值 + 代码落点

| 阈值 | 建议值（阶段1 批量） | 代码落点（现状/应落点） |
|---|---|---|
| 并发上限 | 同时在途 ≤ 3 | **现无全局并发上限**；worker 串行处理 batch（`worker.go:106-110`）。应落点：`worker.go` ProcessRunnableTasks 处加全局 in-flight ceiling |
| 单任务超时 | 15 分钟（默认） | `config.go:177-183` `task_timeout_minutes`；强制失败逻辑 `worker.go:118-136` |
| 在途上限 | batch size ≤ 5 | `worker_batch_size`（`worker.go:34-36`，`videoDefaultBatchSize` `service.go:21`） |
| 预算硬上限 | ¥200（≤20 条） | **现无预算/花费上限**；`cost_estimate` 记录在 `video_usage_logs` 但从不校验。应落点：`submitTask`/`processTask`（`worker.go:164-180`）前置累计花费检查 |
| Kill-switch | 一键停 worker | `worker_enabled`（`config.go`），`ProvideVideoGatewayWorker` 若 false 跳过 `Start()`（`worker.go:48-54`）。另有 env `SUB2API_VIDEO_REAL_SMOKE_ENABLED` 置 `0` 即停真实链路 |
| 每用户每日上限 | 1/day（trial 现状） | `service.go:444-449` |
| HTTP 超时 | 30 秒（硬编码） | `adapter.go:184,258` |

> **缺口提示（Recon B §5）**：当前**无 circuit breaker、无并发上限、无预算上限、rate limit 不强制执行**。阶段1 批量前，这三项熔断必须先补到上述落点，否则无自动熔断保护。

---

## 4. 凭据注入方式设计

### 核心原则：绝不落盘明文

1. **绝不**把真实密钥写入任何文件（含 yaml、.env 提交物、代码常量）。
2. **绝不**在 chat / PR / commit message / 日志中粘贴密钥值。
3. **绝不**让密钥进 git（git 历史不可逆）。
4. 客户端永远只看到 masked 形式 `first4***last4`（`MaskVideoAPIKey` `service.go:1199-1208`）。

### 注入路径（三层，按推荐优先级）

**(A) Provider 真实密钥 → admin 加密存储（推荐）**

- 通过 `VideoProviderCreateParams.APIKey` / `VideoProviderUpdateParams.APIKey`（`video_gateway_types.go:175,185`）传入明文。
- 由 `applyProviderAPIKey → encryptor.Encrypt`（`service.go:1095-1104`）用 **AES-256** 加密，密文存入 `video_provider_accounts.encrypted_api_key`（`repo:43,67,113`）。
- 运行时 `decryptProviderKey → encryptor.Decrypt`（`service.go:1173-1189`）解出 transient `PlainAPIKey`，**从不回写磁盘**。
- 加密 master key：`video_gateway.encryption_key`（`config.go:178`，yaml `deploy/config.example.yaml:866`），32 字节 hex，校验在 `config.go:1895-1902`；dev fallback 到 `totp.encryption_key`（`video_key_encryptor.go:16-20`，带 warning）。加密器构造 `NewVideoKeyEncryptor` 强制 `len==32`（`video_key_encryptor.go:21-28`）。
- **master key 本身也走环境/secret manager 注入，绝不入库 yaml 明文。**

**(B) Hono → Go 的员工网关 key → 环境变量 / Worker binding**

- A 侧从 env / Worker binding 读取（`readRuntimeText` `sub2api.video-mock-gateway.service.ts:34-36`），**从不内联**。
- upstream Bearer 用 `SUB2API_API_KEY`（`:74,155`），base 用 `SUB2API_BASE_URL`（`:73`）。

### 需要注入的 env 变量名（仅名称）

| 用途 | env NAME | 落点 |
|---|---|---|
| Go 加密 master key | `video_gateway.encryption_key`（config 字段，经 env/secret 注入） | `config.go:178` |
| 真实 smoke 总闸 | `SUB2API_VIDEO_REAL_SMOKE_ENABLED` | `adapter.go:360`, `service.go:306` |
| 脱敏事件日志路径 | `SUB2API_VIDEO_REDACTED_EVENT_LOG` | `adapter.go:366`, `service.go:312` |
| Hono→Go 网关 base | `SUB2API_BASE_URL` | gateway.service.ts:73 |
| Hono→Go 网关 key | `SUB2API_API_KEY` | gateway.service.ts:74,155 |
| Hono 网关启用 | `SUB2API_VIDEO_MOCK_GATEWAY_ENABLED` | gateway.service.ts:38-78 |
| 人工授权位 | `SUB2API_REAL_HUMAN_AUTHORIZED` | sub2api.routes.ts:198-224 |

> Provider 的 Seedance 真实密钥**不进 env**，走 (A) 加密入库（仅密文落 DB）。

---

## 5. 写入边界

**默认只写 Day0 候选池，绝不写 /wujie。**

- **Day0 写入**（允许）：成功快照 → `writeCandidateFromSnapshot`（`studioV2BusinessBridge.ts:260-331`）→ `createDay0LocalCandidateFromPayload`（`day0CandidateStore.ts:136-169`），仅 **localStorage**，key `qcanvas.day0CandidatePool.v1:<project>:<chapter>`（`day0CandidateStore.ts:25,42-44`），上限 80 条（`:104`）。Payload 固定 `persistable:false, reuseAllowed:false`，note 明示 "Day0 localStorage only ... not a /wujie staging persistence contract"（`day0CandidateStore.ts:26-27`）。
- **/wujie 强制 OFF**（enforcing gate）：
  - 前端：`studioV2BusinessBridge.ts:14` 明确 "No writes to /wujie. The /wujie adapter is read-only here"；`studioToWujieAdapter.ts` 仅做校验/规范化，**无任何 I/O**；其唯一调用 `computeWujieBlockHints`（`:419-436`）只用于解释为何不可持久化。
  - gate 逻辑 `studioToWujieAdapter.ts:102-150`：凡 `isMock` / `dryRun` / bridgeMode 为 mock/dry-run/preflight / URL 为 `mock://`/`dry-run://`/`data:` / objectKey 含 `mock/`/`dry-run/` → `persistStatus:'not-persistable'`。当前所有路径都带这些标记，故**任何路径都进不了 wujie**。
  - 后端确认必须保持 OFF：`studio-v2-mock-real.service.ts:69-70,152-153` 硬编码 `wujieWriteEnabled:false, wujieWriteCount:0`；`sub2api.video-mock-gateway.service.ts:64` 同；wujie staging 本身 memory-only（`wujie.staging.service.ts:135,145`，block reason `"no-real-persistence-contract"` `:13`）。
- **必须保持 OFF 的开关**：`wujieWriteEnabled`（永远 false）、任何使 `studioToWujieAdapter` 产生 `persistable:true` 的旁路、wujie staging 的真实持久化契约（当前不存在，不得新建）。

---

## 6. 失败处理：成功或失败立即停，不 retry / 不 fallback

### 当前行为（Recon C §4）

P2 内部轮询循环 `agents-tool-bridge.generate-video-to-canvas.ts:243-273`：

- `while (Date.now() < deadline)` 仅在 `queued/running` 时按 `intervalMs=3000` 重新轮询**同一任务**（正常轮询，非重试）。
- `status==="failed"` → **立即 `break`（:270）并 throw 一次** 502 `agents_tool_video_generate_failed`（`:288-299`）。
- 成功要求 `status==="succeeded"` **且** 非空 `videoUrl`（`:260-269`）；succeeded 但无 url → 502 `agents_tool_video_missing_url`（`:301-310`）。
- 超时 → 504 `agents_tool_video_generate_timeout`（`:276-286`）。

P1 直连网关 `sub2api.video-mock-gateway.service.ts:160-171`：非 2xx 时**吞掉 HTTP 错误**，合成 `{status:"failed", error_message}` 终态任务 — **终态失败，不重试**。

### 需要的控制流变更（vs 现状）

现状**已基本满足"成功/失败即停、不 retry、不 fallback"**，需要做的是**显式锁死并补一个隐患**：

1. **确认无重试**：`adapter.go` `CreateTask`/`PollTask` 单次调用，worker `processTask`（`worker.go:114-162`）对失败任务不重新 submit。✅ 保持。
2. **确认无 vendor fallback**：轮询循环内单一 vendor，无 alternate-vendor 级联（Recon C §4）。
3. **唯一隐患（需锁死）**：vendor 选择处 `runPublicTask` 用 `vendor:"auto"` + `vendorCandidates`（`generate-video-to-canvas.ts:416-420`）。seedance trial 路径**必须显式钉死单一 provider=seedance**，禁止 `auto` 候选级联，否则失败后可能切换 vendor — 这违反"不 fallback"。落点：创建任务时强制 `provider='seedance'`，不传 `vendorCandidates`。
4. **后端层**：seedance 失败时 worker 将任务置 `failed` 并写 `error_message`（`UpdateTask` `repo:238-267`），**不**重新入队。确认 `ListRunnableTasks` 不会捞回已 `failed` 任务。

---

## 7. 分阶段执行

### 阶段 0：单条 Seedance 2.0 冒烟（人工盯着，1 条，预算 ≤ 单条成本）

**前置预算**：本阶段硬上限 = 1 条 × 高端 ¥10 = **¥10**。

步骤：

1. 在 Go backend 建 `video_provider_accounts` 行：`provider='seedance'`、`enabled=true`、`base_url`=Ark、`metadata_json.single_smoke_authorized=true`；真实密钥经 admin API（明文入参→AES 加密入库，§4-A），**绝不落盘明文**。
2. 注入 env：`SUB2API_VIDEO_REAL_SMOKE_ENABLED=1`、`SUB2API_VIDEO_REDACTED_EVENT_LOG=<path>`、`video_gateway.encryption_key`（32B hex，经 secret 注入）；确认 `worker_enabled=true`。
3. 注入 Hono env：`SUB2API_BASE_URL`、`SUB2API_API_KEY`、`SUB2API_VIDEO_MOCK_GATEWAY_ENABLED=1`、`SUB2API_REAL_HUMAN_AUTHORIZED=1`。
4. 前端开浏览器本地开关 `studioV2RealChainReady=true`（仅操作员本机）。
5. 发**单条** 请求：`provider=seedance, trial_mode=tiny_real, model` 含 `seedance`, `duration` ≤ 5 秒, `task_type=text_to_video`。
6. 人工实时盯：观察 `video_task_events` / 脱敏日志，确认走 `CreateAPIKeySeedanceTinyTrialTask`（`video_handler.go:267`），upstream 命中 Ark `/contents/generations/tasks`。
7. 轮询 `GET /v1/video/tasks/:id` 直到 `succeeded`+`result_url` 或 `failed`。
8. 成功则确认结果落 Day0 池（`resultSource='sub2api-real-trial'`），**确认未写 /wujie**（`wujieWriteCount:0`）。

**Go / No-Go 闸门**：
- ✅ Go：真实 upstream 调用成功 1 次、result_url 可播、Day0 写入正确、wujie 计数为 0、实际账单 ≤ ¥10 且接近预估、失败不重试已验证。
- ✖ No-Go：任一 gate 误放行、出现 retry/fallback、写入 /wujie、成本远超预估、密钥出现在任何日志/文件。

### 阶段 1：通过后员工批量（启用熔断）

**前置**：阶段0 全 Go，且 §3.3 三项缺口熔断**已补齐并验证**（并发上限、预算硬上限、circuit breaker）。

步骤：

1. 补熔断代码（READ-ONLY 方案外，需单独实现 PR）：
   - 全局 in-flight ≤ 3（落点 `worker.go` ProcessRunnableTasks）。
   - 预算累计校验 ≤ ¥200（落点 `worker.go:164-180`，基于 `video_usage_logs`）。
   - circuit breaker：连续 N 次失败自动 open + cooldown（落点 `service.go:935-1008` 健康判定旁）。
2. 校准真实单条成本（用阶段0 实账单替换 §3.1 ASSUMPTION），重算 200 元条数。
3. 设 `worker_batch_size ≤ 5`、`task_timeout_minutes=15`、保留 `worker_enabled` kill-switch。
4. 评估是否放宽 seedance 1/user/day 上限（`service.go:444-449`）；批量需放宽则连同熔断一并审。
5. 灰度：先 1 名员工跑 ≤ 5 条，再扩量；每批后核对花费 vs ¥200 硬上限。

**Go / No-Go 闸门**：
- ✅ Go：熔断三项均可触发验证通过、kill-switch 实测可一键停、批量花费严格 ≤ ¥200、无 retry/fallback、无 wujie 写入。
- ✖ No-Go：任一熔断不生效、花费突破硬上限、kill-switch 失灵、出现 vendor fallback。

---

## 8. 风险与未决问题

1. **真实定价未知（最高优先级）**：recon 未读到任何真实 Seedance 定价；§3 全部为 ASSUMPTION（¥0.5–2.0/秒）。`cost_estimate` 真实 adapter 不填（`adapter.go` seedance 不 populate cost），无法在代码层做精确预算熔断，直到阶段0 拿到真实账单。
2. **缺三大熔断**：当前无全局并发上限、无预算/花费上限、无 circuit breaker，rate limit（`RateLimitPerMinute`）仅存储不强制（Recon B §5）。批量前必须补，否则无自动保护。
3. **vendor auto 级联隐患**：`runPublicTask` 的 `vendor:"auto"` + `vendorCandidates`（`generate-video-to-canvas.ts:416-420`）可能在失败后切 vendor，违反"不 fallback"；需显式钉死 provider。recon 标注此为 out-of-scope，未确认 `runPublicTask` 内部是否真有 seedance 级联。
4. **两条 A→B 路径易混淆**：P1（直连网关，真实 HTTP）才是跨项目契约；P2（QCanvas 内部 pipeline）不达 Go backend（Recon C §0）。需确认 seedance trial 实际走的是 P1，且 P1 是否有服务端轮询循环（recon 称 P1 轮询循环在 recon 范围外）。
5. **字段类型 mismatch 已被吸收但需回归**：B 返回 `id` 为 int64，A 强制 stringify（`gateway.service.ts:119-126`）；B 用 `id` 非 `task_id`；A 丢弃 `upstream_task_id`。改动后需回归这些映射。
6. **auth-contract split**：upstream 用员工 `SUB2API_API_KEY` 而非用户 JWT；需确认该员工 key 有 group-assignment（`gateway.go:41` `requireGroupAnthropic`），否则即便鉴权通过仍被拒。
7. **admin TestProvider 不可用于验证**：`service.go:211-218` 对非 mock provider 返回 "disabled in P0"，无法用 admin test 路径预检真实链路，只能靠阶段0 真实单条冒烟。
8. **encryption master key 的 dev fallback**：`video_key_encryptor.go:16-20` 会 fallback 到 `totp.encryption_key`，生产必须显式配置 `video_gateway.encryption_key`，避免误用 totp key。
9. **Day0 池容量与易失性**：localStorage only、上限 80 条、非正式资产 DB，批量产出可能溢出/丢失；是否需要正式持久化契约未决（当前明确无）。