<!-- 单一收口物证 / 2026-06-15 / sub2api @ phase-3.8.2-overnight-readiness / 零真实调用·零真实凭据·未 commit 未 push -->

# SUB2API · Git 地图 · 单一收口物证

- **日期**：2026-06-15
- **仓库 / 分支**：`02_source/sub2api` @ `phase-3.8.2-overnight-readiness`（HEAD `4dd599af`）
- **范围**：把本工作树当前的 git 拓扑（remote / 分支 / worktree / 提交血统 / 工作树改动）**逐字**落成一张地图，并把每一个改动路径映射到【三个变更家族】。本文件**只读**采集，全程未改任何 git 状态。
- **结论(一句话)**：HEAD `4dd599af` 领先 `origin/main` **16 个 commit、落后 0**（→ **未 push**）；工作树 11 个被跟踪文件被改、14 个未跟踪条目，**全部未 staged**（→ **未 add、未 commit**）；改动可干净归入【甲 blocker 安全修复】【乙 白标/部署卫生】【丙 未跟踪文档+部署产物】三家族。
- **铁律遵守声明**：
  - ✅ 仅执行**只读** git（`status` / `log` / `branch -vv` / `worktree list` / `remote -v` / `rev-list` / `diff --stat`）；**未** `add` / `commit` / `push` / `restore` / `checkout` / `stash` / `reset`。
  - ✅ **零真实调用**任何 provider；本文档不含任何真实 key / token / JWT / secret。
  - ✅ 本物证包仅新建本 markdown 一个文件，未触碰其它任何文件。
  - ✅ 本文为**物证**，不声称已执行真实冒烟。

---

## 1. 仓库身份

`git remote -v`（逐字）：

```
origin	https://github.com/Wei-Shaw/sub2api.git (fetch)
origin	https://github.com/Wei-Shaw/sub2api.git (push)
```

- 唯一 remote：`origin = https://github.com/Wei-Shaw/sub2api.git`，fetch 与 push 同址。
- **当前分支**：`phase-3.8.2-overnight-readiness`。
- **HEAD**：`4dd599afa8a677c2585c22d4ca470c317f2b406e`（短 sha `4dd599af`），subject **`feat(video): prepare gated seedance tiny real trial`**。

---

## 2. 分支表

`git branch -vv`（逐字）：

```
  main                                            69f648e2 [origin/main: ahead 5] chore: lock phase 3.8.1 usability acceptance
+ p0-9b-seedance-real-smoke                       7ac5335b (/mnt/d/Codex创业任务/企业 API 管理后台项目/02_source/sub2api-p0-9b-clean) feat: guard seedance single smoke gateway
* phase-3.8.2-overnight-readiness                 4dd599af feat(video): prepare gated seedance tiny real trial
  safety-sub2api-before-dirty-resolution-20260605 7ac5335b feat: guard seedance single smoke gateway
```

逐条解释：

| 分支 | sha | 标记 | 与上游/彼此关系 |
|---|---|---|---|
| `main` | `69f648e2` | — | 跟踪 `origin/main`，`[origin/main: ahead 5]` → **本地 main 领先 origin/main 5 个 commit**（尚未 push 的 main 侧提交）。注意：main 当前并**不**指向 `4dd599af`。 |
| `p0-9b-seedance-real-smoke` | `7ac5335b` | `+`（被另一 worktree 占用） | `+` 表示该分支**已被另一个 worktree 检出**（`...02_source/sub2api-p0-9b-clean`，见第 3 节），故在本工作树不可同时检出。subject `feat: guard seedance single smoke gateway`。 |
| `phase-3.8.2-overnight-readiness` | `4dd599af` | `*`（当前） | `*` = 本工作树当前 HEAD。即本物证包的工作分支，subject `feat(video): prepare gated seedance tiny real trial`。 |
| `safety-sub2api-before-dirty-resolution-20260605` | `7ac5335b` | — | 一个**安全快照分支**，与 `p0-9b-seedance-real-smoke` 指向**同一 sha** `7ac5335b`（dirty 处理前的留底锚点）。 |

