# M1-B 采集口施工 · 执行日志

> 分支：`wujie/trunk` ｜ 日期：2026-06-18 ｜ 7 项决策全批 + 施工授权
> 安全姿态：**feature flag `gateway.content_capture.enabled` 默认 false**——整套功能默认 dark，生产零影响；采集 fail-open。

## 工程判断：M1-B 拆为两个可验证子步（判断基准4 不冒险主路由）
response 全文采集需在中继**流式热路径**的 ~8 个 emit 出口插限容 tee，且本地无法对真实上游做流式集成验证。故拆：
- **M1-B.1（本次，已完成+全绿）**：全套基础设施 + **prompt 采集**端到端打通（messages + chat-completions 两个在范围端点）。
- **M1-B.2（下一焦点步）**：把 `cappedSink` 接到 emit 出口填充 response，配合流式测试 harness 验证。

---

## M1-B.1 已交付（全部已编译+测试通过）

### 新建文件（8）
| 文件 | 作用 |
|---|---|
| `backend/migrations/140_ai_generation_content.sql` | 新独立内容表（含 prompt/response 列 + 预留 task_id/adoption_status/quality_score；唯一索引 (api_key_id,request_id)） |
| `backend/internal/service/generation_content.go` | `GenerationContent`/仓接口/`GenerationContentCaptureArgs`/`GenerationContentCollector`（fail-open）+ GatewayService 的 `SetGenerationContentCollector`/`CollectGenerationContent`/`contentCaptureEnabled` |
| `backend/internal/service/generation_content_redact.go` | 入库前脱敏编排（`RedactJSON`/`RedactText` + 新 PII pass + `redactContentModerationSecrets`）+ `redaction_version` |
| `backend/internal/service/cappedsink.go` | 限容 tee（fail-open、永不超额分配、缓冲独立持有）——为 M1-B.2 备好 |
| `backend/internal/repository/generation_content_repo.go` | 裸 SQL 仓，`INSERT ... ON CONFLICT DO NOTHING`（幂等）；仿 content_moderation 先例 |
| `*_test.go` ×3 | cappedSink / 脱敏 / 采集器（fail-open + 网关门控）单测 |

### 修改文件（5，共 +48 行，外科级）
- `backend/internal/service/gateway_service.go` — `GatewayService` 加 `generationCollector` 字段；`ForwardResult` 加 `ResponseSample/ResponseTruncated/ResponseBytes`（B.1 留空，B.2 填充）。
- `backend/internal/config/config.go` — 加 `ContentCaptureConfig`（Enabled/ResponseMaxBytes/PromptMaxBytes）+ 挂到 `GatewayConfig.ContentCapture`。
- `backend/internal/handler/gateway_handler.go`（messages 闭包）+ `gateway_handler_chat_completions.go`（CC 闭包）— `RecordUsage` 之后并列调 `CollectGenerationContent`（与计费隔离）。
- `backend/cmd/server/wire_gen.go` — 注入采集器（`SetGenerationContentCollector(NewGenerationContentCollector(NewGenerationContentRepository(db), configConfig))`）。

### 7 项决策落地
1 cap 64KiB（`defaultGenerationResponseMaxBytes`，B.2 用）｜1b prompt 限容 256KiB（采集时强制）｜2 内容表独立 + 预留清理（保留期/清理任务=后续）｜3 LLM 先行（messages+CC）｜4 PII 最小集（email+phone）｜5 flag 默认 false｜6 超大截断（不上对象存储）｜7 仅预留 task_id/adoption_status/quality_score。

### 验证（本次实跑，全绿）
- `go build ./...` = 0
- `go vet ./internal/service/... ./internal/repository/... ./internal/handler/...` = 0
- 新单测 10 个全 PASS（cappedSink 3 / 脱敏 3 / 采集器 4）
- 回归：`go test ./internal/service/...` ok 48.4s ｜ `./internal/repository/...` ok ｜ `-run C1 ./internal/server/routes/...` ok
- 物证安全：脱敏单测断言 access_token/Bearer/sk-ant/password/email 被剥、model 名不误伤；采集器 fail-open（仓报错/ nil 不 panic）；门控（flag off / 未注入 → 不写）。

---

## M1-B.2 待施工（精确插入点已定位，供下一焦点步）
- **native Anthropic Forward**：`gateway_service.go:4959` 构建 `ForwardResult`；流式 `handleStreamingResponse`（:7156，写出点 ~:7478 `restored`）；非流式 `handleNonStreamingResponse`（:7778，`c.Data` 前）。建议：Forward 内按 flag 建 `cappedSink` 传入两函数，返回后在 :4959 回填 3 字段（或 streamingResult 加字段 + named-return defer）。
- **CC 路径**：`gateway_forward_as_chat_completions.go` 的流式 `writeChunk` / buffered 出口。
- **关键正确性**：`cappedSink` 必须新 `[]byte`，禁用池化 `getSSEScannerBuf64K`（gateway_service.go:7180）；写客户端字节不变（仅旁路 `capSink.Write(restored)`）。
- **验证**：复用 `gateway_streaming_test.go` harness 断言 `ResponseSample` 填充且限容、`Truncated` 正确、**客户端响应字节不变**；扩 cappedSink 大压力用例。

---

## 红线遵守
未 push、未开真门、未起服务、未真实调用、未碰真库迁移（迁移文件已写，等部署时 embed 自动跑）。改动未 commit（等学者指示）。feature flag 默认 false → 生产 dark。
