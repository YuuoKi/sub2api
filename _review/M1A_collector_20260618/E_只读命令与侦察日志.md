# E · 只读命令与侦察日志 — M1-A

> 全程只读：Read / Grep / Glob + 只读 git。无写操作、无凭据提取、无服务启动、无真实调用。

## 侦察方法
3 个并行 Explore 只读子代理 + 自读关键文件交叉验证 + 1 个 Plan 只读子代理做实现设计压测。

### 子代理 1 — 采集口写入点 + 燃料可达性（§6 核心）
读：`backend/internal/repository/usage_log_repo.go`、`backend/internal/service/usage_log.go`、`backend/internal/service/gateway_service.go`(RecordUsage/recordUsageCore/buildRecordUsageLog/ForwardResult/流式&非流式 handler)、`backend/internal/service/gateway_request.go`(ParsedRequest)、`backend/internal/handler/gateway_handler.go`(submitUsageRecordTask)。
产出：甲/乙 混合判定（见 [02](./02_燃料可达性结论.md)）。

### 子代理 2 — schema 字段 + 归因 + migration + 读取侧
读：`backend/ent/schema/usage_log.go`(全字段)、`backend/migrations/*`(编号机制/035 分区/042 清理/077-078 加列先例/139 最新)、`backend/internal/repository/usage_log_repo.go`(select/insert/scan)。
产出：字段分类表、归因缺口（员工✅团队✅任务❌）、加列/建表流程、行量信号（分区+批量+清理→不宜把大文本塞 usage_logs）。

### 子代理 3 — 脱敏 + 质量信号 + 结果形态
读：`backend/internal/service/video_gateway_redact.go`、`backend/internal/service/content_moderation_redact.go`、`backend/internal/util/logredact/redact.go`、`backend/internal/service/drama_gateway_service.go`(ReuseStatus/QualityScore/ReviewStatus/RecordDramaShotDecision)、`backend/internal/service/video_gateway_types.go`。
产出：可复用脱敏函数清单 + PII 缺口、质量信号现状（散在 drama、未关联 usage_log）、结果形态/体积表。

### 自读交叉验证（关键命门，亲眼确认）
- `gateway_request.go:55-99`（ParsedRequest.Body 等字段）
- `gateway_service.go:8270-8474`（RecordUsage→recordUsageCore 丢弃 ParsedRequest；buildRecordUsageLog/writeUsageLogBestEffort）
- `gateway_service.go:485-519`（ForwardResult 无响应正文；UpstreamFailoverError.ResponseBody）
- `gateway_handler.go:470-544`（submitUsageRecordTask 异步旁路 + parsedReq/result 同在闭包）

### Plan 子代理 — 实现设计压测
产出 6 个 response emit 出口精确行号、cappedSink/ForwardResult 字段方案、新表 SQL+索引（仿 `content_moderation_logs` 裸 SQL 先例，不走 ent）、脱敏顺序+PII pass、改动清单、风险/回滚、验证。

## 只读 git
```
git rev-parse --abbrev-ref HEAD   → wujie/trunk
git rev-parse HEAD                → 40e83bf4...
git status --porcelain            → 仅 ?? _review/（侦察全程跟踪树未改）
```

## 凭据安全
侦察涉及脱敏/采集链路，未读取/打印/落盘任何 key/token/凭据；schema 与脱敏代码仅只读查看，未提取任何密钥样例。停止条件 #4 未触发。