> 说明：`p0-9b-seedance-real-smoke` 与 `safety-sub2api-before-dirty-resolution-20260605` 同处 `7ac5335b`，前者被外部 worktree 占用，后者是纯留底引用——两者构成"处理脏工作区前"的回滚保险。

---

## 3. Worktree 地图

`git worktree list`（逐字，含 git 的八进制转义路径）：

```
"D:/Codex\345\210\233\344\270\232\344\273\273\345\212\241/\344\274\201\344\270\232 API \347\256\241\347\220\206\345\220\216\345\217\260\351\241\271\347\233\256/02_source/sub2api"                 4dd599af [phase-3.8.2-overnight-readiness]
"/mnt/d/Codex\345\210\233\344\270\232\344\273\273\345\212\241/\344\274\201\344\270\232 API \347\256\241\347\220\206\345\220\216\345\217\260\351\241\271\347\233\256/02_source/sub2api-p0-9b-clean" 7ac5335b [p0-9b-seedance-real-smoke] prunable
```

逐条解释：

| Worktree 路径 | HEAD sha | 检出分支 | 标记 |
|---|---|---|---|
| `...02_source/sub2api`（**本工作树**，Windows `D:` 盘符表示） | `4dd599af` | `phase-3.8.2-overnight-readiness` | 主工作树，本物证包全部采集发生于此。 |
| `...02_source/sub2api-p0-9b-clean`（同一物理目录的 WSL `/mnt/d` 表示） | `7ac5335b` | `p0-9b-seedance-real-smoke` | **`prunable`** |

- **`prunable` 的含义**：git 认为该 worktree 的工作目录已**不可达/可被回收**（典型为目录已被移动、删除或在另一文件系统视图下失效）。它仍占用着 `p0-9b-seedance-real-smoke` 分支（故第 2 节该分支带 `+`），但 `git worktree prune` 可将其登记清除。**本物证只读，不执行 prune**——`prunable` 的处置方案（plan-only：`git worktree prune` / `repair` 取舍与安全前提）见关联文档③[《SUB2API_WORKTREE_CLEANUP》](./SUB2API_WORKTREE_CLEANUP_2026_06_15.md)第 5 节。
- 两条路径分别用 Windows（`D:/…`）与 WSL（`/mnt/d/…`）前缀指向同一磁盘位置，是跨 Win/WSL 视图导致 git 对同一物理目录登记出两份记录的根因；这也是该外部 worktree 被判为 `prunable` 的可能来源之一。

---

## 4. 提交血统

`git log --oneline -8`（逐字）：

```
4dd599af feat(video): prepare gated seedance tiny real trial
3351338d feat: add api-key video mock gateway for qcanvas
4143673f docs: record sub2api dirty resolution checkpoint
c4e2337d wip: checkpoint sub2api day0 video gateway and white-label worktree
7ac5335b feat: guard seedance single smoke gateway
1b8865ff feat(video-gateway): Seedance 2.0 真实 API 适配器
884415ac feat: 内部试点视图 + 视频路由注册 + i18n/路由/认证优化
0774feeb feat(video-gateway): add api-first drama skill learning pilot
```

最近 5 个 commit 各自加了什么：

| # | sha | subject | 这条 commit 加了什么 |
|---|---|---|---|
| 1（HEAD） | `4dd599af` | `feat(video): prepare gated seedance tiny real trial` | 在闸门保护下**准备** seedance tiny real 试点（gated、未真实调用）；本工作分支顶点。 |
| 2 | `3351338d` | `feat: add api-key video mock gateway for qcanvas` | 为 QCanvas 落地 **api-key 视频 mock gateway**（A 侧契约对接所依赖的 Go 侧 mock 入口）。 |
| 3 | `4143673f` | `docs: record sub2api dirty resolution checkpoint` | 文档 commit：记录 sub2api **脏工作区处理 checkpoint**。 |
| 4 | `c4e2337d` | `wip: checkpoint sub2api day0 video gateway and white-label worktree` | WIP checkpoint：day0 视频网关 + **白标 worktree**（白标/部署线的早期落点）。 |
| 5 | `7ac5335b` | `feat: guard seedance single smoke gateway` | 为 **seedance 单次冒烟网关**加守卫（即 `p0-9b-seedance-real-smoke` / `safety-…20260605` 两分支所在 sha）。 |

