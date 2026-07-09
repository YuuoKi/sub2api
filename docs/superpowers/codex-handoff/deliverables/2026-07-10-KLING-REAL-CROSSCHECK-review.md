# KLING-REAL Crosscheck Review - 2026-07-10

## 1. 总评

**结论：`pass_with_findings`。**

Sub2API 侧 KX-A～E 与 KX-G 关键实现/契约核查基本属实，硬禁用串已消失，Kling AK/SK、JWT、adapter gate、API-key production fail-closed、计费与 Drama 分叉均有代码和测试证据。KX-A 有一个 Important 级别补强点：`metadata_json` 仍允许管理员误填凭证并原样存储/响应。真实付费烟测按任务边界标记为 **blocked: awaiting AK/SK**。

QCanvas 侧 Kling 静态路径（KX-F）成立，但用户指定的 `@tapcanvas/web` 测试命令未全绿：`studioV2ShellMockRealWiring.test.ts` 中一个 `image_to_video` first-frame 用例失败。该失败不是 Kling 专属断言，但发生在同一 Studio V2 real-chain/ARM 测试文件内，因此 KX-F 不能判定为全绿。

## 2. 工作区与边界

| 项 | 结果 |
|---|---|
| Sub2API root | `D:/sub2api-trunk/.worktrees/kling-real` |
| Sub2API branch | `feature/kling-real-integration` |
| Sub2API status | 未提交 worktree diff 为本次审查对象；未 commit / 未 push |
| QCanvas root | `D:/Codex创业任务/QCanvas（无界版）/QCanvas` |
| QCanvas branch | `work/night-hardening-20260702` |
| QCanvas scope | 只审 Kling 相关文件；无关 asset/auth/chat/review backlog WIP 未纳入结论 |

未读取 `.env`、真实 AK/SK、JWT、密码、cookie；未运行付费真实烟测；未 commit / push。

## 3. KX-A～G 逐项表

