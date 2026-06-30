# A · 分支测绘总表 — Sub2API 分支收编 M0-A 只读侦察

> 复核日期：2026-06-18 ｜ 维护人：学者 / 方浩臣 ｜ 状态：`[DRAFT 待学者校准]`
> 全程只读，零写操作。证据均为 commit hash / file:line。

---

## 0. 头号结论（一句话）

**所谓"10+ 条从未并回 main 的分支"，其实是同一条线性开发链上的 25 个 commit 的不同检查点标签。**
它们全部从 `main` 当前 HEAD（`69f648e2`）切出、**落后 main = 0**、彼此严格祖先关系，**互相之间零冲突**。
最末端 `night-run/20260618-D-c1-alive`（`40e83bf4`）是**超集**，线性包含其余所有分支的全部工作。

> 这彻底改变了收编策略：原任务书假设"优先 cherry-pick 单 commit 以避开旧上游 590 落差"——但这 25 个 commit **没有一个**是上游 590 commit 的后代，它们是纯 fork 工作，ff 收编 D **不会**带入任何上游包袱。详见 [C_trunk_plan.md](./C_trunk_plan.md)。

---

## 1. 仓库基线快照（步骤 1）

| 项 | 值 | 证据 |
|---|---|---|
| origin | `https://github.com/Wei-Shaw/sub2api.git`（fetch=push 同一上游） | `git remote -v` |
| main HEAD | `69f648e2` chore: lock phase 3.8.1 usability acceptance | `git rev-parse main` |
| main vs origin/main | **落后 590 / 领先 5** | `git rev-list --left-right --count origin/main...main` → `590	5` |
| main 自有 5 commit | `4c5de849`→`bade34de`→`43003a00`→`58f79542`→`69f648e2` | `git log origin/main..main --oneline` |

**main 自有 5 commit 摘要（视频网关白送底座 + 白标 demo）：**

| hash | message | 改动摘要 |
|---|---|---|
| `4c5de849` | feat(video-gateway): add P0 mock provider workflow | 38 文件 +4140：视频网关全套骨架（adapter/service/worker/types/repo/migration 136 + 前端 video 视图） |
| `bade34de` | chore(video-gateway): checkpoint P0.5 localized QA flow | 9 文件：前端 video 视图本地化 QA |
| `43003a00` | feat(video-gateway): add demo-safe white label mode | 17 文件：productMode 白标开关 + THIRD_PARTY_NOTICES |
| `58f79542` | chore: lock phase 3.8 final gateway baseline | 20 文件 +2065：handler/service/worker_test + migration 137（reuse-first demo accounts） |
| `69f648e2` | chore: lock phase 3.8.1 usability acceptance | 6 文件：前端 video 视图可用性收尾 |

> 这 5 个 commit 即"护城河底座"，是本次所有未并工作的共同起点（merge-base）。

---

## 2. 全分支清点（步骤 2）

### 2.1 本地分支（领先 main / 落后 main / 最近提交）

| 分支 | 领先 main | 落后 main | tip commit | 最近提交 | 说明 |
|---|---:|---:|---|---|---|
| `night-run/20260618-D-c1-alive` ⭐ | **25** | 0 | `40e83bf4` | 2026-06-18 | **超集 tip**（含全部）。在主工作树检出（见 §3） |
| `night-run/20260618-C-skeleton` | 24 | 0 | `486f52fe` | 2026-06-18 | D 的祖先 |
| `night-run/20260618-B-contract` | 22 | 0 | `831e9c98` | 2026-06-18 | C 的祖先 |
| `night-run/20260618-A-truth` | 20 | 0 | `ed83030b` | 2026-06-18 | B 的祖先 |
| `feat/phase2b-real-smoke-20260616` | 19 | 0 | `c35049a4` | 2026-06-16 | A 的祖先 |
| `feat/phase2a-pre-arming-20260616` | 18 | 0 | `85b6347f` | 2026-06-16 | 2b 的祖先 |
| `feat/phase1-eng-baseline-20260616` | 17 | 0 | `7b78f9ca` | 2026-06-16 | 2a 的祖先 |
| `phase-3.8.2-overnight-readiness` | 16 | 0 | `bcb5fd32` | 2026-06-16 | phase1 的祖先 |
| `p0-9b-seedance-real-smoke` | 7 | 0 | `7ac5335b` | 2026-06-05 | 3.8.2 的祖先；在 WSL 工作树检出（**prunable**） |
| `safety-sub2api-before-dirty-resolution-20260605` | 7 | 0 | `7ac5335b` | 2026-06-05 | **与 p0-9b 同一 commit，纯重复备份** |
| `claude/nostalgic-chatterjee-703296` | 0 | 0 | `69f648e2` | 2026-05-18 | = main（worktree 占位，无自有工作） |
| `claude/objective-goldstine-b9bf89` | 0 | 0 | `69f648e2` | 2026-05-18 | = main（**本侦察工作树**） |

