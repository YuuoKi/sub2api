<!-- 单一收口物证 / 2026-06-15 / sub2api @ phase-3.8.2-overnight-readiness / 零真实调用·零真实凭据·未 commit 未 push -->

# SUB2API · 工作树收口 = commit 切分【方案】

> ## ⚠️ 本文件只给【方案】，不执行任何改变 git 状态的命令；不 push。
> 下面所有 `git add` / `git commit` 文本都是**给用户本人 review 后自行执行**的脚本草案。本任务**只成文、不执行**：撰写本文时**未** `git add`、**未** `git commit`、**未** `git push`、**未** `restore/checkout/stash/reset`。仓库全程【只读】。

- **日期**：2026-06-15
- **仓库 / 分支**：`02_source/sub2api` @ `phase-3.8.2-overnight-readiness`（HEAD `4dd599af`）
- **范围**：把当前**混在一个工作树里的三个变更家族**切成有序、可审计的 commit 序列；给出精确 `git add` 路径与拟用 commit message（conventional commits 风格，对齐现有 log）。
- **结论(一句话)**：工作树同时压着【blocker 安全修复】【白标/部署卫生】【部署基建+文档】三家族；其中**已被双家族签字 GO 的核心安全改动**应**第一个、单独**落 commit（Commit A），其余三类按 B→C→D 顺序跟进；本文只给方案，**现在不要执行**。

---

## 0. 实时地面真相（撰写时只读采集）

逐字来自 `git status --porcelain=v1` / `git diff --stat` / `git rev-list` / `git worktree list` / `git remote -v`（本文撰写时实际运行的只读命令）：

- **HEAD**：`4dd599af` "feat(video): prepare gated seedance tiny real trial"。
- **相对 origin/main**：`git rev-list --left-right --count origin/main...HEAD` = `0   16` → 落后 0、**领先 16，尚未 push**。
- **工作树改动全部未暂存**（`porcelain` 第二列为空）→ **未 commit、未 `git add`**。
- **`git diff --stat`**：11 个被跟踪文件被改，**+154 / −51**（伴随两条 `LF will be replaced by CRLF`（`.gitignore`、`deploy/Caddyfile`）的 git 无害换行 warning，如实保留，不影响切分）。
- **remote**：`origin` = `https://github.com/Wei-Shaw/sub2api.git`（fetch+push）。
- **worktree**：本工作树（`4dd599af` / `phase-3.8.2-overnight-readiness`）+ 另一个 `…/02_source/sub2api-p0-9b-clean`（`7ac5335b` / `p0-9b-seedance-real-smoke`，标 `prunable`）。worktree **拓扑全貌**见姊妹文档②[《SUB2API_GIT_MAP》](./SUB2API_GIT_MAP_2026_06_15.md)第 3 节；`prunable` worktree 的**处置方案（plan-only）见本文第 5 节**。本文 §1–§4 专管"工作树内文件 → commit 切分"，§5 附 prunable worktree 处置方案。

---

## 1. 为什么必须切分：一个工作树压了三个家族

当前 `phase-3.8.2-overnight-readiness` 工作树把**三类互不相干的变更**叠在了一起。若一把 `git add -A && git commit` 会把安全修复、部署卫生、未跟踪文档糊成一坨，**无法被单独 revert / cherry-pick / 审计**，也违背"已签字的核心安全改动应可独立追溯"的要求。

| 家族 | 性质 | 与 blocker 关系 | 签字状态 |
|---|---|---|---|
| **甲** Seedance blocker 安全修复 | 7 改 + 3 新 | **就是 blocker 修复本体** | ✅ codex(GPT) + Claude **双家族 GO**（见前序评审第 5/6 节） |
| **乙** 白标 / 部署卫生 | 4 改 + config 内 1 个 hunk | 无关 | 无（普通卫生改动） |
| **丙** 部署基建产物（未跟踪） | 一批脚本 / compose | 无关 | 无 |
| **丁** 项目/参考文档（未跟踪） | 一批 `*.md` + 本物证包 | 无关 | 无 |

**核心原则**：**家族甲（已签字的安全修复）必须第一个、单独提交（Commit A）**，使其在历史中干净可追溯；其余按 B→C→D 跟进。

---

## 2. ⚠️ 混合文件风险（必须显式点名）

切分前必须知道：**`deploy/config.example.yaml` 是一个"混合文件"**——它**同时**承载家族甲与家族乙的改动，**不能整文件 `git add`**，否则会把乙的卫生改动误并入安全 Commit A。撰写本文时已 `git --no-pager diff` 逐字核对，确认它含**两个独立 hunk**：