| KX | 状态 | 证据与结论 |
|---|---|---|
| KX-A 凭证与安全 | pass_with_finding | `applyKlingCredentials` 将 AK/SK pack 到加密 blob，并只设置 `EncryptedAPIKey` 与 masked AK：`backend/internal/service/video_gateway_service.go:1544`、`:1547`、`:1583`。响应/列表清空 transient 明文：`:285`～`:288`、`:1601`～`:1603`。blob 使用 `v=1/auth=kling_aksk`：`video_gateway_kling_cred.go:15`、`:40`、`:48`、`:58`、`:67`。JWT 只在进程内缓存到 `exp-60s`：`:78`～`:110`。redaction 对 PlainAPIKey/AK/SK/JWT 做 account-aware strip：`video_gateway_redact.go:125`～`:150`。adapter Create/Poll 用 `videoProviderHasPlainCredentials`，不是 PlainAPIKey：`video_gateway_kling_adapter.go:49`、`:156`；测试覆盖 `TestKlingAdapterGatesOnAKSKNotPlainAPIKey`。发现：admin `metadata_json` 任意透传，若误填 `access_key` / `secret_key` 等字段，会绕开专用凭证列的保护并进入存储/响应：`backend/internal/handler/admin/video_handler.go:21`、`:31`、`:45`、`:159`。 |
| KX-B 真实 adapter 契约 | pass | 硬禁用串命令 `rg "KLING_REAL_CALL_DISABLED|KLING_REAL_POLL_DISABLED|kling is disabled in API-key"` 结果 `NO_MATCH`。路径覆盖 t2v/i2v/multi/omni/extend/avatar：`video_gateway_kling_adapter.go:348`～`:436`，extend/avatar 使用 model alias 或 `PricingSource`，不扩 DB `task_type`。model map fail-closed：`:348`～`:367`，`kling-3.0 -> kling-v2-6`、omni/o1 走 omni。duration 只允许 5/10，非 production smoke 只能 5：`:732`～`:771`。UpstreamTaskID 字符集校验 + PathEscape：`:39`～`:42`、`:172`、`:193`、`:440`～`:449`。SSRF/allowlist 对外部媒体 URL fail-closed：`video_gateway_ssrf.go:18`～`:47` 与 adapter 调用 `:519`、`:538`、`:606`。Cancel local-only：`video_gateway_kling_adapter.go:277`～`:286`。 |
| KX-C API-key 产品路径 | pass | List providers 不再写死 disabled，Kling 暴露 tiny_real metadata：`video_gateway_service.go:366`～`:396`。handler 有 Kling tiny_real 与 production 分支：`backend/internal/handler/video_handler.go:384`、`:403`。API-key Kling production 要求 `RouteAvailable && klingProductionAuthorized`，否则 `VIDEO_PRODUCTION_NOT_AUTHORIZED`：`video_gateway_service.go:709`～`:786`。未授权测试：`TestAPIKeyVideoGatewayBlocksKlingWithoutProductionAuth`；放行测试：`TestAPIKeyVideoGatewayAllowsKlingTinyTrial`。 |
| KX-D 计费 | pass | Kling 估算/结算走 catalog CNY/sec，并写 `PricingVersion=kling-video-2026-07`：`video_gateway_pricing.go:11`～`:14`、`:80`～`:87`，`video_gateway_billing.go:131`～`:150`。测试覆盖 `TestKlingBudgetEstimateUsesPerSecondCatalogWhenConfigRateUnset`、`TestKlingSucceededTaskSettlesPerSecondCostAndCarriesPricingVersion`、`TestKlingCatalogRateCNYPerSecondSelectsStdAndPro`。费率已明确 `PLACEHOLDER/provisional`，不可作生产对账真理：`video_gateway_pricing.go:12`～`:13`、`:140`。Seedance billing 测试仍在同一必跑 service gate 中通过。 |
| KX-E Drama | pass | 授权账号让 `kling_real_*` profile 标为 `RealProviderVerified`：`drama_gateway_service.go:274`～`:278`。未授权仍 `kling_safe_demo`：`:826`～`:832`。t2v 不静默改写成 safe-demo i2v，测试覆盖 `kling_real_text_to_video` 与 `kling_real_image_to_video`：`drama_gateway_service_test.go:132`～`:154`、`:262`～`:275`。Drama provider selection 仍要求 `APIKeyConfigured + RouteAvailable + production_authorized`：`drama_gateway_service.go:908`、`:948`、`:966`。 |
| KX-F QCanvas 产品面 | fail | 静态 Kling 链路通过：`assertProviderAllowed` 放行 `kling + allowRealCalls + tiny_real`：`apps/web/src/ui/studio-v2/studioV2RealTaskAdapter.ts:248`～`:252`；ARM catalog -> provider=kling：`:259`～`:275` 与 `studioV2ShellStore.ts:1183`～`:1281`；Kling catalog `preflightOnly=false`：`studioV2ModelCapabilities.ts:231`～`:273`；Hono map 与 Sub2API catalog/upstream 分层一致：`sub2api.video-mock-gateway.service.ts:32`～`:54`；`rg 127.0.0.1:7781 apps/web/src/ui/studio-v2` 为 `NO_MATCH`；Yunwu diff 无输出。但 §4 指定 web 测试命令失败：`studioV2ShellMockRealWiring.test.ts` 1 个 `image_to_video` first-frame 用例报 `Cannot read properties of undefined (reading '0')`。 |
| KX-G 文档诚实度 | pass | `docs/api/video-gateway-contract.md` §4.2 写明 Kling 不再 disabled/skeleton、AK/SK -> JWT、duration 5/10、production/tiny_real gates、真实烟测 `blocked: awaiting AK/SK`：`:122`～`:167`。`docs/api/qcanvas-integration-guide.md` 写明 dual-key 配置、model map、Studio ARM 与 smoke blocked：`:70`～`:98`。实现方自审包对 QCanvas 测试写为 agent-reported pass；本复查实跑发现 web gate 有红点，见差异节。 |
| 真付费烟测 | blocked | 官方 AK/SK 尚未到位；按任务书不运行真实付费 smoke。当前只证明代码/fixture/test gate，不证明上游 paid create/poll 闭环。 |

## 4. 必跑命令输出摘要