证据：`git rev-list --count main..<b>` / `git merge-base --is-ancestor` / `git rev-parse`。

### 2.2 远端追踪分支（上游 Wei-Shaw/sub2api，**非我方工作**，不收编）

`origin/main`(=`4a5665da`)、`origin/dev`、`origin/preview`、`origin/preview-dev`、`origin/cla-signatures`、`origin/feat/api-key-ip-restriction`、`origin/revert-114-feature/atomic-scheduling`、`origin/HEAD→origin/main`。
→ 这些是开源上游的开发线，与本次收编无关；仅说明 origin = 活跃上游（590 落差来源）。

### 2.3 链上里程碑 tag（4 个，落在 main..D 区间内）

| tag | → commit | 含义 |
|---|---|---|
| `phase-4b4-internal-pilot-ready`（附注 tag，对象 `f3f538e8`） | `0774feeb` | 内部试点就绪（drama 引擎） |
| `checkpoint/p0-9a-seedance-single-smoke-gate` | `7ac5335b` | Seedance 单条 smoke 门（=p0-9b tip） |
| `checkpoint/sub2api-apikey-video-mock-gateway-20260609` | `3351338d` | qcanvas api-key mock 网关 |
| `checkpoint/sub2api-seedance-tiny-real-c0-prep-20260609` | `4dd599af` | tiny-real 试点准备 |

（另有 `v0.1.0`–`v0.1.137` 等 137 个上游发布 tag，与我方工作无关。）

---

## 3. 线性 DAG 全貌（步骤 3 核心 · 一条直线，无分叉）

```
night-run/D  40e83bf4  [25] test: 阶段0+1定标准 + C1进程内活体(keyless ¥0, 未开真门)        keep-test
night-run/C  486f52fe  [24] docs: 00_黎明总结                                                docs
             47cf1146  [23] test: C1骨架契约测试(mock)                                       keep-test
night-run/B  831e9c98  [22] fix: B2分辨率分层"死代码"修复 + B1直通收紧                        keep-code
             1be53de3  [21] fix: ★B1 aspect_ratio→ratio请求侧修复★ + B2轮询窗 + B3换皮草案    keep-code
night-run/A  ed83030b  [20] docs: 阶段A M0收敛真相源                                          docs
phase2b      c35049a4  [19] feat: ★预算门DI接线★ + 命门级脱敏 + 形态A harness + 反双发守卫     keep-code
phase2a      85b6347f  [18] feat: ★StaticBudgetGuard 预算拦截原语★(未接DI)                    keep-code
phase1       7b78f9ca  [17] feat: ★工程地基★ VA2轮询上限/VA1预算门骨架/ratio响应侧/wsl卫生    keep-code
phase-3.8.2  bcb5fd32  [16] docs(收口) + 部署归档 + 惰性realsmoke harness + 不透明token脱敏    keep-code(mixed)
             379da544  [15] docs: Caddyfile 注释订正                                          docs
             0f68d5ab  [14] docs: blocker评审包(5份)                                          docs
             9af63819  [13] chore: 白标 + 部署卫生(admin_password弱默认→占位)                 keep-code
             1d5badd8  [12] fix: ★Seedance blocker B1/B2/B3★ SSRF门+脱敏+0600审计+key必填     keep-code
             4dd599af  [11] feat: ★gated seedance tiny_real 试点★(1次/日/人限额, 不自动开火)  keep-code
             3351338d  [10] feat: qcanvas api-key 视频MOCK网关(/video路由, 拒seedance/kling)  superseded
             4143673f  [09] docs: dirty-resolution checkpoint                                docs
p0-9b/safety c4e2337d  [08] wip: day0视频网关+白标 checkpoint(migration 138白标文案)          keep-code
             7ac5335b  [07] feat: ★Seedance 单条 smoke 门★(env+元数据+时长1-5s授权)           keep-code
             1b8865ff  [06] feat: ★Seedance 2.0 真实Ark适配器★(POST/GET ark.cn-beijing)      keep(基础)
             884415ac  [05] feat: 内部试点视图 + 路由注册 + i18n/白标(企业AI视频API调度中台)  keep-code
phase-4b4    0774feeb  [04] feat: drama技能学习引擎(drama_gateway_service.go 1362行内存引擎)   keep-code
             53148df4  [03] docs: AGENTS.md codex 护栏                                        keep-docs
             fede33aa  [02] feat: phase4b内部MVP(APIKeyConfigured只认EncryptedAPIKey安全修复) keep-code
             fa442586  [01] chore: phase3.8.2 打包工具(tools/phase_3_8_2/*.js/.ps1)          checkpoint
─────────────  main  69f648e2  [00] (共同基线 = 护城河底座 5 commit 之顶)  ────────────────────
```

