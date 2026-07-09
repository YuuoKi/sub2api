# 审查包：DUAL-SURFACE-ALIGN — 双产品面官方对齐

> 执行者：Cursor Agent  
> 完成时间：2026-07-09  
> 关联规划：双产品面官方对齐 Implementation Plan  
> 状态：`done`

---

## 1. 本任务做了什么（给 Claude / 老板看）

- Seedance 2.0：`duration=-1` 现在会**显式**写入 Ark create payload（不再省略）。
- QCanvas Studio/Hono：首尾帧与全能参考 **互斥**；仅全局参考图时派生 `reference_to_video`；混发被 Zod 400。
- 作图：修复 `/v1beta` 路径 `"512"` → `0.5K` 计费归一；打包价表补上 `gpt-image-2`；重写 image 契约文档。
- 新增现行调用说明 `docs/api/qcanvas-integration-guide.md`（两条 API + 可灵下轮必接）。
- 可灵本轮保持 disabled/skeleton；即梦/LLM 明确不做。

---

## 2. 改了哪些文件

### Sub2API

| 文件 | 变更摘要 |
|------|----------|
| `backend/internal/service/video_gateway_adapter.go` | `duration != 0` 时写入 payload（含 `-1`） |
| `backend/internal/service/video_gateway_content_contract_test.go` | `TestSeedanceCreateSendsExplicitAutoDurationMinusOne` |
| `docs/api/video-gateway-contract.md` | `-1` 显式下发 + 可灵 disabled 下轮必接 |
| `backend/internal/service/antigravity_gateway_service.go` | `/v1beta` extractImageSize 对齐 `"512"`→`0.5K` |
| `backend/internal/service/antigravity_image_test.go` | 覆盖 512 归一 |
| `backend/internal/pkg/antigravity/gemini_types.go` | 注释 |
| `backend/resources/model-pricing/model_prices_and_context_window.json` | `gpt-image-2` 价表项 |
| `backend/internal/service/pricing_service_test.go` | gpt-image-2 回退链测试 |
| `docs/api/image-gateway-contract.md` | 官方对齐重写 |
| `docs/api/qcanvas-integration-guide.md` | **新增**现行调用说明 |

### QCanvas（跨仓）

| 文件 | 变更摘要 |
|------|----------|
| `apps/web/src/ui/studio-v2/studioV2RealTaskAdapter.ts` | content 互斥 + `deriveStudioV2VideoTaskKind` |
| `apps/web/src/ui/studio-v2/shell/studioV2ShellStore.ts` | taskKind 按模式派生 |
| `apps/hono-api/src/modules/sub2api/sub2api.schemas.ts` | Zod 互斥校验 |
| 相关 vitest | 修正混发错误期望 |

---

## 3. 验收结果（必须可核对）

| 验收项 | 结果 | 证据 |
|--------|------|------|
| Seedance `duration=-1` 显式下发 | pass | `TestSeedanceCreateSendsExplicitAutoDurationMinusOne` |
| 全能参考 content[] role 映射仍绿 | pass | `TestSeedanceCreateMapsContentArrayWithRoles` |
| Studio 首帧+全局参考不再混发 | pass | QCanvas vitest 42/42 |
| Hono Zod 拒混发 | pass | hono-api vitest 16/16 |
| NB2 `/v1beta` 512→0.5K | pass | antigravity image unit tests |
| GPT Image 2 价表项存在 | pass | pricing_service_test + JSON key |
| 调用说明落盘 | pass | `docs/api/qcanvas-integration-guide.md` |
| 可灵真接 | skip（下轮必接） | 文档已写明 |
| 真实付费冒烟 | skip | 未授权 |

---

## 4. 验证命令与结果

```text
# Sub2API
cd backend
go test -tags=unit ./internal/service/ -run "TestSeedanceCreate(SendsExplicitAutoDurationMinusOne|MapsContentArrayWithRoles|PayloadSnapshotMatchesArkContract)" -count=1
# ok

# WS-C focused (agent-reported)
go test -tags=unit ./internal/service/ -run "TestExtractImageSize_|TestGetModelPricing_GptImage|..." -count=1
# ok

# QCanvas (agent-reported)
pnpm --filter @tapcanvas/web test -- _test/unit/studioV2RealTaskAdapter.test.ts _test/unit/studioV2ShellMockRealWiring.test.ts
# 42/42 passed
pnpm --filter @tapcanvas/api test -- src/modules/sub2api/sub2api.video-mock-gateway.service.test.ts
# 16/16 passed
```

---

## 5. 给 Claude 的前端接口说明（如有）

- **互斥**：首尾帧模式与全能参考不可同请求；Studio 有帧时会丢弃全局参考图 roles。
- **taskKind**：仅全局参考时应为 `reference_to_video`。
- **duration=-1**：正式路径可传，中台会原样发给 Ark。
- **可灵**：UI 若仍展示，须标不可用；勿接真链。
- **作图中台契约**：见 `docs/api/image-gateway-contract.md`；画布强制走 Sub2API 另开。

---

## 6. 风险与遗留

- 未解决问题：Studio 仍无参考视频/音频 UI（全能参考不完整）。
- 需要老板决策：即梦是否进 Sub2API；画布作图何时强制收口中台。
- 建议下一任务：**可灵真接** + 全能参考 video/audio UI。

---

## 7. 阻塞项（若 status=blocked）

无。
