# D2 任务B Sub2API 采集武装与 retention 收口审查包

> 执行日期：2026-07-02  
> 执行目录：`D:\sub2api-trunk`  
> 分支：`wujie/video-capture-moat-20260702`  
> 状态：内部可用 / 可演示；G3 本机实跑已阻塞；未 push、未部署、未调用真实 provider、未读取 secret/env/token/cookie。

---

## 0. 结论

G0/G1/G2/G4 已通过代码与测试收口；G3 的真实 dev 链路验证因本机无可用 dev DB/Redis/server 监听而标记为 **已阻塞**，未伪造 SQL 或截图证据。

| 目标 | 状态词 | 判定 |
|---|---|---|
| G0 运行时门禁补跑 | 内部可用 | `go build ./...`、`go test ./...`、`golangci-lint run` 绿；`make` 在 Windows 不可用，已按任务书 fallback。 |
| G1 config.example 样例 | 内部可用 | `deploy/config.example.yaml` 已补 `gateway.content_capture` / `gateway.content_retention` 中文注释样例，默认仍为 false。 |
| G2 retention 接启动 | 内部可用 / 待复核 | 接入 `ProvideGenerationContentRetentionService`，双闸 `content_capture.enabled && content_retention.enabled` 才启动，cleanup 会 Stop；单测覆盖关/开。真实 daemon 环境验证留给 G3。 |
| G3 受控 dev 验证 | 已阻塞 | 本机无 5432/5433/6379/6380/8080/8090/18081 监听，未能执行 chat + mock video + SQL + Dashboard C。 |
| G4 契约快照 v1 | 内部可用 | 新增 JSON 快照和 Go 契约测试；API-key `/v1/video/tasks` 响应保留 `result_url`，新增兼容 `ResultURL`。 |

---

## 1. 背景与边界

- 只碰 Sub2API 仓：`D:\sub2api-trunk`。
- 起始 Git 真相：`git rev-parse --show-toplevel` = `D:/sub2api-trunk`，分支 `wujie/video-capture-moat-20260702`。
- 起始未跟踪项：`.impeccable/`、`MORNING_RESULT_2026_06_28.md`，本轮未触碰。
- 红线执行：无 push、无 deploy、无真实付费 provider、无 secret/env/token/cookie 读取或打印、无生产 DB 迁移、无 QCanvas 仓改动。

---

## 2. 变更清单

### G1 配置样例

- `deploy/config.example.yaml`
  - 新增 `gateway.content_capture` 示例：`enabled`、`response_max_bytes`、`prompt_max_bytes`。
  - 新增 `gateway.content_retention` 示例：`enabled`、`retention_days`、`batch_size`、`interval_seconds`、`dry_run`。
  - 样例全部保持暗态：`enabled: false`；没有改 Go 默认值。

### G2 retention 启动接线

- `backend/internal/service/wire.go`
  - 新增 `ProvideGenerationContentRetentionService(repo, cfg)`。
  - 仅当 `cfg.Gateway.ContentCapture.Enabled` 与 `cfg.Gateway.ContentRetention.Enabled` 同时为 true 时创建并 `Start()`。
- `backend/cmd/server/wire_gen.go`
  - 复用 `generationContentRepository` 接入 retention provider。
  - `provideCleanup` 传入 retention service。
- `backend/cmd/server/wire.go`
  - mirror cleanup 参数与 Stop 步骤。
- `backend/cmd/server/wire_gen_test.go`
  - 更新 cleanup 最小依赖测试入参。
- `backend/internal/service/generation_content_retention_service_test.go`
  - 新增 flag 关不启动、双 flag 开启动并触发 startup cleanup 的单测。

### G4 契约快照

- `backend/internal/handler/testdata/api_key_video_task_contract_v1.json`
  - 固定 Sub2API 侧 API-key video 创建请求顶层字段和响应关键字段。
- `backend/internal/handler/video_handler_c1_contract_test.go`
  - 新增 `TestD2ApiKeyVideoContractMatchesSnapshotV1`。
- `backend/internal/handler/video_handler.go`
  - 仅在 API-key video response 上新增 `ResultURL` PascalCase alias；保留既有 `result_url`。

### G0 lint 修复

为让 `golangci-lint run` 归零，修复本任务相关/相邻 video capture 文件的 gofmt、errcheck、staticcheck 问题：

- `backend/internal/repository/generation_content_repo.go`
- `backend/internal/server/routes/api_key_video_gateway_test.go`
- `backend/internal/service/cappedsink_test.go`
- `backend/internal/service/generation_content_redact.go`
- `backend/internal/service/generation_content_redact_structured_test.go`
- `backend/internal/service/video_gateway_adapter.go`
- `backend/internal/service/video_gateway_b1b2b3_test.go`
- `backend/internal/service/video_gateway_redact.go`
- `backend/internal/service/video_gateway_redact_blindspot_test.go`
- `backend/internal/service/video_gateway_security_test.go`
- `backend/internal/service/video_gateway_va2_test.go`
- `backend/internal/service/video_gateway_worker.go`

