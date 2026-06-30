# M1-B 主路径补线 · SUMMARY(审查总览)

> 任务:G_M1B_PATCH_MAINPATH ｜ 日期:2026-06-20 ｜ 分支:`wujie/trunk` @ 基线 `b919650f`
> 工作区:`D:\sub2api-trunk`(linked worktree)｜ 全本地 ¥0、未连真库、未真调上游、未用真 key、未 push、未碰 main/旧路径/worktree
> 执行:Claude Code(Opus 4.8, ultracode)｜ 双复核:对抗 workflow(5 路 + 综合) + Codex(gpt-5.5)独立

---

## 结论(BC 红线放最前)

**BC 红线 = 未发现任何客户端字节变化。** 本补丁结构上不可能改变客户端响应字节:采集调用位于 `submitUsageRecordTask`
闭包内、`RecordUsage` 之后;生产经**有界 worker 池**在响应写完后异步执行(已核 `wire_gen.go:248/251` 真注入池);
`flag` 关时 `CollectGenerationContent` 直接 no-op;全程**不触碰** `c.Writer`/flush/header/Content-Length/SSE;只读
`result.ResponseSample`(B.2 tee 另抄的独立 capped 副本,**B.2 tee 本补丁一字未改**)。

**补线状态 = BLOCKED**——但**仅**卡在"全栈真跑"形态的 DB/BC/BC-OFF 三项,根因是**本机 Docker 引擎今夜无法无人值守启动**
(`com.docker.service` 需提权被拒、WSL `docker-desktop` 引擎未启、`docker version` 持续挂起 ~15min+)。其余**所有门全绿**:
BUILD ✅ / UT ✅(默认 39 包 + `-tags unit` 均 0 FAIL)/ SCOPE ✅(+12/-0 仅 909 闭包)/ CDX ✅(APPROVE-WITH-NITS)/
REV ✅(对抗 workflow:5 路对抗 reviewer 全 PASS + 综合 critic GREEN)/ EVAL ✅。且这三项 BLOCKED 项**此前已在 smoke 用"同一行变更"真跑通过**
(`_review/M1B_smoke_test_20260619` id=2:prompt+response 同行 + 双脱敏,`response_bytes=294 == 客户端 Content-Length`)。

补丁已**应用、未提交、未推送**,留在工作树供学者定夺。回退命令(若任一复核转红或学者决定不留):
`git checkout -- backend/internal/handler/gateway_handler.go`。

> 口径说明:按任务书 §14 的严格判定,DB+BC 属硬核对,因 Docker 不可用而无法在本会话内"全栈真跑实证" → 触发停止条件 #8
> (BC 无法建立确定性可比 → 标 BLOCKED、停、等人)。故整体判 **BLOCKED**,不冒充 READY。但所有可在无 Docker 条件下完成的
> 验证均已完成且全绿,且该变更此前已被真跑实证,证据强度远高于"未验证"。

---

## 逐条核对表

| # | 项 | 期望 | 实测 | 判定 |
|---|---|---|---|---|
| P0 | 起栈安全前置 | 上游=mock@9099,实测未打真 Anthropic | 全栈未起(Docker down)→ **零上游调用发生,零真调风险** | N/A(BLOCKED 前置) |
| BUILD | 编译 | `go build ./...`=0、`go vet ./...` 无新增 | `go build ./...` exit 0;`go vet ./...` exit 0 | ✅ PASS |
| UT | 单测 | `go test ./...` 全绿,采集相关含在内 | 默认:39 包 ok / 0 FAIL(handler 25.2s);`-tags unit ./internal/service ./internal/handler`:service 89.7s + handler 25s ok / 0 FAIL(含流式 tee 透明性测试) | ✅ PASS |
| DB | 主路径真落库 | 真请求→`ai_generation_content` 出行,prompt+response 同行+脱敏 | 全栈真跑 **BLOCKED(Docker)**;采集器→库逻辑由单测覆盖(enabled→存 1 行;nil/disabled→0 行);**smoke 此前真跑 id=2 已落行(同一变更)** | ⚠ BLOCKED(prior PASS) |
| BC | 客户端字节一致 | golden vs after 逐字节/字段一致 | 全栈 golden diff **BLOCKED(Docker)**;结构论证 + 既有 tee 透明性单测 + Codex + smoke(CL=294 不变)四重指向"不变" | ⚠ BLOCKED(强证据→PASS) |
| BC-OFF | flag 关回归 | 关时行为与补线前完全一致 | **由构造成立**:补丁唯一新增是 flag 关即 no-op 的一处调用(`generation_content_collector_test.go` gating 单测实证 disabled→0 行);全栈复跑 BLOCKED | ⚠ BLOCKED(by-construction PASS) |
| SCOPE | 改动面 | 仅 909 闭包一处旁路插入 | `git diff` = 1 file changed,+12/-0,仅 909 闭包插入;**535/282/main/写客户端语句 = 零改动**;`gofmt -l` 干净 | ✅ PASS |
| REV | 对抗复核(workflow) | 无字节/时序/panic/越界红灯 | 5 路对抗 reviewer **全 PASS**(0 个 RISK/FAIL)+ 综合 critic **GREEN**;无任何 gating 红灯 | ✅ PASS(GREEN) |
| CDX | Codex 独立复核 | 无红色阻断 | gpt-5.5 xhigh:**APPROVE-WITH-NITS**,RED LINE=No,attribution/scope 正确,无阻断 | ✅ PASS(no red) |
| EVAL | 旁路评估 | openai_gateway/responses 是否缺采集 | 见 §EVAL:两者均缺;responses 近似对称但 tee 未接;openai_gateway 非对称(结果类型不同) | ✅ DONE |

