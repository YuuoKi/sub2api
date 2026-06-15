<!-- 单一收口物证 / 2026-06-15 / sub2api @ phase-3.8.2-overnight-readiness / 零真实调用·零真实凭据·未 commit 未 push -->

# SUB2API · Seedance 2.0 Blocker 修复 · 签字物证（Sign-off Evidence）

- **日期**：2026-06-15
- **仓库 / 分支**：`02_source/sub2api` @ `phase-3.8.2-overnight-readiness`（HEAD `4dd599af`；未新建分支，未 push，未 `git add`）
- **本文件定位**：四份"只读收口物证包"中的【①签字物证】。目标只有一个——用**逐字、未截断**的一手物证坐实三件事：
  1. **当前工作树真实通过**全量 `go test ./...`（不靠缓存，全新非缓存运行 exit 0）。
  2. 工作树改动**未 commit、未 add、未 push**（喂给 codex 评审的 diff == 本工作树真实内容）。
  3. codex（GPT 家族）与 Claude **双家族签字均已 GO**，唯一留给冒烟的是 Ark 字段名核对。
- **结论(一句话)**：工作树 = 双家族评审通过的状态；非缓存 `go test ./...` exit 0、零 FAIL/零 panic；改动落后 origin/main 0、领先 16 且全未 staged → 未 push、未 commit；前序评审喂给 codex 的 diff 就是本工作树本身。
- **铁律遵守声明**：
  - ✅ 本文件**未执行任何真实冒烟**——它只是【物证/准备包】，不声称跑过真实 Ark/Seedance 调用。
  - ✅ 全程**零真实上游调用**、**零真实 API key / token / JWT / secret**（凡涉真实凭据处一律占位符）。
  - ✅ 仓库**只读**：除创建本文件外未触碰任何源码，未 `git add / commit / push / restore / checkout / stash / reset`。
- **house style 基准**：本文件与 `SUB2API_SEEDANCE_BLOCKER_FIX_REVIEW_2026_06_15.md`（权威前序评审）对齐。该评审第 4 节给的是**节选/包装**的 `go test`；**本文件用下方第 1 节的逐字全文替换它**，作为"当前工作树真实通过"的权威物证。

---

## 1. `go test ./...` 逐字全文（核心物证）

> 本节是本文件的核心。两次运行**都 exit 0、零 FAIL、零 panic**，所有包均 `ok` / `[no test files]`。
> **非缓存运行（1.1）证明"当前工作树真实通过"，而非只靠测试缓存**；缓存命中运行（1.2）作为对照。

### 1.1 全新【非缓存】运行 —— 权威物证（exit 0）

- **命令**：`cd backend && go test ./... -count=1 2>&1`
- **性质**：`-count=1` 强制全部重跑、**不命中任何缓存**；逐行可见编译耗时（如 `internal/service 43.375s`、`internal/handler 21.906s`），证明各包真实重新构建并执行。
- **结果**：全部包 `ok` 或 `[no test files]`，**无一 FAIL / 无一 panic**，末行 `GO_TEST_FRESH_EXIT=0`。
- **日志逐字全文（92 行，原样粘贴，未截断、未省略）**：