---

## 3. 实现细节

### retention 双闸

`GenerationContentRetentionService` 原本已具备 `Start/Stop/RunOnce`。本轮只补生产启动入口：

```go
if !cfg.Gateway.ContentCapture.Enabled || !cfg.Gateway.ContentRetention.Enabled {
    return nil
}
svc := NewGenerationContentRetentionService(repo, cfg)
svc.Start()
```

因此：

- 采集未打开时，retention 不空转。
- retention 未配置时，即使采集打开也不启动。
- `RunOnce(ctx, dryRun)` 仍可通过构造函数独立调用，不受 daemon 开关限制。

### 契约差异记录

只读侦察确认任务书所说 `ResultURL` PascalCase 与当前 Go wire JSON 有差异：原响应只有 `result_url`。本轮选择兼容扩展：

- 保留 `result_url`，不破坏现有 QCanvas C1 与旧调用方。
- API-key `/v1/video/tasks` 响应新增 `ResultURL`，满足 D2 快照。
- 用户 JWT `/api/v1/video/tasks` 不额外扩展，避免扩大响应面。

---

## 4. 验证命令与结果

### G0 baseline

```powershell
make test-backend
```

结果：失败，Windows 当前环境无 `make` 命令；按任务书 fallback。

```powershell
cd backend
go build ./...
go test ./...
golangci-lint run
```

结果：

- `go build ./...`：PASS。
- `go test ./...`：PASS。
- 首轮 `golangci-lint run`：17 issues；修复后最终 `0 issues`。

### TDD RED/GREEN

RED：

```powershell
go test ./internal/service -run "TestProvideGenerationContentRetentionService" -count=1
```

结果：FAIL，`ProvideGenerationContentRetentionService` 未定义。

```powershell
go test ./internal/handler -run "TestD2ApiKeyVideoContractMatchesSnapshotV1" -count=1
```

结果：FAIL，响应缺少 `ResultURL`。

GREEN：

```powershell
go test ./internal/service -run "TestProvideGenerationContentRetentionService|TestGenerationContentRetention" -count=1
go test ./internal/handler -run "TestD2ApiKeyVideoContractMatchesSnapshotV1|TestC1ApiKeyVideo" -count=1
go test ./cmd/server -run "TestProvideCleanup" -count=1
```

结果：全部 PASS。

### 最终 backend 门禁

```powershell
cd backend
go build ./...
go test ./...
golangci-lint run
```

结果：

- `go build ./...`：PASS。
- `go test ./...`：PASS。
- `golangci-lint run`：`0 issues`。

### secret scan

```powershell
python tools/secret_scan.py --include-untracked
```

结果：`secret-scan: no high-confidence tracked-plus-untracked findings`。

### frontend 质量门

```powershell
cd frontend
pnpm test:run
pnpm typecheck
pnpm lint:check
```

结果：

- `pnpm test:run`：单独复跑 PASS，97 files / 581 tests passed。首次与 `typecheck/lint` 并行跑时出现 3 个 `vue-i18n` mock 收集失败，单独复跑消失，按最终串行结果判定通过。
- `pnpm typecheck`：PASS。
- `pnpm lint:check`：PASS。

---

## 5. G3 本机验证状态

本机只读端口探测：

```powershell
Get-NetTCPConnection -State Listen -ErrorAction SilentlyContinue |
  Where-Object { $_.LocalPort -in 5432,5433,6379,6380,8080,8090,18081 }
```

结果：无输出。当前没有可用 dev PostgreSQL、Redis 或 Sub2API server 监听。

因此 G3 判定为 **已阻塞**：

- 未执行 chat 请求。
- 未执行 mock video HTTP 任务。
- 未执行 `ai_generation_content` SELECT。
- 未验证 Dashboard C `is_live=true`。
- 未截图，因为没有可访问服务。

### 授权 dev 环境的人类操作步骤

1. 在隔离 dev/内网环境准备 PostgreSQL + Redis + Sub2API server，不连接生产 DB，不配置真实付费 provider。
2. 复制 `deploy/config.example.yaml` 为 dev 专用配置，确认：

```yaml
gateway:
  content_capture:
    enabled: true
    response_max_bytes: 65536
    prompt_max_bytes: 262144
  content_retention:
    enabled: true
    retention_days: 90
    batch_size: 500
    interval_seconds: 3600
    dry_run: true
```

3. 启动服务后，用受控 API key 打 1 条 chat mock 请求；请求体可包含测试手机号/假 token，用于确认脱敏，但不要使用真实密钥。
4. 用受控 API key 打 1 条 mock video：

```bash
curl -sS -X POST "http://127.0.0.1:<port>/v1/video/tasks" \
  -H "Authorization: Bearer <DEV_API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{"provider":"mock","task_type":"reference_to_video","model":"mock-video-v1","prompt":"D2 dev capture check, phone 13800138000, token sk-test-placeholder","reference_image_url":"https://example.invalid/ref.png","aspect_ratio":"16:9","duration":5,"resolution":"720p"}'
```

