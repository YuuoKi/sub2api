# 阶段 A 审查包 · M0 收敛真相源（Sub2API）

> 夜间无人值守任务 · 2026-06-18 · Claude Code（opus-4-8）
> 本阶段范围：双仓各建五件套单一真相锚点 + 归档散落探索包。**零代码改动、零删除（只移动）。**

---

## 0. 执行环境 · 实际路径核对（开工第一步）

任务书给的是 WSL 路径（`/mnt/d/...`），**实际执行在 Windows + Git Bash**，以下为 `git` 实测真相：

| 项 | Sub2API | QCanvas |
|---|---|---|
| 实际仓库根 | `D:/Codex创业任务/企业 API 管理后台项目/02_source/sub2api` | `D:/Codex创业任务/QCanvas（无界版）/QCanvas` |
| 起始分支 | `feat/phase2b-real-smoke-20260616` | `rebuild/studio-v2-product-shell-20260607` |
| 起始 HEAD | `c35049a4` | `17f11f1d` |
| 本阶段工作分支 | `night-run/20260618-A-truth`（off `c35049a4`） | `night-run/20260618-A-truth`（off `17f11f1d`，下方记录） |
| origin | `github.com/Wei-Shaw/sub2api`（开源上游，**push=泄露公司代码，全程禁止**） | `github.com/YuuoKi/qcanvas-wujie`（**禁止 push**） |

> 起始分支保留两处未提交改动（`M .gitignore` + 未跟踪 `deploy/docker-compose.b1-emptybrake.yml`），属前序工作，**全程不动、不提交、不删除**，随分支携带。

---

## 1. 五件套锚点（Sub2API）

任务书五件套理想命名为 `00_START_HERE / 01_BASELINE / 02_当前真相 / 03_当前目标 / 最新审查包/`。
本仓**已存在英文命名的等价锚点**，为避免「目录越整理越乱」再生熵增，**就地更新、补齐缺件，不新建中文重名锚点**：

| 五件套角色 | 本仓实际文件 | 本轮动作 |
|---|---|---|
| 00 入口 | `00_START_HERE.md` | 更新为当前真相（旧文 06-12，已过期） |
| 01 基线 | `01_PROJECT_BASELINE.md` | 保留（基线版本/commit） |
| 02 当前真相 | `02_CURRENT_REALITY_STATUS.md` | 更新为 06-18 真相（含 B1/B2/B3 缺陷定位） |
| 03 当前目标 | `03_CURRENT_GOAL.md`（新建于根，统一指向 C1） | 新建，C1 端到端为唯一焦点 |
| 最新审查包/ | `最新审查包/`（指针目录） | 新建，指向 `_NIGHT_RUN_20260618/` |

---

## 2. 归档散落探索包（只移动，git mv，不删除）

详见下方「归档执行记录」（在实际 move 后回填）。

---

## 3. 验证证据

- `git status`：仅新增/重命名，无删除（见下方回填）。
- 归档前后目录树：见下方回填。

---

## 4. 停止条件

五件套就位 + 归档完成 → 进阶段 B。

---

## 归档执行记录

`_review_packages/` 整目录被 `.gitignore:144` 忽略 → 散落探索包对 git 不可见。归档为**纯文件系统移动**（`mv`，非 `git mv`，因 git mv 仅作用于已跟踪文件），git 无任何 delete。

移入 `_review_packages/_archive_20260618/`（10 项，索引见该目录 `INDEX.md`）：
`codex_review_prompt.txt`、`阶段0收口审查包.md`、`阶段1工程地基审查包.md`、`阶段2A_前置加固审查包.md`、`阶段2B_single_smoke_authorized写入.ps1`、`阶段2B_真实冒烟审查包.md`、`发射前检查_2026-06-17.md`、`真打首枪_2026-06-17.md`、`多发勘察_2026-06-17.md`、`多发实打结果_2026-06-18.md`。

根级 `真实Seedance2.0全链路接入方案.md` / `真实Seedance2.0方案_对抗评审.md` / `单条冒烟执行任务书_形态B.md` **保留原位**（属现行参考，且被 harness/任务书引用，非散落）。

## git status 证据（Phase A 提交前）

```
 M .gitignore                                  ← 前序未提交，不动、不提交
 M 00_START_HERE.md                            ← 本轮：更新当前真相
 M 02_CURRENT_REALITY_STATUS.md                ← 本轮：更新 06-18 真相 + B1/B2/B3
?? 03_CURRENT_GOAL.md                          ← 本轮：新建 C1 目标锚点
?? _NIGHT_RUN_20260618/                        ← 本轮：审查包
?? 最新审查包/                                  ← 本轮：指针目录
?? deploy/docker-compose.b1-emptybrake.yml     ← 前序未提交，不动、不提交
```

**无任何 delete**；归档对 git 不可见（ignored）。Phase A 提交仅含上述 5 个本轮新增/修改锚点。

## 提交记录

分支 `night-run/20260618-A-truth`；提交仅 `git add` 5 个锚点（显式逐一，绝不 `git add .`）。
确切 commit hash 见 `git log` 与 `00_黎明总结.md`。**未 push。**