| 命令 | 结果 |
|---|---|
| `go test -tags=unit ./internal/service/ -run "Kling|kling" -count=1` | pass：`ok github.com/Wei-Shaw/sub2api/internal/service 4.693s` |
| `go test -tags=unit ./internal/server/routes/ -run "Kling|VideoGateway" -count=1` | pass：`ok github.com/Wei-Shaw/sub2api/internal/server/routes 4.118s` |
| `rg -n "KLING_REAL_CALL_DISABLED|KLING_REAL_POLL_DISABLED|kling is disabled in API-key" --glob "*.go"` | pass：`NO_MATCH` |
| `pnpm --filter @tapcanvas/web test -- _test/unit/studioV2RealTaskAdapter.test.ts _test/unit/studioV2ShellMockRealWiring.test.ts` | fail：93 files run, 92 passed, 1 failed. Failed test: `studioV2ShellMockRealWiring.test.ts > image_to_video 字段缝补全...` at line `432`, `adapterMocks.create.mock.calls[0][0]` undefined. |
| `pnpm --filter @tapcanvas/api test -- src/modules/sub2api/sub2api.video-mock-gateway.service.test.ts src/modules/sub2api/sub2api.routes.test.ts` | pass：2 files, 66 tests passed. stderr contains expected negative-path logs from tests, not failure. |
| `rg -n "127.0.0.1:7781" apps/web/src/ui/studio-v2` | pass：`NO_MATCH` |

可选 `make secret-scan` 未运行：本轮必跑门已覆盖任务书 §4；且未读取 secrets/环境文件。

## 5. 发现的问题

### Critical

无。

### Important

1. **QCanvas 指定 web gate 未全绿。**  
   `studioV2RealTaskAdapter.test.ts` 本身通过，但同命令中的 `studioV2ShellMockRealWiring.test.ts` 失败 1 个 adjacent real-chain 用例：`image_to_video` first-frame 内容没有触发预期 adapter create mock。虽然不是 Kling 专属断言，但它位于 Studio V2 real-chain/ARM 关键测试文件内；在修复或解释前，不能把 QCanvas KX-F 判为全绿。

2. **Sub2API admin `metadata_json` 缺少凭证字段 denylist/scrub。**  
   正常 top-level `access_key` / `secret_key` 会进入加密 `encrypted_api_key`，不会写入 metadata；但 admin handler 仍允许调用方把任意 `metadata_json` 透传到服务层与响应。如果前端或人工调用误把 `access_key`、`secret_key`、`jwt`、`api_key` 等字段塞进 metadata，会被保留并可能在 admin response 中回显。建议补服务端 metadata scrub/denylist 与测试。

### Minor

1. **QCanvas surface 文档 env gate 命名偏 Seedance。**  
   `docs/integrations/QCANVAS_SUB2API_OFFICIAL_SURFACE_V1.md:76` 仍把 `SUB2API_VIDEO_REAL_SMOKE_ENABLED` / `SUB2API_REAL_HUMAN_AUTHORIZED` / `SUB2API_REAL_ENABLED` 描述为 `Seedance tiny_real gates`，但当前代码已覆盖 Seedance/Kling。建议后续改成 `Video tiny_real gates (Seedance/Kling)`。

2. **Sub2API admin `CreateTask` 参数命名仍是 Seedance-only。**  
   `VideoTaskCreateParams` 只有 `RequireSeedanceProductionAuthorization`，admin handler 也只设置该字段：`video_gateway_types.go:282`、`video_handler.go:246`。实际 Kling 调用仍会在 API-key production 和 adapter `klingSmokeGateBlockedReasons` fail-closed，但命名/对称性容易让后续维护者误读。

3. **真实上游仍缺 paid smoke 证据。**  
   这是任务书预期 blocked，不是实现缺陷；但所有“已真通”表述都应继续限定为代码路径就绪、等待 AK/SK。

## 6. 与实现方自审包的差异

| 实现方自述 | 交叉复查结论 |
|---|---|
| WS-A AK+SK blob/JWT/redact pass | 基本属实，但有 Important 补强点：top-level AK/SK 处理安全，`metadata_json` 仍可被误填凭证并透传，建议补 scrub/denylist。 |
| WS-B adapter pass | 基本属实。路径、model map、duration、ID 校验、SSRF、audit、local cancel 均有证据。 |
| WS-C API-key tiny_real + production pass | 属实。Kling production 未授权 fail-closed。 |
| WS-D billing pass | 属实，但必须保留 `PLACEHOLDER/provisional` 限定，不可作生产对账。 |
| WS-E Drama pass | 属实。授权走 `kling_real_*`，未授权保留 `kling_safe_demo`。 |
| WS-F QCanvas vitest pass（agent-reported） | 被本复查部分打脸：静态 Kling 路径成立，`@tapcanvas/api` tests 绿，但用户指定 `@tapcanvas/web` 命令红 1 个 Studio V2 real-chain 用例。 |
| WS-G docs pass | 基本属实。另有一处 QCanvas surface env gate 文案偏 Seedance。 |
| 真实付费 smoke blocked | 属实，继续 blocked: awaiting AK/SK。 |