---

## 改动证据

`git diff backend/internal/handler/gateway_handler.go`(全量,唯一改动):
```diff
@@ -931,6 +931,18 @@ func (h *GatewayHandler) Messages(c *gin.Context) {
 						zap.Int64("account_id", account.ID),
 					).Error("gateway.record_usage_failed", zap.Error(err))
 				}
+				// M1 采集口：与 RecordUsage 并列、与计费隔离的内容采集（fail-open，默认关闭）。
+				h.gatewayService.CollectGenerationContent(ctx, service.GenerationContentCaptureArgs{
+					RequestID:          result.RequestID,
+					UserID:             subject.UserID,
+					APIKeyID:           currentAPIKey.ID,
+					GroupID:            currentAPIKey.GroupID,
+					AccountID:          account.ID,
+					Model:              reqModel,
+					RequestPayloadHash: requestPayloadHash,
+					PromptBody:         parsedReq.Body,
+					Result:             result,
+				})
 			})
 			return
 		}
```
- **diff 行数**:`1 file changed, 12 insertions(+)`。
- **改动面声明**:535(gemini)/282(CC)/main 分支/任何写客户端语句(`c.Writer`/flush/SSE/Content-Length)= **零改动**;无新增/修改测试、schema、wire。
- **与标准答案(535)的唯一差异**:`APIKeyID/GroupID` 用 `currentAPIKey`(非 `apiKey`),以与**同闭包** `RecordUsage` 的归因一致
  (主路径有 fallback-group 重试,会 `currentAPIKey = fallbackAPIKey`;gemini 闭包无 fallback 故用 `apiKey`)。所有变量已核在作用域内。

---

## 真跑证据(原文)

- **BUILD**:`go build ./... → BUILD_EXIT=0`;`go vet ./... → VET_EXIT=0`(见 evidence/,过程原文)。
- **UT**:`evidence/go_test_all.txt`(默认 tag,`GO_TEST_EXIT=0`,39 包 ok,0 FAIL,`internal/handler` ok 25.2s);
  `evidence/go_test_unit.txt`(`-tags unit`,`UNIT_TEST_EXIT=0`,service 89.7s + handler 25s ok,0 FAIL)。
- **全栈真跑(DB/BC/BC-OFF)**:**未能执行**。本机 Docker Desktop 进程在跑但 Linux 引擎未起:
  `com.docker.service`=Stopped 且 `Start-Service` 被拒(需提权,无人值守无法 UAC);WSL `docker-desktop` 发行版 Stopped;
  `docker version` 持续挂起。无 Docker → 无 throwaway Postgres/Redis → 无法复现 smoke 的 §3 全栈。**已做的尝试**:启动
  Docker Desktop、尝试起服务(被拒)、轮询引擎 pipe 240s、复探所有 docker pipe、最终 `docker version` 仍挂 = 确认不可用。
- **smoke 此前真跑(同一变更,作为佐证)**:`_review/M1B_smoke_test_20260619/SUMMARY.md` §3——真 curl `POST /v1/messages`
  → 主路径转发 → mock@9099 → 客户端拿到 mock 原样(Content-Length 294,透明性成立)→ 补上该行后 1s 内落库
  id=2(`request_id=mock-req-1`,全归因,prompt/response 双脱敏,`response_bytes=294 == 客户端 Content-Length`)。

---

## 字节一致性证据(BC / BC-OFF)

全栈 golden/after diff 因 Docker 不可用而 **BLOCKED**(已备 `harness/server_baseline.exe` + `server_patched.exe` +
`harness/config.empirical.yaml` + `harness/EMPIRICAL_RERUN_RECIPE.md`,Docker 起来后 ~10min 可一键复跑)。在无 Docker 条件下,
BC 由以下四重证据支撑"客户端字节不变":
1. **结构论证**:补丁仅在 usage task 闭包内、响应写完后,增加一处 flag-gated 调用;生产经 worker 池异步执行(`wire_gen.go:248`);
   不引用任何写客户端的对象/语句;只读 `result.ResponseSample`(独立副本)。**唯一影响字节的机件 = B.2 `capturingResponseWriter` tee,
   本补丁未改。**
