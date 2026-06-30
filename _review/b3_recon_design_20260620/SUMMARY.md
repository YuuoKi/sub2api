# B.3 只读侦察 + 方案设计 · 两条漏采路径(responses + openai_gateway)· SUMMARY

> 任务编号:`G_B3_RECON_DESIGN` ｜ 生成:2026-06-20 ｜ 执行:Claude Code(ultracode 只读 verify+对抗 workflow)
> 性质:**纯只读侦察 + 方案设计 · 为 B.3 动手备料**。零代码、零服务、零数据库、零 git 改动。**回来不急着拍板。**
> 并行安全:与"修房子"主线(`G_M1B_REALRUN_ENV_AND_VERIFY`,正起本地 PG/占 5433·6380·9099·8090)同时跑;全程只读、未起服务/连库/占端口、未改任何代码/分支/worktree;本文件写在独立目录 `_review/b3_recon_design_20260620/`,与主线 `_review/M1B_realrun_verify_*`、支线 `_review/selling_value_output_*` 文件级不碰撞。
> 方法:Dynamic Workflow,3 只读 verify(独立读真码核对) + 2 path-designer + 4 对抗 skeptic + 1 综合,共 10 个 `Explore`(工具级无 Edit/Write)。**关键事实经 workflow verify 与我本人自查双确认。**

---

## 0. 一句话结论(先读)

两条漏采路径**难度天差地别**:

- **路径1 · responses(anthropic GatewayHandler):完全对称,~8 行 2 处改动即可全采(prompt+response)。** `ForwardAsResponses` 返回的就是 `*ForwardResult`(同主路径同类型),且只经 `c.Writer` 写客户端 → 现成 tee 照搬即抓。低工作量、中风险。**建议先做。**
- **路径2 · openai_gateway(OpenAIGatewayHandler,3 处):结构性适配,不能简单镜像。** 根因三条:① 结果类型不符(`*OpenAIForwardResult` 无 response 字段)② 采集器/tee/开关都长在另一个 service(anthropic `GatewayService`,openai 侧够不到)③ WebSocket 子路(:1363)响应走 `clientConn.Write` 绕开 `c.Writer`,**tee 物理抓不到**。

**我对 workflow 对抗结论的三处修正(重要,见 §4):**
1. 有 skeptic 把 openai 判成 **fatal "架构级不可采"——过度**。真相:HTTP 两路(:426/:800)response **可以采**,只需把采集器与 `*ForwardResult` **解耦**(args 显式带 3 个 response 值)。**唯一真·采不到的是 WS 的 response**(channel 绕开 c.Writer)。
2. designer 自相矛盾(一处说 ForwardAsAnthropic 返回 `*ForwardResult` 无 mismatch)——**可证伪**::426/:800 的 `result` 都被塞进 `OpenAIRecordUsageInput{Result *OpenAIForwardResult}`,按 Go 类型推断 `result` 必为 `*OpenAIForwardResult`。**三处 openai 都是类型不符。**
3. "responses 请求体脱敏覆盖不足/blocking"——**过度**。`RedactJSON` 是**按敏感键名 + 正则内容**脱敏(access_token/password/Bearer/PII 等,与消息 schema 无关),Responses/OpenAI 体得到的保护与 Anthropic 体**一样**。降级为 verify 项,非 blocker。

---

## 1. 现状核实(采集三件套 + 两条漏采路径,file:line,双确认)

### 1.1 采集机制三件套(均在 anthropic `GatewayService`)
| 件 | 位置 | 作用 |
|---|---|---|
| 采集器 | `service/generation_content.go` | `Collect(ctx, GenerationContentCaptureArgs)`,response **仅**读 `args.Result.{ResponseSample,ResponseBytes,ResponseTruncated}`(:93-97);prompt=`args.PromptBody`(≤256KiB);fail-open(recover+吞错) |
| 门 | `CollectGenerationContent`(:141-146) | 仅 collector 已注入 **且** `contentCaptureEnabled()`(`cfg.Gateway.ContentCapture.Enabled`,默认 false)时执行 |
| tee | `service/gateway_response_capture.go` | `capturingResponseWriter` 包 `c.Writer`,逐字节抄进 `cappedSink`(透明/字节相同/独立缓冲);`(s *GatewayService) beginResponseCapture(c)` 开关开时换 `c.Writer`;`fillResponseSample(*ForwardResult,*cappedSink)` 回填 3 字段 |

**tee 现已接的出口(仅 2 处):** `gateway_service.go:4357-4359`(主 Forward,/v1/messages)、`gateway_forward_as_chat_completions.go:188+199`。
**已工作的采集调用(金标准):** `gateway_handler.go:535`、`:935`(已提交 eca1b65c)、`gateway_handler_chat_completions.go:282` —— 均在 `submitUsageRecordTask` 闭包里、与 `RecordUsage` 並列调 `CollectGenerationContent`,传 `PromptBody=parsedReq.Body`、`Result=result(*ForwardResult)`。

