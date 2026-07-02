# A1 跨仓只读契约快照报告

状态：待复核

## 结论

A1 已完成只读契约快照。QCanvas -> Hono -> Sub2API 的 video 创建字段缝在核心字段上是可对齐的：Hono 将画布嵌套参数摊平为 Sub2API Go handler 绑定的顶层 snake_case 字段，Sub2API 响应中的 `id/status/result_url/ResultURL` 也被 QCanvas 归一为 `taskId/status/resultUrl`。

本报告不把 QCanvas 整体状态升格。原因是 QCanvas ABC progress 仍有 RUNNING/阻塞记录，且 QCanvas 五件套与最新 LATEST 审查包存在真相源漂移。本轮未写 QCanvas、未触发真实供应商、未读取敏感配置内容、未 push。

## 执行目录

- 写入仓：`D:/sub2api-trunk`
- 只读仓：`D:/Codex创业任务/QCanvas（无界版）/QCanvas`
- 产物目录：`D:/sub2api-trunk/_review/a1-contract-snapshot-20260703/`
- 机器快照：`contract_snapshot.json`

## Git 边界

Sub2API：

- root：`D:/sub2api-trunk`
- branch：`wujie/video-capture-moat-20260702`
- HEAD：`2d77007885be545ebb4a4daf0e3162ae7cd0a27c`
- 进入 A1 前仍有非本轮 dirty：`.cursor/SETUP.md`、`.cursor/mcp.json`、`.cursor/skills/*`、`nango-integrations/`

QCanvas：

- root：`D:/Codex创业任务/QCanvas（无界版）/QCanvas`
- branch：`work/night-hardening-20260702`
- HEAD：`b1aa25f23d0af92f7b31258e509173f56b2c6843`
- ABC progress：`docs/reviews/超级任务书_ABC_20260703/progress.md` 仍显示子线 RUNNING/阻塞，本轮不写 QCanvas。

## 来源文件与哈希

QCanvas：

| 文件 | SHA256 |
|---|---|
| `packages/schemas/generation-contract/sub2api-video-field-seam-v2.snapshot.json` | `d479077995b302bb2d7b70625ef034160d22d08b88288d91f57a07b2eca3c418` |
| `apps/hono-api/src/modules/sub2api/sub2api.video-mock-gateway.service.ts` | `9863c58fa34cd237bfea37497143e8858c50f9d3d4fd88bd6f57fb2bf2c12821` |
| `apps/hono-api/src/modules/sub2api/sub2api.video-mock-gateway.service.test.ts` | `8088bbbdcb3080acc6c692ad85c417b3505ff83204da617c371cad967f9e8550` |
| `docs/reviews/超级任务书_ABC_20260703/progress.md` | `c112fbdcc90fb59b379369bf0c83fe796abe30bf350d7e9263092af0f3cb7f22` |
| `docs/reviews/LATEST_REVIEW_PACKAGE.html` | `d303044c2c383342605bb3afcc093421421f992f417902a107473d9097df1080` |
| `docs/LATEST_REVIEW_PACKAGE.md` | `42c8d1e401b86594eac6e87eb9a4b6fcc9c66c9a87003dae45b191c07ed63438` |

Sub2API：

