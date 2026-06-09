# Sub2API API-Key 视频任务网关 Mock-Only 实现包

## 当前状态

状态：内部可用（mock-only），真实 provider 禁用。

执行目录：`D:\Codex创业任务\企业 API 管理后台项目\02_source\sub2api`

当前 HEAD：`4143673f docs: record sub2api dirty resolution checkpoint`

分支：`phase-3.8.2-overnight-readiness`

## 本轮目标

为 QCanvas 后续机器调用新增 API Key 认证的视频任务网关入口：

- `GET /v1/video/providers`
- `POST /v1/video/tasks`
- `GET /v1/video/tasks/:id`
- `POST /v1/video/tasks/:id/cancel`

第一版只允许 `provider=mock` 或省略 provider 后默认 mock；`seedance` / `kling` 明确 blocked，不触发真实上游。

## 改动文件列表

- `backend/internal/server/routes/gateway.go`
- `backend/internal/handler/video_handler.go`
- `backend/internal/service/video_gateway_service.go`
- `backend/internal/service/video_gateway_types.go`
- `backend/internal/service/video_gateway_adapter.go`
- `backend/internal/server/routes/api_key_video_gateway_test.go`
- `docs/reviews/SUB2API_APIKEY_VIDEO_MOCK_GATEWAY_REVIEW_2026_06_09.md`

未修改：`Dockerfile`、`deploy/*`、QCanvas、真实 provider smoke gate、生产部署配置。

## 新增 Endpoint 表

| Method | Path | 鉴权 | 行为 |
| --- | --- | --- | --- |
| GET | `/v1/video/providers` | `Authorization: Bearer <api-key>` | 返回 mock provider，并把 seedance/kling 标记为 mock-only disabled/unavailable |
| POST | `/v1/video/tasks` | `Authorization: Bearer <api-key>` | 创建 mock 视频任务；provider 空值默认 mock |
| GET | `/v1/video/tasks/:id` | `Authorization: Bearer <api-key>` | 查询当前 API Key 用户可见的 mock 任务 |
| POST | `/v1/video/tasks/:id/cancel` | `Authorization: Bearer <api-key>` | 取消非终态 mock 任务；终态任务返回明确不可取消 |

## 鉴权方式

路由注册在现有 `/v1` gateway group 下，继承：

- `RequestBodyLimit`
- `ClientRequestID`
- `OpsErrorLoggerMiddleware`
- `InboundEndpointMiddleware`
- `APIKeyAuthMiddleware`
- `RequireGroupAssignment`

API Key 读取仍走现有规则：`Authorization: Bearer <api-key>` 为目标形态，`x-api-key` 等由既有中间件兼容。未新增 JWT Cookie 机器调用方案。

## Mock-Only 边界

- `provider=""` -> 默认 `mock`
- `provider="mock"` -> 允许创建任务
- `provider="seedance"` / `provider="kling"` -> 返回 `403 VIDEO_PROVIDER_DISABLED`
- blocked 响应 metadata 包含 `real_provider_dispatch_count=0`
- 创建的任务固定落到可用 mock provider account
- 真实 provider 不会 fallback，也不会被自动选择
- 不读取 `.env` / key / token / cookie / secret
- provider 列表响应不返回 `encrypted_api_key`、`PlainAPIKey` 或真实 masked placeholder

## 真实 Provider 未调用证据

代码边界：

- `CreateAPIKeyMockOnlyTask` 在进入 `CreateTask` 前校验 provider。
- 非 mock provider 直接返回 `VIDEO_PROVIDER_DISABLED`。
- mock task 创建时写入事件 `mock_only_gateway`，payload 包含 `real_provider_dispatch_count: 0`。
- mock adapter 的 payload 只使用本地字段，并补齐 `reference_video_url`。

测试证据：

- `TestAPIKeyVideoGatewayBlocksRealProviders` 验证 `provider=seedance` 返回 403，且内存 repo 的 real provider task count 为 0。
- `TestAPIKeyVideoGatewayMockCreateGetCancel` 验证 mock 任务创建、查询、取消，响应 `real_provider_dispatch_count=0`。
- `TestAPIKeyVideoGatewayProvidersDoNotExposeSecrets` 验证 providers 响应不包含测试用加密占位值或 masked placeholder。

## 测试命令和结果

