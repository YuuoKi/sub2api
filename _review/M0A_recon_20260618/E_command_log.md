# E · 只读命令执行日志 — M0-A

> 全部为只读命令（show/diff/log/rev-list/ls-tree/merge-base/for-each-ref/grep/status）。
> **无任何写副作用命令**（无 merge/cherry-pick/rebase/commit/checkout/reset/stash/push）。
> 无凭据输出。下方为原样命令 + 关键输出摘录。

---

## 1. 基线（步骤 1）

```bash
$ git remote -v
origin  https://github.com/Wei-Shaw/sub2api.git (fetch)
origin  https://github.com/Wei-Shaw/sub2api.git (push)

$ git rev-parse --abbrev-ref HEAD ; git rev-parse HEAD
claude/objective-goldstine-b9bf89
69f648e20e6f194b08fb120c215e96e88b30e84e

$ git rev-list --left-right --count origin/main...main
590	5

$ git log origin/main..main --oneline
69f648e2 chore: lock phase 3.8.1 usability acceptance
58f79542 chore: lock phase 3.8 final gateway baseline
43003a00 feat(video-gateway): add demo-safe white label mode
bade34de chore(video-gateway): checkpoint P0.5 localized QA flow
4c5de849 feat(video-gateway): add P0 mock provider workflow
# （另跑 git log origin/main..main --stat 取每 commit 改动文件，见 A 表 §1）
```

## 2. 全分支清点（步骤 2）

```bash
$ git branch -a
# 本地：claude/nostalgic-chatterjee-703296, claude/objective-goldstine-b9bf89(*),
#   feat/phase1-eng-baseline-20260616, feat/phase2a-pre-arming-20260616,
#   feat/phase2b-real-smoke-20260616, main,
#   night-run/20260618-{A-truth,B-contract,C-skeleton,D-c1-alive(+)},
#   p0-9b-seedance-real-smoke(+), phase-3.8.2-overnight-readiness,
#   safety-sub2api-before-dirty-resolution-20260605
# 远端：origin/{HEAD,cla-signatures,dev,feat/api-key-ip-restriction,main,preview,preview-dev,revert-114-feature/atomic-scheduling}

$ git for-each-ref --sort=-committerdate --format='%(refname:short) | %(committerdate:short) | %(objectname:short) | %(subject)' refs/heads refs/remotes
# （结果见 A 表 §2.1，含日期/tip/subject）

# 逐分支领先/落后 main：
$ for b in <每条本地分支>; do
    echo "$b  ahead=$(git rev-list --count main..$b)  behind=$(git rev-list --count $b..main)"; done
# feat/phase1=17/0  phase2a=18/0  phase2b=19/0
# night-run A=20/0 B=22/0 C=24/0 D=25/0
# p0-9b=7/0  phase-3.8.2=16/0  safety=7/0  claude/*=0/0

# 各分支与 main 的 merge-base：
$ for b in <…>; do git merge-base main "$b"; done
# 全部 = 69f648e2（main HEAD）→ 全部从 main 切出、落后 0
```

## 3. 线性祖先验证（步骤 3）

```bash
$ check(){ git merge-base --is-ancestor "$1" "$2" && echo "YES $1 anc-of $2" || echo "no"; }
$ check p0-9b…           phase-3.8.2…         # YES
$ check phase-3.8.2…     feat/phase1…         # YES
$ check feat/phase1…     feat/phase2a…        # YES
$ check feat/phase2a…    feat/phase2b…        # YES
$ check feat/phase2b…    night-run/A          # YES
$ check night-run/A      night-run/B          # YES
$ check night-run/B      night-run/C          # YES
$ check night-run/C      night-run/D          # YES
$ check p0-9b…           night-run/D          # YES（端到端超集确认）
# safety <-> p0-9b 互为祖先（同一 commit 7ac5335b）

$ git log --graph --oneline main..night-run/20260618-D-c1-alive
# 输出全为 "*"，零 "|/" 分叉 → 严格线性 25 commit（图见 A 表 §3）

$ git log --reverse --pretty='%h %ad %s' --date=short --shortstat main..night-run/20260618-D-c1-alive
# 25 commit 逐条 insertions/deletions（见 B 表）
```

## 4. 命门文件定位 + 只读查看

