# Codex 任务书 · 可灵真接交叉复查（KLING-REAL Crosscheck）

> **性质**：交叉复查（audit），**不是重做** WS-A～G。  
> **执行者**：Codex（独立于 Cursor 实现代理）  
> **前置实现**：Cursor Agent 已在 worktree 完成可灵全量真接；官方 AK/SK **尚未到位**，付费真烟测预期 `blocked`。  
> **交付**：`docs/superpowers/codex-handoff/deliverables/2026-07-10-KLING-REAL-CROSSCHECK-review.md`（一份总审查包，含 pass/fail 表 + 发现的问题分级）。

---

## 0. 工作区（先对齐路径）

| 项 | 值 |
|----|-----|
| Sub2API worktree | `D:\sub2api-trunk\.worktrees\kling-real` |
| 分支 | `feature/kling-real-integration`（**未 commit / 未 push**，以工作区 diff 为准） |
| QCanvas | `D:\Codex创业任务\QCanvas（无界版）\QCanvas`（Kling 相关改动与其它 WIP 混在脏树，**只审 Kling 相关文件**） |
| 实现方自述审查包 | [deliverables/2026-07-09-KLING-REAL-review.md](./deliverables/2026-07-09-KLING-REAL-review.md) |
| 契约 | [docs/api/video-gateway-contract.md](../../api/video-gateway-contract.md) §4.2、[qcanvas-integration-guide.md](../../api/qcanvas-integration-guide.md) |

**不要**在主工作区 `D:\sub2api-trunk`（`wujie/video-capture-moat-20260702`）上改可灵代码；以 worktree 为准。

---

## 1. 背景（实现方声称）

上一轮「双产品面对齐」把可灵标成 **下轮必接 / disabled skeleton**。本轮 Cursor 声称已完成：

| WS | 声称 |
|----|------|
| A | AK+SK 打进 `encrypted_api_key` 的 v1 blob；`klingMintJWT`；admin 双密钥 UI；脱敏 |
| B | 真实 `klingVideoAdapter`（删 `KLING_REAL_*_DISABLED`）；t2v/i2v/multi/omni/extend/avatar；duration 仅 5\|10 |
| C | API-key `tiny_real` + `production_authorized` 放行；翻转 BlocksKling 测试 |
| D | 占位 CNY/秒价表 `kling-video-2026-07` + estimate/settle |
| E | Drama：`production_authorized` → 真 Kling；否则 `kling_safe_demo` |
| F | QCanvas：`assertProviderAllowed` + provider-aware ARM；Hono model map |
| G | 契约/文案/交付物；真烟测 `blocked: awaiting AK/SK` |

**你的工作**：独立验证上述声称是否属实；找出实现方漏测、过度乐观、安全回退、契约漂移。**不要**因为「单测绿」就照抄审查包结论。

---

## 2. 明确禁止

- 不要重写 adapter / 大重构；本轮是复查。发现 P0 可给最小补丁，但须在审查包写清「复查发现 → 补丁」。
- 不要提交 `.env`、真实 AK/SK、JWT、密码。
- 不要跑付费真烟测（无密钥）；审查包标 `blocked: awaiting AK/SK` 即可。
- 不要改 Yunwu 旁路（`task.yunwu-video.ts`）除非发现它被误改。
- 不要把 QCanvas 无关 WIP（binding ciphertext 等）卷进本审查结论。
- 不要 force-push；不要擅自 commit（老板未要求则只出审查包）。

---

## 3. 复查清单（必须逐项给证据）

### KX-A — 凭证与安全（P0）

| 检查项 | 如何验 | 期望 |
|--------|--------|------|
| AK+SK 不进 `metadata_json` | 读 `applyKlingCredentials` / admin response | 仅密文列 + masked AK |
| blob 格式 | `pack/unpack` + 单测 | `{"v":1,"auth":"kling_aksk",...}`；Seedance 仍是裸 key |
| leave-empty 更新 | 单测 / 读代码 | 空 AK/SK 保留原值 |
| JWT 不落库 | 搜 persist / DB write | 仅内存缓存至 exp-60s |
| 脱敏 | redact + echo 测试 | body/事件/错误信息不含 AK/SK/JWT |
| adapter 门控 | Create/Poll | 用 `videoProviderHasPlainCredentials`（AK+SK），不是 `PlainAPIKey` |

### KX-B — 真实 adapter 契约（P0）

| 检查项 | 如何验 | 期望 |
|--------|--------|------|
| 无硬禁用串 | `rg KLING_REAL_CALL_DISABLED` 等 | **零匹配** |
| 路径映射 | 读 `video_gateway_kling_adapter.go` | t2v/i2v/multi/omni/extend/avatar 与文档一致 |
| model map | 对照契约表 | `kling-3.0`→`kling-v2-6`；omni/o1 走 omni；未知 fail-closed |
| duration | payload + smoke + `validateVideoGenerationContract` | **仅 5 或 10**；smoke 未 production 时仅 5 |
| UpstreamTaskID | create/poll | 字符集校验 + PathEscape；敌对 ID 不发网 |
| SSRF / 审计 fail-closed | 夹具测试 | 与 Seedance 同级 |
| Cancel | 读代码 | 仅本地 cancel，不假装上游取消 |

### KX-C — API-key 产品路径（P0）

