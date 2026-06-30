# M1-B 验新家 · SUMMARY

> 任务编号 G_M1B_VERIFY_NEWHOME · 生成 2026-06-20 · 执行 Agent: Claude Code
> 工作区 `D:\sub2api-trunk` · 主线 `wujie/trunk @ b919650f`
> 性质=验证(只读核对 + 一次可撤回 throwaway commit)· 全程未改任何业务代码

## 结论

**新家 `D:\sub2api-trunk` 状态 = PASS(READY)。**

理由:全部 ★ 硬核对(C1/C2/C3/C5 + 实测 C7)与全部软核对(C4/C6/C8)逐条 PASS——分支/HEAD/main 全对,工作树健康无锁无损,throwaway commit 成功且 `reset` 后 HEAD 干净回到 `b919650f`,证据包 `_review/` 随迁完整可读。新家可正常提交,可作为主路径补线的可信工作区。

## 逐条核对表

| # | 核对项 | 期望 | 实测 | PASS/FAIL |
|---|---|---|---|---|
| C1 ★ | git status 健康 | 工作树正常,仅 `_review/` 未跟踪 | `On branch wujie/trunk`,仅 `_review/` 未跟踪,无 lock/corrupt | **PASS** |
| C2 ★ | 分支 wujie/trunk | `wujie/trunk` | `wujie/trunk` | **PASS** |
| C3 ★ | HEAD b919650f | `b919650f…` 开头 | `b919650f21921165362c05f934cebc5a014fad21` | **PASS** |
| C4 | log 顶端 b919650f | 顶端 b919650f,下含 6478237a | `b919650f` → `6478237a` → `40e83bf4` | **PASS** |
| C5 ★ | main 69f648e2 | `69f648e2…` 开头 | `69f648e20e6f194b08fb120c215e96e88b30e84e` | **PASS** |
| C6 | worktree 干净唯一 | 一个有效树指向 D:\sub2api-trunk | 恰一个 `D:/sub2api-trunk b919650f [wujie/trunk]`;其余 4 棵为已登记历史残留(见异常登记),均未锁、不挡 commit | **PASS(带登记)** |
| C7 ★ | 能 commit + reset 回 b919650f | throwaway commit 成功,reset 后 HEAD=b919650f,工作树干净 | commit `8e38737c` 成功压在 b919650f 上;`reset --hard HEAD~1` 后 `rev-parse HEAD`=`b919650f2192…`;throwaway 文件已消失;工作树干净 | **PASS** |
| C8 | _review 证据在位可读 | 三份 SUMMARY 在位可读 | 7 个评审目录全部在位且可读(多于任务书所说"三份"),含 M1B 系列三份 + 历史四份 | **PASS** |

## git 证据(原文)

### 步骤 B/STEP1 · 只读核对原文

```
$ git status
On branch wujie/trunk
Untracked files:
  (use "git add <file>..." to include in what will be committed)
	_review/
nothing added to commit but untracked files present (use "git add" to track)

$ git branch --show-current
wujie/trunk

$ git rev-parse HEAD
b919650f21921165362c05f934cebc5a014fad21

$ git log --oneline -3
b919650f feat(collector): M1-B.2 response 限容 tee(包一层旁路写出器, flag 默认关)
6478237a feat(collector): M1-B.1 采集口基础设施 + prompt 采集(flag 默认关)
40e83bf4 test(video-gateway): 阶段0+1 定标准+C1进程内活体(keyless ¥0, 未开真实门)

$ git rev-parse main
69f648e20e6f194b08fb120c215e96e88b30e84e

$ git worktree list
"D:/Codex创业任务/企业 API 管理后台项目/02_source/sub2api"                                                   69f648e2 [main]
"D:/Codex创业任务/企业 API 管理后台项目/02_source/sub2api/.claude/worktrees/clever-gauss-162c23"             69f648e2 [claude/clever-gauss-162c23]
"D:/Codex创业任务/企业 API 管理后台项目/02_source/sub2api/.claude/worktrees/nostalgic-chatterjee-703296"     69f648e2 [claude/nostalgic-chatterjee-703296]
"D:/Codex创业任务/企业 API 管理后台项目/02_source/sub2api/.claude/worktrees/wizardly-haibt-f4c9a2"           69f648e2 [claude/wizardly-haibt-f4c9a2]
D:/sub2api-trunk                                                                                            b919650f [wujie/trunk]
（注:git 原始输出把非 ASCII 路径以八进制转义引号包裹,上面为可读还原;路径与 commit 完全一致。）

$ ls _review
M0A_recon_20260618  M0B_trunk_20260618  M1A_collector_20260618  M1B2_collector_impl_20260619
M1B_collector_impl_20260618  M1B_smoke_test_20260619  wujie_trunk_unlock_20260619
```

### 前置闸门(reset 前确认无已跟踪改动)

```
$ git status --porcelain --untracked-files=no
（空输出 = 已跟踪文件无未提交改动,闸门通过,可安全 reset --hard）
```

### 步骤 C/C7 · throwaway 试写 + 撤回原文