5. 轮询 `GET /v1/video/tasks/<id>`，确认状态进入 `succeeded`，响应带 `result_url` 与 `ResultURL`。
6. 用只读 SQL 验证近 15 分钟至少 2 行：

```sql
SELECT id, task_id, model, left(prompt_redacted, 120) AS prompt_preview,
       left(response_redacted, 120) AS response_preview,
       prompt_bytes, response_bytes, response_truncated
FROM ai_generation_content
WHERE created_at >= NOW() - INTERVAL '15 minutes'
ORDER BY created_at DESC;
```

7. 验证没有原始密钥/PII 明文：

```sql
SELECT COUNT(*) AS suspicious_rows
FROM ai_generation_content
WHERE created_at >= NOW() - INTERVAL '15 minutes'
  AND (
    prompt_redacted ~ 'sk-[A-Za-z0-9_-]{20,}'
    OR response_redacted ~ 'sk-[A-Za-z0-9_-]{20,}'
    OR prompt_redacted ~ '13800138000'
    OR response_redacted ~ '13800138000'
  );
```

期望：`suspicious_rows = 0`。

8. 用 admin JWT 访问：

```bash
curl -sS "http://127.0.0.1:<port>/api/v1/admin/generation-content/stats" -H "Authorization: Bearer <ADMIN_JWT>"
curl -sS "http://127.0.0.1:<port>/api/v1/admin/generation-content/samples" -H "Authorization: Bearer <ADMIN_JWT>"
```

期望：`is_live=true`，样本墙可见脱敏 preview。

---

## 6. 风险

- G3 未在本机实跑，真实 `chat + mock video + SQL + Dashboard C` 仍待授权 dev 栈复核。
- `wire_gen.go` 是手工维护文件，本轮未运行 wire codegen；后续若运行 codegen，必须确认 retention 接线不被抹掉。
- `ResultURL` 是兼容 alias；旧 `result_url` 仍是既有主字段，调用方迁移前不要删除。
- retention daemon 真实启动依赖部署配置双闸，默认仍为关闭。

---

## 7. 回滚方案

代码回滚：

```bash
git revert <本轮本地提交>
```

配置回滚：

- 将 `gateway.content_capture.enabled` 保持或恢复为 `false`。
- 将 `gateway.content_retention.enabled` 保持或恢复为 `false`。

局部文件回滚：

- G2：回滚 `backend/internal/service/wire.go`、`backend/cmd/server/wire.go`、`backend/cmd/server/wire_gen.go`、`backend/cmd/server/wire_gen_test.go`、retention provider 测试。
- G4：回滚 `backend/internal/handler/video_handler.go`、`video_handler_c1_contract_test.go`、`testdata/api_key_video_task_contract_v1.json`。
- G1：回滚 `deploy/config.example.yaml` 的新增配置示例段。

---

## 8. 文件索引

- `deploy/config.example.yaml`
- `backend/internal/service/wire.go`
- `backend/cmd/server/wire.go`
- `backend/cmd/server/wire_gen.go`
- `backend/cmd/server/wire_gen_test.go`
- `backend/internal/service/generation_content_retention_service_test.go`
- `backend/internal/handler/video_handler.go`
- `backend/internal/handler/video_handler_c1_contract_test.go`
- `backend/internal/handler/testdata/api_key_video_task_contract_v1.json`
- `backend/internal/repository/generation_content_repo.go`
- `backend/internal/server/routes/api_key_video_gateway_test.go`
- `backend/internal/service/cappedsink_test.go`
- `backend/internal/service/generation_content_redact.go`
- `backend/internal/service/generation_content_redact_structured_test.go`
- `backend/internal/service/video_gateway_adapter.go`
- `backend/internal/service/video_gateway_b1b2b3_test.go`
- `backend/internal/service/video_gateway_redact.go`
- `backend/internal/service/video_gateway_redact_blindspot_test.go`
- `backend/internal/service/video_gateway_security_test.go`
- `backend/internal/service/video_gateway_va2_test.go`
- `backend/internal/service/video_gateway_worker.go`

---

## 9. 可复制后续提示词

```text
继续 Sub2API D2 G3 受控 dev 验证。只允许在隔离 dev/内网环境打开 gateway.content_capture.enabled=true 与 gateway.content_retention.enabled=true；不得读取或打印 secret/env/token/cookie；不得调用真实付费 provider；video 只走 provider=mock。按 _review/capture-arming-D2-20260702/SUMMARY.md 的 G3 步骤执行：打 1 条 chat mock 请求 + 1 条 mock video 任务，SQL 验证 ai_generation_content 近 15 分钟至少 2 行且 suspicious_rows=0，再验证 /api/v1/admin/generation-content/stats 与 /samples 的 is_live=true。输出 SELECT 摘要、HTTP 状态、截图或明确阻塞原因；不 push、不 deploy。
```
