# 审查包：UNIFIED-SWEEP — Sub2API 统合冲刺（P0–P2）

> 执行者：Grok  
> 完成时间：2026-07-09 Asia/Shanghai  
> 关联任务：[CODEX_TASK_UNIFIED_SWEEP_20260709.md](../CODEX_TASK_UNIFIED_SWEEP_20260709.md)  
> 基线 HEAD：`8e401f42`（Night Phase B）  
> 状态：`partial`（G0–G5 done；G6 真实付费 blocked 等授权）

---

## 1. 本任务做了什么（给 Claude / 老板看）

- **G1**：总览成员排行 / 模型分布下钻到 AI 调用记录（`?user_id=` / `?model=`）；提示词 tab 接入 `ContentWall` 采纳 + 周报摘要。
- **G2**：公司月度总预算 `company_monthly_budget_cny`；dashboard stats 追加花费进度；总览可设置/更新预算。
- **G3**：成功视频任务异步归档到 `DATA_DIR/assets/video/{id}/`；DB 记 `local_asset_path`；`GET /api/v1/video/tasks/:id/local-asset` 受控下载。
- **G4**：任务列表/详情视频预览 + 本地归档入口；总览备份超期黄条（>7 天或无成功备份）。
- **G5**：全量 go test / golangci / eslint / typecheck / consoleUtils vitest 绿。
- **G6**：无老板当面授权，真实付费冒烟记 `blocked`（不读 `.env`、不触发付费）。

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
| G6 真实付费 | blocked | 缺老板授权 / 预算 / 停止条件 |

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
- G6 R2-B/C / 9:16 / v2v 仍需老板授权后另开会话
- 归档依赖 `result_url` 可下载 + media allowlist；失败仅打日志
- 备份黄条依赖 data-management API；无权限时静默
- 已加 image realsmoke 脚手架测试（默认 skip，启用后仍拒绝无授权调用）

---

## 7. Commits

1. `991d3117` — docs: start unified sweep g0 baseline  
2. `e3b59346` — feat(console): overview drill-down and prompt adoption  
3. `75094c48` — feat(billing): company monthly budget cny progress  
4. `69077a5d` — feat(video): archive succeeded results to local assets  
5. `01ae4c3f` — feat(console): video preview and backup stale alert  
6. （本收口）`docs: close out unified sweep gates`（含 lint 修复 + image smoke scaffold + 文档）

---

## 9. G6 阻塞说明（真实付费）

**状态：`blocked`**

缺：
1. 老板当面授权本轮真实付费调用（R2-B NB2 1 次 + 可选 R2-C / 9:16 / v2v）
2. `production_authorized` 账号 + 已入库 Key（禁止读 `.env` 明文）
3. 单次预算上限与停止条件
4. 火山/Google 账单核对入口（人工）

授权后从 `CODEX_TASK_PRODUCTION_VERIFY.md` / 本审查包 G6 续跑；脚手架：`image_gateway_realsmoke_scaffold_test.go`（默认 skip）。

---

## 8. Verdict

**partial / 内部可用** — 零成本运营闭环与成品归档已落地；真实付费签字证据（G6）停等老板授权。