## 7. 给老板的下一步建议

1. **Sub2API 可进入合并审前补一个 metadata scrub 小补丁。**  
   建议在 admin provider create/update 的 `metadata_json` 入口增加凭证字段 denylist/scrub，覆盖 `access_key`、`secret_key`、`api_key`、`jwt`、`token` 等明显敏感键，并补回归测试。

2. **不要声称 paid Kling 已闭环。**  
   当前证据支持“代码路径与 fail-closed gate 就绪”，不支持“真实上游已成功 create/poll”。

3. **QCanvas 需先让 Cursor/实现方修复或解释 web gate 红点。**  
   最小位置：`apps/web/_test/unit/studioV2ShellMockRealWiring.test.ts:406`～`:432` 对应的 image_to_video first-frame real-chain 断言，及其实现侧 `apps/web/src/ui/studio-v2/shell/studioV2ShellStore.ts` 相关 `runNodeRehearsal` 分支。

4. **AK/SK 到位后再做授权 smoke。**  
   必须配置 dual-key、`single_smoke_authorized` 或 `production_authorized`、redacted event log、media URL allowlist，并验证 create/poll/result_url/audit redaction 全链路。

5. **上线前替换 Kling 占位价表。**  
   `kling-video-2026-07` 当前是 provisional rate，不可直接用于真实成本对账。

## 8. Fix follow-up - 2026-07-10

**Updated verdict:** `pass_with_findings`.

This section supersedes the two Important findings above for the current working tree. The paid Kling smoke remains **blocked: awaiting AK/SK** and was not run.

### Fixed / no longer reproduces

| Item | Current status | Evidence |
|---|---|---|
| Sub2API admin `metadata_json` credential leakage | fixed | `backend/internal/service/video_gateway_service.go` now scrubs credential-like metadata keys before provider create/update persistence and defensively before provider responses. Regression coverage added in `backend/internal/service/video_gateway_single_smoke_authorized_test.go`. Focused red test failed before the fix with `access_key`, `secret_key`, `api_key`, `jwt`, `authorization`, `token`, `credential` retained; it passes after the fix. |
| QCanvas required web gate | no longer reproduces in current QCanvas worktree | `pnpm --filter @tapcanvas/web test -- _test/unit/studioV2RealTaskAdapter.test.ts _test/unit/studioV2ShellMockRealWiring.test.ts` now passes: 93 files, 538 tests. No QCanvas source edit was needed in this fix pass. |

### Commands rerun after fix

| Command | Result |
|---|---|
| `go test -tags=unit ./internal/service/ -run "TestVideoProviderMetadataScrubsCredentialLikeKeys" -count=1` | pass: `ok github.com/Wei-Shaw/sub2api/internal/service 5.569s` |
| `go test -tags=unit ./internal/service/ -run "Metadata\|Kling\|kling\|SingleSmoke\|ProviderKey" -count=1` | pass: `ok github.com/Wei-Shaw/sub2api/internal/service 5.252s` |
| `go test -tags=unit ./internal/server/routes/ -run "Kling\|VideoGateway" -count=1` | pass: `ok github.com/Wei-Shaw/sub2api/internal/server/routes 4.902s` |
| `rg -n "KLING_REAL_CALL_DISABLED\|KLING_REAL_POLL_DISABLED\|kling is disabled in API-key" --glob "*.go"` | pass: no matches |
| `pnpm --filter @tapcanvas/web test -- _test/unit/studioV2RealTaskAdapter.test.ts _test/unit/studioV2ShellMockRealWiring.test.ts` | pass: 93 files, 538 tests |
| `pnpm --filter @tapcanvas/api test -- src/modules/sub2api/sub2api.video-mock-gateway.service.test.ts src/modules/sub2api/sub2api.routes.test.ts` | pass: 2 files, 66 tests; stderr contains expected negative-path logs from tests |
| `rg -n "127.0.0.1:7781" apps/web/src/ui/studio-v2` | pass: no matches |

### Files changed by this fix pass

- `backend/internal/service/video_gateway_service.go`
- `backend/internal/service/video_gateway_single_smoke_authorized_test.go`
- `docs/superpowers/codex-handoff/deliverables/2026-07-10-KLING-REAL-CROSSCHECK-review.md`

No commit and no push were performed.
