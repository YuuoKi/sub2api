# 审查包：DBug-6 — MLA-P2-007 KeysView 错误信息

> 执行者：Codex
> 完成时间：2026-07-08 02:04 +08:00
> 关联规划：[CODEX_TASK_MLA_DBUG.md](../CODEX_TASK_MLA_DBUG.md)
> 状态：`done`

---

## 1. 本任务做了什么（给 Claude / 老板看）

- `KeysView.handleSubmit` 的 catch 块改为使用 `extractApiErrorMessage`，与项目 axios 拦截器 reject 的 plain object 形状对齐。
- 创建 API Key 失败时，toast 展示后端 `message`（如配额/限制类错误），不再误落默认 `keys.failedToSave`。

---

## 2. 改了哪些文件

| 文件 | 变更摘要 |
|------|----------|
| `frontend/src/views/user/KeysView.vue` | import `extractApiErrorMessage`；submit catch 改用统一提取 |
| `frontend/src/views/user/__tests__/KeysView.spec.ts` | 新增 create 失败展示 message 的测试 |

---

## 3. 验收结果（必须可核对）

| 验收项 | 结果 | 证据 |
|--------|------|------|
| create 失败展示后端 message | pass | `KeysView.spec.ts` |
| vitest | pass | 1 test |
| eslint + vue-tsc | pass | exit 0 |

---

## 4. 验证命令与结果

```text
cd D:\sub2api-trunk\frontend
npx.cmd vitest run src/views/user/__tests__/KeysView.spec.ts --reporter=basic
# Tests  1 passed (1)
npx.cmd eslint src/views/user/KeysView.vue src/views/user/__tests__/KeysView.spec.ts --ext .ts,.vue --max-warnings=0
npx.cmd vue-tsc --noEmit
```

---

## 5. 给 Claude 的前端接口说明（如有）

无 API 变更。错误对象仍为拦截器形态 `{ status, code, message, error }`；UI 通过 `extractApiErrorMessage` 读取 `message`。

---

## 6. 风险与遗留

- `handleDelete` 仍使用 `error?.message`（与 delete 路径一致，本次未改）。
- 建议下一任务：**DBug-7** — MLA-REV-SUP1 `VideoTasksView` 列表轮询。

---

## 7. 阻塞项（若 status=blocked）

无。
