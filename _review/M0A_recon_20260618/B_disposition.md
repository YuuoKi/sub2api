# B · 逐分支 / 逐 commit 处置建议 — M0-A

> `[DRAFT 待学者校准]` ｜ 处置词：**保留(线性收编)** / **重复-丢弃** / **可折叠(docs)** / **待学者决策**
> 前提：全链线性、零互冲突（见 [A 表 §3](./A_branch_map.md)）。因此"处置"的真问题不是"哪条分支取哪条舍"，而是
> **"这条单链整体收编，还是先剔掉 docs/checkpoint 噪声 commit 再收编"**。

---

## 1. 分支级处置（先收掉"重复/空"分支）

| 分支 | 处置 | 理由 / 证据 |
|---|---|---|
| `night-run/20260618-D-c1-alive` (`40e83bf4`) | **以此为准（超集）** | 线性包含其余全部 24 commit（`merge-base --is-ancestor` 全 YES）。收编只需盯它。 |
| night-run/C、B、A ／ phase2b、2a、phase1 ／ phase-3.8.2 ／ p0-9b | **同链祖先，不单独收** | 都是 D 的祖先检查点；收 D 即全收。保留分支名仅作历史书签。 |
| `safety-sub2api-before-dirty-resolution-20260605` | **重复-可删（待学者）** | 与 `p0-9b-seedance-real-smoke` 同为 `7ac5335b`，纯备份。M0-B 后可删，本任务不动。 |
| `claude/nostalgic-chatterjee-703296` | **空-可删（待学者）** | = main HEAD，无自有工作，worktree 占位。 |
| `origin/*`（dev/preview/…） | **不收（上游）** | Wei-Shaw 上游开发线，非我方工作。 |

---

## 2. 逐 commit 处置（main..D，旧→新，25 条）

> "命中"列：①Seedance真调 ②ratio修复 ④C1活体 ⑥护城河其他（SSRF/脱敏/预算/drama/白标）；docs/checkpoint 不标。
> "冲突风险"全部 = **低**：链上每个 commit 已基于 main 的自有 5 commit，ff 收编对 main 零冲突（见 §3）。