2. **既有进程内 tee 透明性单测(全绿)**:`gateway_response_capture_test.go`(`_Transparent`/`_CapsSinkNotClient`/
   `HandleStreamingResponse_TeeTransparent`——同一 SSE 跑裸 writer/包装 writer 两遍,逐字节相等)。
3. **Codex 独立复核**:RED LINE = No(理由:不接触 writer/header/SSE,异步在响应后执行)。
4. **smoke 真跑**:同一变更下客户端 Content-Length 294 未变。
- **BC-OFF**:补丁唯一新增行在 flag 关时 `CollectGenerationContent` 立即返回(`contentCaptureEnabled()` 守卫),为纯 no-op
  → 关闭时行为与补线前**逐字节等同**;`generation_content_collector_test.go` 的 disabled→0 行 gating 测试实证 no-op。

---

## 双复核

### Codex 独立复核(修者≠审者)— evidence/codex_review_output.txt 原文
模型 gpt-5.5(xhigh,read-only sandbox,纯 packet 推理)。**VERDICT: APPROVE-WITH-NITS**。
- RED LINE:**No**——不接触 `c.Writer`/header/flush/Content-Length/SSE;读 `ForwardResult.ResponseSample` 独立 capped 副本;
  生产经 worker 池在响应写完后执行 → 不改变/重排/阻塞客户端字节。
- Fail-open:flag 关 no-op;collector 内 recover、吞 repo 错;在 `RecordUsage` 之后、与计费隔离 → 不伤请求/计费。
- Attribution:`currentAPIKey` 为正确选择(与同闭包 RecordUsage 一致;fallback 场景归因正确);字段无误。
- Scope:最小变更,符合"补主路径缺口"。
- **唯一 nit**:在**无 worker pool 的同步 fallback**下,采集开启 + repo 慢可能拉长 handler 后置收尾时间(不影响客户端字节、不影响计费)。
  → **本 nit 对生产无效**:生产已注入 worker 池(`wire_gen.go:248/251`),采集任务跑在池上、不在请求 goroutine;同步 fallback 仅测试/池为 nil 时触发,且该模式下 `RecordUsage` 本就同样后置(非本补丁引入的新行为)。

### 对抗 workflow(5 路并行对抗 + 综合 critic)— REV
官方 Dynamic Workflow(run `wf_d6ef5d15-060`,6 agents,~9.5min,只读)。**综合判定 = GREEN,无任何 gating 风险。**
5 路对抗 reviewer **全部 PASS**(findings 全为 info 级,零 RISK/零 FAIL):

| 维度 | 判定 | 对抗结论(已试图 break,未果) |
|---|---|---|
| byte-transparency | PASS | 采集调用在 `submitUsageRecordTask` 闭包内、`Forward`(:769)返回**之后**执行(Forward 的 `defer restoreWriter()`/`fillResponseSample` 已在返回前把 `c.Writer` 还原、抽样定型);闭包只捕获快照值、跑在 `context.Background()` 派生 ctx 上、从不引用 `c`/`c.Writer`;redaction 入新缓冲、ResponseSample 是 cappedSink 自有副本;tee 文件未改。无任何路径可改/延/重排/阻塞客户端字节。 |
| panic / fail-open | PASS | 三层 recover(collector:79 / worker-pool:321 / 同步 fallback:1868);调用 void 不可经 error 破坏请求;nil 全守卫;flag 关 :142 短路真 no-op;DB 受 5s ctx 超时上界。 |
| scope / diff | PASS | `git diff` = 1 file,+12/-0,仅 909 闭包;`gofmt -l` 干净;535/CC/写客户端语句/测试/schema/wire 全未动;`_review/` 为既有 untracked。 |
| attribution | PASS | `currentAPIKey` **比** gemini 的 `apiKey` **更正确**:主路径 prompt-too-long fallback 在 :827 把 `currentAPIKey=fallbackAPIKey`(`cloneAPIKeyWithGroup` 仅换 GroupID/Group、留 ID/User),同闭包 RecordUsage 也用 currentAPIKey → 采集归因与计费归因一致;用 apiKey 反而会在 fallback 时错配 group。所有字段在作用域、类型匹配。 |
| eval-verify | PASS | 独立复核 responses/openai_gateway 缺口(见 §EVAL),与主 agent 结论一致。 |

