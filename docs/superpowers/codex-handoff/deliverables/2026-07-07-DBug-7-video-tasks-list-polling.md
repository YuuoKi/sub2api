# 审查包：DBug-7 — MLA-REV-SUP1 视频任务列表轮询

> 执行者：Codex
> 完成时间：2026-07-08 02:08 +08:00
> 关联规划：[CODEX_TASK_MLA_DBUG.md](../CODEX_TASK_MLA_DBUG.md)
> 状态：`done`

---

## 1. 本任务做了什么（给 Claude / 老板看）

- `VideoTasksView` 在列表含非终态任务（`queued`/`submitted`/`running` 等，对齐 `isTerminalStatus`）时每 4s 自动 `loadTasks()`。
- `onUnmounted` 清理 interval，避免离开页面后继续请求。

---

## 2. 改了哪些文件

| 文件 | 变更摘要 |
|------|----------|
| `frontend/src/views/admin/video/VideoTasksView.vue` | 4s 轮询 + onUnmounted 清理 |
| `frontend/src/views/admin/video/__tests__/VideoTasksView.spec.ts` | running 任务时 list API 调用 >1 次 |

---

## 3. 验收结果（必须可核对）

| 验收项 | 结果 | 证据 |
|--------|------|------|
| 非终态任务存在时轮询 | pass | `VideoTasksView.spec.ts` |
| vitest | pass | 1 test |
| eslint + vue-tsc | pass | exit 0 |

---

## 4. 验证命令与结果

```text
cd D:\sub2api-trunk\frontend
npx.cmd vitest run src/views/admin/video/__tests__/VideoTasksView.spec.ts --reporter=basic
# Tests  1 passed (1)
npx.cmd eslint src/views/admin/video/VideoTasksView.vue src/views/admin/video/__tests__/VideoTasksView.spec.ts --ext .ts,.vue --max-warnings=0
npx.cmd vue-tsc --noEmit
```

---

## 5. 给 Claude 的前端接口说明（如有）

无 API 变更。列表页行为与详情页 2s 轮询一致，间隔为 4s。

---

## 6. 风险与遗留

- 轮询 tick 仍走完整 `loadTasks()`（含 `loading` 状态）；若需静默刷新可后续优化。
- 建议下一任务：**DBug-8** — MLA-P2-013 secret-scan hook fail-closed。

---

## 7. 阻塞项（若 status=blocked）

无。
