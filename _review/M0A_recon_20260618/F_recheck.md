# F · G1–G9 二手情报逐条复核 — M0-A

> 复核日期：2026-06-18 ｜ 方法：每条用 file:line / commit 独立坐实（自查 + 独立子代理对抗复核双通道）
> 图例：✅ 完全对得上 ｜ ⚠️ 部分对（实质成立、细节有出入） ｜ ❌ 不成立

| 编号 | 二手线索 | 判定 | 真实情况（含证据） |
|---|---|:--:|---|
| **G1** | origin = `github.com/Wei-Shaw/sub2api`（push=泄露红线） | ✅ | `git remote -v`：fetch 与 **push 均**指向 `https://github.com/Wei-Shaw/sub2api.git`。这是公开上游，向 origin push 任何分支都会泄露公司私有视频网关代码。**红线成立且 origin 仍是上游（未被改成可 push 的私仓）→ 任务书停止条件 #3 未触发，但 push 永久禁止。** |
| **G2** | main 落后 origin/main ~590、自有 5 | ✅ | `git rev-list --left-right --count origin/main...main` → `590	5`。落后 590、领先 5，精确吻合。 |
| **G3** | main 自有 5 = `4c5de849`/`bade34de`/`43003a00`/`58f79542`/`69f648e2` | ✅ | `git log origin/main..main --oneline` 五个 hash 与 message 全部吻合（P0 mock + P0.5 QA + 白标 demo + phase3.8 两次锁定）。HEAD=`69f648e2`。 |
| **G4** | 10+ 未并分支，每条领先 7–25、0 回合 | ✅（并修正） | 逐条 `git rev-list --count` 确认（见 [A 表 §2.1](./A_branch_map.md)）。**修正**：这 10+ 条**不是平行分叉**，而是同一线性链上的检查点标签；`night-run/D`(25) 线性包含全部；`safety-…20260605` 与 `p0-9b`（均 `7ac5335b`）**完全重复**。 |
| **G5** | 竖屏 `aspect_ratio→ratio` 修复在 `1be53de3`（未并入） | ⚠️ | **内容属实**：`git show 1be53de3 -- …adapter.go` 确认把 `payload["aspect_ratio"]=task.AspectRatio` 改为 `payload["ratio"]=normalizeSeedanceRatio(...)`，并新增纯函数 `normalizeSeedanceRatio`（根因：发 `aspect_ratio` 被 Ark 静默忽略→默认 16:9→竖屏做不出）。**需补充**：① 这是**请求侧**修复；**响应侧**早在 `7b78f9ca`（phase1）已对齐解析 `ratio`——两半合起来才让 9:16 真正打通。② `1be53de3` **不在 main 上**（`git branch --contains` 只返回 night-run/B、C、D），main 仍是 bug 态（见 G8）。 |
| **G6** | Seedance 真调用疑似在 `p0-9b-seedance` | ✅ | `git show 1b8865ff`：在 `p0-9b` 谱系内新增真实 `http.NewRequest` 到 `ark.cn-beijing.volces.com/api/v3` 的 create+poll（`1b8865ff` 加真适配器，`7ac5335b`=p0-9b tip 加 smoke 门）。`git grep` main 的 adapter 对 `http.NewRequest/ark.cn-beijing` **零命中** → main 上 Seedance 不发任何真实请求。 |
| **G7** | 禁用开关 adapter:**134**(Seedance)、:**189**(Kling) | ⚠️ | Seedance **精确**：main 第 **134** 行 `infraerrorsUnavailable("SEEDANCE_REAL_CALL_DISABLED", …)`（poll 禁用在 144）。Kling **行号偏差**：`KLING_REAL_CALL_DISABLED` 实际在第 **198** 行（poll 禁用 208）；189 行只是 `klingVideoAdapter.CreateTask` 的**函数声明**。实质（两家真调用在 main 上均硬禁用）成立，仅 Kling 行号差 9。 |
| **G8** | 竖屏 bug：adapter 发 `aspect_ratio` 于 :**110/180/238** | ✅ | `git show main:…adapter.go \| grep -n aspect_ratio` → 精确命中 **110**（mock）/ **180**（seedance，真 bug 处，Ark 要 `ratio`）/ **238**（kling），各一处 `BuildCreatePayload`。补充：main 上此 bug **潜伏**——真调用在 134 行被禁，payload 不会真正抵达 Ark，bug 只在真适配器（G6）并入后才发作。 |
| **G9** | `ent/schema/usage_log.go` 只存 token/成本/SHA256 哈希，无 prompt/结果 | ⚠️ | **核心隐私结论成立**：`git show main:…usage_log.go` 全字段为 user_id/api_key_id/account_id/request_id/model 三件套/channel_id/billing_*/group_id/subscription_id/各类 token 计数/各类 cost/rate_multiplier/billing_type/stream/duration_ms/first_token_ms/user_agent/ip_address/image_count/image_size/cache_ttl_overridden/created_at——`grep prompt\|result\|content\|video_url\|payload` **零命中**：**无 prompt、无结果内容**。**修正**："SHA256 哈希"描述不准——**没有任何 hash/sha 字段**；`request_id` 是普通 `MaxLen(64) NotEmpty` 字符串（请求标识，非内容哈希）。 |

---

## 附 · 凭据安全扫描（任务书停止条件 #4）

对 `69f648e2..40e83bf4` 全 25 commit 做了 `git log -p` + 逐文件审查（config/env/compose/Caddyfile/day0脚本/SQL/Go/测试）。

**结论：CLEAN — 零明文凭据入库。** 所有"像密钥"的字符串只落两类安全桶：

1. **`*.example` 模板占位**：`change_this_secure_password`、`change-me-*`、`CHANGE_ME_STRONG_PASSWORD`、空 `encryption_key`（带 `openssl rand -hex 32` 提示）；compose 里全是 `${VAR:?}`/`${VAR}` 引用，零字面值。
2. **`*_test.go` 合成测试夹具**：如 `sk-…-PLAINTEXT-…`、`seedance-real-key-placeholder-…`、`akltsecret…`——**故意造的假密钥**，用途是断言脱敏层把它们剥掉；真实 Ark key 仅运行时从 env `SUB2API_SEEDANCE_SMOKE_API_KEY` 读取，从不入库/落日志。

仅一处历史占位被**标注为"已修复"**：`AUDIT_REPORT.md:232`（commit `bcb5fd32`）列了旧 `JWT_SECRET=dev-jwt-secret-not-for-production`，非 live。另：`c35049a4` 还**改善**了卫生（把弱默认 `admin123` 换成占位）。

> ⚠️ 注意（非 commit 内）：真实出片物证按记忆可能以**未提交改动**存在于主工作树（检出 `night-run/D`）或 WSL `p0-9b-clean` 工作树。本侦察只覆盖**已提交历史**，不进入其他工作树查看其脏区——交学者在那两个工作树各自 `git status` 自查。详见 [C_trunk_plan.md §5](./C_trunk_plan.md)。

---

## 复核可信度声明

- G1/G2/G3/G6/G8 = ✅ 完全坐实。
- G4 = ✅ 并发现"线性单链 + 重复分支"的重要修正。
- G5/G7/G9 = ⚠️ 实质成立，细节修正（G5 不在 main+另有响应侧一半；G7 Kling 行号 198≠189；G9 无 SHA256 字段、request_id 非哈希）。
- ❌ 项：无。无任何线索被推翻。
- 凭据扫描：CLEAN（停止条件 #4 未触发）。