```
?   	github.com/Wei-Shaw/sub2api/cmd/jwtgen	[no test files]
ok  	github.com/Wei-Shaw/sub2api/cmd/server	0.553s
?   	github.com/Wei-Shaw/sub2api/ent	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/account	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/accountgroup	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/announcement	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/announcementread	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/apikey	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/authidentity	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/authidentitychannel	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/channelmonitor	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/channelmonitordailyrollup	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/channelmonitorhistory	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/channelmonitorrequesttemplate	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/enttest	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/errorpassthroughrule	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/group	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/hook	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/idempotencyrecord	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/identityadoptiondecision	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/intercept	[no test files]
ok  	github.com/Wei-Shaw/sub2api/ent/migrate	1.741s
?   	github.com/Wei-Shaw/sub2api/ent/paymentauditlog	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/paymentorder	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/paymentproviderinstance	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/pendingauthsession	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/predicate	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/promocode	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/promocodeusage	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/proxy	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/redeemcode	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/runtime	[no test files]
ok  	github.com/Wei-Shaw/sub2api/ent/schema	4.628s
?   	github.com/Wei-Shaw/sub2api/ent/schema/mixins	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/securitysecret	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/setting	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/subscriptionplan	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/tlsfingerprintprofile	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/usagecleanuptask	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/usagelog	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/user	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/userallowedgroup	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/userattributedefinition	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/userattributevalue	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/usersubscription	[no test files]
ok  	github.com/Wei-Shaw/sub2api/internal/config	2.159s
ok  	github.com/Wei-Shaw/sub2api/internal/domain	0.749s
ok  	github.com/Wei-Shaw/sub2api/internal/handler	21.906s
ok  	github.com/Wei-Shaw/sub2api/internal/handler/admin	0.786s
ok  	github.com/Wei-Shaw/sub2api/internal/handler/dto	0.480s
ok  	github.com/Wei-Shaw/sub2api/internal/middleware	1.452s
?   	github.com/Wei-Shaw/sub2api/internal/model	[no test files]
ok  	github.com/Wei-Shaw/sub2api/internal/payment	0.994s
ok  	github.com/Wei-Shaw/sub2api/internal/payment/provider	0.387s
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/antigravity	0.736s
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/apicompat	0.605s
?   	github.com/Wei-Shaw/sub2api/internal/pkg/claude	[no test files]
?   	github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey	[no test files]
?   	github.com/Wei-Shaw/sub2api/internal/pkg/errors	[no test files]
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/gemini	0.500s
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/geminicli	0.665s
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/googleapi	0.790s
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/httpclient	0.724s
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/httputil	0.748s
?   	github.com/Wei-Shaw/sub2api/internal/pkg/ip	[no test files]
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/logger	1.181s
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/oauth	0.574s
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/openai	0.538s
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat	0.579s
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/pagination	0.566s
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/proxyurl	0.519s
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/proxyutil	0.584s
?   	github.com/Wei-Shaw/sub2api/internal/pkg/response	[no test files]
?   	github.com/Wei-Shaw/sub2api/internal/pkg/sysutil	[no test files]
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/timezone	0.495s
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint	0.589s [no tests to run]
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/usagestats	0.466s
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/websearch	1.026s
ok  	github.com/Wei-Shaw/sub2api/internal/repository	1.758s
?   	github.com/Wei-Shaw/sub2api/internal/server	[no test files]
ok  	github.com/Wei-Shaw/sub2api/internal/server/middleware	0.432s
ok  	github.com/Wei-Shaw/sub2api/internal/server/routes	1.880s
ok  	github.com/Wei-Shaw/sub2api/internal/service	43.375s
ok  	github.com/Wei-Shaw/sub2api/internal/service/openai_ws_v2	3.570s
ok  	github.com/Wei-Shaw/sub2api/internal/setup	0.340s
?   	github.com/Wei-Shaw/sub2api/internal/util/httputil	[no test files]
ok  	github.com/Wei-Shaw/sub2api/internal/util/logredact	0.435s
ok  	github.com/Wei-Shaw/sub2api/internal/util/responseheaders	0.579s
ok  	github.com/Wei-Shaw/sub2api/internal/util/urlvalidator	0.438s
?   	github.com/Wei-Shaw/sub2api/internal/web	[no test files]
ok  	github.com/Wei-Shaw/sub2api/migrations	0.557s
GO_TEST_FRESH_EXIT=0
```

### 1.2 缓存命中运行 —— 对照物证（exit 0）

