<!-- 单一收口物证 / 2026-06-15 / sub2api @ phase-3.8.2-overnight-readiness / 零真实调用·零真实凭据·未 commit 未 push -->

# SUB2API · 单条真实冒烟【准备包草案】

> **本文件是单条真实 Seedance 冒烟的【准备包草案】。本任务【只准备、不执行】：不真实调用 Ark / 任何 provider，不填任何真实 API key / 域名实测值（除占位符与公开示例域名 `volces.com`）。** 供用户醒来 + QCanvas 侧契约补验通过后再由用户本人执行；执行前需先确认本物证包另三份（签字物证、git 地图、worktree 收口）均已 review。

- **日期**：2026-06-15
- **仓库 / 分支**：`02_source/sub2api` @ `phase-3.8.2-overnight-readiness`（HEAD `4dd599af`，未新建分支、未 `git add`、未 commit、未 push）
- **范围**：把 Seedance blocker 修复后【唯一留给冒烟】的一步——单条受闸门的真实 tiny-real 调用——所需的运维配置、Go/No-Go 闸门、Ark 字段名待核清单成文备齐。本文件不代表已执行。
- **权威前序**：[`SUB2API_SEEDANCE_BLOCKER_FIX_REVIEW_2026_06_15.md`](./SUB2API_SEEDANCE_BLOCKER_FIX_REVIEW_2026_06_15.md)（双家族签字 GO / GO-WITH-FOLLOWUPS，唯一剩余 follow-up = 本文件第 C 节的 Ark 字段名核对）。
- **铁律声明**：
  - 本文件是**草案**，**绝不声称已执行过真实冒烟**。
  - **零真实调用**：不调用 Ark / 任何 provider。
  - **零真实凭据**：所有命令与 `export` 一律占位符；凡真实 key/域名实测值的位置一律留空或写 `<...>` 占位，**绝不编造看似真实的凭据**。
  - **未 commit / 未 push / 未 add**（见 git 地图物证）。

---

## A. 运维配置 checklist（执行冒烟前逐项配置；命令全用占位符）

> 下列闸门均为 blocker 修复（B1/B3 fail-closed）后**新增的真实前提**。任一缺失，Go 侧 `seedanceSmokeGateBlockedReasons`（`backend/internal/service/video_gateway_adapter.go:411`）会在 create/poll 前直接拦截并返回 `real_call_executed=false`，**不会发起真实上游调用**。每项给"为什么 / 怎么配 / 校验"。

### A.1 `video_gateway.encryption_key`（`backend/config.yaml` 的 `video_gateway:` 段下）

- **为什么**：B3b 修复把"空则静默 fallback 到 `totp.encryption_key`"改为**硬失败**（`backend/internal/repository/video_key_encryptor.go`：空则 `return error`）。这是密钥域混淆防护——避免用 TOTP key 误加密真实 Seedance 凭据。**该 key 现为必填，空则服务启动即报错（不可启动）。**
- **怎么配**：32 字节 / 64 hex 字符；**须不同于** `totp.encryption_key`。生成命令（占位，请在你自己的终端跑）：

  ```bash
  openssl rand -hex 32   # 生成 64 位 hex；输出请勿粘到本文档/任何会入库的文件
  ```

  在 `backend/config.yaml`：

  ```yaml
  video_gateway:
    encryption_key: "<在此粘贴-openssl-rand-hex-32-的64位hex>"   # 必填，须 ≠ totp.encryption_key
  ```

- **校验**：服务能正常启动即说明 key 已被接受（B3b 路径不抛 `video_gateway.encryption_key is required`）；并目视确认其值与 `totp.encryption_key` 不同。

### A.2 `SUB2API_VIDEO_URL_ALLOWLIST`（环境变量）

- **为什么**：B3a fail-closed SSRF 姿态。闸门要求该变量非空，否则 `validateExternalVideoURL`（`backend/internal/service/video_gateway_ssrf.go`）会落入"松分支"。配置后，校验进入严格 allowlist 分支：`result_url` 与 `reference_image_url` 必须命中白名单域名后缀，**域名 allowlist 天然拒绝十进制/十六/八进制/CGNAT 等混淆 IP**（它们无法匹配域名后缀）。空则 smoke gate 直接拦截真实调用（`blocked_reasons: media url allowlist ... is missing`）。
- **怎么配**：逗号分隔的可信媒体域名**后缀**；代码 `videoURLAllowlistHosts` 会把每项展开成 apex + `*.子域`（即 `volces.com` 同时匹配 `volces.com` 与任意子域）。须含：**Ark 结果 CDN 域名** + 允许的**参考图域名**。

  ```bash
  # 占位示例：确切 Ark 结果 CDN 域名于首条真实冒烟核对后再补全
  export SUB2API_VIDEO_URL_ALLOWLIST="volces.com,<在此补-ark-结果-CDN-域名后缀>,<在此补-允许的参考图域名后缀>"
  ```

