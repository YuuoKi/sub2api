# Codex 任务书 — 请先完整阅读本文件再动手

> **给 Codex 的开场白**  
> 你是 Sub2API（无界 AI 生产控制台）仓库里的**后端执行代理**。前端控制台业务化重构（总览 / 密钥库 / 员工与开卡 / 任务记录）已由 Claude 完成并验证。  
> **你的任务**：只实现路线图里标注【Codex】的后端与配置工作；**不要改** `frontend/src/views/admin/console/` 下的新页面（除非 roadmap 明确写了后端配套且无可避免的小改动）。  
> **做完后**：在 `docs/superpowers/codex-handoff/deliverables/` 下新建一份审查包 Markdown（用 [DELIVERABLE_TEMPLATE.md](./DELIVERABLE_TEMPLATE.md)），然后告诉老板「已完成，审查包路径是 xxx」。

---

> **2026-07-05 R2：R2-A `done`** — 真实 Seedance task #4 已跑通（见 [R2-A 审查包](./deliverables/2026-07-05-R2-A-production-smoke-review.md)）。**请废弃临时 Key。**
>
> **2026-07-05 R1（已完成）**：见 [CODEX_TASK_R1.md](./CODEX_TASK_R1.md) 与 `deliverables/2026-07-05-R1-backend-review.md`。

## 第一步：读这些（顺序固定）

1. 本文件（你正在读）
2. **R1 复查任务书（当前执行）**：[CODEX_TASK_R1.md](./CODEX_TASK_R1.md)
3. 已有审查包：`deliverables/2026-07-05-*.md`（9 份，了解已交付范围）
4. 契约文档：`docs/api/video-gateway-contract.md`、`docs/api/image-gateway-contract.md`

---

## 你的执行范围（按优先级）

### 第一批（P0，必须先做）

| ID | 任务 | 说明 |
|----|------|------|
| **P0-2** | 视频计费从估算改为可对账 | `video_tasks.cost_estimate` → 按 Seedance 等规则写实际费用；`video_usage_logs` 可对账 |
| **P0-3 后端** | 外部工具归属 | 工具账号与员工区分（`user_attributes` 或 `notes` 前缀 `[工具]`）；**不要改前端导航** |
| **P0-1** | 真实通道冒烟 | 核对 `SUB2API_VIDEO_URL_ALLOWLIST`、`VIDEO_GATEWAY_PER_CALL_BUDGET`；在老板提供真实 Key 后跑最小真实任务；无 Key 则审查包标 `blocked` 并写清缺什么 |

### 第二批（P1，P0 完成后）

| ID | 任务 |
|----|------|
| **P1-1 后端** | 卡额度 80%/100% 告警事件（复用 `balance_notify_*` 体系） |
| **P1-3 后端** | 任务成功后归档到 `/app/data/assets/`，DB 记本地路径 + 清理策略 |

### 第三批（P2，按需）

P2-1 组长权限、P2-3 通用 AI 计费校准、P2-4 备份超期提示 — 见 roadmap，**不要抢在 P0 未完成前做**。

### 明确禁止

- 不要实现 P0-4、P1-2 前端、P1-4 前端（Claude 负责）
- 不要大改演示模式导航或 `BossOverviewView.vue` 等 console 页面
- 不要提交 `.env`、真实 API Key、员工密码

---

## 分层与质量（强制）

```
handler → service → repository
```

- 遵守 [backend/.golangci.yml](../../backend/.golangci.yml) depguard
- 每个任务结束前在 `backend/` 下执行：
  - `go test ./...`
  - `golangci-lint run ./...`
- 动到密钥/计费/权限：在审查包里写「已做安全自查」，列出攻击面

---

## 交付物：审查包放哪里

**目录（固定）：**

```text
docs/superpowers/codex-handoff/deliverables/
```

**命名（固定）：**

```text
YYYY-MM-DD-<任务ID>-review.md
```

示例：

- `2026-07-04-P0-2-review.md`
- `2026-07-04-P0-3-backend-review.md`

**每个任务一个文件**；若一次会话完成多个任务，可一个文件里分章节，但文件名用最高优先级任务 ID，并在文内列出子任务。

**必须用模板：** [DELIVERABLE_TEMPLATE.md](./DELIVERABLE_TEMPLATE.md)

审查包里必须包含：

1. 做了什么（老板能看懂）
2. 改了哪些文件
3. 验收表（pass/fail + 证据）
4. 验证命令输出摘要
5. **给 Claude 的前端接口说明**（新 API、字段、建议改哪些页面）— 有则必填
6. 阻塞项与需要老板做的事

---

## 完成后你怎么汇报

对老板说一句话即可，例如：

> 「P0-2 和 P0-3 后端已完成，审查包：`docs/superpowers/codex-handoff/deliverables/2026-07-04-P0-2-P0-3-review.md`，请转 Claude 继续做前端。」

**不要**假设 Claude 会自动扫 git；**必须**写好 deliverables 里的 Markdown。

---

## 建议执行顺序（Codex 会话）

```text
会话 A：P0-2 视频计费对账 → 出 review → go test/lint
会话 B：P0-3 后端工具账号 → 出 review → go test/lint
会话 C：P0-1（依赖老板 Key）→ 出 review 或 blocked 说明
会话 D：P1-1 / P1-3（可选）
```

---

## 当前仓库真值（2026-07-04）

- 演示模式前端：总览、密钥库、员工与开卡、任务记录（含 AI 调用记录）、系统折叠组
- 管理员开卡 API：`POST /api/v1/admin/users/:id/api-keys`（已实现）
- 本地 dev：`http://127.0.0.1:18081`（docker compose dev）
- 生成内容看板 API 已存在：`/api/v1/admin/generation-content/*`（前端未全暴露，**不是 Codex 本阶段重点**）

---

## 有疑问时

- 产品优先级以 [2026-07-04-console-v2-roadmap.md](../plans/2026-07-04-console-v2-roadmap.md) 为准
- 与前端界面相关的决策：**写入审查包「给 Claude 的建议」**，不要自己改 Vue
- 需要老板提供：Seedance Key、官方单价表、是否启用真实冒烟

**现在从 P0-2 或 P0-3 后端开始执行。**