- `@@ -858,10 +858,11 @@`（家族甲 / 属 Commit A）：`video_gateway.encryption_key` 文案改为"必填 / 启动即失败 / 不再回退 totp / 须不同于 totp.encryption_key"。
- `@@ -942,7 +943,7 @@`（家族乙 / 属 Commit B）：`admin_password: "admin123"` → `"CHANGE_ME_STRONG_PASSWORD"` 的**无关**卫生改动。

> **对照另一个被点名的混合候选 `.gitignore`**：已逐字核对，`.gitignore` 本次**仅**含一个 hunk（脱敏事件日志忽略段：`backend/video-redacted-events.log` + `*.redacted-events.log`），**干净、整文件归入 Commit A 即可**，无需拆 hunk。

**处理混合文件 `deploy/config.example.yaml` 的两条路线（A 计划首选第一条）**：

- **【推荐】按 hunk 暂存**：`git add -p deploy/config.example.yaml`，交互式**只 stage `encryption_key` 那个 hunk（即第 858 行附近的 `y`）**，对 `admin_password` 那个 hunk 选 `n`，留到 Commit B。
- **【可接受的替代】两 hunk 一起进 A**：若不想拆 hunk，可整文件归入 Commit A，**但必须在 Commit A 的 message body 里显式说明"本提交顺带含一处无关的 `admin_password` 占位符卫生改动"**，以保留审计可读性（不推荐，因混淆了"安全修复"与"卫生"的边界）。

---

## 3. commit 切分【方案】（给用户本人执行的脚本草案）

> 下列每条 `git add` / `git commit` **均为方案文本，本任务一律不运行**。message 采用 conventional commits 风格，对齐现有 log（`feat(video):` / `feat:` / `docs:` / `chore:`）。占位/凭据一律占位符，绝不编造真实密钥。

### ⭐ Commit A —— Seedance blocker 安全修复（**第一个、单独提交**）

家族甲：**7 改 + 3 新**。这是已被 codex + Claude 双家族签字 GO 的核心安全改动，**应优先单独落历史**。

```bash
# —— 先 stage 7 个被跟踪的"改"文件中 6 个可整文件加的 ——
git add backend/internal/service/video_gateway_adapter.go
git add backend/internal/service/video_gateway_service.go
git add backend/internal/service/video_gateway_types.go
git add backend/internal/repository/video_key_encryptor.go
git add backend/internal/server/routes/api_key_video_gateway_test.go
git add .gitignore                       # 本次仅脱敏日志段，干净，整文件可加

# —— 混合文件：仅 stage encryption_key 那个 hunk（admin_password 留给 Commit B）——
git add -p deploy/config.example.yaml    # 对 858 行附近 encryption_key hunk 选 y；对 admin_password hunk 选 n

# —— 再 stage 3 个新文件 ——
git add backend/internal/service/video_gateway_redact.go
git add backend/internal/service/video_gateway_ssrf.go
git add backend/internal/service/video_gateway_security_test.go
```

**提交前自检（强烈建议，仍为只读 / 暂存区检查）**：

```bash
git status --porcelain=v1                 # 确认 deploy/config.example.yaml 同时出现在已暂存(第1列M)与未暂存(第2列M)——证明 hunk 拆分成功
git --no-pager diff --cached --stat       # 复核进入 A 的就是这 10 个落点、且 config 只进了 encryption_key 行
```

拟 commit message：

```text
fix(video): redact upstream secrets, add SSRF guards, require dedicated encryption key

Close the Go-side parts of the three Seedance 2.0 blockers (B1 secret
leakage / B2 cost contract / B3 auth+SSRF), reviewed and signed off by
both codex (GPT) and Claude families.

- B1a: redact upstream error bodies (non-2xx AND 200-OK business errors)
  before they reach DB / events / API response
- B1b: actually implement the redacted audit log (0600, append-only,
  fail-closed on write failure)
- B1c: mask VideoProviderAccount.PlainAPIKey in String/GoString/LogValue
  and add json:"-"
- B2: send duration/resolution/aspect_ratio in the create payload so the
  smoke-gate duration cap is enforced upstream (Ark field names UNVERIFIED,
  to be confirmed at first real smoke)
- B3a: SSRF-validate reference_image_url and result_url (https-only,
  allowlist, reject backslash/userinfo parser-differential)
- B3b: hard-fail when video_gateway.encryption_key is empty instead of
  silently falling back to totp.encryption_key (key-domain confusion)

No real upstream calls; tests use httptest + dummy key. Not pushed.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>
```

> 进入 A 的精确落点（10 个）：`video_gateway_adapter.go`、`video_gateway_service.go`、`video_gateway_types.go`、`video_key_encryptor.go`、`api_key_video_gateway_test.go`、`.gitignore`[仅脱敏日志段]、`deploy/config.example.yaml`[**仅 encryption_key hunk**] + 新增 `video_gateway_redact.go`、`video_gateway_ssrf.go`、`video_gateway_security_test.go`。

