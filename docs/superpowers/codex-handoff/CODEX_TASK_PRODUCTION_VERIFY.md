# Codex 任务书 — 生产验证与收尾（R2）

> **前置**：PR `wujie/video-capture-moat-20260702` 已含 V-1→R1 后端 + 控制台 v2 前端。Claude 浏览器走查通过（mock 链路）。  
> **你的目标**：把「能演示」推进到「真钱真任务可验收」，并处理剩余后端/配置项。  
> **不要改** `frontend/src/views/admin/console/`（P0-4 通道告警由 Claude 做）。

---

## 第一步：读这些

1. [CODEX_START_HERE.md](./CODEX_START_HERE.md)
2. [docs/api/video-gateway-contract.md](../../api/video-gateway-contract.md)
3. [deliverables/2026-07-05-R1-backend-review.md](./deliverables/2026-07-05-R1-backend-review.md)
4. 本任务书

---

## R2-A — 正式 Seedance 端到端（P0-1，老板已授权付费）

### 配置（dev：`http://127.0.0.1:18081`，env 文件 `C:\tmp\sub2api-b1-dev.env`）

1. 在密钥库 → 视频供应商，确认 Seedance 账号已录入且 `RouteAvailable=true`
2. 更新 provider metadata：`production_authorized: true`（保留既有 `single_smoke_authorized` 亦可）
3. Docker env（`deploy/docker-compose.dev.yml` 或 env-file）：
   - `SUB2API_VIDEO_REAL_SMOKE_ENABLED=1`
   - `SUB2API_MEDIA_URL_ALLOWLIST=<Ark CDN + QCanvas 参考图域名>`（未设则 fallback `SUB2API_VIDEO_URL_ALLOWLIST`）
4. `docker compose -p deploy -f deploy/docker-compose.dev.yml --env-file ... up -d --build --force-recreate sub2api`

### 验证（API-key 路径，QCanvas 契约）

**正式路径**（无需 `trial_mode`）：

```http
POST /v1/video/tasks
Authorization: Bearer <员工或工具 sk-...>
Content-Type: application/json

{
  "provider": "seedance",
  "task_type": "reference_to_video",
  "model": "doubao-seedance-2-0-260128",
  "prompt": "R2 production verify: 9:16 portrait v2v",
  "content": [
    {"type": "text", "text": "R2 production verify"},
    {"type": "image_url", "role": "reference_image", "url": "https://<allowlisted>/ref.png"},
    {"type": "video_url", "role": "reference_video", "url": "https://<allowlisted>/ref.mp4"}
  ],
  "aspect_ratio": "9:16",
  "duration": 10,
  "resolution": "720p",
  "generate_audio": false
}
```

轮询 `GET /v1/video/tasks/{id}` 直到 `succeeded`，确认：

- `result_url` / `last_frame_url` 可访问且在 allowlist 内
- `usage.total_tokens` 有值
- 用户余额实际扣减（USD，经 `usd_cny_rate`）
- `video_usage_logs` 仅一条（`UNIQUE(video_task_id)`）
- 控制台任务记录显示花费（¥ 或 $，视 `currency`）

### 交付

`deliverables/YYYY-MM-DD-R2-A-production-smoke-review.md`：配置 diff、任务 ID、扣费前后余额、供应商侧费用截图或日志摘要（脱敏）。

---

## R2-B — 图片网关生产验证（A-3 回归）

对 `gemini-3.1-flash-image-preview`（NB2 四档）跑一条 API-key 或 JWT 作图：

- `imageConfig` / `responseModalities` 透传
- `usage_logs.media_type=image`、多图 count、512 档计价

交付：`deliverables/YYYY-MM-DD-R2-B-image-smoke-review.md`

---

## R2-C — 计费对账抽检（P0-2 收尾）

抽 3 条已完成视频任务 + 3 条图片 usage_log：

- 控制台 `unified_total_actual_cost` 与明细之和一致
- Seedance CNY 与供应商账单 ±1%
- 文档化 `usd_cny_rate` 当前值与调整方式

交付：`deliverables/YYYY-MM-DD-R2-C-billing-reconciliation-review.md`

---

## R2-D — 运维小项（可选，P1）

| ID | 任务 |
|----|------|
| R2-D1 | 卡额度 80%/100% 告警（P1-1 后端，复用 `balance_notify_*`） |
| R2-D2 | 任务成功后资产归档到 `/app/data/assets/`（P1-3） |
| R2-D3 | 统一路由 meta title：`员工与开卡` → `成员与开卡`（若改，仅 `router/index.ts` 一行，或标给 Claude） |

---

## 明确禁止

- 不要大改 console 页面（BossOverviewView / StaffView 等）
- 不要提交 `.env`、真实 Key、密码
- 不要 force-push main

---

## 完成标准

- R2-A 必须 PASS 或明确 `blocked` + 缺什么
- R2-B、R2-C 至少各 1 条真实样本
- `go test ./...` + `golangci-lint run ./...` 全绿
