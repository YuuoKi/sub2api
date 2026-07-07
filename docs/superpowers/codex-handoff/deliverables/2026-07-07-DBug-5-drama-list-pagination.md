# 审查包：DBug-5 — MLA-P2-001 Drama 列表分页 total

> 执行者：Codex
> 完成时间：2026-07-08 02:00 +08:00
> 关联规划：[CODEX_TASK_MLA_DBUG.md](../CODEX_TASK_MLA_DBUG.md)
> 状态：`done`

---

## 1. 本任务做了什么（给 Claude / 老板看）

- `ListDramaTasks` 不再在 service 层对单页结果做内存过滤后返回 `total=len(out)`。
- 新增 `VideoGatewayRepository.ListDramaTasks`：在 repo 层按 `drama_context` 事件 JSON 与任务列下推过滤，DB 级 `COUNT(*)` + `LIMIT/OFFSET` 分页。
- 返回的 `total` 为全量匹配数，分页每页条数与 `page_size` 一致（末页除外）。

---

## 2. 改了哪些文件

| 文件 | 变更摘要 |
|------|----------|
| `backend/internal/service/drama_gateway_service.go` | `ListDramaTasks` 委托 repo 分页 |
| `backend/internal/service/drama_gateway_service_test.go` | 新增 `TestListDramaTasks_FilteredPagination` |
| `backend/internal/service/video_gateway_types.go` | 接口新增 `ListDramaTasks` |
| `backend/internal/service/video_gateway_worker_test.go` | memory repo 实现过滤+分页 |
| `backend/internal/repository/video_gateway_repo.go` | PostgreSQL `LATERAL` join + JSON 过滤 |
| `backend/internal/server/routes/api_key_video_gateway_test.go` | memory stub 满足接口 |

---

## 3. 验收结果（必须可核对）

| 验收项 | 结果 | 证据 |
|--------|------|------|
| 过滤后 total=全量匹配数 | pass | `TestListDramaTasks_FilteredPagination` |
| 分页 page1/2/3 条数正确 | pass | 同上 |
| `go test ./internal/service/...` | pass | exit 0 |

---

## 4. 验证命令与结果

```text
cd D:\sub2api-trunk\backend
$env:GOCACHE='D:\sub2api-trunk\.cache\go-build'
go test ./internal/service -run TestListDramaTasks_FilteredPagination -count=1 -v
# --- PASS: TestListDramaTasks_FilteredPagination (0.00s)
go test ./internal/service/... -count=1
# ok  github.com/Wei-Shaw/sub2api/internal/service  52.954s
```

---

## 5. 给 Claude 的前端接口说明（如有）

无 API 形状变更。`GET /user/drama/tasks`（或等价 drama list）的 `total` 字段现为过滤后全量计数，分页 `page`/`page_size` 行为与常规 list 一致。

---

## 6. 风险与遗留

- SQL 过滤与内存 `dramaTaskMatchesFilters` 语义对齐（`EqualFold`、engine family、mode 回退 task_type）。
- 仅返回含 `drama_context` 事件的任务（比旧逻辑更严格，排除无 drama 上下文的普通 video task）。
- 建议下一任务：**DBug-6** — MLA-P2-007 KeysView 错误信息展示。

---

## 7. 阻塞项（若 status=blocked）

无。