| # | hash | 类型 | 真正干了什么（diff 级） | 命中 | 价值 | 处置 |
|--:|---|---|---|---|---|---|
| 01 | `fa442586` | checkpoint | `tools/phase_3_8_2/*` 打包/校验脚本（JS+PS1，+1260），无运行时码 | — | 打包工具 | **可折叠/丢弃**（与产品码无关，M0-B 可不收） |
| 02 | `fede33aa` | code | 安全修复：`APIKeyConfigured` 只认 `EncryptedAPIKey`（不再被 MaskedKey 误判已配）；前端 video 视图 + ListTasks 员工/管理员过滤 | ⑥ | 安全修复 | **保留** |
| 03 | `53148df4` | docs | `AGENTS.md` codex 护栏（禁 push/部署、禁读打印真密钥、薄适配器） | — | 护栏文档 | **保留(docs)**，可入主线根 |
| 04 | `0774feeb` | code | `drama_gateway_service.go`（1362 行内存 drama 推荐/技能引擎，无网络）+ admin/user 路由 | ⑥ | 产品功能 | **保留（待学者：是否纳入 MVP 范围）** |
| 05 | `884415ac` | code/白标 | `InternalPilotView.vue` + 路由注册测试；`embed_on.go` 改名"企业 AI 视频 API 调度中台"；i18n/setup 中文化 | ⑥ | 白标 | **保留** |
| 06 | `1b8865ff` | code | **★Seedance 2.0 真实 Ark 适配器★**：真 POST `/contents/generations/tasks`+GET，默认 `ark.cn-beijing.volces.com/api/v3`、model `doubao-seedance-2-0-260128`；poll 解析 `content.video_url`。**此版 create 只发 model+content+negative_prompt（尚无 ratio）** | ① | **基础（被后续增强）** | **保留（链基，不可单摘）** |
| 07 | `7ac5335b` | code | **★Seedance 单条 smoke 门★**：env `SUB2API_VIDEO_REAL_SMOKE_ENABLED=1`+provider 元数据授权+model=seedance+时长1-5s 才放行；qcanvas 契约状态映射；redacted_event 标志 | ① | 安全门 | **保留**（=p0-9b tip） |
| 08 | `c4e2337d` | code/白标 | day0 白标 checkpoint：`migration 138` UPDATE 白标文案；`embed_on` 改"无界互娱 API 控制台"；`VideoSystemCheckView.vue` 等前端大改 | ⑥ | 白标+迁移 | **保留（待学者：白标文案 138 是否最终版）** |
| 09 | `4143673f` | docs | `docs/SUB2API_DIRTY_RESOLUTION_2026_06_05.md`（230 行） | — | 过程记录 | **可折叠** |
| 10 | `3351338d` | code | **qcanvas api-key 视频 MOCK 网关**：`/video` 路由(providers/tasks/cancel)，`mock_only`/`provider_boundary`，拒 seedance/kling | ④ | **被#11 取代** | **保留（链基）**；其 mock-only 语义被 #11 trial 路径扩展 |
| 11 | `4dd599af` | code | **★gated seedance tiny_real 试点★**：`trial_mode oneof tiny_real`、1 任务/日/人限额、smoke 门校验、`trial_gate` 事件，**不自动开火** | ①④ | 真试点骨架 | **保留** |
| 12 | `1d5badd8` | code | **★Seedance blocker B1/B2/B3 修复★**：`video_gateway_ssrf.go`(validateExternalVideoURL+allowlist+反 SSRF)；`video_gateway_redact.go`(脱敏+AKLT 模式+fail-closed 0600 审计日志)；poll 不再盲信 `video_url`→校验；`PlainAPIKey json:"-"`+掩码；`encryption_key` **必填**（去掉 totp 回退） | ①②⑥ | **护城河核心** | **保留（强烈）** |
| 13 | `9af63819` | code/deploy | 部署卫生：Dockerfile GOPROXY、`build:fast\|\|build`；`admin_password admin123→CHANGE_ME`（卫生改善）；Caddyfile `{$CADDY_DOMAIN:localhost}`/env 化 | ⑥ | 部署卫生 | **保留** |
| 14 | `0f68d5ab` | docs | `docs/reviews/*` 5 份评审/物证 md（+1444） | — | 评审包 | **可折叠** |
| 15 | `379da544` | docs | Caddyfile 注释订正（5/3 行，无行为变化） | — | 注释 | **可折叠** |
| 16 | `bcb5fd32` | code(mixed) | phase0 收口：大量 docs/部署归档(`deploy/day0/*.sh`)+**惰性 realsmoke harness**(`//go:build realsmoke` 三层解保险)+**不透明 token 脱敏 pass**（补脱敏盲点） | ①②④ | 收口+脱敏增强 | **保留（代码部分）**；docs 部分可折叠 |
| 17 | `7b78f9ca` | code(mixed) | **★工程地基★**：VA2 `migration 139`(poll_count 列)+轮询上限(默认72)+worker 长生命周期 ctx；VA1 `VideoBudgetGuard` 接口(fail-closed CheckBudget+Charge)；**ratio 响应侧**对齐(poll 解析 `ratio`)；wsl 部署卫生 | ②⑤⑥ | **工程地基** | **保留（强烈）** |
| 18 | `85b6347f` | code | **★StaticBudgetGuard★** fail-closed 预算拦截原语+穿 CreateTask 拦截测试；`deploy/README` 钉死单实例/poll_count/poll 窗约束。**未接 DI，生产零变化** | ⑥ | 预算门原语 | **保留** |
| 19 | `c35049a4` | code(mixed) | **★预算门 DI 接线★**(`per_call_budget>0` 才武装)+**命门级脱敏双补**(key-aware `seedanceRedactBody`+pre-arm 自检)+**回显凭证 fail-closed 守卫**+**worker 反双发**(post-create queued→submitted 防二次计费 create)+形态A harness | ①④⑥ | **护城河核心** | **保留（强烈）** |
| 20 | `ed83030b` | docs | 重写 night-run M0 真相源五锚点（189/70） | — | 锚点文档 | **可折叠**（其内容由本审查包 D_anchors 取代/校准） |
| 21 | `1be53de3` | code(mixed) | **★ratio 请求侧修复★**：create/BuildCreatePayload 改发 `ratio`+`normalizeSeedanceRatio`；B2 轮询窗随分辨率(48/72/300)；B3 v2v `video_url` 草案(字段名 UNVERIFIED)。*worker 配置钉死早返回路径在生产为死代码，被 #22 修* | ② | **竖屏必修** | **保留（强烈）** |
| 22 | `831e9c98` | code(mixed) | **B2 死代码修复**：`maxPollAttemptsForTask` 改为**始终**按分辨率缩放(config 72 视为 720p 基线→1080p 得 300/25min 而非 72/6min)；ratio 直通收紧到 shape-valid W:H | ② | B2 真生效 | **保留（强烈，必与#21同行）** |
| 23 | `47cf1146` | test | `video_handler_c1_contract_test.go`(97 行)：断言 mock 任务响应 JSON 键匹配 QCanvas 契约+mock-only 边界 | ④ | 契约测试 | **保留(test)** |
| 24 | `486f52fe` | docs | `_NIGHT_RUN_20260618/00_黎明总结.md`(70 行) | — | 总结 | **可折叠** |
| 25 | `40e83bf4` | test | `api_key_video_gateway_c1_alive_test.go`(151 行)：真 gin 路由进程内驱动 mock 至 succeeded，断言 9:16 竖屏存活 + 真门关闭(`real_provider_dispatch_count==0`)+147 行 doc | ②④ | **C1 活体证** | **保留(test)** |