| 检查项 | 如何验 | 期望 |
|--------|--------|------|
| List providers | `ListAPIKeyTrialProviders` | 不再写死 `kling is disabled…` |
| Create 分支 | `video_handler.go` | `tiny_real` + production 分支存在 |
| 未授权 fail-closed | routes 测试 | 无 production_authorized → 403/禁用 |
| 旧 BlocksKling | routes 测试名 | 已翻转为 Allow / WithoutProductionAuth |

### KX-D — 计费（P1）

| 检查项 | 如何验 | 期望 |
|--------|--------|------|
| Kling estimate/settle | billing 测试 | 非零 + `PricingVersion=kling-video-2026-07` |
| Seedance 未破坏 | 既有 Seedance billing 测试 | 仍绿 |
| 占位费率标注 | 代码注释 / 审查包 | 明确 provisional，不可当生产对账真理 |

### KX-E — Drama（P1）

| 检查项 | 如何验 | 期望 |
|--------|--------|------|
| 授权 → 真链 | drama 测试 | `kling_real_*` + RealProviderVerified |
| 未授权 → safe_demo | drama 测试 | 行为与改前一致 |
| 授权 t2v | 测试 | **不**静默改写成 demo-only i2v |

### KX-F — QCanvas（P0 产品面）

| 检查项 | 如何验 | 期望 |
|--------|--------|------|
| `assertProviderAllowed` | 读 adapter + 单测 | `kling` + `allowRealCalls` + `tiny_real` 放行 |
| ARM | shell store | Kling catalog → `provider=kling` |
| `preflightOnly` | capabilities | 已映射 id 为 false |
| Hono map | `KLING_FORWARD_MODEL_IDS` | 与 Sub2API allowlist 一致（catalog 入、上游由 Go 再映射） |
| 调试 ingest | `rg 127.0.0.1:7781` under studio-v2 | **零匹配**（shell 路径） |
| Yunwu | `git diff` / 读文件 | **未改** |
| 官方 surface 文档 | `QCANVAS_SUB2API_OFFICIAL_SURFACE_V1.md` | 不再写「kling blocked until trunk unlock」 |

### KX-G — 文档诚实度（P1）

| 检查项 | 如何验 | 期望 |
|--------|--------|------|
| video contract §4.2 | 读文档 | 无「本轮 disabled/skeleton」 |
| integration guide | 读文档 | 写清凭证步骤 + smoke blocked |
| 实现方审查包 | 对照本复查 | 标出「写得比代码乐观」的条目 |

---

## 4. 必跑命令（自己跑，贴输出摘要）

```bash
# Sub2API worktree
cd D:/sub2api-trunk/.worktrees/kling-real/backend
go test -tags=unit ./internal/service/ -run "Kling|kling" -count=1
go test -tags=unit ./internal/server/routes/ -run "Kling|VideoGateway" -count=1
rg -n "KLING_REAL_CALL_DISABLED|KLING_REAL_POLL_DISABLED|kling is disabled in API-key" --glob "*.go"
# 期望：无匹配

# QCanvas（仅 Kling 相关）
cd "D:/Codex创业任务/QCanvas（无界版）/QCanvas"
pnpm --filter @tapcanvas/web test -- _test/unit/studioV2RealTaskAdapter.test.ts _test/unit/studioV2ShellMockRealWiring.test.ts
pnpm --filter @tapcanvas/api test -- src/modules/sub2api/sub2api.video-mock-gateway.service.test.ts src/modules/sub2api/sub2api.routes.test.ts
rg -n "127.0.0.1:7781" apps/web/src/ui/studio-v2
```

可选（有时间再跑）：`make secret-scan`（在 Sub2API worktree 根）。

---

## 5. 重点怀疑点（实现方自审已提过 / 容易漏）

1. **占位价表当真理** — 费率未与官方对齐，审查包必须写「不可对账」。  
2. **QCanvas 脏树污染** — 只审 Kling 文件；发现无关改动被误提交风险要写进审查包。  
3. **model 双向映射** — Hono 转发 catalog id，Go 再映射 upstream；两端不一致会导致 400。  
4. **Drama 真链是否绕过 smoke env** — 授权账号创建后，worker/adapter 门控是否仍 fail-closed。  
5. **extend/avatar** — 靠 model alias / `PricingSource`，DB `task_type` CHECK 未扩；确认文档与代码一致，无半开入口。  
6. **未 commit** — 审查以 worktree diff 为准；不要假设已入库。

---

## 6. 交付格式

新建：`docs/superpowers/codex-handoff/deliverables/2026-07-10-KLING-REAL-CROSSCHECK-review.md`

必须包含：

1. **总评**：`pass` / `pass_with_findings` / `fail`（一句人话）  
2. **KX-A～G 表**：每项 `pass` / `fail` / `blocked` + 证据（文件:行 或 测试名）  
3. **发现的问题**：Critical / Important / Minor（可空）  
4. **命令输出摘要**（绿/红，勿贴超长日志）  
5. **与实现方审查包的差异**（哪些声称被打脸 / 哪些属实）  
6. **给老板的下一步建议**：可合并审 / 需 Cursor 补丁 / 等 AK/SK 再烟测

完成后告诉老板：「交叉复查完成，审查包路径是 xxx」。

---

## 7. 成功标准

- 独立跑通 §4 命令，不抄 Cursor 输出。  
- 硬禁用串确认消失。  
- 至少抽查 JWT、duration 5\|10、API-key 未授权 fail-closed、QCanvas ARM 四条路径的**代码**（不只测试名）。  
- 真烟测诚实标 blocked。  
- 若有 Critical：审查包 `fail`，并给出最小复现与建议补丁位置。