### 1.2 漏采路径1 · responses(`handler/gateway_handler_responses.go`,`GatewayHandler.Responses`,POST /v1/responses)
- Forward:`h.gatewayService.ForwardAsResponses(ctx, c, account, forwardBody, parsedReq)` 返回 `(*ForwardResult, error)`(handler:227;service `gateway_forward_as_responses.go:31`)。**返回同主路径同类型 `*ForwardResult`。**
- **tee 未接** `ForwardAsResponses`(`beginResponseCapture` 只在 Forward + ForwardAsChatCompletions)。
- **采集调用缺失**:RecordUsage 闭包(handler:267-287)无 `CollectGenerationContent`。
- 闭包内已在 scope:`parsedReq`(:159,有 `.Body`)、`result(*ForwardResult)`、`requestPayloadHash`(:263)、`subject.UserID`、`apiKey(.ID/.GroupID/.User)`、`account.ID`、`reqModel`。
- ✅ **独立 verify(high)+我自查双确认**:`ForwardAsResponses` 只经 `c.Writer` 写客户端 —— buffered `c.Data`(:342)、streaming `fmt.Fprint(c.Writer)`+`c.Writer.Flush()`(:434/442/455/457)、error `c.JSON`。tee 装在函数顶端即可全抓。

### 1.3 漏采路径2 · openai_gateway(`handler/openai_gateway_handler.go`,`OpenAIGatewayHandler`)
- handler 字段 `gatewayService *service.OpenAIGatewayService`(:30)——**不是** anthropic `*GatewayService`;无 collector、无 `contentCaptureEnabled`、无 `beginResponseCapture`。持有 `cfg`。
- `OpenAIForwardResult`(`openai_gateway_service.go:213-239`)有 `RequestID`/`Model`/`Usage`…**但无** `ResponseSample/ResponseBytes/ResponseTruncated`。
- **三处 RecordUsage 站点**(均传 `OpenAIRecordUsageInput{Result *OpenAIForwardResult}`):

| 站点 | 端点 | forward | prompt | 客户端写法 | response 可 tee? |
|---|---|---|---|---|---|
| `:426` | Responses(`/openai/v1/responses`) | `OpenAIGatewayService.Forward`(handler:333) | `body`(:420) | HTTP `c.Writer`(`openai_gateway_service.go:4212 w:=c.Writer`,`:4711 c.Data`) | ✅ 可(需接) |
| `:800` | Messages dispatch(`openai_messages`) | `OpenAIGatewayService.ForwardAsAnthropic`(handler:716) | `body`(:795) | HTTP `c.Writer`(`openai_gateway_messages.go:725 fmt.Fprint`+`:735 Flush`,`:456 c.JSON`) | ✅ 可(需接) |
| `:1363` | WebSocket `AfterTurn`(每轮) | WS forwarder | `firstMessage`(:1373) | **WebSocket** `clientConn.Write`(`openai_ws_forwarder.go:2605/2807`) | ❌ **绕开 c.Writer,tee 抓不到** |

- ✅ **双确认**:`OpenAIGatewayService` 结构(:315-345)无 `generationCollector`;无 `contentCaptureEnabled/beginResponseCapture/fillResponseSample`(grep 零命中);WS response 确经 `clientConn.Write(coderws.MessageText,…)` 非 HTTP。

---

## 2. 路径1 · responses —— 方案(完全对称,推荐先做)

**对称性结论:HIGH。** 与主 Forward / ForwardAsChatCompletions 同 service、同 tee、同 collector、同 args、同 `*ForwardResult`。差的只是没接而已。

**tee 是否需延伸:不需。** 现成 tee 照搬;`ForwardAsResponses` 只经 `c.Writer` 写 → 顶端装 `beginResponseCapture(c)` 即抓 buffered+streaming(+error)。

**采集接法(2 处改动,~8 行):**
1. `service/gateway_forward_as_responses.go` 顶端(`startTime` 后)镜像 `gateway_service.go:4357-4359`:
   ```go
   respSink, restoreWriter := s.beginResponseCapture(c)
   defer restoreWriter()
   defer func() { fillResponseSample(result, respSink) }()   // result 为命名返回/闭包捕获
   ```
2. `handler/gateway_handler_responses.go` 的 RecordUsage 闭包内(:281 后)镜像 `:935`:
   ```go
   h.gatewayService.CollectGenerationContent(ctx, service.GenerationContentCaptureArgs{
     RequestID: result.RequestID, UserID: subject.UserID, APIKeyID: apiKey.ID,
     GroupID: apiKey.GroupID, AccountID: account.ID, Model: reqModel,
     RequestPayloadHash: requestPayloadHash, PromptBody: parsedReq.Body, Result: result,
   })
   ```