- **命令**：`go test ./...`（同一工作树，未加 `-count=1`，故复用上一次结果缓存）。
- **结果**：与 1.1 一致——全部 `ok` / `[no test files]`，零 FAIL，凡有测试的包标 `(cached)`，末行 `GO_TEST_EXIT=0`。两次 exit code 同为 0，互相印证。
- **日志逐字全文（原样粘贴，未截断、未省略）**：

```
?   	github.com/Wei-Shaw/sub2api/cmd/jwtgen	[no test files]
ok  	github.com/Wei-Shaw/sub2api/cmd/server	(cached)
?   	github.com/Wei-Shaw/sub2api/ent	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/account	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/accountgroup	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/announcement	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/announcementread	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/apikey	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/authidentity	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/authidentitychannel	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/channelmonitor	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/channelmonitordailyrollup	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/channelmonitorhistory	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/channelmonitorrequesttemplate	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/enttest	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/errorpassthroughrule	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/group	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/hook	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/idempotencyrecord	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/identityadoptiondecision	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/intercept	[no test files]
ok  	github.com/Wei-Shaw/sub2api/ent/migrate	(cached)
?   	github.com/Wei-Shaw/sub2api/ent/paymentauditlog	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/paymentorder	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/paymentproviderinstance	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/pendingauthsession	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/predicate	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/promocode	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/promocodeusage	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/proxy	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/redeemcode	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/runtime	[no test files]
ok  	github.com/Wei-Shaw/sub2api/ent/schema	(cached)
?   	github.com/Wei-Shaw/sub2api/ent/schema/mixins	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/securitysecret	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/setting	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/subscriptionplan	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/tlsfingerprintprofile	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/usagecleanuptask	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/usagelog	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/user	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/userallowedgroup	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/userattributedefinition	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/userattributevalue	[no test files]
?   	github.com/Wei-Shaw/sub2api/ent/usersubscription	[no test files]
ok  	github.com/Wei-Shaw/sub2api/internal/config	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/domain	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/handler	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/handler/admin	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/handler/dto	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/middleware	(cached)
?   	github.com/Wei-Shaw/sub2api/internal/model	[no test files]
ok  	github.com/Wei-Shaw/sub2api/internal/payment	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/payment/provider	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/antigravity	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/apicompat	(cached)
?   	github.com/Wei-Shaw/sub2api/internal/pkg/claude	[no test files]
?   	github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey	[no test files]
?   	github.com/Wei-Shaw/sub2api/internal/pkg/errors	[no test files]
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/gemini	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/geminicli	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/googleapi	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/httpclient	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/httputil	(cached)
?   	github.com/Wei-Shaw/sub2api/internal/pkg/ip	[no test files]
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/logger	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/oauth	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/openai	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/pagination	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/proxyurl	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/proxyutil	(cached)
?   	github.com/Wei-Shaw/sub2api/internal/pkg/response	[no test files]
?   	github.com/Wei-Shaw/sub2api/internal/pkg/sysutil	[no test files]
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/timezone	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint	(cached) [no tests to run]
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/usagestats	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/pkg/websearch	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/repository	(cached)
?   	github.com/Wei-Shaw/sub2api/internal/server	[no test files]
ok  	github.com/Wei-Shaw/sub2api/internal/server/middleware	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/server/routes	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/service	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/service/openai_ws_v2	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/setup	(cached)
?   	github.com/Wei-Shaw/sub2api/internal/util/httputil	[no test files]
ok  	github.com/Wei-Shaw/sub2api/internal/util/logredact	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/util/responseheaders	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/util/urlvalidator	(cached)
?   	github.com/Wei-Shaw/sub2api/internal/web	[no test files]
ok  	github.com/Wei-Shaw/sub2api/migrations	(cached)
GO_TEST_EXIT=0
```

### 1.3 两次运行的结论