| 文件 | SHA256 |
|---|---|
| `backend/internal/handler/video_handler.go` | `9cf0e5c747d6e45a9b4b2b4fdb1d3390e6234e6f3864fafefff5839404421620` |
| `backend/internal/handler/video_handler_c1_contract_test.go` | `aa7eb33b6b555e8e629474c3d2ae30b3dd89c0519b1d49915304edf099eff689` |
| `backend/internal/handler/testdata/api_key_video_task_contract_v1.json` | `ebf16a280d07a7cd662251bd910c4def037cdc2fb993ef70c05bc5f436e7b5fc` |
| `backend/internal/service/generation_content.go` | `771d6e8f4c51034031f1a5a8c0c406265c73f80effcbb39dad080831b830b886` |
| `backend/internal/repository/generation_content_repo.go` | `3e96d2ec9d30ef0cb61718761c0c5344f66f56bdcd30ab885ebba5fb92aa5e25` |
| `backend/internal/handler/admin/generation_content_handler.go` | `cd0ac5144e8fde3c293616587341147f47e0beb75a67c02b17b0dc2316a0626a` |
| `backend/migrations/140_ai_generation_content.sql` | `4587a9b54a58e0e2f3c0f9234298bd4baef2350981da1d3f53c689a8cc5c89b9` |
| `backend/migrations/146_ai_generation_content_task_id_unique_notx.sql` | `40a8c15d905acbce1d7074ccdcd781ee1c478f99c214a8b73f588bbfd8d471c3` |

## 契约快照

请求路径：

- QCanvas snapshot：`POST /v1/video/tasks`
- Sub2API handler：`apiKeyVideoTaskCreateRequest`

Go 绑定的顶层字段：

- `provider`
- `trial_mode`
- `task_type`
- `prompt`
- `model`
- `negative_prompt`
- `reference_image_url`
- `reference_video_url`
- `aspect_ratio`
- `duration`
- `resolution`

Hono 摊平规则：

- `aspect_ratio` <- `params.aspect` / `params.aspect_ratio` / `params.aspectRatio`
- `resolution` <- `params.quality` / `params.resolution`
- `duration` <- `params.duration`，只保留 1 到 60 秒整数
- `reference_image_url` <- `params.reference_image_url` / `params.referenceImageUrl` / `params.firstFrameUrl`
- `reference_video_url` <- `params.reference_video_url` / `params.referenceVideoUrl`
- `task_type` <- 有首帧或 `metadata.taskKind=image_to_video` 时为 `image_to_video`，否则为 `text_to_video`

响应归一：

- QCanvas `taskId` 来源：`task_id` / `taskId` / `id`
- Sub2API 当前响应主键：`id`
- QCanvas `resultUrl` 来源：`result_url` / `resultUrl` / `ResultURL` / `url` / nested result variants
- Sub2API 当前结果 URL 字段：`result_url`，并兼容 `ResultURL`

## 差异

| 编号 | 结论 | 证据 | 风险 |
|---|---|---|---|
| A1-D1 | 待复核 | QCanvas snapshot 把 `params` / `metadata` 列为 requiredTopLevelFields；Sub2API Go binding 不消费这两个字段，Gin 对未知 JSON 字段按忽略处理。 | 不是当前阻塞，但后续若要把 metadata 入账，需要新增 Sub2API 字段或日志设计。 |
| A1-D2 | 待复核 | Sub2API 接受 `reference_to_video`，且自身 testdata 使用该枚举；QCanvas Hono 当前只派生 `text_to_video` / `image_to_video`。 | 当前画布路径不受影响；若将来需要 reference-to-video，需要扩 QCanvas 派生和快照。 |
| A1-D3 | 内部可用 | Sub2API 返回 `id`，QCanvas 归一逻辑读取 `id` 作为 `taskId` 来源。 | 若后续改 response wrapper，需要保留 `id` 或 `task_id`。 |
| A1-D4 | 内部可用 | Sub2API `apiKeyVideoTaskToResponse` 同时保留 `result_url` 和 `ResultURL`，QCanvas 侧大小写兼容读取。 | 若未来移除 Pascal 兼容，需要先改 QCanvas snapshot 与测试。 |
| A1-D5 | 内部可用 | `ai_generation_content.task_id` 有唯一索引，仓库 `CreateVideoTaskContent` 使用 `ON CONFLICT (task_id) ... DO NOTHING`。 | 只能保证 video task 级幂等；采用率和质量分仍是后续人工回填规划。 |

## 采集账本

Sub2API 的 video 成功采集链路：