- **领先量**：`git rev-list --left-right --count origin/main...HEAD` →

  ```
  0	16
  ```

  左 `0` = `origin/main` 相对 HEAD 独有 0 个 commit（HEAD 未落后）；右 `16` = **HEAD 领先 `origin/main` 16 个 commit**。结合 remote 单一且未推送 → 这 16 个 commit **均未 push**。

---

## 5. 工作树改动清单

`git status --porcelain=v1`（逐字全文）：

```
 M .gitignore
 M Dockerfile
 M backend/internal/repository/video_key_encryptor.go
 M backend/internal/server/routes/api_key_video_gateway_test.go
 M backend/internal/service/video_gateway_adapter.go
 M backend/internal/service/video_gateway_service.go
 M backend/internal/service/video_gateway_types.go
 M deploy/Caddyfile
 M deploy/Dockerfile
 M deploy/config.example.yaml
 M frontend/src/views/admin/video/videoUtils.ts
?? 00_START_HERE.md
?? 01_PROJECT_BASELINE.md
?? 02_CURRENT_REALITY_STATUS.md
?? AUDIT_REPORT.md
?? BOSS_DEPLOY_GUIDE.md
?? backend/internal/service/video_gateway_redact.go
?? backend/internal/service/video_gateway_security_test.go
?? backend/internal/service/video_gateway_ssrf.go
?? deploy/backup.sh
?? deploy/day0/
?? deploy/docker-compose.wsl.prod.yml
?? deploy/docker-compose.wsl.yml
?? "\347\234\237\345\256\236Seedance2.0\345\205\250\351\223\276\350\267\257\346\216\245\345\205\245\346\226\271\346\241\210.md"
?? "\347\234\237\345\256\236Seedance2.0\346\226\271\346\241\210_\345\257\271\346\212\227\350\257\204\345\256\241.md"
```

> porcelain 列读法：第 1 列=已暂存状态，第 2 列=工作区状态。这里 ` M`（**前导空格** + `M`）表示"**仅工作区修改、未暂存**"；`??` 表示"**未跟踪**"。**没有任何一行第 1 列非空** → 工作区零暂存 → 未 `add`、未 `commit`。

`git diff --stat`（逐字，含两条 LF→CRLF 无害 warning，如实保留）：

```
warning: in the working copy of '.gitignore', LF will be replaced by CRLF the next time Git touches it
warning: in the working copy of 'deploy/Caddyfile', LF will be replaced by CRLF the next time Git touches it
 .gitignore                                         |  5 ++
 Dockerfile                                         |  9 +--
 backend/internal/repository/video_key_encryptor.go | 12 ++--
 .../server/routes/api_key_video_gateway_test.go    |  3 +
 backend/internal/service/video_gateway_adapter.go  | 69 ++++++++++++++++++++--
 backend/internal/service/video_gateway_service.go  |  3 +
 backend/internal/service/video_gateway_types.go    | 32 +++++++++-
 deploy/Caddyfile                                   | 45 ++++++--------
 deploy/Dockerfile                                  |  9 +--
 deploy/config.example.yaml                         | 11 ++--
 frontend/src/views/admin/video/videoUtils.ts       |  7 ++-
 11 files changed, 154 insertions(+), 51 deletions(-)
```

→ **11 个被跟踪文件被改，+154 / -51**。下表把**每一个路径**映射到三家族之一。

### 5.1 路径 → 家族映射表（被跟踪 11 改 + 未跟踪 14 项）

