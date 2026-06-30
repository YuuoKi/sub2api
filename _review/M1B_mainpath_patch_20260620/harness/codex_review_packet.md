# Independent code review request — M1-B main-path collection patch

You are an INDEPENDENT reviewer (reviewer ≠ author). Review ONLY the material in this packet using pure
reasoning. Do NOT run shell commands or read files — everything you need is inline. Output a clear verdict.

## What changed
A 12-line patch (uncommitted) on branch `wujie/trunk`, baseline `b919650f`, in
`backend/internal/handler/gateway_handler.go`. It adds content collection to the **main Anthropic
`/v1/messages`** forward path (the `Messages` handler), which previously recorded usage but did NOT
collect prompt/response content. It mirrors an already-in-production call in the gemini closure of the
same file and in the chat-completions handler.

## THE RED LINE
The bytes the CLIENT receives in its HTTP response must be **byte-for-byte unchanged** by this patch.
A change that alters/delays/reorders/blocks client bytes is an automatic FAIL.

## The diff (git diff, the only change in the repo)
```diff
diff --git a/backend/internal/handler/gateway_handler.go b/backend/internal/handler/gateway_handler.go
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

## Surrounding context: the main closure (gateway_handler.go ~907-940, AFTER patch)
```go
				// 使用量记录通过有界 worker 池提交，避免请求热路径创建无界 goroutine。
				h.submitUsageRecordTask(func(ctx context.Context) {
					if err := h.gatewayService.RecordUsage(ctx, &service.RecordUsageInput{
						Result:             result,
						ParsedRequest:      parsedReq,
						APIKey:             currentAPIKey,
						User:               currentAPIKey.User,
						Account:            account,
						Subscription:       currentSubscription,
						// ... (other fields)
						ChannelUsageFields: channelMapping.ToUsageFields(reqModel, result.UpstreamModel),
					}); err != nil {
						logger.L().With( /* ... */ ).Error("gateway.record_usage_failed", zap.Error(err))
					}
					// >>> INSERTED 12 lines: CollectGenerationContent(...) <<<
				})
				return
```

## The mirror it copies — gemini closure (gateway_handler.go ~534-545, UNCHANGED, in production)
```go
					// M1 采集口：与 RecordUsage 并列、与计费隔离的内容采集（fail-open，默认关闭）。
					h.gatewayService.CollectGenerationContent(ctx, service.GenerationContentCaptureArgs{
						RequestID:          result.RequestID,
						UserID:             subject.UserID,
						APIKeyID:           apiKey.ID,        // <-- gemini uses apiKey (no fallback in that closure)
						GroupID:            apiKey.GroupID,
						AccountID:          account.ID,
						Model:              reqModel,
						RequestPayloadHash: requestPayloadHash,
						PromptBody:         parsedReq.Body,
						Result:             result,
					})
```
NOTE: the main-path patch uses `currentAPIKey.ID/.GroupID` instead of `apiKey.*`, to match the
**same closure's** RecordUsage attribution (which uses `currentAPIKey`/`currentAPIKey.User`/`currentSubscription`,
because the main path supports a fallback-group retry that reassigns `currentAPIKey = fallbackAPIKey`).

## How the task is dispatched — submitUsageRecordTask (gateway_handler.go ~1857)
```go
func (h *GatewayHandler) submitUsageRecordTask(task service.UsageRecordTask) {
	if task == nil { return }
	if h.usageRecordWorkerPool != nil {
		h.usageRecordWorkerPool.Submit(task)   // PROD: runs on a bounded worker pool, AFTER handler returns the response
		return
	}
	// fallback (no pool): run synchronously with panic recovery
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer func() { if r := recover(); r != nil { /* logged, recovered */ } }()
	task(ctx)
}
```

## Collector entry + body (service/generation_content.go)
```go
func (s *GatewayService) CollectGenerationContent(ctx context.Context, args GenerationContentCaptureArgs) {
	if s == nil || s.generationCollector == nil || !s.contentCaptureEnabled() { return } // flag off -> no-op
	s.generationCollector.Collect(ctx, args)
}
func (s *GatewayService) contentCaptureEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.Gateway.ContentCapture.Enabled // default false
}
func (c *GenerationContentCollector) Collect(ctx context.Context, args GenerationContentCaptureArgs) {
	if c == nil || c.repo == nil { return }
	defer func() { if r := recover(); r != nil { /* log, swallow */ } }() // fail-open
	// reads args.PromptBody (caps), args.Result.ResponseSample/Bytes/Truncated, redacts, then repo.Create (errors swallowed+logged)
}
type GenerationContentCaptureArgs struct {
	RequestID, RequestPayloadHash, Model string
	UserID, APIKeyID, AccountID int64
	GroupID *int64
	PromptBody []byte
	Result *ForwardResult   // ResponseSample populated by the B.2 response-capture tee inside Forward (UNCHANGED by this patch)
}
```
The collector NEVER references `c.Writer`, headers, flush, Content-Length, or the SSE stream. The response
sample it reads (`Result.ResponseSample`) is a separate capped copy filled by the B.2 tee during Forward.

## Your review questions (answer each)
1. RED LINE: Can this patch change the bytes the client receives? Consider timing (the call runs inside the
   usage-record task, which in prod is submitted to a worker pool that runs after the response is written; in the
   sync fallback it runs after Forward returns, i.e. after the response is fully written). Yes/No + why.
2. Fail-open: Can the inserted call panic, error, or block in a way that harms the request or billing path?
3. Attribution: Is `currentAPIKey` (not `apiKey`) the correct choice here, given the same closure's RecordUsage
   uses `currentAPIKey`? Any field wrong?
4. Scope: Is adding only this block (no change to the gemini mirror, the chat-completions handler, or any
   client-write statement) the right minimal change?
5. Overall verdict: APPROVE / APPROVE-WITH-NITS / REJECT — and the single most important risk, if any.