证据：`git log --graph --oneline main..night-run/20260618-D-c1-alive`（输出全为 `*`，零 `|/` 分叉标记 → 严格线性）。

### 3.1 关键工作命中索引（任务书 §3 步骤3 的 6 类）

| 关键工作 | 命中 commit（file:line 证据） | 现状 |
|---|---|---|
| ① Seedance 真调用 | `1b8865ff` 引入真实 Ark HTTP（POST `/contents/generations/tasks`、GET `…/tasks/{id}`，默认 `ark.cn-beijing.volces.com/api/v3`，model `doubao-seedance-2-0-260128`）；`7ac5335b` 加 smoke 门；`c35049a4` 形态A harness | 仅在链上，**main 上为禁用骨架**（见 F·G6/G7） |
| ② `aspect_ratio→ratio` | **响应侧** `7b78f9ca`（poll 解析 `ratio`）；**请求侧** `1be53de3`（`BuildCreatePayload`/`CreateTask` 改发 `ratio` + `normalizeSeedanceRatio`）；`831e9c98` 收紧直通 | 两半都在链上，**main 仍发 `aspect_ratio`**（adapter:110/180/238，见 F·G8） |
| ③ Kling 真调用 | **无**。全链 Kling 始终禁用（`KLING_REAL_CALL_DISABLED`） | 未做 |
| ④ C1 活体 / 真实出片 | `4dd599af`（tiny_real 试点骨架）；`47cf1146`（C1 契约测试 mock）；`40e83bf4`（C1 进程内活体 mock，断言真门关闭 `real_provider_dispatch_count==0`） | 均为 **mock 活体**，未真实出片 |
| ⑤ 采集相关 | `7b78f9ca` 加 migration 139（`video_tasks.poll_count` 列）；`c4e2337d` migration 138（白标文案 UPDATE）。**未动 `ent/schema/usage_log.go`**（采集口命门，见 F·G9） | 采集口未改 |
| ⑥ 其他护城河 | `1d5badd8` SSRF门(`video_gateway_ssrf.go`)+脱敏(`video_gateway_redact.go`)+0600审计+key加密必填；`c35049a4` 预算门 DI+回显凭证 fail-closed；`0774feeb` drama 引擎 | 链上 keep-code |

---

## 4. 与本表配套的文件

- 逐 commit 处置建议 + 理由 + 证据 → [B_disposition.md](./B_disposition.md)
- 干净主线方案（新分支名/基线/收编顺序/冲突点） → [C_trunk_plan.md](./C_trunk_plan.md)
- G1–G9 逐条复核 → [F_recheck.md](./F_recheck.md)
- 只读命令执行日志 → [E_command_log.md](./E_command_log.md)
- 五件套草稿 → [D_anchors/](./D_anchors/)