| 路径 | git 状态 | 家族 | 说明 |
|---|---|---|---|
| `backend/internal/service/video_gateway_adapter.go` | ` M` 改 | **甲** blocker 安全修复 | B1a 错误体脱敏（非2xx + 200-OK 业务错误）、B2 下发 duration/resolution/aspect_ratio、B3a reference/result URL SSRF 校验、写脱敏审计、smoke gate 强制 allowlist。 |
| `backend/internal/service/video_gateway_service.go` | ` M` 改 | **甲** | 状态层 `blocked_reasons` 同步加入 URL allowlist 必填项。 |
| `backend/internal/service/video_gateway_types.go` | ` M` 改 | **甲** | `VideoProviderAccount` 加 `String()`/`GoString()`/`LogValue()` 屏蔽 `PlainAPIKey`，并加 `json:"-"`。 |
| `backend/internal/repository/video_key_encryptor.go` | ` M` 改 | **甲** | B3b：`video_gateway.encryption_key` 为空时由 totp fallback 改为**硬失败**。 |
| `backend/internal/server/routes/api_key_video_gateway_test.go` | ` M` 改 | **甲** | 成功用例 fixture 补 allowlist env（新必需前提，非弱化断言）。 |
| `.gitignore` | ` M` 改 | **甲** | 仅"脱敏事件日志忽略段"（`backend/video-redacted-events.log`、`*.redacted-events.log`）。 |
| `deploy/config.example.yaml`（**混合文件**） | ` M` 改 | **甲 + 乙** | 甲：`video_gateway.encryption_key` 改"必填/启动即失败"文案段；乙：另一处 `admin_password` 改 `CHANGE_ME` 的 hunk。**同一文件横跨两家族**。 |
| `backend/internal/service/video_gateway_redact.go` | `??` 新 | **甲**（新增） | `redactVideoUpstreamSecrets`（pattern 脱敏 + `AKLT`）+ `appendRedactedVideoEvent`（0600、fail-closed）。 |
| `backend/internal/service/video_gateway_ssrf.go` | `??` 新 | **甲**（新增） | `validateExternalVideoURL`（拒反斜杠/空白/控制字符/userinfo + 归一尾点 + https-only + 内网封锁 + allowlist）。 |
| `backend/internal/service/video_gateway_security_test.go` | `??` 新 | **甲**（新增） | 13 个安全行为测试（脱敏/审计/SSRF/key 不外泄/duration/result_url 等）。 |
| `Dockerfile` | ` M` 改 | **乙** 白标/部署卫生 | 与 blocker 无关。 |
| `deploy/Dockerfile` | ` M` 改 | **乙** | 同上。 |
| `deploy/Caddyfile` | ` M` 改 | **乙** | 同上（带 LF→CRLF warning）。 |
| `frontend/src/views/admin/video/videoUtils.ts` | ` M` 改 | **乙** | 前端白标工具调整。 |
| `00_START_HERE.md` | `??` 新 | **丙** 未跟踪文档+部署产物 | 未跟踪文档。 |
| `01_PROJECT_BASELINE.md` | `??` 新 | **丙** | 未跟踪文档。 |
| `02_CURRENT_REALITY_STATUS.md` | `??` 新 | **丙** | 未跟踪文档。 |
| `AUDIT_REPORT.md` | `??` 新 | **丙** | 未跟踪文档。 |
| `BOSS_DEPLOY_GUIDE.md` | `??` 新 | **丙** | 未跟踪文档。 |
| `真实Seedance2.0全链路接入方案.md` | `??` 新 | **丙** | 未跟踪文档（porcelain 中显示为八进制转义路径）。 |
| `真实Seedance2.0方案_对抗评审.md` | `??` 新 | **丙** | 未跟踪文档（同上转义）。 |
| `deploy/backup.sh` | `??` 新 | **丙** | 部署产物脚本。 |
| `deploy/day0/`（目录：`backup.sh`、`check.sh`、`start.sh`、`stop.sh`、`windows_disable_lan_access.ps1`、`windows_enable_lan_access.ps1`） | `??` 新 | **丙** | 未跟踪目录，porcelain 折叠成单行 `?? deploy/day0/`；内含 6 个 day0 运维脚本。 |
| `deploy/docker-compose.wsl.prod.yml` | `??` 新 | **丙** | 部署产物（WSL prod compose）。 |
| `deploy/docker-compose.wsl.yml` | `??` 新 | **丙** | 部署产物（WSL compose）。 |