```powershell
cd D:\Codex创业任务\企业 API 管理后台项目\02_source\sub2api\backend
go test ./internal/server/routes -run "TestAPIKeyVideoGateway|TestVideoAndDramaTaskRoutesRequireAuth|TestGatewayRoutesOpenAIImagesPathsAreRegistered"
```

结果：PASS

```powershell
cd D:\Codex创业任务\企业 API 管理后台项目\02_source\sub2api\backend
go test ./internal/service -run "TestVideoGateway|TestVideoProvider|TestVideoAdapterContractSafeProviderBehavior"
```

结果：PASS

```powershell
cd D:\Codex创业任务\企业 API 管理后台项目\02_source\sub2api
git diff --check
```

结果：PASS；仅提示既有 `deploy/Caddyfile` LF/CRLF warning，非本轮改动。

## 前端截图 / 日志证据

本轮为后端 API-only mock gateway 实现，未启动前端页面，未做浏览器截图。

可用证据为 Go 路由/服务测试、diff、审查包。没有把规划文档里的 curl 当作真实接口证据。

## 对 QCanvas 的最小契约

QCanvas 后续 adapter 可按以下最小契约接入：

- Base path：`/v1/video`
- Auth：`Authorization: Bearer <api-key>`
- Providers：先调用 `GET /v1/video/providers`，仅当 `provider=mock` 且 `route_available=true` 时创建任务
- Create：`POST /v1/video/tasks`
- Provider：第一版传 `mock` 或不传；禁止传 `seedance` / `kling`
- Task types：`text_to_video`、`image_to_video`、`reference_to_video`
- Supported fields：`prompt`、`negative_prompt`、`reference_image_url`、`reference_video_url`、`aspect_ratio`、`duration`、`resolution`、`model`
- Query：`GET /v1/video/tasks/:id`
- Cancel：`POST /v1/video/tasks/:id/cancel`
- Mock result URL：只有 mock worker 推进任务后才会出现 `mock.sub2api.local` 风格结果；创建响应不承诺已有 result_url

## 当前阻塞项 / 待复核

无阻塞项影响 mock-only API-key gateway 合入。

待复核项：

- 未使用真实 DB API Key fixture 做本地 HTTP smoke，原因是本轮禁止读取/打印真实 key/token/cookie，且不做真实 provider 调用。
- 未启动真实 server/worker 验证 mock 任务自动从 queued 推进到 succeeded。
- 未验证 QCanvas adapter 调用，因为本轮禁止改 QCanvas。

## 下一轮 QCanvas Adapter 建议

1. QCanvas 新增 Sub2API video adapter，只调用 `/v1/video/*`，不要复用 JWT Cookie `/api/v1/video/*`。
2. 先做 mock-only 开关：默认 `provider=mock`，UI 标记 Seedance/Kling disabled。
3. 本地 smoke 只验证 API Key 鉴权、创建 task id、查询状态、取消或不可取消状态。
4. 若要验证结果资产，先开启/确认 Sub2API 本地 worker 只处理 mock provider，再检查 `result_url` 是否为 `mock.sub2api.local`。
5. 下一阶段如要接 Seedance/Kling，必须另开授权目标，显式打开真实 provider gate，并单独产出 real-call 审查包。

## 回滚方案

不需要 reset/clean/restore。

可人工回滚本轮文件：

- 移除 `backend/internal/server/routes/gateway.go` 中 `/v1/video` 注册块
- 移除 `backend/internal/handler/video_handler.go` 中 API Key video DTO 和 handler 方法
- 移除 `backend/internal/service/video_gateway_service.go` 中 API Key mock-only 方法
- 移除 `backend/internal/service/video_gateway_types.go` 新增错误
- 移除 `backend/internal/service/video_gateway_adapter.go` 中 mock payload 的 `reference_video_url`
- 删除 `backend/internal/server/routes/api_key_video_gateway_test.go`
- 删除本审查包

## 可复制后续提示词

```text
在 Sub2API 仓库继续 QCanvas adapter 接入，只调用 /v1/video/* API Key mock-only 网关。
禁止真实 provider、禁止读取 .env/key/token/cookie、禁止 push/deploy。
先用 mock provider 验证 QCanvas 创建任务、拿 task id、查询状态、取消或不可取消状态，并产出审查包。
```