```bash
$ git ls-tree -r --name-only main | grep -iE "video_gateway_adapter.go|usage_log.go"
backend/internal/service/video_gateway_adapter.go
backend/ent/schema/usage_log.go
backend/internal/service/usage_log.go

# G7/G8：禁用开关 + aspect_ratio 行号
$ git show main:backend/internal/service/video_gateway_adapter.go | grep -nE "REAL_(CALL|POLL)_DISABLED|aspect_ratio"
110:		"aspect_ratio":        task.AspectRatio,        # mock BuildCreatePayload
134:	... "SEEDANCE_REAL_CALL_DISABLED" ...               # G7 ✅ 精确
144:	... "SEEDANCE_REAL_POLL_DISABLED" ...
180:		"aspect_ratio":    task.AspectRatio,            # seedance（真 bug 处）
198:	... "KLING_REAL_CALL_DISABLED" ...                  # G7 实际(≠189)
208:	... "KLING_REAL_POLL_DISABLED" ...
238:		"aspect_ratio":    task.AspectRatio,            # kling

# G9：采集口 schema 字段（只读，无密钥）
$ git show main:backend/ent/schema/usage_log.go | grep -nE "field\.|Fields|prompt|result|hash|sha"
# 仅 id/计费/token/ip/ua/image/timing 字段；prompt|result|hash|sha 零命中

# G5：ratio 修复 commit
$ git show 1be53de3 --stat
$ git show 1be53de3 -- backend/internal/service/video_gateway_adapter.go | grep -nE "^[-+].*(aspect_ratio|ratio)"
# -	payload["aspect_ratio"]=task.AspectRatio
# +	payload["ratio"]=normalizeSeedanceRatio(...)   等
```

## 5. tag / worktree / 重复确认

```bash
$ git for-each-ref --format='%(refname:short) -> %(objectname:short) %(subject)' refs/tags | grep -E "checkpoint|phase-4b4"
checkpoint/p0-9a-seedance-single-smoke-gate -> 7ac5335b
checkpoint/sub2api-apikey-video-mock-gateway-20260609 -> 3351338d
checkpoint/sub2api-seedance-tiny-real-c0-prep-20260609 -> 4dd599af
phase-4b4-internal-pilot-ready -> f3f538e8 (附注 tag，对象指向 commit 0774feeb)

$ git worktree list
…/02_source/sub2api                       40e83bf4 [night-run/20260618-D-c1-alive]
/mnt/d/…/sub2api-p0-9b-clean              7ac5335b [p0-9b-seedance-real-smoke] prunable
…/.claude/worktrees/nostalgic-chatterjee 69f648e2 [claude/nostalgic-chatterjee-703296]
…/.claude/worktrees/objective-goldstine  69f648e2 [claude/objective-goldstine-b9bf89]  ← 本侦察工作树

$ git rev-parse safety-sub2api-before-dirty-resolution-20260605 p0-9b-seedance-real-smoke
7ac5335b1ac77780bfe9119b33ee8f7aac87b02e
7ac5335b1ac77780bfe9119b33ee8f7aac87b02e   # 完全相同 → safety 是 p0-9b 的纯备份
```

## 6. 深度逐 commit 测绘（并行只读子代理）

通过只读工作流派出 7 个子代理，每个只跑 `git show/diff/log/ls-tree/grep`（同样禁写）：
5 个分段映射（A:`69f648e2..7ac5335b` / B:`7ac5335b..bcb5fd32` / C:`bcb5fd32..7b78f9ca` / D:`7b78f9ca..c35049a4` / E:`c35049a4..40e83bf4`）+ 1 个全 25-commit 凭据扫描 + 1 个 G1–G9 独立对抗复核。
逐 commit 结论汇入 [B_disposition.md](./B_disposition.md)，凭据扫描与复核汇入 [F_recheck.md](./F_recheck.md)。

## 7. 工作树洁净证明（任务书 §10 / 验收）

```bash
$ git status --porcelain=v1        # 侦察开始时
（空）                              # 跟踪树纯净

$ git status --porcelain=v1        # 侦察结束时（详见本目录末尾或最终回传）
?? _review/                        # 唯一变化：审查包（任务书 §5/§8 授权产物，不入 git）
# 零 tracked 文件被修改、零 staged、零 merge/checkout/commit/push 痕迹
```

> 说明：`_review/` 未被 `.gitignore` 覆盖（我**未**修改 `.gitignore`——那属禁止的写操作），故以未跟踪条目形式出现；这是任务书唯一许可的产物，**不 commit、不 push**。
