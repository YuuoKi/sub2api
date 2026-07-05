# 审查包：Sub2API S1 收尾

> 完成时间：2026-07-05 Asia/Shanghai  
> 仓库：`D:\sub2api-trunk`  
> 分支：`wujie/video-capture-moat-20260702`  
> PR：[#3726](https://github.com/Wei-Shaw/sub2api/pull/3726)  
> 当前状态：**内部可用 / 待复核**（本地 `go test` 与 secret-scan 通过；`golangci-lint` 工具上下文加载失败；CLA/CI 需老板在 GitHub 侧复核）

---

## 目标

按 `CODEX_TASK_S1_CLOSEOUT.md` 完成 S1 工程收口：锁定本仓文件范围，跑本地门禁，输出最终审查包，记录 R2-B/C 终态和 PR #3726 合并门禁。

---

## S1 终态表

| 项 | 终态 | 说明 |
|----|------|------|
| R2-A 正式 Seedance | done | 已有 task #4 真实样本，R2-A 审查包已存在；临时 Key 需老板废弃/轮换 |
| R2-B NB2 图片冒烟 | blocked | 本轮无 NB2 真实调用授权、预算上限、停止条件和无明文 Key 调用方式 |
| R2-C 3+3 扩样 | partial | task #4 仍为唯一真实视频样本；新增 2 视频 + 3 图片未授权，不触发真实付费调用 |
| S1-0 测试修复 | done | `api_key_video_gateway_test.go` 期望 reason 已对齐为 `VIDEO_PRODUCTION_NOT_AUTHORIZED` |
| S1 收尾文档 | done | 本审查包已新增；`CODEX_START_HERE.md` 与 R2 收口总览已补 CLOSEOUT 指针 |

---

## 执行目录

| 步骤 | 结果 |
|------|------|
| S1-C0 仓库卫生 + 范围锁定 | `git diff --check` 初次发现 `CODEX_START_HERE.md` 一处尾随空格；已仅在允许文件内修复，复跑通过 |
| S1-C1 全量验证 + secret-scan | `go test ./...` 通过；secret-scan 通过；`golangci-lint` 因工具上下文加载失败待复核 |
| S1-C2 审查包与文档终稿 | 新增本文件；更新 R2 closeout summary 与 handoff start 指针 |
| S1-C3 R2-B/C 条件分支 | 无授权，按任务书写终态 blocked/partial；未触发真实供应商调用 |
| S1-C4 PR #3726 门禁终检 | `gh` 本机不可用，GitHub 远端 CI/mergeable 状态待老板复核；CLA 按任务书记录为待签 |

---

## 变更清单

| 文件 | 状态 | 说明 |
|------|------|------|
| `backend/internal/server/routes/api_key_video_gateway_test.go` | modified | S1-0 测试对齐，reason 改为 `VIDEO_PRODUCTION_NOT_AUTHORIZED` |
| `00_START_HERE.md` | modified | 顶层入口已对齐北极星 V5.0 / R2 状态 |
| `docs/superpowers/codex-handoff/CODEX_START_HERE.md` | modified | 当前执行指向 S1 CLOSEOUT；补 S1-CLOSEOUT 审查包链接；修复尾随空格 |
| `docs/superpowers/codex-handoff/CODEX_TASK_S1_R2BC_MERGE.md` | untracked | S1-R2BC 任务书入库，需老板 commit |
| `docs/superpowers/codex-handoff/CODEX_TASK_S1_CLOSEOUT.md` | untracked | 本轮 S1 收尾任务书，需老板 commit |
| `docs/superpowers/codex-handoff/deliverables/2026-07-05-R2-closeout-summary.md` | modified | 补 S1-CLOSEOUT 指针和真实门禁状态 |
| `docs/superpowers/codex-handoff/deliverables/2026-07-05-S1-R2BC-review.md` | untracked | S1-R2BC 审查包，需老板 commit |
| `docs/superpowers/codex-handoff/deliverables/2026-07-05-S1-CLOSEOUT-review.md` | new | 本轮新增最终收尾审查包 |

---

## 验证命令和结果

| 命令 | 结果 | 摘要 |
|------|------|------|
| `git rev-parse --show-toplevel` | pass | `D:/sub2api-trunk` |
| `git branch --show-current` | pass | `wujie/video-capture-moat-20260702` |
| `git rev-parse --git-dir` / `git rev-parse --git-common-dir` | pass | 当前是 linked worktree；未创建新 worktree |
| `git diff --check` | pass | 修复允许文件内一处尾随空格后 exit 0；仅剩 CRLF 提示 |
| `git -c core.quotepath=false status --short` | pass | 仅任务书允许清单内文件 modified/untracked；有本机 git ignore 权限噪声 |
| `cd backend; go test ./...` | pass | 全量 Go 测试通过，exit 0 |
| `cd backend; golangci-lint run ./...` | blocked / 待复核 | exit 1：`context loading failed: no go files to analyze`; `go list ./...` 可正常列出包； scoped lint 同样失败 |
| `cd .; make secret-scan` | fallback | Windows 环境无 `make` 命令 |
| `python tools/secret_scan.py --include-untracked` | fallback | Windows 环境无 `python` 命令 |
| bundled Python `tools/secret_scan.py --include-untracked` | pass | `secret-scan: no high-confidence tracked-plus-untracked findings`；脚本跳过 `.env` / `.key` 等敏感文件且不打印命中值 |
| `gh pr view 3726 ...` | 待复核 | 本机无 `gh` 命令，无法读取 GitHub 远端 PR/CI 状态 |

---

## PR #3726 合并门禁表

| 门禁 | 状态 | 证据 / 说明 |
|------|------|-------------|
| 本地 go test | pass | `go test ./...` exit 0 |
| 本地 golangci-lint | fail / 待复核 | `golangci-lint run ./...` 工具上下文加载失败；未发现业务 lint 命中输出 |
| secret-scan | pass | bundled Python 跑 `tools/secret_scan.py --include-untracked` 无 high-confidence 命中 |
| CLA | 待老板 | 任务书记录 PR #3726 `CLA failed`，需 PR 提交者 / 老板 GitHub 账号处理 |
| CI 其他项 | 待复核 | 本机无 `gh`，未读取远端 checks |
| conflict with main | 待复核 | 未 fetch / merge；不做合并模拟 |
| deliverables 已 track | 待 commit | 文件已在工作树存在；Codex 本轮不 commit、不 push |

---

## 风险 / 阻塞

- `golangci-lint` 未获得绿色结果；当前证据指向工具上下文加载失败，但合并前仍需在可用 lint 环境复跑。
- PR #3726 的 CLA 和 CI 状态未在本机验证；任务书已标 CLA failed，合并前必须由老板在 GitHub 侧处理。
- R2-B/C 未授权扩样，不影响内部测试，但不能宣称“3+3 对账完成”。
- R2-A 使用过的临时 Key 需要老板废弃或轮换；Codex 未读取任何 Key 明文。

---

## 回滚方案

- 文档回滚：还原 `CODEX_START_HERE.md`、`2026-07-05-R2-closeout-summary.md`，删除本文件即可撤销 S1-CLOSEOUT 文档收口。
- 测试修复回滚：如需要，可单独还原 `api_key_video_gateway_test.go` 的 reason 期望；不建议在生产授权 gate 已改名后回滚。
- Git 操作回滚：本轮未 commit、未 push、未 merge、未 deploy，可由老板选择仅提交允许清单内文件。

---

## 给老板的 3 条待办

1. 在 GitHub 上签署 / 修复 PR #3726 CLA，并复核 CI checks。
2. 废弃或轮换 R2-A 正式 Seedance 冒烟使用过的临时 Key。
3. 决定是否授权 R2-B/C 扩样；若授权，需同时给出 NB2 / Seedance 范围、预算上限、停止条件，以及无需 Codex 读取 Key 明文的调用方式。

---

## 可复制后续提示词

```text
在 D:\sub2api-trunk 继续 Sub2API S1 合并前复核。禁止读 .env/Key/token/cookie，禁止 push/merge/deploy，禁止真实付费调用。
请只复跑：cd backend; golangci-lint run ./...；若可用，gh pr view 3726 --json statusCheckRollup,mergeable,mergeStateStatus。
把结果补进 docs/superpowers/codex-handoff/deliverables/2026-07-05-S1-CLOSEOUT-review.md 的 PR 门禁表。
```