- **校验**：用一条**已知的合法 https CDN URL**（属于上面某后缀）走 poll 路径应放行；用 `http://`、内网 IP、`localhost`、混淆 IP 应被拒。Go 侧 `TestValidateExternalVideoURL` / `...Allowlist`（全绿，见前序物证第 4 节）已覆盖这些断言。

### A.3 `SUB2API_VIDEO_REDACTED_EVENT_LOG`（环境变量）

- **为什么**：B1b 修复把"幽灵"env 变成真正落地的**脱敏审计日志**（`appendRedactedVideoEvent`）。create/poll 在读到上游响应体后都会写一行**已脱敏**的 JSON 审计；**写失败 fail-closed**——`f.Chmod(0o600)` 或写入失败会让 create/poll 返回 `SEEDANCE_AUDIT_LOG_FAILED` 并中止真实调用（"没审计就不放行"）。文件以 `O_CREATE|O_WRONLY|O_APPEND` 打开并强制 `0600`（owner 读写）。
- **怎么配**：指向 `*.log` 或 `backend/data/` 下的文件（两者均被 `.gitignore` 忽略，审计文件**绝不入库**）。

  ```bash
  export SUB2API_VIDEO_REDACTED_EVENT_LOG=backend/data/video-redacted-events.log
  ```

- **校验**：① 该路径所在目录可写、最终文件权限为 `0600`（非 0644/0640）；② 冒烟后该文件应出现 create + poll 的脱敏行（含 `phase`/`status_code`，`body` 已脱敏）；③ 确认该路径命中 `.gitignore`（`git status` 不应出现它）。

### A.4 两道真实调用闸门（环境变量 + provider metadata）

- **`SUB2API_VIDEO_REAL_SMOKE_ENABLED=1`**（Go 侧 smoke gate 必检）：Go `seedanceSmokeGateBlockedReasons` 要求该值严格等于 `"1"`，否则拦截。
- **`SUB2API_REAL_HUMAN_AUTHORIZED=1`**（QCanvas / Hono 侧 trial gate 必检）：前序评审 Q3（`sub2api.routes.ts` trial gate）要求 `SUB2API_REAL_HUMAN_AUTHORIZED==='1'` 与 `SUB2API_VIDEO_REAL_SMOKE_ENABLED==='1'` 同时为真且 `trialMode==='tiny_real'`，否则 403 `SUB2API_VIDEO_PROVIDER_BLOCKED`。**此项须在 QCanvas 仓库核实（本仓库看不到 Hono 代码）。**
- **provider metadata 单次授权**（Go 侧另检）：smoke gate 还要求 provider 账户 metadata 记录 `single_smoke_authorized`（或 `real_smoke_authorized`）为真，否则 `provider metadata does not record single smoke authorization`。

  ```bash
  # 两道 env 闸门（占位；执行冒烟前临时置 1）
  export SUB2API_VIDEO_REAL_SMOKE_ENABLED=1
  export SUB2API_REAL_HUMAN_AUTHORIZED=1
  # provider metadata single_smoke_authorized=true 通过后台/数据写入（非 env），冒烟后复位
  ```

- **🔁 冒烟后强制复位（方案 §8 收口强制项）**：冒烟一结束**立即**把上述两道 env 闸门复位为 `0`，并复位 provider metadata 的 `single_smoke_authorized`（恢复 false）。这三处任一未复位都意味着真实调用边界仍处于开启态。

  ```bash
  export SUB2API_VIDEO_REAL_SMOKE_ENABLED=0
  export SUB2API_REAL_HUMAN_AUTHORIZED=0
  # 同步复位 provider metadata single_smoke_authorized=false
  ```

---

## B. Go / No-Go 闸门清单（冒烟现场逐项勾选）

> **GO 必须【全绿】。任一未过 = NO-GO / 立即中止**：复位 A.4 两道 env 闸门 + `single_smoke_authorized`；**不换用户、不清当天额度就不得重试**（失败的冒烟任务也计入该用户当天 daily limit，见前序评审 CB3）。

### B.1 GO 闸门（全绿才放行下一步）

- [ ] **result_url 可播**：https + 命中 `SUB2API_VIDEO_URL_ALLOWLIST` 域名后缀 + `validateExternalVideoURL` 通过（poll 路径未把 `result_url` 置为 rejected）+ 前端能实际播放该视频。
- [ ] **Day0 写入成功**：真实成功快照写入 Day0，且 `resultSource='sub2api-real-trial'`、`persistable:false`、`reuseAllowed:false`（需在 QCanvas 侧核实，前序评审 Q6）。
- [ ] **wujie 计数 0**：`wujieWriteCount:0` / `persistStatus:'not-persistable'`（真实成功路径不得溜进 /wujie，前序评审 WB1/WB2）。
- [ ] **实账单 ≤ 预估**：单条 tiny-real 实际计费 ≤ 预估上限 `¥<在此填预算上限>`（注意 poll 是否计费为 ASSUMPTION，轮询次数须有上限，见前序评审 VA2）。
- [ ] **失败不重试 / 无级联**：`failed` → 单次 throw，**无 fallback / 无级联 / 无 vendor auto 自动切换**（`vendor:'auto'` 须钉死 seedance、candidates 空，前序评审 VA1）、**无无上限轮询**。
- [ ] **审计落地**：`SUB2API_VIDEO_REDACTED_EVENT_LOG` 出现 create + poll 两条脱敏审计行，文件权限 `0600`。

