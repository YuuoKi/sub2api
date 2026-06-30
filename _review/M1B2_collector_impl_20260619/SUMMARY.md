# M1-B.2 · 采集口 response 限容 tee · 复审包(SUMMARY)

> 分支:`wujie/trunk`(基线 `6478237a` = B.1)｜ 日期:2026-06-19 ｜ 接法:包一层旁路写出器(学者拍板)
> 修者:Claude Code｜交 Codex 独立复核(修者≠审者)｜复核重点:客户端透明性 / buffer 独立 / fail-open / flag 默认关 / 3 个黄点

---

## 0. 一句话

把 B.1 留空的 `ForwardResult.ResponseSample/ResponseTruncated/ResponseBytes` 填上:在「构建 ForwardResult 的转发总入口」临时把 `c.Writer` 包一层 `capturingResponseWriter`,每写一笔给客户端的字节就顺手抄一份进限容 `cappedSink`,**绝不改变发给客户端的任何字节**。flag 关时不分配/不包装。附带修掉 B.1 标黄的电话正则误伤 + 补测试。

---

## 1. 主线改动(包一层写出器)

### 新增:`backend/internal/service/gateway_response_capture.go`
- `capturingResponseWriter`:内嵌 `gin.ResponseWriter`,**只重写 `Write`/`WriteString`** —— 先调底层写客户端、再 `sink.Write(已写出的 p[:n])`,**原样返回底层 `(n,err)`**。`Status/Size/Written/Flush/Hijack/Pusher` 全由内嵌接口自动提升到底层。
- `(s *GatewayService) beginResponseCapture(c) (*cappedSink, func())`:flag 关 / `c==nil` → 返回 `(nil, 空函数)`,**不分配不包装**(热路径零开销);flag 开 → 建 sink(cap=`response_max_bytes`,默认 64KiB)、把 `c.Writer` 换成包装器、返回还原函数。
- `fillResponseSample(r, sink)`:任一为 nil 即 no-op;否则回填三字段(`Bytes()/Truncated()/Total()`,均 nil 安全)。
- `(s *GatewayService) responseCaptureMaxBytes()`:读配置,默认 64KiB。

### 接入点(2 处,均**不碰**写客户端的语句本身)
| 文件:行 | 作用 | 覆盖出口 |
|---|---|---|
| `gateway_service.go:4349` | `Forward` 签名改具名返回 `(result, err)` | — |
| `gateway_service.go:4357-4359` | `beginResponseCapture` + `defer restore` + `defer fillResponseSample(result,…)` | 出口 ①②(原生流/非流)③④(透传流/非流) + bedrock/web-search 兜底 |
| `gateway_forward_as_chat_completions.go:188` | `beginResponseCapture` + `defer restore`(正常响应前;错误分支已先 return) | — |
| `gateway_forward_as_chat_completions.go:199` | `fillResponseSample(result, respSink)`(return 前) | 出口 ⑤⑥(CC 流式/缓冲) |

`Forward` 用**具名返回 + defer** → 无论走哪条分支(passthrough/bedrock/web-search/native/error)返回的 `result` 都被回填;error 路径 `result==nil` → no-op。原生 `handleStreamingResponse`/`handleNonStreamingResponse` 内 `w := c.Writer` 进入时取到包装器,**这两个函数及其 ~20 处测试调用一行未改**。

### 范围与缓做
- **覆盖**:任务书 6 个在范围出口 ①-⑥,外加 bedrock(`Forward` 总入口免费兜住)。
- **缓做(待 B.3)**:`gateway_forward_as_responses.go`(出口 ⑦⑧)未接 → 其 `ResponseSample` 保持零值、fail-open,符合 M1-A 既定。

---

## 2. 黄点修复(B.1 Codex 标黄,并入本步)

`generation_content_redact.go:18` 电话正则由 `\+?\d[\d().\-\t ]{7,}\d`(过宽)改为三分支:
```
\+\d[\d().\-\t ]{6,16}\d | \b\d{2,4}[.\-\t ]\d{3,4}[.\-\t ]\d{3,4}\b | \b1[3-9]\d{9}\b
```
国际(带+) / 分隔分组(分隔符非数字,强制 3 组) / 中国手机(1[3-9] 起 11 位)。长度锚 + `\b` 使**12 位 UUID 末段、16 位纯数字 id、模型名、日期**均不命中。
> 关键洞察:旧正则会把 UUID 末段 `446655440000` 切成 `[PHONE]`,**破坏**了下游 `content_moderation` 对完整 UUID 的整体脱敏(残片 `...a716-[PHONE]` 反而泄露)。新正则不碰 UUID → 由密钥脱敏整体脱为 `[已脱敏]`。已加端到端断言。

