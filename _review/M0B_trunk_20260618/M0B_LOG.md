# M0-B 收编立主线 · 执行日志

> 日期：2026-06-18 ｜ 执行：Claude Code（边做边教）｜ 决策：ff 全收 D，不 squash、不 cherry-pick
> 全程：零 push / 零真实调用 / 零真门开启 / 不删任何分支 / main 未动。`_review/` 不入 git。

---

## 阶段 0 · 物证抢救（只读 + GATE 1）

### 工作树清单
```
…/02_source/sub2api                       40e83bf4 [night-run/20260618-D-c1-alive]   ← 主工作树
/mnt/d/…/02_source/sub2api-p0-9b-clean    7ac5335b [p0-9b-seedance-real-smoke] prunable
…/.claude/worktrees/nostalgic-chatterjee  69f648e2 [claude/nostalgic-chatterjee-703296]
…/.claude/worktrees/objective-goldstine   69f648e2 [claude/objective-goldstine-b9bf89]   ← M0-B 执行树
```

### 主工作树（检出 D）状态 —— 不干净（非出片物证）
```
modified:   .gitignore                                  # 仅加一行 _review_packages/
Untracked:  deploy/docker-compose.b1-emptybrake.yml     # 21行 B-1预算门"空踩刹车"测试配置，无密钥
stash: （空）
```
- 两项均**非**真实出片 mp4 物证，而是测试脚手架 + gitignore 小改。
- ff 收编**不触碰**主工作树，两项安然留在原地。
- 新文件无凭据：用 `${VIDEO_GATEWAY_ENCRYPTION_KEY:?}` 占位引用，真实本地对称密钥在 git 忽略的 `deploy/.env`（非供应商 key）。

### WSL p0-9b-clean 状态 —— Windows 侧核查不到
- 目录在硬盘存在，但 `.git` 链路失效（`fatal: not a git repository`）+ 本就标 `prunable`。
- 浅扫无明显 mp4/output/日志物证（但≠git 干净）。
- 权威 `git status` 需学者在 WSL 自查：`cd /mnt/d/…/sub2api-p0-9b-clean && git status`。

### GATE 1 拍板（学者）
1. 主工作树两项 → **原地保留 + 复制归档，继续**。已复制到 `stage0_rescued/`（原件未动）。
2. p0-9b-clean → **放行，学者稍后在 WSL 自查**（M0-B 全程不碰它）。

---

## 阶段 1 · 建主线 + ff 收编

| 步 | 命令 | 结果 |
|---|---|---|
| 1.1 | `git status`（执行树） | 跟踪树干净，仅 `_review/` 未跟踪 ✅ |
| 1.2 | `git switch -c wujie/trunk main` | `Switched to a new branch 'wujie/trunk'`；HEAD=`69f648e2`(=main) ✅ |
| 1.3 | `git merge --ff-only night-run/20260618-D-c1-alive` | **`Updating 69f648e2..40e83bf4` / `Fast-forward`**；**126 文件改动，+15417 / −1090**；HEAD=`40e83bf4` ✅ |
| 1.4 | `git rev-parse main` | 仍 `69f648e2`（main 未动）✅ |

收编后：`wujie/trunk` 领先 `main` = **25** commit（全部纳入）。

---

## 阶段 2 · 验证（亲自复跑，修者≠审者）

| # | 命令 | 结果 |
|---|---|---|
| 1 | `go build ./...` | 退出 `0`（无报错）✅ |
| 2 | `go vet ./...` | 退出 `0`（无报错）✅ |
| 3 | `go test ./internal/service/...` | `ok …/internal/service 47.820s` + `ok …/openai_ws_v2 4.231s` ✅ |
| 4 | `go test -run C1 ./internal/server/routes/...` | `ok …/internal/server/routes 3.737s`（C1 进程内活体）✅ |

环境：`go1.26.3 windows/amd64`，GOMODCACHE 本机已缓存（未额外联网安装）。
realsmoke 真实冒烟测试带 `//go:build realsmoke` 标记，默认 `go test` 不编译 → **零真实调用，¥0**。

**结论：全绿，无 FAIL。**

---

## 阶段 3 · 旧分支归档（只 tag，不删）

```
git tag archive/pre-trunk-D     night-run/20260618-D-c1-alive
git tag archive/pre-trunk-p0-9b p0-9b-seedance-real-smoke
```
| 标签 | → 快照 |
|---|---|
| `archive/pre-trunk-D` | `40e83bf4` |
| `archive/pre-trunk-p0-9b` | `7ac5335b` |

旧分支全部**健在**（`night-run/20260618-D-c1-alive`、`p0-9b-seedance-real-smoke` 等一条未删）。

---

## 收尾快照

```
$ git log --oneline -5    （wujie/trunk）
40e83bf4 test(video-gateway): 阶段0+1 定标准+C1进程内活体(keyless ¥0, 未开真实门)
486f52fe docs(night-run): 00_黎明总结……
47cf1146 test(video-gateway): 阶段C C1骨架契约测试……
831e9c98 fix(video-gateway): 阶段B复审闭环……
1be53de3 fix(video-gateway): 阶段B Seedance契约修复 B1比例字段……

wujie/trunk = 40e83bf4 ｜ main = 69f648e2 ｜ 领先 = 25
$ git status --porcelain → 仅 `?? _review/`（跟踪树 = D 内容，无额外改动）
```

---

## 验收对照（学者侧）

| 标准 | 实测 | ✅ |
|---|---|---|
| `wujie/trunk` HEAD = `40e83bf4`（含全部 25 commit） | 40e83bf4，领先 main 25 | ✅ |
| `main` 仍 = `69f648e2`（未动） | 69f648e2 | ✅ |
| 验证全绿（或失败如实报告、没偷偷修） | build/vet/service/C1 全绿 | ✅ |
| 旧分支都还在 + 有 `archive/` 标签 | 旧分支健在 + 2 个 archive tag | ✅ |
| 零 push / 零真实调用 / 零真门开启 | 全程未触 push/env smoke 开关 | ✅ |

---

## 下一锹（守主线提醒）
M0-B = 把散砖码成一条干净主线（止血定基的**收尾**，不是往前走）。
真正往前的下一步 = **M1 开采集口**：那个被整晚工程绕开的 `backend/ent/schema/usage_log.go` 命门（当前只存计费元数据、无 prompt/无结果）。收编完别停太久，下一锹挖采集口。