---

## 3. 与 main 自有 5 commit 的冲突评估

- **冲突风险：全链低 → 实际为零（对 ff 收编）。** 25 commit 都已**基于** main 的 5 个自有 commit（merge-base 全 = `69f648e2`），它们是 main 的**直系后代**，不是平行分叉。
- 唯一"改了同一批文件"的是视频网关那组文件（`video_gateway_*.go`、前端 `video/*.vue`、`migration 136/137`）——但因为是后代而非并行线，这些是**顺序演进**而非冲突。`git merge --ff-only night-run/D` 可零冲突推进。
- **真正会冲突的场景**只有一个：若将来要把这条链 rebase 到 **origin/main（领先 590）**之上做开源同步——那时视频网关文件与上游 590 commit 可能撞车。但 M0-B 的目标是**干净私有主线**，不做上游同步，故不触发。详见 [C_trunk_plan.md](./C_trunk_plan.md)。

---

## 4. 重复 / 取舍标注（任务书要求）

- **同一份工作重复出现**：仅 `safety-…20260605` 与 `p0-9b`（同 commit）。**以 p0-9b 为准，safety 标重复丢弃（M0-B 删，本任务不动）。**
- **被后续取代但不可单摘**：`1b8865ff`(#06)、`3351338d`(#10) 被后续增强/扩展，但在线性链里是基座 commit——**收 D 即自然含其最终态**，无需也不应单独摘出。
- **docs/checkpoint 噪声（共 7 条：#01,09,14,15,20,24 + 部分 #16/bcb5fd32 的 docs 体积）**：是否保留 = **待学者决策**。建议：M0-B 收编时若走 squash 路线则自然剔除；若走 ff 全收则保留为历史（无害）。

---

## 5. 拿不准 → 待学者决策清单

1. **前端去留**：drama 引擎(#04)、白标文案两版(#05 企业AI视频 / #08 无界互娱)——哪个是最终品牌？是否纳入 MVP？→ **待学者**。
2. **收编粒度**：ff 全收 25（含 docs）vs squash 成几个干净 commit（剔 docs）→ 见 C 方案，**待学者拍板**。
3. **重复/空分支删除**（safety、claude/*）：M0-B 授权后删，**本任务不动**。
4. **未提交真实出片物证**：可能在主工作树(检出 D)或 WSL p0-9b-clean 工作树的脏区——本侦察不进入查看，**待学者在那两处各自 `git status` 自查后并入**。