- 两次运行**均 exit 0**（`GO_TEST_FRESH_EXIT=0` 与 `GO_TEST_EXIT=0`）。
- **非缓存运行（1.1）是权威物证**：它强制全部重新编译并执行，证明的是"当前工作树（含三家族改动）真实通过测试"，而非"测试缓存恰好命中旧结果"。
- 凡有测试的包全部 `ok`、其余 `[no test files]`，**无一 FAIL、无一 panic**。13 个新增安全测试函数所在的 `internal/service` 包在 1.1 中以 `43.375s` 真实重跑通过（函数清单见前序评审第 4 节与本文件第 5 节）。

---

## 2. `git status` 逐字全文

> 证明：11 个被跟踪文件**改动均未 staged**（落在 "Changes not staged for commit" 段）；另有未跟踪文件群（家族丙）。无任何文件位于 "Changes to be committed" → **未 `git add`、未 commit**。

```
On branch phase-3.8.2-overnight-readiness
Changes not staged for commit:
  (use "git add <file>..." to update what will be committed)
  (use "git restore <file>..." to discard changes in working directory)
	modified:   .gitignore
	modified:   Dockerfile
	modified:   backend/internal/repository/video_key_encryptor.go
	modified:   backend/internal/server/routes/api_key_video_gateway_test.go
	modified:   backend/internal/service/video_gateway_adapter.go
	modified:   backend/internal/service/video_gateway_service.go
	modified:   backend/internal/service/video_gateway_types.go
	modified:   deploy/Caddyfile
	modified:   deploy/Dockerfile
	modified:   deploy/config.example.yaml
	modified:   frontend/src/views/admin/video/videoUtils.ts

Untracked files:
  (use "git add <file>..." to include in what will be committed)
	00_START_HERE.md
	01_PROJECT_BASELINE.md
	02_CURRENT_REALITY_STATUS.md
	AUDIT_REPORT.md
	BOSS_DEPLOY_GUIDE.md
	backend/internal/service/video_gateway_redact.go
	backend/internal/service/video_gateway_security_test.go
	backend/internal/service/video_gateway_ssrf.go
	deploy/backup.sh
	deploy/day0/
	deploy/docker-compose.wsl.prod.yml
	deploy/docker-compose.wsl.yml
	"\347\234\237\345\256\236Seedance2.0\345\205\250\351\223\276\350\267\257\346\216\245\345\205\245\346\226\271\346\241\210.md"
	"\347\234\237\345\256\236Seedance2.0\346\226\271\346\241\210_\345\257\271\346\212\227\350\257\204\345\256\241.md"

no changes added to commit (use "git add" and/or "git commit -a")
```

> 注：上方两个 `\347...` 文件名是 git 对中文文件名的八进制转义显示，对应 `真实Seedance2.0全链路接入方案.md` 与 `真实Seedance2.0方案_对抗评审.md`（属家族丙未跟踪文档），非乱码。

---

## 3. `git log --oneline -5` 逐字全文

> 证明：HEAD = `4dd599af`，且最近 5 个提交**不含**本次三家族工作树改动（改动仍未 commit，见第 2、4 节）。

```
4dd599af feat(video): prepare gated seedance tiny real trial
3351338d feat: add api-key video mock gateway for qcanvas
4143673f docs: record sub2api dirty resolution checkpoint
c4e2337d wip: checkpoint sub2api day0 video gateway and white-label worktree
7ac5335b feat: guard seedance single smoke gateway
```

---

## 4. 签字归因（未 commit · 未 add · 未 push）

本节把第 2、3 节物证与 push 状态量化，逐条坐实"工作树就是评审对象"。

### 4.1 未 push —— HEAD 领先 origin/main 16、落后 0

- **命令与输出**：`git rev-list --left-right --count origin/main...HEAD` → `0	16`
  - 左 `0` = origin/main 相对 HEAD 多出的提交数（**落后 0**）。
  - 右 `16` = HEAD 相对 origin/main 多出的提交数（**领先 16**）。