```
$ echo check> _newhome_writecheck.txt
$ git add _newhome_writecheck.txt
warning: in the working copy of '_newhome_writecheck.txt', LF will be replaced by CRLF the next time Git touches it   # 良性 Windows 行尾告警

$ git commit -m "chore: new-home write check (throwaway)"
[wujie/trunk 8e38737c] chore: new-home write check (throwaway)
 1 file changed, 1 insertion(+)
 create mode 100644 _newhome_writecheck.txt

$ git log --oneline -2
8e38737c chore: new-home write check (throwaway)
b919650f feat(collector): M1-B.2 response 限容 tee(包一层旁路写出器, flag 默认关)

$ git reset --hard HEAD~1
HEAD is now at b919650f feat(collector): M1-B.2 response 限容 tee(包一层旁路写出器, flag 默认关)

$ git rev-parse HEAD
b919650f21921165362c05f934cebc5a014fad21          # == b919650f,回到位

$ git status
On branch wujie/trunk
Untracked files:
	_review/
nothing added to commit but untracked files present

$ ls _newhome_writecheck.txt
ls: cannot access '_newhome_writecheck.txt': No such file or directory   # 文件随 reset 消失 = 预期
```

### C8 · _review 子目录内容(可读性证据)

```
M0A_recon_20260618/         A_branch_map.md  B_disposition.md  C_trunk_plan.md  D_anchors/  E_command_log.md  F_recheck.md
M0B_trunk_20260618/         M0B_LOG.md  stage0_rescued/
M1A_collector_20260618/     00_START_HERE.md  01_采集链路图.md  02_燃料可达性结论.md  03_设计方案_字段存法脱敏架构.md  04_改动清单_风险_回滚_验证.md  05_待学者拍板清单.md  E_只读命令与侦察日志.md
M1B2_collector_impl_20260619/   SUMMARY.md
M1B_collector_impl_20260618/    M1B_LOG.md
M1B_smoke_test_20260619/        SUMMARY.md
wujie_trunk_unlock_20260619/    SUMMARY.md
```

## 异常登记(已登记 · 全程未动手)

### 1) C6 历史残留 worktree —— 4 棵,已登记,严禁 prune/删除
`git worktree list` 共 5 棵树。我们这棵之外的 4 棵全部 `[69f648e2]`、均未标 locked/prunable、均不挡 C7 commit(各 worktree 索引/HEAD 独立)。原文路径:

- `D:/Codex创业任务/企业 API 管理后台项目/02_source/sub2api`  `[main]` —— 原始克隆,承载共享 `.git`(对象库 + refs)。
- `D:/Codex创业任务/企业 API 管理后台项目/02_source/sub2api/.claude/worktrees/clever-gauss-162c23`  `[claude/clever-gauss-162c23]`
- `D:/Codex创业任务/企业 API 管理后台项目/02_source/sub2api/.claude/worktrees/nostalgic-chatterjee-703296`  `[claude/nostalgic-chatterjee-703296]`
- `D:/Codex创业任务/企业 API 管理后台项目/02_source/sub2api/.claude/worktrees/wizardly-haibt-f4c9a2`  `[claude/wizardly-haibt-f4c9a2]`

判定:按任务书 C6 异常处理记为"有残留待清理(已登记),不影响本任务"= PASS。本任务**未 prune、未删除**(清理属须授权动作)。

### 2) 关键结构事实 —— 新家是链接工作树,仓库主干仍在旧路径
`D:\sub2api-trunk\.git` 是指针文件,内容:
```
gitdir: D:/Codex创业任务/企业 API 管理后台项目/02_source/sub2api/.git/worktrees/objective-goldstine-b9bf89
```
- 新家 = **链接工作树(linked worktree)**,不是独立仓库;其记账名 `objective-goldstine` **确实指向 D:\sub2api-trunk** → 即任务/记忆预期的"已重指的无害记账名",**PASS,未 prune**。
- 仓库主干(`.git` 对象库 + refs + `main` 主工作树)仍在旧路径 `D:/Codex创业任务/…/02_source/sub2api`。这是 `git worktree move` 的正常行为(只搬工作文件,gitdir 留在主仓 `.git/worktrees/` 下),**非搬家没搬干净**。
- **含义(供学者知情,非阻断项)**:新家可正常提交,但其存续依赖旧路径继续在位且健康——若旧路径被删/移,新家会失联。建议后续若要彻底独立,再单独评估是否 `git worktree repair` 或迁移 `.git` 主干(须另开授权任务,本步不做)。
- 健康面证据:共享 `.git` 根与 `objective-goldstine-b9bf89/` 管理目录均**无任何 `*.lock`**,管理目录 HEAD/index/refs/logs 齐全且 mtime 新至 06-20;C7 实测写入成功已坐实可提交。

### 3) C8 数量差异(非异常,记录在案)
任务书 C8 期望"三份 SUMMARY",实测 `_review/` 下有 **7 个**评审目录(全部可读,优于期望)。M1B 系列三份(M1B_collector_impl / M1B2_collector_impl / M1B_smoke_test)+ 历史四份(M0A / M0B / M1A / wujie_trunk_unlock)。判定 PASS,差异仅为措辞宽泛,不构成 FAIL。

## 给学者的下一步建议

- **PASS → 新家 READY**:可把本 SUMMARY 回贴战略脑,进入**主路径补线任务书**(补 Anthropic `/v1/messages` 路径 `gateway_handler.go:908` 闭包漏接采集;碰热路径中风险,需本地回归 + Codex 独立复核,红线=客户端字节一字不变)。
- **两项无需现在处理、但要让学者知道的事实**:① 4 棵历史残留 worktree 已登记未动(何时清理由学者授权,不在本步);② 新家是链接工作树、主干仍在旧路径(补线提交完全不受影响,但旧路径要保持在位)。
- 本任务**不交 Codex 复核**(纯只读 + 可撤回试写,无代码改动)。Codex 独立复核留给补线那一步(修者≠审者)。