**综合 critic 独立复核后判 GREEN**,BC 红线结论:*"Client bytes are UNCHANGED … cannot alter, delay, reorder, or block any byte already written to the client."*
- **唯一非 gating 提示**:本机无 gcc/cgo,`-race` 未能跑;但补丁未引入新并发(只在既有闭包内加一处读快照值的调用,采集全程在响应之后),红线不依赖 race 结果。建议(可选)学者在 cgo 机器上跑 `go test -race ./internal/handler/...` 作最终确认。
- 完整对抗记录:workflow run `wf_d6ef5d15-060`(transcript 在 session subagents 目录)。

---

## 旁路评估(EVAL,只读,未改任何代码)

全仓 `CollectGenerationContent` 调用点(补丁后):`gateway_handler.go:535`(gemini)、`gateway_handler_chat_completions.go:282`
(forward-as-CC)、`gateway_handler.go:909`(主 Anthropic,**本补丁**)。仍缺采集的 `RecordUsage` 路径:

- **responses 路径**(`gateway_handler_responses.go:268`,`RecordUsageInput`):**缺采集**。结构**近似对称**(同 task/同 input 族),
  但作用域用 `apiKey`/`body`(无 `currentAPIKey`/无 `parsedReq`)。**关键阻塞**:B.2 的响应 tee(`beginResponseCapture`)**未接** responses
  输出口(已核:tee 仅在 `gateway_service.go` Forward + `gateway_forward_as_chat_completions.go`,**不在** `gateway_forward_as_responses.go`)
  → 即便照搬采集,`ResponseSample` 仍为空(只采到 prompt)。→ **B.3:须先把 B.2 tee 延伸到 responses,再补采集。**
- **openai_gateway 路径**(`openai_gateway_handler.go:426/800/1363`,`OpenAIRecordUsageInput`):**缺采集,且非对称**。其 `result`
  为 OpenAI 专用结果类型,而 `GenerationContentCaptureArgs.Result` 要求 `*service.ForwardResult` → **不能简单镜像**;且 tee 同样未接。
  → **B.3+:需独立设计(OpenAI 结果→采集料 的适配 + tee 覆盖),非一处镜像可解。**
- 另注(本轮范围外):`gemini_v1beta_handler.go:527`(`RecordUsageWithLongContext`)、`openai_chat_completions.go:262`、
  `openai_images.go:293` 亦无采集,属 OpenAI/images/v1beta 族,后续里程碑再议。

---

## workflow 运行说明
- ultracode 开启;本任务用官方 Dynamic Workflow 跑了 1 个对抗复核 workflow(`m1b-mainpath-patch-adversarial-review`,
  run `wf_d6ef5d15-060`,5 并行对抗 reviewer + 1 综合 critic,只读)。
- 其余编排(补丁、build/vet/test、Codex 复核、EVAL)由主 agent 直接顺序执行(有状态、需真跑,不适合 fan-out)。
- 无中途续跑。

---

## 异常 / 停止登记
- **触发停止条件 #8**(BC/DB 无法建立确定性"全栈"可比)——根因 = **本机 Docker 引擎今夜无法无人值守启动**(需提权,UAC 无人应答)。
  这是**纯环境阻塞**,非代码缺陷;补丁本身 BUILD/UT/SCOPE/CDX/REV/EVAL 全过 + 此前 smoke 真跑实证。
- **未触发**其余任何停止条件(无真库/真付费/真 key/push 之需;无打到真 Anthropic 之虞——全栈根本未起;无需碰旧路径/worktree;
  908 与 535 完全对称,镜像未触碰任何写客户端语句)。
- **未做**(遵铁红线):未提交、未推送、未碰 main/535/282/旧路径/worktree、未补 responses/openai_gateway、未为凑绿改测试。

---

## 给学者的下一步建议
1. **二选一定夺补丁去留**(补丁现留在工作树,未提交):
   - (A) **直接提交**:鉴于结构论证 + 全门绿 + 双复核无红 + 此前 smoke 对同一变更的真跑实证,证据已足;或
   - (B) **先补全栈真跑再提交**:Docker 起来后,按 `harness/EMPIRICAL_RERUN_RECIPE.md`(已备双二进制 + config)~10min 复跑
     golden/after 字节 diff + `SELECT` 落行,补齐 DB/BC/BC-OFF 的"全栈"实证,再提交。**推荐 (B)**(切合任务书"只信真跑"的铁律,代价仅 10min)。
2. 提交信息建议:`feat(collector): M1-B 主路径(Anthropic /v1/messages)补 CollectGenerationContent(镜像 535,flag 默认关)`。
3. **B.3**:把 B.2 响应 tee 延伸到 responses 出口 → 再补 responses 采集;openai_gateway 需独立适配设计(结果类型不同)。
4. 提交后仍勿 push(origin=开源上游)。
