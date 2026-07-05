# Codex 任务书 · Sub2API S1 收尾（仅此仓库）

> **仓库**：`D:\sub2api-trunk`（Sub2API / 无界 AI 管理中台）  
> **性质**：S1 工程收口 — **不是** QCanvas 任务，**不要**打开 QCanvas 仓改代码。  
> **前置**：[CODEX_TASK_S1_R2BC_MERGE.md](./CODEX_TASK_S1_R2BC_MERGE.md) 已执行；[2026-07-05-S1-R2BC-review.md](./deliverables/2026-07-05-S1-R2BC-review.md) 已存在。  
> **规划真相源**：`D:\Codex创业任务\QCanvas（无界版）\北极星\北极星V5.0_...20260705.html` `#roadmap` S1  
> **交付**：`deliverables/YYYY-MM-DD-S1-CLOSEOUT-review.md`

---

## 给 Codex 的开场白（复制即用）

```text
你在 D:\sub2api-trunk 执行 Sub2API S1 收尾。只读本仓 docs/superpowers/codex-handoff/CODEX_TASK_S1_CLOSEOUT.md。
禁止改 QCanvas 仓、禁止改 frontend/src/views/admin/console/、禁止读 .env/Key/token、禁止 push/merge/deploy（除非老板明确要求）。
从 S1-C0 开始，完成后出 deliverables/YYYY-MM-DD-S1-CLOSEOUT-review.md。
```

---

## 本仓当前状态（2026-07-05 晚）

| 项 | 状态 |
|----|------|
| R2-A 正式 Seedance | **done**（task #4） |
| R2-B NB2 图片冒烟 | **blocked**（缺授权） |
| R2-C 3+3 扩样 | **partial**（仅 task #4） |
| S1-0 测试修复 | **done**（1 行 `api_key_video_gateway_test.go`） |
| PR #3726 | Open；**CLA failed** |
| 未 track 文件 | `CODEX_TASK_S1_R2BC_MERGE.md`、`2026-07-05-S1-R2BC-review.md` |

**老板判定**：Sub2API **可以内部测试**；R2-B/C 为加强项；PR 合并前须 CLA + CI。

---

## 执行顺序（单会话 · 仅 Sub2API）

```text
S1-C0  仓库卫生 + 范围锁定（只 stage 本任务文件）
S1-C1  全量验证 + secret-scan
S1-C2  审查包与文档终稿
S1-C3  R2-B/C 条件分支（有授权才跑，无则终态 blocked）
S1-C4  PR #3726 合并门禁终检（不自行 merge）
```

---

## S1-C0 — 仓库卫生

### 只允许改动 / stage 的文件

| 文件 | 说明 |
|------|------|
| `backend/internal/server/routes/api_key_video_gateway_test.go` | S1-0 测试对齐 |
| `00_START_HERE.md` | 入口同步；**修掉 line 7 等尾随空格**（若 `git diff --check` 报错） |
| `docs/superpowers/codex-handoff/CODEX_START_HERE.md` | 指向 CLOSEOUT |
| `docs/superpowers/codex-handoff/CODEX_TASK_S1_R2BC_MERGE.md` | 任务书入库 |
| `docs/superpowers/codex-handoff/CODEX_TASK_S1_CLOSEOUT.md` | 本文件 |
| `docs/superpowers/codex-handoff/deliverables/2026-07-05-R2-closeout-summary.md` | R2 收口 |
| `docs/superpowers/codex-handoff/deliverables/2026-07-05-S1-R2BC-review.md` | S1 审查包 |
| `docs/superpowers/codex-handoff/deliverables/YYYY-MM-DD-S1-CLOSEOUT-review.md` | 本轮新增 |

### 禁止

- 不要 stage/commit QCanvas 路径下的任何文件
- 不要 stage 无关 tools/、`.vscode/` 等（除非老板明确要求）
- **不要** `git add -A` 全仓

### 验收

```powershell
cd D:\sub2api-trunk
git diff --check          # exit 0
git status --short        # 仅上述文件在 staged 或 intentional modified
```

---

## S1-C1 — 全量验证 + secret-scan

### 必跑命令