### Commit B —— 白标 / 部署卫生（与 blocker 无关）

家族乙：4 个被跟踪文件 + **config 的 admin_password hunk**（若 A 已按推荐路线把它留下，则在此收尾）。

```bash
git add Dockerfile
git add deploy/Dockerfile
git add deploy/Caddyfile
git add frontend/src/views/admin/video/videoUtils.ts

# 若 Commit A 用了 git add -p 只取 encryption_key，则 config 的 admin_password hunk 此刻仍未暂存——整文件加即可收尾它
git add deploy/config.example.yaml        # 此时只剩 admin_password hunk
```

拟 commit message：

```text
chore(deploy): white-label hygiene and harden default admin password placeholder

- Dockerfile / deploy/Dockerfile / deploy/Caddyfile: white-label and
  deployment hygiene
- frontend/.../videoUtils.ts: white-label tweak
- config.example.yaml: replace the admin_password sample "admin123" with
  CHANGE_ME_STRONG_PASSWORD so the example never ships a weak default

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>
```

> **混合文件收口提醒**：若 Commit A 选择了"两 hunk 一起进 A"的替代路线，则 `deploy/config.example.yaml` 不再出现在 B（B 仅 4 个文件）；务必据实调整本块。

### Commit C —— 部署基建产物（未跟踪）

家族丙：未跟踪的部署脚本与 compose。

```bash
git add deploy/backup.sh
git add deploy/day0/                      # backup.sh, check.sh, start.sh, stop.sh,
                                          # windows_disable_lan_access.ps1, windows_enable_lan_access.ps1
git add deploy/docker-compose.wsl.yml
git add deploy/docker-compose.wsl.prod.yml
```

拟 commit message：

```text
chore(deploy): add day0 ops scripts and WSL compose files

- deploy/backup.sh and deploy/day0/* (backup/check/start/stop +
  windows LAN-access toggles)
- deploy/docker-compose.wsl.yml and .wsl.prod.yml

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>
```

> **提交前请人工确认**：这些脚本中**不得**含任何真实凭据/密钥/内网地址；如有，先用占位符（如 `<在此粘贴-openssl-rand-hex-32-的64位hex>` / `<REDACTED>`）替换后再提交。本任务**未读/未改**这些脚本内容，仅据 `git status` 列出文件名。

### Commit D —— 项目 / 参考文档（未跟踪，含本物证包）

家族丁：未跟踪的项目文档、参考方案、以及**本物证包四份 `docs/reviews/SUB2API_*_2026_06_15.md`**。