- **判定**：HEAD（`4dd599af`）领先远端 16 个提交且零落后 → 这 16 个提交**尚未推送到 `origin`**。remote 经核实为 `origin = https://github.com/Wei-Shaw/sub2api.git`（fetch+push 同一地址）。
- **对照 `git branch -vv`**（逐字）：

```
  main                                            69f648e2 [origin/main: ahead 5] chore: lock phase 3.8.1 usability acceptance
+ p0-9b-seedance-real-smoke                       7ac5335b (/mnt/d/Codex创业任务/企业 API 管理后台项目/02_source/sub2api-p0-9b-clean) feat: guard seedance single smoke gateway
* phase-3.8.2-overnight-readiness                 4dd599af feat(video): prepare gated seedance tiny real trial
  safety-sub2api-before-dirty-resolution-20260605 7ac5335b feat: guard seedance single smoke gateway
```

  - 当前分支 `* phase-3.8.2-overnight-readiness @ 4dd599af` **无上游跟踪标记**（无 `[origin/...]`），进一步印证本分支未 push。
  - `main` 标 `[origin/main: ahead 5]`、`p0-9b-seedance-real-smoke` 标 `+`（被另一 worktree 占用）——均非本次签字对象，仅作完整地图（详见关联文档②《GIT_MAP》、③《WORKTREE_CLEANUP》）。

### 4.2 未 commit / 未 add —— 改动全部 unstaged

- 第 2 节 `git status` 中 11 个被跟踪文件全部落在 **"Changes not staged for commit"** 段；末行明确 `no changes added to commit`。等价于 `git status --porcelain=v1` 中这些条目**第二列（暂存区列）为空**（形如 ` M <file>`，前导空格 = 未 staged）。
- 结论：**未 `git add`、未 commit**。家族丙的新文件停在 "Untracked files" 段，同样未纳入索引。
- `git diff --stat`（被跟踪文件）规模：**11 文件 +154 / -51**（会附带两条 LF→CRLF 的 git 无害 warning，如实保留，不影响内容）。

### 4.3 喂给 codex 的 diff == 本工作树真实内容（逐行权威源）

- 前序评审（codex/Claude 双家族）评审的对象，**逐行权威源就是本工作树本身**，即：
  `git --no-pager diff`（11 个被跟踪文件的未暂存改动） **＋** 3 个未跟踪新文件（`backend/internal/service/video_gateway_redact.go`、`video_gateway_ssrf.go`、`video_gateway_security_test.go`）。
- 因为改动**未 commit、未 add、未 push**（4.1/4.2），所以"被评审的 diff"与"当前磁盘上的工作树"之间**没有任何 commit/push 间隙**——评审看到的就是冒烟将要运行的字节。这正是把工作树作为签字对象（而非某个已推送 commit）的核心保证。
- 注意：工作树改动含**三个变更家族**，其中只有**家族甲（Seedance blocker 安全修复：7 改 + 3 新）**是本签字物证的安全范围；家族乙（白标/部署卫生）、家族丙（未跟踪文档与部署产物）与 blocker 无关，其边界划分见关联文档②《GIT_MAP》。`deploy/config.example.yaml` 是**混合文件**（同时含家族甲的 `encryption_key` 必填文案段 + 家族乙的 `admin_password: "admin123"` → `"CHANGE_ME_STRONG_PASSWORD"` hunk）。

---

## 5. 双家族签字回执

> 详尽的逐轮裁决全文见权威前序评审 `SUB2API_SEEDANCE_BLOCKER_FIX_REVIEW_2026_06_15.md` 第 5、6 节；本节仅给签字回执，不重复全文。

| 家族 | 评审方式 | 最终裁决 | 指向 |
|---|---|---|---|
| **codex（GPT 家族）** | 只读评审（`codex -s read-only -a never exec`，内容内联、零写盘、零真实调用）；用户本人 `codex login --device-auth` 登录后由助手执行 | **GO-WITH-FOLLOWUPS** | 前序评审 §5 |
| **Claude** | 4 镜头对抗评审 → skeptic 复核 → 综合（两轮 + 机械修复 #1–#3） | **GO** | 前序评审 §6 |