```powershell
cd D:\sub2api-trunk\backend
go test ./...
golangci-lint run ./...

cd D:\sub2api-trunk
make secret-scan
# 或: python tools/secret_scan.py --include-untracked
```

### 验收

| 项 | 期望 |
|----|------|
| `go test ./...` | exit 0 |
| `golangci-lint run ./...` | 0 issues |
| secret-scan | 无 high-confidence 命中；若有，审查包列明且不得 commit 密钥 |

---

## S1-C2 — 审查包与文档终稿

### 新建 `deliverables/YYYY-MM-DD-S1-CLOSEOUT-review.md`

必须包含：

1. **S1 终态表**：R2-A done / R2-B blocked|done / R2-C partial|done
2. **验证命令输出摘要**（S1-C1）
3. **secret-scan 结果**
4. **Git 就绪清单**：哪些文件应被老板 commit；**Codex 不自行 commit**（除非老板在本轮明确说「请 commit」）
5. **PR #3726 合并门禁表**（见 S1-C4）
6. **给老板的 3 条待办**：签 CLA、废弃 R2-A 临时 Key、是否授权 R2-B/C

### 更新 `2026-07-05-R2-closeout-summary.md`

- 补充 S1-CLOSEOUT 指针
- 若 secret-scan pass，在商用签字条件行注明

### 更新 `CODEX_START_HERE.md`

- 当前执行 → 指向本 CLOSEOUT 任务书
- S1-R2BC 标为「已完成，见 S1-R2BC-review + S1-CLOSEOUT」

---

## S1-C3 — R2-B/C 条件分支

> **默认路径（无新授权）**：保持 blocked/partial，在 CLOSEOUT 审查包写终态，**不触发真实付费调用**。

### 若老板在本轮消息中明确授权（须同时满足）

- 书面授权：NB2 / Seedance 扩样、预算上限、停止条件
- 确认 R2-A 临时 Key 已废弃或轮换
- 提供**无需 Codex 读取 Key 明文**的调用方式（如 dev 已录入 provider account）

则执行：

| 任务 | 动作 |
|------|------|
| R2-B | 1–3 条 NB2 作图；记录 usage_log id、cost、余额 Δ |
| R2-C | 补 2 条 Seedance 视频 + 纳入 R2-B 图片；更新对账表 |
| 交付 | 追加到 CLOSEOUT 审查包 §扩样，或新建 `R2-B-review` / 更新 `R2-C` |

### 若无授权

审查包写：

```text
R2-B: blocked（终态，不挡内部测试）
R2-C: partial（task #4 仍为唯一真实视频样本）
```

---

## S1-C4 — PR #3726 合并门禁终检

### Codex 做

| 项 | 动作 |
|----|------|
| 本地门禁 | S1-C1 已通过 |
| 分支 | 确认 `wujie/video-capture-moat-20260702` |
| 文件卫生 | S1-C0 清单内文件 ready to commit |
| GitHub | 打开 PR checks 页或 `gh pr view 3726`（若可用） |

### Codex 不做

- 不签 CLA（需 PR 提交者 / 老板 GitHub 账号）
- 不 `git push`、不 merge、不 deploy

### 合并门禁表（审查包必填）

| 门禁 | pass/fail/待老板 |
|------|----------------|
| 本地 go test | |
| 本地 golangci-lint | |
| secret-scan | |
| CLA | 预期 fail 直至老板签署 |
| CI 其他项 | |
| conflict with main | |
| deliverables 已 track | 待 commit |

---

## 明确禁止（Sub2API 专用）

- ❌ 不要打开 `D:\Codex创业任务\QCanvas（无界版）\QCanvas` 改代码
- ❌ 不要改 `frontend/src/views/admin/console/`
- ❌ 不要提交 `.env`、Key、token、cookie
- ❌ 不要 force-push main
- ❌ 不要声称「QCanvas 已验收」或「3+3 对账已完成」（除非 S1-C3 真跑过）

---

## 完成后汇报

> 「Sub2API S1 收尾完成，审查包：`docs/superpowers/codex-handoff/deliverables/YYYY-MM-DD-S1-CLOSEOUT-review.md`。PR #3726：CLA 待签；R2-B blocked / R2-C partial（或已扩样）。请老板 commit 清单内文件。」

**现在从 S1-C0 开始。**
