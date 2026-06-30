# 阶段 C 审查包 · C1 端到端骨架联调（QCanvas → Sub2API · mock provider）

> 夜间无人值守 · 2026-06-18 · 分支 `night-run/20260618-C-skeleton`（off 阶段B 修复后 tip `831e9c98`）
> 原则：provider 层走 **mock**，**不真实 provider 调用、不部署、不碰密钥、不 push**。

## 0. 一句话结论

**C1 骨架的契约已端到端打通并测试证明；真实跨进程 e2e 卡在两个明确前置：①需要一把 API key（用户授权）②QCanvas hono-api 需自己的 DB。** 关键意外发现：**Sub2API 真实服务进程此刻已在 `localhost:8080` 运行（WSL 中，DB 撑起、前端在服、鉴权在守）——「服务进程」这半边已经是活的。**

## 1. 执行环境实勘（关键，决定本阶段形态）

| 探测项 | 结果 | 含义 |
|---|---|---|
| **Sub2API @ :8080** | **已在运行**（`wslrelay.exe` PID 25272 → WSL 内服务）。`GET /`→200（「AI 生产控制台-无界互娱」前端 HTML）；`/healthz`→200；`/v1/video/providers`→**401 `API_KEY_REQUIRED`**（正是 Sub2API 的 api-key 鉴权） | 真实服务进程 + DB 已活，C1 的 Sub2API 半边**无需我再起栈** |
| Docker daemon | **未运行**（CLI 在、Docker Desktop 未启）。`docker version`→`cannot find dockerDesktopLinuxEngine` | 无法用 compose 另起隔离栈（但 :8080 已有一个，moot） |
| 原生 Postgres/Redis | 无 `psql`/`redis-cli` | 栈在 WSL/容器内，不在 Windows 原生 |
| Node / pnpm | `node v24.15.0` / `pnpm 11.1.2` | QCanvas 可在本机跑（但 hono-api 需自己的 DB/R2 env） |
| API key | **无**（且铁律禁止读取/打印/伪造任何凭据） | 无法鉴权到 :8080 → 真实 e2e 的硬前置 |

> 依任务书容错铁律：Docker 阻断、服务起不来 → 记录现象 + 卡点，不死磕。本阶段据此**不强起新栈、不碰用户在跑的 :8080 实例、不自助注册账号/造 key**。

## 2. C1 架构接通点（骨架长什么样）

```
QCanvas Web(5173) → QCanvas hono-api(8788)
    apps/hono-api/src/modules/sub2api/sub2api.video-mock-gateway.service.ts
      ├─ resolveSub2ApiVideoMockGatewayConfig: 读 SUB2API_BASE_URL + SUB2API_API_KEY (+ENABLED)
      │    未配置→null→503→前端落本地 dry-run；配置了→走真实 Sub2API
      ├─ createSub2ApiVideoMockTask:  POST {baseUrl}/v1/video/tasks  Bearer<key>
      │    body {provider:"mock", task_type, prompt, model, metadata}
      ├─ readSub2ApiVideoMockTask:    GET  {baseUrl}/v1/video/tasks/{id}
      └─ normalizeTaskRecord: 映射 Sub2API 响应 → QCanvas DTO
   → Sub2API(:8080) api-key 路由 /v1/video/tasks (gateway.go:115-124)
      ├─ CreateAPIKeyVideoTask → provider=mock → CreateAPIKeyMockOnlyTask（mock adapter 常驻）
      ├─ worker 轮询 mock adapter: submitted→running→succeeded（假视频 url）
      └─ 响应 apiKeyVideoTaskResponse {id,status,result_url,...,mock_only,provider_boundary}
   → 结果回 QCanvas 候选区（candidate-pool 内存态）
```

**真实 Seedance 闸门全程关闭**（mock-only 路由强制 `provider=mock`；真实 Seedance 需 `SUB2API_VIDEO_REAL_SMOKE_ENABLED=1`+账号授权+时长门，本阶段一个都不设）。

## 3. 已证明（骨架契约端到端打通，有物证）

