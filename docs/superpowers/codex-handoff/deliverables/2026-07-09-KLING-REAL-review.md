# 审查包：KLING-REAL — 可灵真实接入（WS-A..G）

> 执行者：Cursor Agent  
> 完成时间：2026-07-10  
> 关联规划：Kling real integration（worktree `feature/kling-real-integration`）  
> 状态：`done`（真实付费冒烟 `blocked: awaiting AK/SK`）

---

## 1. 本任务做了什么（给 Claude / 老板看）

- **WS-A**：Kling Access Key + Secret Key 加密 blob（`auth_mode=kling_aksk`）、JWT 签发/缓存、脱敏与 admin 双密钥 UI。
- **WS-B**：真实 `klingVideoAdapter`（替换 skeleton）：JWT Bearer、t2v/i2v/multi/omni/extend/avatar、模型 allowlist、smoke gate、SSRF、审计脱敏。
- **WS-C**：API-key 产品路径解锁 — `tiny_real` 试跑 + `production_authorized` 正式，与 Seedance 对称。
- **WS-D**：Kling 计价目录 + estimate/settle 计费路径（占位 CNY 费率，`kling-video-2026-07`）。
- **WS-E**：Drama 网关在 `production_authorized` + healthy 时走真实 Kling；未授权仍 `kling_safe_demo`。
- **WS-F**（QCanvas）：`assertProviderAllowed` 放行 Kling tiny_real；Studio ARM 按模型选 provider；Hono model map 对齐；`preflightOnly` 清除。
- **WS-G**（本包）：契约 / 集成说明 / admin 文案更新 + 本审查交付物。

**真实付费冒烟：** `blocked: awaiting AK/SK` — 官方密钥尚未到位；单元/夹具绿不代表上游付费闭环。

---

## 2. 改了哪些文件（按工作流摘要）

### Sub2API — WS-A..E（实现）

| 区域 | 代表文件 |
|------|----------|
| 凭证 / JWT | `video_gateway_kling_cred.go`、`video_gateway_redact.go`、admin `video_handler.go` |
| 真实适配器 | `video_gateway_kling_adapter.go`（+ tests） |
| API-key 路径 | `video_gateway_service.go`、`video_handler.go`、`api_key_video_gateway_test.go` |
| 计费 | `video_gateway_pricing.go`、`video_gateway_billing.go`（+ tests） |
| Drama | `drama_gateway_service.go`（+ tests） |
| Admin UI（双密钥） | `VideoProvidersView.vue`、`KeyVaultView.vue`、`frontend/src/api/admin/video.ts` |

### QCanvas — WS-F

| 文件 | 变更摘要 |
|------|----------|
| `studioV2RealTaskAdapter.ts` | Kling `assertProviderAllowed` + real-trial 标记 |
| `studioV2ShellStore.ts` / model capabilities | provider-aware ARM；Kling `preflightOnly: false` |
| `sub2api.video-mock-gateway.service.ts` | `KLING_FORWARD_MODEL_IDS` 对齐 Sub2API allowlist |

### Sub2API — WS-G（文档 + 文案）

| 文件 | 变更摘要 |
|------|----------|
| `docs/api/video-gateway-contract.md` | 删除 Kling disabled/skeleton；新增 §4.2（JWT、模式、duration、gate、model map） |
| `docs/api/qcanvas-integration-guide.md` | Kling 可调用；凭证步骤；model map；smoke blocked |
| `docs/api/image-gateway-contract.md` | 去掉「本轮 disabled」；指向 video 契约 |
| `frontend/.../videoUtils.ts` | 去掉「预留」口吻 → 已接入但 gated |
| `frontend/.../VideoProvidersView.vue` | 能力矩阵 / 说明文案 |
| `frontend/.../VideoDashboardView.vue` | 总览 / 前置条件文案 |
| 本文件 | 审查交付物 |

---

## 3. 验收结果（必须可核对）

