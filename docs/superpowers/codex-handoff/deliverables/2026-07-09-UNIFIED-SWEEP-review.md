# 审查包：UNIFIED-SWEEP — Sub2API 统合冲刺（P0–P2）

> 执行者：Grok  
> 完成时间：2026-07-09 Asia/Shanghai  
> 关联任务：[CODEX_TASK_UNIFIED_SWEEP_20260709.md](../CODEX_TASK_UNIFIED_SWEEP_20260709.md)  
> 基线 HEAD：`8e401f42`（Night Phase B）  
> 状态：`partial`（G0–G5 done；G6 9:16 Form A **done**；R2-B NB2 产品链 **done**；v2v skip）

---

## 1. 本任务做了什么（给 Claude / 老板看）

- **G1**：总览成员排行 / 模型分布下钻到 AI 调用记录（`?user_id=` / `?model=`）；提示词 tab 接入 `ContentWall` 采纳 + 周报摘要。
- **G2**：公司月度总预算 `company_monthly_budget_cny`；dashboard stats 追加花费进度；总览可设置/更新预算。
- **G3**：成功视频任务异步归档到 `DATA_DIR/assets/video/{id}/`；DB 记 `local_asset_path`；`GET /api/v1/video/tasks/:id/local-asset` 受控下载。
- **G4**：任务列表/详情视频预览 + 本地归档入口；总览备份超期黄条（>7 天或无成功备份）。
- **G5**：全量 go test / golangci / eslint / typecheck / consoleUtils vitest 绿。
- **G6**：老板授权临时 Ark Key + ¥50 硬顶后，跑 **Form A 全链路** Seedance **9:16 / 5s / 720p** 真实付费冒烟 → `succeeded`（详见 §9）。

---

## 2. 改了哪些文件

| Phase | 关键变更 |
|-------|----------|
| G0 | 任务书 + 审查包 + `CODEX_START_HERE` 指针 |
| G1 | `BossOverviewView` / `AiRecordsView` / `consoleUtils` + vitest |
| G2 | `company_monthly_budget*.go`、dashboard handler/routes、总览预算 UI |
| G3 | migration `155_*`、`video_asset_archive*.go`、repo/handler/routes |
| G4 | `VideoTasksView` / `VideoTaskDetailView` 预览；总览备份黄条 |
| G5 | 本审查包、`00_START_HERE` |

---

## 3. 验收结果

| 验收项 | 结果 | 证据 |
|--------|------|------|
| G0 基线 | pass | HEAD 自 `8e401f42` 起 |
| G1 下钻 + 采纳 | pass | `parseAiRecordsQuery` 5 tests；ContentWall 复用 |
| G2 月度预算 | pass | `TestMonthlyBudgetUsagePercent` + dashboard stats 测试 |
| G3 成品归档 | pass | `TestArchiveSucceededVideoResult*`；失败不阻断 |
| G4 预览 + 备份告警 | pass | typecheck 绿；UI 已接 |
| G5 全量门禁 | pass | 见 §4 |
| G6 真实付费 9:16 | pass | Form A `succeeded`；`ratio=9:16`；tokens=108900；upstream `cgt-20260709141516-ptcrw` |
| G6 R2-B NB2 | pass | 产品链 `/v1/messages`：usage_log#1 `media_type=image` cost=$0.045；余额 18.6085→18.5635 — [R2-B](./2026-07-09-R2-B-image-smoke-review.md) |
| G6 v2v | skip | 竖屏样本已覆盖优先缺口；剩余预算保留，未再开火 |

---

## 4. 验证命令与结果

```text
cd backend
go test ./... -count=1
=> GO_TEST_EXIT:0

golangci-lint run ./...
=> 0 issues. LINT_EXIT:0

cd frontend
npx eslint . --ext .ts,.vue --max-warnings=0
=> ESLINT_EXIT:0

npx vue-tsc --noEmit
=> EXIT:0

npx vitest run src/views/admin/console/__tests__/consoleUtils.spec.ts
=> 5 tests pass

secret-scan
=> partial：本机 WindowsApps Python 为 stub；Docker 经 WSL 挂载 D: 失败（`/work/tools/secret_scan.py` 不可见）。
   未在源码中引入密钥文件；建议日间用可用 Python 重跑 `make secret-scan` / `python tools/secret_scan.py --include-untracked`
```