**改动面:** 2 文件 / ~8 行新增 / 0 删除 / 0 结构改动 / 0 新基建。
**风险分级:中**(触 ForwardAsResponses 转发热路径,字节红线适用;但 tee 机制已两路生产验证,fail-open+热路径零开销均现成)。
**能否镜像:能,100% 镜像主 Forward。**

**对抗发现 + 我的裁决:**
- skeptic「error 响应漏抓」(serious)→ **我裁:基本 moot。** tee 装在 `ForwardAsResponses` 顶端则其内 error 写(c.JSON)也被抓;且 collect 只在 forward 成功后的闭包触发,forward-error 本就不采(与现有 Forward 一致)。capture 完整性对 error 是 nice-to-have,非阻断。
- skeptic「Responses 体脱敏覆盖不足」(serious/blocking)→ **我裁:降为 LOW/verify 项**(§4 修正③:RedactJSON 按键名+正则,format-agnostic)。
- `defer` 顺序需 `restoreWriter` 先于 `fillResponseSample`(后进先出);`result` 用命名返回确保 defer 读到终值 —— 实现期注意。

---

## 3. 路径2 · openai_gateway —— 方案(结构性适配,分子路)

**对称性结论:HTTP 两路中等(需先拆掉三个结构障碍);WS response 无对称(物理采不到)。**

**根因(为何不能简单镜像):**
- ① **类型不符**:`GenerationContentCaptureArgs.Result` 是 `*ForwardResult`;openai 三处 `result` 都是 `*OpenAIForwardResult`(无 3 个 response 字段)。
- ② **基建在另一 service**:collector/tee/开关都在 anthropic `GatewayService`,openai handler/service 够不到。
- ③ **WS channel**:`:1363` response 走 `clientConn.Write`,tee(包 c.Writer)抓不到。

**修正后的方案(我对 workflow 的纠偏:HTTP response 可采,非 fatal):**

### 3.A 三个结构改动(一次性,使 openai 侧具备采集能力)
1. **解耦采集器与 `*ForwardResult`(keystone,~6 行)**:`GenerationContentCaptureArgs` 加显式 `ResponseSample []byte / ResponseBytes int / ResponseTruncated bool`;`Collect` 中 `Result!=nil` 用 Result(现状不变),否则用显式字段。**向后兼容**,且让任何"无 *ForwardResult"的调用方都能采。← 这一步直接化解类型不符。
2. **tee 变可复用(~30 行重构,anthropic 行为不变)**:把 `beginResponseCapture`/`fillResponseSample` 的体抽成 service 内**与 receiver 无关的自由函数**(入参 `enabled bool, maxBytes int`),`GatewayService` 与 `OpenAIGatewayService` 各加薄包装。proven tee 代码零行为变化。
3. **给 openai 侧注入同一 collector + 开关(~15 行)**:`GenerationContentCollector` 本就 service-agnostic(`repo+cfg` 构造)。给 `OpenAIGatewayService` 加 `generationCollector` 字段 + `SetGenerationContentCollector` + `contentCaptureEnabled()` + `CollectGenerationContent`(镜像 anthropic 侧),DI 处把**同一个 collector** 注入两边。

> 备选(workflow 提的):handler 多持一个 `*GatewayService` 字段借方法 —— **不推荐**(跨 service 字段耦合、泄露内部、code smell)。3.A.2/3.A.3 的"抽自由函数 + 各自薄包装"更干净。

### 3.B 三处站点接法(结构就绪后)
- **:426 Responses / :800 Messages(HTTP,可全采)**:各在 forward 外/内装 tee(`beginResponseCapture(c)`+`defer` 回填到本地 sink),RecordUsage 闭包内 `CollectGenerationContent`,传 `PromptBody=body`、`ResponseSample=sink.Bytes()`/`ResponseBytes=sink.Total()`/`ResponseTruncated=sink.Truncated()`、`RequestID=result.RequestID`、`Model=reqModel`、归因(apiKey/subject/account)。**不依赖 `*ForwardResult`。** 各 ~15 行。
- **:1363 WebSocket(response 真采不到)**:`prompt-only`——`PromptBody=firstMessage`、无 response;或整段 defer。**response 需在 `openai_ws_forwarder.go:2605/2807` 的 `clientConn.Write` 旁加 WS-frame tap**(独立子任务,B.4+)。诚实硬限,不可用 c.Writer tee 糊弄。