### 5.2 三家族小结

- **家族甲（Seedance blocker 安全修复）= 7 改 + 3 新**：
  - 改：`video_gateway_adapter.go`、`video_gateway_service.go`、`video_gateway_types.go`、`video_key_encryptor.go`、`api_key_video_gateway_test.go`、`.gitignore`（仅脱敏日志忽略段）、`deploy/config.example.yaml`（仅 encryption_key 必填文案段）。
  - 新：`video_gateway_redact.go`、`video_gateway_ssrf.go`、`video_gateway_security_test.go`。
- **家族乙（白标/部署卫生，与 blocker 无关）**：`Dockerfile`、`deploy/Dockerfile`、`deploy/Caddyfile`、`frontend/src/views/admin/video/videoUtils.ts`，外加 `deploy/config.example.yaml` 里 `admin_password→CHANGE_ME` 的那处 hunk。
- **家族丙（未跟踪文档 + 部署产物）**：5 份英文/编号 markdown + 2 份"真实Seedance2.0…"中文 markdown + `deploy/backup.sh` + `deploy/day0/`（6 脚本）+ 2 份 `docker-compose.wsl*.yml`。

> 关键提醒：`deploy/config.example.yaml` 是**唯一一个跨家族（甲+乙）的混合文件**——若后续要按家族拆分提交，这一文件需按 hunk 切分。

---

## 6. "已跟踪 vs 未跟踪"澄清 + 未 commit/未 push 归因

- **已跟踪、被修改（11 项，porcelain 第二列 `M`）**：分布于家族甲（7 项含混合文件）与家族乙（4 项 + 混合文件的另一 hunk）。**第一列全为空格 → 全部未暂存**。
- **未跟踪（14 个 porcelain 条目，`??`）**：全部归家族丙（其中 `deploy/day0/` 是被折叠的目录，实含 6 个脚本）。git 从未追踪过它们，故既不在 `diff --stat`（仅统计被跟踪改动）内，也不会被任何只读命令改动。
- **未 add / 未 commit 归因**：`git status --porcelain=v1` 中**无任何一行第 1 列非空白** → 暂存区为空 → 没有 `git add`、没有 `git commit`（与 `git status` 文末 `no changes added to commit` 一致）。
- **未 push 归因**：`origin/main...HEAD` 计数为 `0 16` → HEAD 领先 16、落后 0；唯一 remote `origin` 未收到这些 commit → 未 push。
- 以上工作树状态与"未 commit / 未 push"的完整签字口径与本物证包《签字物证》一致，详见关联文档①。

---

## 关联文档

本文件为单一收口物证包 4 份之一，互相交叉引用：

- ① [签字物证 · SUB2API_SIGNOFF_EVIDENCE_2026_06_15](./SUB2API_SIGNOFF_EVIDENCE_2026_06_15.md)
- ②（本文）[Git 地图 · SUB2API_GIT_MAP_2026_06_15](./SUB2API_GIT_MAP_2026_06_15.md)
- ③ [Worktree 清理 · SUB2API_WORKTREE_CLEANUP_2026_06_15](./SUB2API_WORKTREE_CLEANUP_2026_06_15.md)
- ④ [冒烟准备 · SUB2API_SMOKE_PREP_2026_06_15](./SUB2API_SMOKE_PREP_2026_06_15.md)
- 权威前序评审：[SUB2API_SEEDANCE_BLOCKER_FIX_REVIEW_2026_06_15](./SUB2API_SEEDANCE_BLOCKER_FIX_REVIEW_2026_06_15.md)
