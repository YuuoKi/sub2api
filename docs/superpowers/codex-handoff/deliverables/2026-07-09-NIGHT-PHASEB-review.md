# 审查包：Night Phase B — adoption 401 + 卡额度告警 + 素材归档

> 执行者：Grok 4.5 / Composer
> 完成时间：2026-07-09 Asia/Shanghai
> 关联任务：[CODEX_TASK_NIGHT_PHASEB_20260709.md](../CODEX_TASK_NIGHT_PHASEB_20260709.md)
> 状态：`done`

---

## 1. 本任务做了什么（给 Claude / 老板看）

- **B1 (P0)**：新增 `POST /v1/generation-content/:task_id/adoption`（API Key 鉴权），仅允许员工 Key 对自己 `video_tasks.created_by` 任务写 adoption；他人任务 403。修复 QCanvas 走员工 Key 打 admin JWT 路由导致的 `401 INVALID_TOKEN`。
- **B2**：后端 `QuotaUsagePercent`（79/80/100 边界）+ API Key DTO 追加 `quota_usage_percent` / `quota_warning_level`；dashboard stats 追加 `quota_warnings`；总览告警条 + 成员开卡额度黄/红进度条。
- **B3**：解析 Ark/CDN 签名 URL 过期（`X-Amz-Date`+`X-Amz-Expires`），否则 `completed_at+24h` 估算；任务列表/详情展示过期提示与「复制链接」。**未做**对象存储转存。
- **B4**：全量门禁通过（见 §4）；无人值守边界遵守：无 push、无真实调用、未碰支付/webhook。

---

## 2. 改了哪些文件

| Phase | 关键文件 | 变更摘要 |
|-------|---------|----------|
| B1 | `generation_content_adoption_service.go`、`generation_content_handler.go`、`gateway.go`、`wire_gen.go` 等 | API Key adoption 路由 + 归属校验 |
| B2 | `api_key_quota_warning.go`、`dashboard_handler.go`、`StaffView.vue`、`BossOverviewView.vue` 等 | 80/100% 额度告警字段与 UI |
| B3 | `video_result_url_expiry.go`、`video_handler.go`、`VideoTasksView.vue`、`VideoTaskDetailView.vue` | 结果链接过期提示 + 复制 |
| B4 | 本审查包、`CODEX_START_HERE.md` | 收工与入口更新 |

---

## 3. 验收结果

| 验收项 | 结果 | 证据 |
|--------|------|------|
| B0 基线 HEAD | pass | 起始 `37d3126c`，分支 `wujie/video-capture-moat-20260702` |
| B1 员工 Key 自有任务 adoption | pass | routes/service 单测：200 saved / 403 foreign / 401 no key |
| B2 79/80/100 边界 | pass | `TestQuotaUsagePercentBoundaries` |
| B3 URL 过期解析 | pass | `TestParseResultURLExpiry*` |
| 禁止 push / 真实调用 / 支付代码 | pass | 本会话未 push；未读 `.env`；未改 payment/webhook |

---

## 4. 验证命令与结果（Final Gate）

| 门禁 | 结果 | 备注 |
|------|------|------|
| `go test ./... -count=1`（backend） | **pass** | `GO_TEST_EXIT=0` |
| `golangci-lint run ./...`（backend） | **pass** | `0 issues`（gofmt 已修） |
| frontend typecheck (`vue-tsc --noEmit`) | **pass** | `TYPECHECK_EXIT=0` |
| frontend eslint | **pass** | `eslint.cmd . --max-warnings 0` → 0 |
| critical vitest + consoleUtils | **pass** | 7 files / 83 tests（含 consoleUtils 3） |
| `make secret-scan` | **partial** | 本机 WindowsApps `python.exe` 为 stub（exit 9009）；Docker 拉 `python:3.12-slim` 超时未完成。**未在源码中引入密钥文件**；建议日间用可用 Python 重跑 `make secret-scan` |

---

## 5. 给 Claude / QCanvas 的接口说明

### B1 Adoption（员工 Key）

- **新接口**：`POST /v1/generation-content/:task_id/adoption`
- **鉴权**：与 `/v1/video/*` 相同，`Authorization: Bearer sk-...`
- **Body**：`{"adoption_status":"adopted"|"rejected"|"pending","quality_score":0~1?,"notes":"..."}`
- **成功**：`200` + `{enabled,saved,task_id,...}`
- **归属错误**：`403 VIDEO_TASK_FORBIDDEN`
- Admin 路径 `/api/v1/admin/generation-content/:task_id/adoption` **保持不变**

### B2 额度字段

- API Key 列表项追加：`quota_usage_percent`、`quota_warning_level`（`none|warn|critical`）
- `GET /api/v1/admin/dashboard/stats` 追加：`quota_warnings: {warn_count,critical_count,top_items[]}`

### B3 结果过期

- 任务详情/列表追加：`result_url_expires_at`、`result_url_expiry_source`（`url_query|estimated|unknown`）

---

## 6. 风险与遗留（Still Open）

- secret-scan 本机 Python 不可用；Docker 拉镜像超时 → 日间补跑
- B3 无签名参数时过期为 **估算**（完成 +24h），UI 已标明
- 视频采集行仍不写 `api_key_id`；归属按 `created_by == Key.user_id`（与视频 GetTask 一致）
- 对象存储转存仍属日间拍板大件

---

## 7. Commits

1. `3541f7b6` — `fix(generation-content): allow employee key adoption on own tasks`
2. `d44a7e07` — `feat(console): quota usage warnings at 80/100 percent`
3. `c2bcf727` — `feat(console): expose result download before expiry`
4. （本收工）`docs: close out night phase b loop`（含 gofmt 对齐）

---

## 8. Verdict

**done** — Phase B P0 adoption 401 已修；B2/B3 运营强化最小版已落地；全量 go test / golangci / typecheck / eslint / critical vitest 绿；secret-scan 因本机 Python stub 记为 partial，不阻断代码交付。