```bash
git add 00_START_HERE.md
git add 01_PROJECT_BASELINE.md
git add 02_CURRENT_REALITY_STATUS.md
git add AUDIT_REPORT.md
git add BOSS_DEPLOY_GUIDE.md
git add 真实Seedance2.0全链路接入方案.md
git add 真实Seedance2.0方案_对抗评审.md

# 本物证包四份（含本文件自身），人审通过后随 D 一起入库。
# ⚠️ 注意：.gitignore 第 135 行 `docs/*` 忽略整个 docs/ 子树，故 docs/reviews/ 下文件
#         默认【不会】被 git status 列为 untracked，普通 `git add` 会被忽略规则静默拦下。
#         必须用 `-f` 强制加入（本物证包既有的 docs/reviews/*REVIEW*.md 也是这样 force-add 进库的）。
git add -f docs/reviews/SUB2API_SIGNOFF_EVIDENCE_2026_06_15.md
git add -f docs/reviews/SUB2API_GIT_MAP_2026_06_15.md
git add -f docs/reviews/SUB2API_WORKTREE_CLEANUP_2026_06_15.md
git add -f docs/reviews/SUB2API_SMOKE_PREP_2026_06_15.md
```

拟 commit message：

```text
docs: add project briefs, seedance reference plans, and signoff evidence pack

- top-level project/onboarding docs (00_START_HERE, 01_PROJECT_BASELINE,
  02_CURRENT_REALITY_STATUS, AUDIT_REPORT, BOSS_DEPLOY_GUIDE)
- 真实Seedance2.0 全链路接入方案 / 对抗评审 reference docs
- docs/reviews/SUB2API_*_2026_06_15.md read-only signoff evidence pack
  (signoff / git-map / worktree-cleanup / smoke-prep)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>
```

> **未跟踪文档同样先人审**：确认无真实凭据/真实 API key/JWT/secret，凡密钥位置一律占位符，再提交。

---

## 4. 建议提交【顺序】与理由

| 顺序 | Commit | 内容 | 为什么这个位置 |
|---|---|---|---|
| 1 | **A** | Seedance blocker 安全修复（7 改 + 3 新） | **已双家族签字 GO** → 最该干净、可独立追溯 / revert / cherry-pick；先落 A 保证安全改动不被卫生/文档改动污染历史 |
| 2 | **B** | 白标 / 部署卫生（+ config 的 admin_password hunk） | 与 blocker 无关的代码/配置卫生，紧随 A |
| 3 | **C** | 部署基建产物（未跟踪脚本/compose） | 新增运维资产，独立成提交便于回溯 |
| 4 | **D** | 项目 / 参考文档 + 本物证包 | 纯文档，放最后；含本物证包四份，待人审定稿后入库 |

**关键约束**：

- **A 必须第一个**（因其已签字、是核心安全改动）。
- 四个 commit 都是**给用户本人执行**的方案；本任务**只成文，不执行、不 push**。
- push 时机由用户自行决定；当前**领先 origin/main 16**，push 前建议先在本工作树跑一遍 `go build ./...` / `go test ./...`（前序评审记录为全绿）做最后把关——**但这同样不在本任务范围内执行**。

---

## 5. 附：prunable worktree 处置（方案，勿执行）

除"工作树内文件 → commit 切分"外，本次收口还剩一项 **git 注册表层面**的清理：第 0 节列出的外部 worktree `…/02_source/sub2api-p0-9b-clean`（检出 `p0-9b-seedance-real-smoke` @ `7ac5335b`）被 git 标为 **`prunable`**——其工作目录在当前视图下不可达（典型成因：同一物理目录被 Windows `D:/…` 与 WSL `/mnt/d/…` 两种路径双视图登记）。worktree 拓扑全貌见关联文档②[《SUB2API_GIT_MAP》](./SUB2API_GIT_MAP_2026_06_15.md)第 3 节。

**处置方案（给用户本人 review 后自行执行；本任务一律不运行）**：

```bash
# 1) 先只读确认它确实 prunable、且没有未保存改动会丢失
git worktree list                                   # 确认该行仍带 prunable
git -C "/mnt/d/Codex创业任务/企业 API 管理后台项目/02_source/sub2api-p0-9b-clean" status  # 若目录仍可达，确认工作区干净

# 2a) 若确认无用：仅清理 git 对失效 worktree 的登记（不删任何真实文件）
git worktree prune -v                               # 清除不可达 worktree 的登记

# 2b) 若目录其实仍有用（只是被双视图误判）：修复登记而非 prune
git worktree repair
```

**安全前提（务必先核对再 prune）**：

- 该 worktree 占用的分支 `p0-9b-seedance-real-smoke` 指向 `7ac5335b`；**同一 sha 另有留底分支 `safety-sub2api-before-dirty-resolution-20260605`**（见②分支表）。即便 prune 掉 worktree 登记，`p0-9b-seedance-real-smoke` 分支引用仍在、`7ac5335b` 提交不会丢——回滚锚点完好。
- `git worktree prune` 只删 git 内部对**不可达** worktree 的登记，**不触碰任何被跟踪文件或提交**。
- 本任务**只成文、不执行** prune/repair；执行时机由用户决定。

---

## 6. 关联文档

本文件属"单一收口物证包"四份之一，互相交叉引用（相对链接，同目录 `docs/reviews/`）：

- [SUB2API_SIGNOFF_EVIDENCE_2026_06_15.md](./SUB2API_SIGNOFF_EVIDENCE_2026_06_15.md) —— 双家族签字物证总览。
- [SUB2API_GIT_MAP_2026_06_15.md](./SUB2API_GIT_MAP_2026_06_15.md) —— 实时 git 地图（分支 / worktree / 领先落后 / 变更家族归属）。
- **SUB2API_WORKTREE_CLEANUP_2026_06_15.md（本文）** —— 工作树收口 = commit 切分方案（§1–§4）+ prunable worktree 处置方案（§5）；均**只方案、不执行**。
- [SUB2API_SMOKE_PREP_2026_06_15.md](./SUB2API_SMOKE_PREP_2026_06_15.md) —— 真实冒烟【准备包】（仅草案，未执行过任何真实冒烟）。
- 权威前序评审：[SUB2API_SEEDANCE_BLOCKER_FIX_REVIEW_2026_06_15.md](./SUB2API_SEEDANCE_BLOCKER_FIX_REVIEW_2026_06_15.md)。

---

> ## ⚠️ 现在不要执行
> 本文件中所有 `git add` / `git commit` / 任何会改变 git 状态的命令都是**方案草案**，供你本人 review 后**自行**执行。撰写本文时：**未** `git add`、**未** `git commit`、**未** `git push`、**未** `restore/checkout/stash/reset`。仓库保持【只读】。执行前请逐条人审混合文件 hunk 与未跟踪文件内容，确保零真实凭据。