- **codex 轨迹**：第一轮 **NO-GO**，抓到 Claude 两轮都漏掉的 3 个真问题（HIGH URL 解析器差异 `\@` 绕过、MED 审计日志 0600/fail-closed、MED `PlainAPIKey` 经 `%#v`/JSON 泄露）→ 3 项全修 → 第二轮复查 **GO-WITH-FOLLOWUPS**（含 follow-up (a) Ark 字段名、(b) `f.Chmod` 失败返回 error；(b) 已立即闭合）。
- **Claude 轨迹**：6 项 round-1 confirmed + 3 项 round-2 新 finding + F2 测试缺口**全部已修并由确定性测试覆盖**；`go build` / `go vet` / `go test ./...` 全绿（见本文件第 1 节）。
- **双家族签字达成**：codex 与 Claude **均 GO**。
- **唯一留给冒烟的项**：**Ark 真实响应字段名核对**（`id`、`status`、`content.video_url`、`duration`，以及 `resolution` / `aspect_ratio` 的确切字段名）。代码中已显式注释 **UNVERIFIED**，留待首条真实冒烟核对。这是冒烟环节、非本仓库代码任务。
- **本文件不声称跑过真实冒烟**：上述签字是对"Go 侧 blocker 修复 + 全绿测试 + 未 push 工作树"的评审签字；真实 Ark/Seedance 调用**未发生**，相关准备见关联文档④《SMOKE_PREP》。

### 5.1 13 个安全测试函数（`internal/service` 包，随第 1.1 节非缓存运行通过）

`TestRedactVideoUpstreamSecrets`、`TestAppendRedactedVideoEvent`、`TestValidateExternalVideoURL`、`TestValidateExternalVideoURLAllowlist`、`TestVideoProviderAccountRedactsPlainKey`、`TestSeedanceCreateSendsDurationAndAuditsRedacted`、`TestSeedanceCreateRedactsUpstreamErrorBody`、`TestSeedancePollRejectsUnsafeResultURL`、`TestSeedancePollAcceptsSafeResultURL`、`TestSeedanceCreateRedactsBusinessErrorMessage`、`TestSeedancePollRedactsBusinessErrorMessage`、`TestSeedancePollRedactsUpstreamErrorBody`、`TestSeedanceCreateRejectsUnsafeReferenceURL`。

> 这些测试全程使用本地 `httptest` 服务器 + dummy key `test-key-not-real`，**零真实上游调用、零真实凭据**。

---

## 6. 关联文档

本文件为"只读收口物证包"四份之①，与以下三份交叉引用（均位于同目录 `docs/reviews/`）：

- 权威前序评审：[SUB2API_SEEDANCE_BLOCKER_FIX_REVIEW_2026_06_15.md](./SUB2API_SEEDANCE_BLOCKER_FIX_REVIEW_2026_06_15.md) —— 三 blocker 修复全文、双家族逐轮裁决、QCanvas 23 条契约清单。
- ② Git 地图：[SUB2API_GIT_MAP_2026_06_15.md](./SUB2API_GIT_MAP_2026_06_15.md) —— 三变更家族归类、分支/远端/领先落后关系。
- ③ Worktree 收口：[SUB2API_WORKTREE_CLEANUP_2026_06_15.md](./SUB2API_WORKTREE_CLEANUP_2026_06_15.md) —— commit 切分方案（家族甲 7 改+3 新单独提交）+ `sub2api-p0-9b-clean`（prunable）worktree 处置方案；均**只方案、不执行**。
- ④ 冒烟准备包：[SUB2API_SMOKE_PREP_2026_06_15.md](./SUB2API_SMOKE_PREP_2026_06_15.md) —— 环境变量/闸门、Ark 字段名核对清单、冒烟后收口（仍为准备包，未执行真实冒烟）。