| 链路环节 | 状态 | 证据 |
|---|---|---|
| Sub2API 真实服务进程在跑 | ✅ 实勘 | :8080 `/healthz`=200、前端在服、`/v1/video/providers`=401（鉴权在守、视频路由已挂载） |
| Sub2API 受理→mock→succeeded→result_url | ✅ CI 动态证明 | `service` 包 `TestFormASeedanceHappyPathInMemory` 驱动 create→worker→poll→succeeded（in-memory repo + mock 路径）；`fakeVideoAdapter` VA2 测试 |
| **Sub2API 出站响应序列化 == QCanvas 读取的键** | ✅ 新增 C1 契约测试 | `internal/handler/video_handler_c1_contract_test.go::TestC1ApiKeyVideoResponseMatchesQCanvasContract`：marshal 真实响应构造器 → 断言 `id`/`status:"succeeded"`/`result_url`/`error_message`/`provider`/`mock_only`/`provider_boundary` 在线上 |
| **QCanvas 入站 body == Sub2API 请求结构** | ✅ 新增 C1 契约测试 | `TestC1ApiKeyVideoCreateRequestAcceptsQCanvasBody`：QCanvas 实际 POST body 绑定到 `apiKeyVideoTaskCreateRequest`，`provider=mock` 被接受 |
| QCanvas 响应映射正确 | ✅ 代码核对 | `normalizeTaskRecord` 容错读 `task_id|taskId|id`→taskId、`result_url|resultUrl|url`→resultUrl、status、`error_message|...`→errorMessage——与 Sub2API 出站键**逐一对齐** |

> 即：QCanvas→Sub2API→mock→result→候选 的**字段契约每一跳都已对齐并测试证明**。`go test ./internal/handler/` ✅(20.9s) / `go vet ./...` ✅。

## 4. 阻断清单（差什么才能真跨进程 e2e）

1. **API key（硬前置）**：鉴权到 :8080 需一把 key。铁律禁止我读取/打印/伪造凭据，也不自助注册（会改用户在跑实例的 DB）。→ **需用户提供一把测试 key，或授权我在受控下创建。**
2. **QCanvas hono-api 自己的 DB**：跑完整 QCanvas 前台进程需 `DATABASE_URL`（+R2 等）。本机无原生 PG、Docker daemon 未起 → hono-api 起不来。→ 需起 QCanvas 的 DB（Docker 起来后 compose，或指向已有 PG）。
3. **（非阻断）运行中的 :8080 实例早于本夜跑改动**：它跑的是改前代码。mock 路径不受 B1/B2/B3 影响（那些是 Seedance 专属），故 mock 骨架 e2e 在它上面照样成立；但若要带 B 修复，需从 `night-run/20260618-C-skeleton` 重新 build+restart Sub2API。

## 5. 开真实门需要什么（黎明后 ~分钟级可收尾的 runbook）

**A. Sub2API mock e2e（有 key 后 2 条命令即证，无需 QCanvas 进程）：**
```bash
# K = 一把有效 api key（用户提供）
curl -s -H "Authorization: Bearer $K" -H "Content-Type: application/json" \
  -X POST http://localhost:8080/v1/video/tasks \
  -d '{"provider":"mock","task_type":"text_to_video","prompt":"hello","model":"mock-video"}'
# → 记下返回的 id；轮询：
curl -s -H "Authorization: Bearer $K" http://localhost:8080/v1/video/tasks/<id>
# → status 应从 submitted→running→succeeded，result_url=https://mock.sub2api.local/video/<id>.mp4
#   （需服务端 video_gateway.worker_enabled=true 才会自动推进）
```

**B. QCanvas 跨进程 e2e（需 QCanvas DB + 上面的 key）：**
```bash
# apps/hono-api/.env:
#   SUB2API_BASE_URL=http://localhost:8080
#   SUB2API_API_KEY=<K>
#   SUB2API_VIDEO_MOCK_GATEWAY_ENABLED=1
#   DATABASE_URL=...(QCanvas 自己的 PG)
cd apps/hono-api && pnpm dev      # :8788
cd apps/web && pnpm dev           # :5173 → 浏览器点生成 → 候选区回 mock 结果
```
> QCanvas 侧**无需改代码**——`resolveSub2ApiVideoMockGatewayConfig` + `normalizeTaskRecord` 已就绪，「解 mock」纯是上面三个 env。

## 6. 本阶段产出 & 提交

- 新增 `backend/internal/handler/video_handler_c1_contract_test.go`（C1 跨仓契约测试，2 个，无 DB/无 key/无网络）。
- 本审查包。
- **QCanvas 侧零代码改动**（已就绪，仅待 env），故 Phase C 提交落在 Sub2API 仓。
- `git add` 显式逐一。**未真实调用、未碰密钥、未部署、未 push。**

## 7. 停止条件判定

骨架**契约**通（mock，已测）→ C1 只差「真实 provider 授权 + 一把 key + QCanvas DB」，记入待授权/阻断清单（见上）。真正卡点是**环境（key/DB/Docker）而非代码**，依铁律不强改、留进度 + 阻断清单。