**改动面:** 主要在 `openai_gateway_handler.go` + `openai_gateway_service.go` + 2 个 service 文件薄改;~3 结构改动(args/tee/注入)+ 3 站点接线 ≈ **中等(半天~1 天)**。
**风险分级:中—高**(动第二个 service + 两条 HTTP 转发热路径 + 1 条 WS;字节红线对 :426/:800 同样适用;WS 部分功能性缺口需如实记录)。
**能否镜像:HTTP 部分接线镜像 :535/:935;但前置三结构改动是净新;WS response 不镜像。**

**对抗发现 + 我的裁决:**
- skeptic「fatal 架构不可采」→ **我裁:过度。** HTTP 可采(3.A.1 解耦即解);**仅 WS response 是真硬限**。
- skeptic「跨 service 借方法破封装」(serious)→ **成立,已采纳**:用 3.A.2/3.A.3 的接口/自由函数法,不要多字段借方法。
- designer「ForwardAsAnthropic 返回 *ForwardResult 无 mismatch」→ **可证伪**:`result` 进 `OpenAIRecordUsageInput.Result(*OpenAIForwardResult)`,故三处均 `*OpenAIForwardResult`。

---

## 4. 对抗收敛 · 我对 workflow 的逐条裁决

| workflow 发现 | 原判 | 我的裁决 |
|---|---|---|
| responses 完全对称、~8 行 | low effort | ✅ 采纳(我自查双确认 tee 适用) |
| openai 三处类型不符 | high/fatal | ⚠️ **改判 medium**:HTTP 解耦即采,仅 WS-response 硬限 |
| openai HTTP `result` 是否 *ForwardResult | designer 自相矛盾 | ✅ **判定 = `*OpenAIForwardResult`**(Go 类型推断可证) |
| responses 体脱敏覆盖不足 | serious/blocking | ⚠️ **改判 LOW/verify**:RedactJSON 按键名+正则,format-agnostic |
| responses error 响应漏抓 | serious | ⚠️ **改判 minor/moot**:顶端装 tee 即抓;forward-error 本不采 |
| 跨 service 借方法破封装 | serious | ✅ 采纳 → 用接口/自由函数,不多字段 |
| WS response 经 clientConn.Write 抓不到 | medium | ✅ **采纳为唯一真硬限** → prompt-only 或 B.4 frame-tap |

---

## 5. 建议实现顺序 + 风险分级

1. **先做 responses(B.3 主切)** —— 解锁、对称、~8 行、中风险。当场可全采 prompt+response。
2. **再做 openai HTTP(:426/:800)** —— 一次性三结构改动(解耦 args + tee 可复用 + 注入 collector)后接两站点;中—高风险、半天~1 天。
3. **WS response(:1363)单列 B.4+** —— 真采不到;先 prompt-only,response 留 `clientConn.Write` frame-tap 独立设计。

两路均按既有铁律(随 main-path eca1b65c 已确立):**修者≠审者 + 全栈真跑 + 字节红线(基准0)**;flag 默认 false 灰度;fail-open;热路径零开销。

---

## 6. 给学者的拍板清单(回来不急着拍)

| # | 决策点 | 选项 | 我的倾向 |
|---|---|---|---|
| **D1** | 先做哪条 | responses 先 / openai 先 / 一起 | **responses 先**(解锁、对称、最小) |
| **D2** | openai HTTP(:426/:800)response 采不采 | 采(三结构改动)/ 只采 prompt | **采**(解耦后可行,价值=覆盖 openai 主流量产出) |
| **D3** | openai tee 复用法 | 抽接口/自由函数 / handler 多持 GatewayService 字段 | **抽接口/自由函数**(免跨 service 耦合) |
| **D4** | WS(:1363)response | prompt-only 现做 / 整段 defer B.4 / 现做 frame-tap | **prompt-only 现做,response 留 B.4** |
| **D5** | 脱敏 verify 门 | 实现前先验 RedactJSON 对 Responses/OpenAI 体 / 信其 format-agnostic 直接做 | **先快验一遍**(低成本,确认键名+正则覆盖) |

---

## 7. 红线遵守声明 + 透明记账
全程只读:未起服务、未连库、未占端口、未改任何代码、未 `git add/commit/push`、未动分支/worktree/main、未实现任何方案、未碰主线/支线 `_review` 目录。`git status` 仅本目录 `_review/b3_recon_design_20260620/` 新增(不入 git)。
透明:workflow 10 agent / ~533k tokens / ~13 分钟;3 verify 全 `confirmed`(high)且与我自查一致;对抗层有 1 处 fatal 误判 + 1 处 designer 自相矛盾 + 1 处 blocking 过判,**均经我用真码/类型推断纠偏**(§4)。本设计以"独立 verify + 我本人双确认"为骨,非机械搬运 agent 结论。