| 验收项 | 结果 | 证据 |
|--------|------|------|
| WS-A AK+SK blob + JWT + redact | pass | `TestPackUnpackKling*` / `TestKlingMintJWT*` / redact tests |
| WS-B 真实 adapter（夹具 HTTP） | pass | `TestKlingCreate*` / poll / gate / duration |
| WS-C API-key tiny_real + production | pass | routes + service Kling API-key tests |
| WS-D Kling pricing/billing | pass | estimate/settle + PricingVersion tests |
| WS-E Drama real vs safe_demo | pass | drama_gateway_service_test（authorized/unauthorized） |
| WS-F QCanvas ARM + model map | pass | QCanvas vitest（agent-reported WS-F） |
| WS-G 契约去掉 disabled | pass | video/qcanvas/image docs + admin copy |
| 真实付费冒烟 | **blocked** | `awaiting AK/SK` |

---

## 4. 验证命令与结果

```text
# Sub2API — Kling focused (WS-G session re-run)
cd backend
go test -tags=unit ./internal/service/ -run "TestKling|TestPackUnpackKling|TestDecryptProviderKeyKling|TestApplyKling|TestUpdateKling|TestRedactVideoUpstreamSecretsStripsKling|TestVideoProviderUpstreamEchoedCredential|TestVideoProviderAccountStringAndLogValueRedactKling" -count=1
# ok  github.com/Wei-Shaw/sub2api/internal/service

# Prior WS sessions (agent-reported, not re-run here)
go test -tags=unit ./internal/server/routes/ -run "Kling|VideoGateway" -count=1   # WS-C ok
go test -tags=unit ./internal/service/ -run "Kling|kling|APIKey|Drama" -count=1 # WS-C/E ok

# QCanvas WS-F (agent-reported)
pnpm --filter @tapcanvas/web test -- studioV2RealTaskAdapter / shell wiring
pnpm --filter @tapcanvas/api test -- sub2api.video-mock-gateway
```

---

## 5. 给 Claude 的前端 / 接入说明

- **可调用**：`provider=kling` + `trial_mode=tiny_real`（试跑）或 production（需 `production_authorized`）。
- **凭证**：Admin 填 Access Key + Secret Key；勿再当「永久预留」。
- **时长**：仅 `5` / `10`；试跑无 production 时仅 `5`。
- **模型**：见 `video-gateway-contract.md` §4.2 model map；未知模型 fail-closed。
- **Studio**：选 Kling catalog + ARM → `provider=kling` + `tiny_real`。
- **冒烟**：密钥到位前不要宣称上游付费闭环。

---

## 6. 风险与遗留

- **阻塞**：官方 AK/SK 未到 → 无法做真实付费冒烟。
- Kling 价表为**占位费率**，官方价确认后需改 `video_gateway_pricing.go`。
- Studio 全能参考 video/audio UI 仍不完整（与 Seedance 同源遗留）。
- `video-gateway-contract.md` 历史段落仍有部分编码噪声；§4.1/§4.2 与本次新增内容为可读 UTF-8。

---

## 7. 密钥到位后如何启用

1. Admin → 视频通道 / Key Vault → `provider=kling` → 填入 **Access Key** + **Secret Key** → 保存。  
2. 账号 metadata：试跑设 `single_smoke_authorized` 或 `real_smoke_authorized`；正式设 `production_authorized=true`。  
3. Env：`SUB2API_VIDEO_REAL_SMOKE_ENABLED=1`、脱敏事件日志路径、媒体 URL allowlist。  
4. 调 `POST /v1/video/tasks`：`provider=kling`、`trial_mode=tiny_real`、`model` 在 allowlist、`duration=5`。  
5. 确认脱敏审计有 create/poll 事件、无明文 AK/SK/JWT；任务终态与 `result_url` 正常。  
6. 正式路径去掉 `trial_mode`，确认 `production_authorized` 与 duration `5|10`。

---

## 8. 阻塞项

| 项 | 状态 |
|----|------|
| 官方 Kling Access Key / Secret Key | **blocked: awaiting AK/SK** |
| 真实付费 tiny_real 冒烟 | 依赖上项 |
| 代码 / 契约 / QCanvas 放行 | 已完成（WS-A..G） |
