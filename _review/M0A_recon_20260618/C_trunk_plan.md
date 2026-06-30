# C · 干净主线方案（仅供拍板）— M0-A

> **本方案仅供拍板，M0-B 授权后才执行。本任务不执行其中任何写命令。**
> `[DRAFT 待学者校准]`

---

## 0. 方案前提（来自实扫的关键事实）

1. **全链线性**：`main(69f648e2)` → 25 commit → `night-run/D(40e83bf4)`，无分叉、零互冲突、D 为超集。
2. **零上游包袱**：这 25 commit **无一**是 origin/main(领先590) 的后代——它们是纯 fork 工作。
   → 任务书"优先 cherry-pick 以避开 590 落差"的**前提在此不成立**：ff 收编 D **不会**带入任何上游 commit。
3. **反而**：在一条**线性依赖链**上做"挑单 commit cherry-pick"会**割裂依赖**（如 #22 修 #21 的死代码、#21 建于 #06 真适配器之上、#19 建于 #18 之上），比整体收编**更难、更易错**。
4. 故本方案把"**整体线性收编**"列为**首选**，"squash 精修"为备选，"逐 commit cherry-pick"**不推荐**。

---

## 1. 新主线分支名（候选）

| 候选 | 理由 |
|---|---|
| **`wujie/trunk`**（推荐） | 与产品白标"无界互娱/企业 AI 视频 API 调度中台"一致；`wujie/` 命名空间清晰区隔上游 `main`/`dev`。 |
| `feat/video-gateway-trunk` | 描述性强，但偏长、未体现产品身份。 |

> 命名最终拍板 = 待学者。下文命令以 `wujie/trunk` 占位。

## 2. 基线

- **基线 = `main` 当前 HEAD `69f648e2`。** 它已含护城河底座 5 commit（P0 mock+白标 demo+phase3.8 锁定），且是全链 merge-base。**无需替代基线。**

---

## 3. 收编方案（二选一，待学者拍板）

### 方案 A（首选）— 整体线性 ff 收编 D

最干净、零冲突、保留完整演进与双签历史。

```bash
# —— 以下为 M0-B 提案命令，未执行 ——
git switch -c wujie/trunk main            # 从 main 建新主线
git merge --ff-only night-run/20260618-D-c1-alive
#   ↑ 因 D 是 main 直系后代，ff 推进，零冲突，纳入全部 25 commit
git status                                 # 应 clean
cd backend && go build ./... && go vet ./... && go test ./internal/service/...   # 验收（见 §6）
```
- **优点**：一步到位、零冲突、可追溯（含 codex+Claude 双签 commit msg）。
- **代价**：保留 7 条 docs/checkpoint 噪声 commit（无害，但历史略杂）。

### 方案 B（备选）— squash 精修后收编

若学者要"干净几个 commit"的主线历史，剔除 docs 噪声。

```bash
# —— M0-B 提案，未执行 ——
git switch -c wujie/trunk main
git merge --squash night-run/20260618-D-c1-alive   # 全部改动入暂存，不自动 commit
# 然后按逻辑分组手工提交（建议 4 组）：
#   1) 真适配器+smoke门+试点(①④)   2) 护城河:SSRF/脱敏/预算/key(⑥)
#   3) ratio竖屏+轮询窗(②)          4) 工程地基+迁移139+测试
```
- **优点**：主线历史干净。
- **代价**：丢失逐步双签历史；需人工分组、回归测试每组；工作量大于 A。
- **风险**：squash 会把"未提交物证"以外的一切压平，**测试基线必须全绿后再压**。

> **推荐：先用方案 A 拉起 `wujie/trunk` 跑通全绿测试，确认可用后**，若学者仍要干净历史，再在其上决定是否方案 B。

### 方案 C（不推荐）— 逐 commit cherry-pick

仅在"只要某几个孤立修复、明确不要其余"时才用。本链依赖紧密，cherry-pick 单 commit 会因缺前置而编译失败或语义割裂——**不建议**。

---

