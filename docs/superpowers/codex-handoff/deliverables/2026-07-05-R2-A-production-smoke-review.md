# 审查包：R2-A — 正式 Seedance 端到端冒烟

> 执行者：Claude（老板授权临时 Key）  
> 完成时间：2026-07-05 16:28  
> 状态：**`done`**

---

## 1. 本任务做了什么

- 将老板提供的临时 Seedance Key 录入 dev 供应商账号 **id=2**（`production_authorized` + `single_smoke_authorized`）。
- `SUB2API_VIDEO_REAL_SMOKE_ENABLED=1` 已开启；`route_available=true`。
- 通过员工 API Key 调用 **正式路径**（`provider:seedance`，**无** `trial_mode`）创建 **text_to_video** 5s 任务。
- 轮询约 2.7 分钟至 **succeeded**；真实上游 `upstream_task_id=cgt-20260705162538-bd5cl`。
- 用户余额真实扣减；`video_usage_logs` 单行幂等写入。

---

## 2. 验收结果

| 验收项 | 结果 | 证据 |
|--------|------|------|
| 正式路径无 trial_mode | pass | `POST /v1/video/tasks` → HTTP 201 task **#4** |
| `result_url` 在 allowlist 内 | pass | `ark-acg-cn-beijing.tos-cn-beijing.volces.com/...` |
| `usage.total_tokens` | pass | **108900** |
| 余额扣减（USD） | pass | $20.00000000 → **$19.30425000**（Δ $0.69575） |
| CNY 费用 | pass | `cost_estimate` **5.009400 CNY**；`5.0094/7.2≈0.69575` USD |
| `video_usage_logs` 幂等 | pass | `video_task_id=4` 仅 **1** 行；`uq_video_usage_logs_video_task_id` |
| `balance_charged_at` | pass | `video_tasks.id=4` charged=true |
| `pricing_source` | pass | `provider_usage` / `ark-seedance-2026-03` |

---

## 3. 任务摘要（脱敏）

| 字段 | 值 |
|------|-----|
| task_id | 4 |
| model | doubao-seedance-2-0-260128 |
| task_type | text_to_video |
| duration | 5s / 720p / 16:9 |
| tokens | 108900 |
| cost | ¥5.0094 → $0.69575 |
| 发起人 | zoucha-test@wujie.local |

---

## 4. 安全提醒

**请老板立即在火山方舟控制台废弃本次使用的临时 Key**（已用于 dev 录入，审查包不含 Key 明文）。

---

## 5. 验证命令

```text
GET /v1/video/tasks/4  → status=succeeded, usage.total_tokens=108900
SELECT balance FROM users WHERE email='zoucha-test@wujie.local'  → 19.30425000
SELECT count(*) FROM video_usage_logs WHERE video_task_id=4  → 1
```

---

## 6. 遗留

- API 响应 `provider_boundary` 仍标 `api-key-video-seedance-tiny-trial`（展示字段，不影响正式路径路由）；可后续小修。
- R2-B 图片 NB2 真实冒烟仍可选。