1. worker poll 任务。
2. 状态归一为 `succeeded`。
3. 更新任务、写事件、写用量。
4. 调用 `CollectVideoTaskGenerationContent`。
5. collector 写入 `ai_generation_content`，以 `task_id` 做幂等键。

Admin 只读视图：

- `GET /api/v1/admin/generation-content/stats`
- `GET /api/v1/admin/generation-content/samples`

Stats 返回 `captured_today`、`captured_week`、`distinct_employees`、`distinct_teams`、`distinct_models`、`total_bytes`、`daily_rate`、`daily_series`、`is_live`。Samples 返回已截断预览，不返回原始完整上下文。

## 验证命令

```powershell
git status --short
git branch --show-current
git rev-parse --show-toplevel
```

结果：已记录两仓 root/branch/status；QCanvas dirty 且 ABC 子线 RUNNING，Sub2API 仅保留非本轮 dirty。

```powershell
node -e "const fs=require('fs'); const p='packages/schemas/generation-contract/sub2api-video-field-seam-v2.snapshot.json'; const j=JSON.parse(fs.readFileSync(p,'utf8')); console.log([j.version,j.request.path,j.request.requiredTopLevelFields.includes('task_type'),j.response.resultUrl.readFrom.includes('ResultURL')].join('|'));"
```

结果：

```text
sub2api-video-field-seam/v2|/v1/video/tasks|true|true
```

```powershell
go test ./internal/handler -run "TestD2ApiKeyVideoContractMatchesSnapshotV1|TestC1ApiKeyVideoResponseMatchesQCanvasContract|TestC1ApiKeyVideoCreateRequestAcceptsQCanvasBody" -count=1
```

结果：

```text
ok  	github.com/Wei-Shaw/sub2api/internal/handler	3.566s
```

说明：首次在沙箱内运行被 Go build cache 权限拦住；同一条窄测试在沙箱外重跑通过。

## 两侧测试计划

Sub2API 后续可重复执行：

```powershell
cd D:\sub2api-trunk\backend
go test ./internal/handler -run "TestD2ApiKeyVideoContractMatchesSnapshotV1|TestC1ApiKeyVideoResponseMatchesQCanvasContract|TestC1ApiKeyVideoCreateRequestAcceptsQCanvasBody" -count=1
```

QCanvas 待 ABC 子线解除 RUNNING 后执行：

```powershell
cd D:\Codex创业任务\QCanvas（无界版）\QCanvas\apps\hono-api
pnpm exec vitest run src/modules/sub2api/sub2api.video-mock-gateway.service.test.ts
```

如遇 Prisma EPERM，不整夜卡 root install，改用包内 vitest/tsc 旁证并记录阻塞原因。

## 风险

- QCanvas 五件套仍有旧阶段表述，和当前 LATEST 审查包存在漂移；A1 只记录差异，不在 QCanvas 仓修正。
- QCanvas ABC 子线仍 RUNNING，本轮不写 QCanvas。
- 本报告未重查数据库，也未启动服务；Phase A' 三证仍以 A0 已提交的成功包为准。
- 后续真实供应商调用仍为已冻结，需单独授权、预算、停止条件和脱敏审查包。

## 回滚方案

删除本目录或回退本轮 Sub2API commit 即可移除 A1 文档产物；未改业务代码、未改 QCanvas、未 push。

## 可复制后续提示词

```text
继续超级循环 Phase A2-Sub2API：先重读北极星锚文件 #current-state/#roadmap/#guardrails、两仓真相源、progress log；边界是不 push、不读敏感配置内容、不触发真实供应商调用。只在 D:\sub2api-trunk 复核 G3 受控 dev/mock 路径和 ai_generation_content/Admin stats/samples，若 Docker 需要用 WSL Ubuntu-24.04 并保持 keepalive；若 gate 撞上则写 BLOCKED:<原因> 到 progress 并跳下一独立 phase。
```