### B.2 NO-GO / 中止（任一 GO 闸门未过即触发）

- [ ] 复位 `SUB2API_VIDEO_REAL_SMOKE_ENABLED=0`、`SUB2API_REAL_HUMAN_AUTHORIZED=0`。
- [ ] 复位 provider metadata `single_smoke_authorized=false`。
- [ ] 记录失败任务所用 `created_by`（CreatedByLabel）；**重试必须换用户或清当天记录**（daily limit 已计入该失败任务）。
- [ ] 不在原因未查清前对同一用户重试（避免静默耗尽额度 → 困惑的限速拒绝）。

---

## C. Ark 真实响应字段名【待核清单】（首条真实冒烟第一步必做）

> Go adapter 当前对 Ark 响应字段名的消费**全部标注 UNVERIFIED**（见 `backend/internal/service/video_gateway_adapter.go` 中 create payload 注释 `NOTE: exact Ark field names ... UNVERIFIED`）。这些假设**必须用首条真实 Ark 响应核对后才可依赖**——这是双家族评审唯一剩余的 follow-up，属冒烟环节、非本仓库代码任务。
>
> **本文件不填真实实测值**：下表"真实响应实测"与"是否一致"两列**留空待冒烟现场填写**。

| 字段（代码假设的 JSON 名） | 代码消费位置 | 代码假设 | 真实响应实测（留空待填） | 是否一致（留空） |
|---|---|---|---|---|
| `id` | CreateTask `parsed.ID`（`adapter.go:231/248`）→ 作为 UpstreamTaskID 复用于 poll URL | 创建响应任务标识字段名为 `id` |  |  |
| `status` | CreateTask `parsed.Status`（`adapter.go:232/249`），经 `NormalizeStatus` 归一化 | 状态字段名为 `status`，取值落入 queued/running/succeeded/failed/cancelled 同义词 |  |  |
| `status` | PollTask `parsed.Status`（`adapter.go:315/330`）| poll 同名 `status` 字段 |  |  |
| `content.video_url` | PollTask `parsed.Content.VideoURL`（`adapter.go:317-318/343`）→ 经 `validateExternalVideoURL` 后存为 result_url | 结果视频地址在 `content.video_url` 嵌套字段 |  |  |
| `error.message` | create/poll `parsed.Error.Message`（`adapter.go:233-235/320-322`）→ 200-OK 业务错误，已脱敏 | 业务错误在 `error.message` |  |  |
| create payload `duration` | CreateTask 下发 `payload["duration"]`（`adapter.go:181-182`）| Ark 接受请求字段名 `duration`（秒），使成本上限非纸面 |  |  |
| create payload `resolution` | CreateTask 下发 `payload["resolution"]`（`adapter.go:184-185`）| Ark 接受请求字段名 `resolution` |  |  |
| create payload `aspect_ratio` | CreateTask 下发 `payload["aspect_ratio"]`（`adapter.go:187-188`）| Ark 接受请求字段名 `aspect_ratio` |  |  |
| `resolution` / `aspect_ratio`（响应回显） | 暂未在响应结构体中解析 | 确切响应字段名待核（是否回显、命名） |  |  |

> 补充待核（非字段名，但属同一冒烟步）：① Ark 结果 CDN 的**确切域名后缀**（用于回填 A.2 的 `SUB2API_VIDEO_URL_ALLOWLIST`）；② 真实 Volcengine AKLT key 的确切形状（用于回填 `volcengineAccessKeyPattern` 脱敏边界，`video_gateway_redact.go:17`）；③ poll 是否计费、单次生成的真实 poll GET 次数（用于 B.1 实账单闸门）。以上均**冒烟现场实测**，本草案不预设。

---

## 关联文档

本文件是 2026-06-15 单一收口物证包四份之一，交叉引用其余三份（相对链接）：

- ① 签字物证：[`SUB2API_SIGNOFF_EVIDENCE_2026_06_15.md`](./SUB2API_SIGNOFF_EVIDENCE_2026_06_15.md)
- ② git 地图：[`SUB2API_GIT_MAP_2026_06_15.md`](./SUB2API_GIT_MAP_2026_06_15.md)
- ③ worktree 收口：[`SUB2API_WORKTREE_CLEANUP_2026_06_15.md`](./SUB2API_WORKTREE_CLEANUP_2026_06_15.md)
- 权威前序评审：[`SUB2API_SEEDANCE_BLOCKER_FIX_REVIEW_2026_06_15.md`](./SUB2API_SEEDANCE_BLOCKER_FIX_REVIEW_2026_06_15.md)