---

## 5. 给 Claude 的前端接口说明

### 月度预算
- `GET /api/v1/admin/dashboard/stats` 追加：`monthly_budget_cny`、`monthly_spend_cny`、`monthly_budget_usage_percent`
- `PUT /api/v1/admin/dashboard/monthly-budget` body：`{"monthly_budget_cny":1000}`（0=清除）

### 本地归档
- 任务响应追加：`local_asset_path`、`local_asset_saved_at`、`local_asset_available`
- `GET /api/v1/video/tasks/:id/local-asset`（admin 或任务归属者；文件流）

### AI 记录下钻
- `/admin/console/ai-records?user_id=N&model=xxx&tab=prompts`

---

## 6. 风险与遗留

- secret-scan：本机 WindowsApps Python 为 stub；Docker/WSL 挂载失败 → **partial**，日间补跑
- R2-B 产品链已补：usage_log#1 / 余额 Δ$0.045；请废弃 Gemini Key 并清理临时账号/API Key
- v2v 仍 skip
- Form A 走内存 harness，**不写** dev DB `video_tasks` / 用户余额；上游火山账单仍真实扣费
- 归档依赖 `result_url` 可下载 + media allowlist；失败仅打日志
- 备份黄条依赖 data-management API；无权限时静默
- 本机 Docker 网络曾坏（`deploy_sub2api-network` TCP 超时），已切到 `deploy_sub2api-network_fix`

---

## 7. Commits

1. `991d3117` — docs: start unified sweep g0 baseline  
2. `e3b59346` — feat(console): overview drill-down and prompt adoption  
3. `75094c48` — feat(billing): company monthly budget cny progress  
4. `69077a5d` — feat(video): archive succeeded results to local assets  
5. `01ae4c3f` — feat(console): video preview and backup stale alert  
6. `3a31cd1a` — docs: close out unified sweep gates  
7. （待提交）Form A `SUB2API_SEEDANCE_SMOKE_ASPECT` + 本审查包 G6 收口

---

## 9. G6 真实付费（2026-07-09 授权续跑）

**状态：`partial`（9:16 done；NB2 产品链 done；v2v skip）**

### 授权与停止条件
| 项 | 值 |
|----|-----|
| 预算硬顶 | ¥50 CNY |
| Key | 临时 Ark + 临时 Gemini（仅 env/运行时；**未**写入审查包） |
| 开火 | ① Form A Seedance 5s 9:16 ② NB2 上游 ③ NB2 产品链 `/v1/messages` 各 1 次 |
| 停止 | 各单次成功后停；不跑 v2v / 更高档位 |

### 验收（脱敏）— Seedance 9:16
| 项 | 结果 |
|----|------|
| 路径 | Form A：`CreateTask` → budget gate → worker submit/poll → terminal |
| 模型 | `doubao-seedance-2-0-260128` |
| 规格 | 5s / 720p / **`ratio=9:16`** |
| upstream_task_id | `cgt-20260709141516-ptcrw` |
| status | `succeeded`（≈286s） |
| tokens / 估费 | 108900 → **≈¥5.01** |

### 验收（脱敏）— NB2 产品链
| 项 | 结果 |
|----|------|
| 路径 | `POST /v1/messages` → gemini account#1 / group#2 |
| 模型 | `gemini-3.1-flash-image-preview`（512） |
| msg id | `msg_d018298a80eaa88b30bf4c15` |
| usage_log | id=1；`media_type=image`；**$0.045** |
| 余额 | 18.6085 → **18.5635**（user_id=3） |
| 详情 | [2026-07-09-R2-B-image-smoke-review.md](./2026-07-09-R2-B-image-smoke-review.md) |

### 预算
累计估 ≈¥5.65；余量 ≈¥44.3。

### 未做
- v2v；NB2 1K/2K/4K 扩样

---

## 8. Verdict

**partial / 内部可用** — 运营闭环已落地；Seedance 9:16 + NB2 产品链付费签字 **done**；v2v 仍 skip。