## 4. 建议收编顺序（若走方案 B/C 的逻辑分组参考）

按任务书"先 ratio → 再 Seedance 真调用 → 再其他"的精神，结合真实依赖（基座在前）：

1. **基座**：`1b8865ff`(真适配器) → `7ac5335b`(smoke门) → `3351338d`(mock网关) → `4dd599af`(tiny_real试点)
2. **护城河**：`1d5badd8`(SSRF+脱敏+key必填) → `85b6347f`(预算原语) → `c35049a4`(预算DI+回显守卫+反双发)
3. **工程地基+采集前置**：`7b78f9ca`(VA2 migration139+VA1预算+ratio响应侧)
4. **竖屏闭环**：`1be53de3`(ratio请求侧) → `831e9c98`(B2死代码修复)
5. **白标/部署/安全杂项**：`fede33aa` `884415ac` `c4e2337d` `9af63819` `bcb5fd32`(代码部分)
6. **测试证据**：`47cf1146` `40e83bf4`
7. **docs/checkpoint**：`fa442586` `53148df4` `4143673f` `0f68d5ab` `379da544` `ed83030b` `486f52fe` → **可丢弃或择 `53148df4`(AGENTS.md 护栏) 保留**

> 注意：方案 A（ff 全收）天然按链上真实顺序纳入，无需手工排序——上表只为方案 B/C 分组时参考。

---

## 5. 预期冲突点清单

| 场景 | 冲突风险 | 说明 |
|---|---|---|
| 方案 A（ff D 入基于 main 的新主线） | **零** | D 是 main 直系后代，纯快进。 |
| 方案 B（squash） | **低** | 单次 squash 无逐 commit 冲突；风险在人工分组提交后的回归。 |
| 方案 C（逐 cherry-pick） | **中-高** | 跳过前置 commit 会导致 `video_gateway_adapter.go`/`worker.go`/`config.go`/`migration` 编译或语义断裂。 |
| **未来**：把主线 rebase 到 origin/main(590) 做上游同步 | **高（点名）** | 大概率冲突文件：`backend/internal/service/video_gateway_*.go`、`backend/internal/config/config.go`、`backend/internal/handler/**`、`frontend/src/views/admin/video/*`、`backend/migrations/*`、`Dockerfile`/`deploy/*`。**M0-B 不做上游同步，故本次不触发；列此仅为后续预警。** |

---

## 6. 收编后建议验收（M0-B 执行，非本任务）

- `cd backend && go build ./... && go vet ./... && go test ./internal/service/...`（链上多 commit 自述全绿，需在收编后复跑坐实）。
- `go test -run C1 ./internal/server/routes/...`（C1 进程内活体）。
- **不**起服务、**不**真实调用、**不** push（红线照旧）。

---

## 7. 收编时的硬约束（红线，写进 M0-B）

1. ⛔ 绝不 `git push`（origin=上游，泄露红线）。
2. ⛔ 不开真实 Seedance 门（真适配器虽收编，但 env/元数据 smoke 门保持关闭，`¥0`）。
3. ✅ `night-run/D` 检出在主工作树——M0-B 前先确认主工作树 `git status`，**保住任何未提交真实物证**再动收编（见 §8）。
4. ✅ 收编全程保留 commit 的 codex+Claude 双签信息。

---

## 8. 收编前必做的两项工作树自查（交学者）

> 本侦察只看已提交历史，**未**进入下列工作树查看脏区（避免任何写风险）。M0-B 前请学者亲自：

1. 主工作树 `…/02_source/sub2api`（检出 `night-run/D`）：`git status` —— 若有**未提交的真实出片物证/脚本**，先决定如何归档/提交再收编。
2. WSL `…/sub2api-p0-9b-clean`（`p0-9b`，**worktree list 标 prunable**）：确认是否已失效；若有未提交内容先抢救，否则 M0-B 可 `git worktree prune`。

---

> 再次声明：**以上全部为 M0-B 提案，本 M0-A 任务一条未执行。** 工作树保持 clean（仅 `_review/` 未跟踪）。