---

## 3. 验证(全部实跑,本地、不真调用、不连真库)

| 命令 | 结果 |
|---|---|
| `go build ./...` | **exit 0** |
| `go vet ./internal/service/... ./internal/handler/...` | **exit 0** |
| `go test ./internal/service/...`(默认) | **ok 47.8s** |
| `go test ./internal/repository/...` | **ok 4.9s** |
| `go test -tags unit ./internal/service/...`(流式 harness) | **ok 90.3s**(确认热路径在两种构建模式都不回归) |
| grep `getSSEScannerBuf64K\|sseScannerBuf64K` in `cappedsink.go` / `gateway_response_capture.go` | **0 命中**(命门:不复用池化缓冲) |

### 新增 9 个单测(全 PASS)
- `gateway_response_capture_test.go`(5):
  - `..._Transparent`:写客户端字节 == 写入字节,sink 抄到同一份。
  - `..._CapsSinkNotClient`:**客户端收到全部 20 字节,sink 仅 8 字节 + Truncated + Total=20**(cap 只约束副本)。
  - `BeginResponseCapture_Enabled_CData` / `_Disabled_NoOp`:开/关行为(关时 `c.Writer` 不被替换)。
  - `HandleStreamingResponse_TeeTransparent`:**同一 SSE 跑裸 writer / 包装 writer 两遍,客户端字节逐字节相等,且 sink==客户端字节**(经真实 `reverseToolNamesIfPresent`/flush 路径)。
- `generation_content_redact_phone_test.go`(3):电话被脱、UUID/16 位 id/模型名/日期不被误伤、UUID 不被电话正则切碎。
- `generation_content_collector_panic_test.go`(1):注入 `Create` 会 panic 的 fake 仓 → `Collect` recover、正常返回(日志见 "generation content collect panic … recovered")。

---

## 4. 命门自查(对照判断基准)

- **客户端透明(基准0)**:包装器 `Write/WriteString` 原样返回底层 `(n,err)`,只额外旁路抄一份 → 结构性透明。透明性测试逐字节断言通过(单元 + 真实流式路径)。**未改任何写客户端的语句**。
- **buffer 独立(基准4)**:`cappedSink` 自持缓冲、`Write` 内 `append` copy 输入;包装器只抄 `p[:n]`;全程不引用池化 SSE 缓冲(grep 0 命中)。
- **限容(基准2)**:cap 默认 64KiB,sink 写满即停、置 `Truncated`、`Total` 仍记真实总字节;内存上界 = 并发流数 × cap。
- **fail-open(基准3)**:`cappedSink.Write` 永不报错/永不超额;`Collector.Collect` recover panic + 吞仓库错误;采集在计费之后、与计费隔离。
- **flag 默认关(基准5)**:`gateway.content_capture.enabled` 仍默认 false → `beginResponseCapture` 直接返回空操作,不分配 sink、不包装 `c.Writer`、热路径零开销;回滚=翻开关。
- **安全核查**:全仓对 `c.Writer` 的类型断言**只有** `http.Flusher`(接口,包装器经内嵌自动满足);无具体类型断言 / `Hijacker`/`Pusher`/`ReaderFrom` 断言会被破坏;中间件无人替换/具体断言 `c.Writer`。

---

## 5. 红线声明
- **未 push**(origin=开源上游)。
- **未开真门**(未碰 env smoke 开关,flag 仍默认 false)。
- **未真实调用付费上游**(全程 mock/`io.Pipe`/recorder)。
- **未连/未迁真库**(迁移 140 已在 B.1,部署时 embed 自动跑)。
- **未碰** `usage_logs` 计费热表、**未动** `main` 分支、**未改** 6 个写客户端语句 / 4 个 handle* 函数签名 / 任何现有测试调用 / schema / wire。
- `_review/` 不入 git、不 push。

## 6. 改动清单
- 新增:`service/gateway_response_capture.go`、`service/gateway_response_capture_test.go`、`service/generation_content_redact_phone_test.go`、`service/generation_content_collector_panic_test.go`。
- 修改:`service/gateway_service.go`(+8/-1)、`service/gateway_forward_as_chat_completions.go`(+6)、`service/generation_content_redact.go`(+4/-3)。
- `git diff --stat`(tracked):3 files changed, 17 insertions(+), 4 deletions(-);另 4 个未跟踪新文件。
