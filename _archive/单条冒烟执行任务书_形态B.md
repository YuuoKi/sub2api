# 单条冒烟执行任务书 · 形态 B（给 Claude Code 执行 / 学者盯着跑）

> 生成日期：2026-06-15。本任务书已对照 **当前代码逐行核实**，并纠正了交接包中
> 三处与代码不符的失真（见 §0）。最高风险：**真实付费**。真实调用前必须停下等学者拍板。

---

## 0. 先纠正交接包的三处失真（重要，全部以代码为准）

| # | 交接包说法 | 代码真相（已核实） | 影响 |
|---|-----------|-------------------|------|
| 失真① | 形态 B 要开「4 闸门」：`SUB2API_VIDEO_REAL_SMOKE_ENABLED` / `SUB2API_REAL_HUMAN_AUTHORIZED` / metadata `single_smoke_authorized` / `trialMode=tiny_real` | 形态 B 直调 `seedanceVideoAdapter` 结构体，它**只读** `seedanceSmokeGateBlockedReasons`（[video_gateway_adapter.go:411-435](backend/internal/service/video_gateway_adapter.go)）。注意区分：`SUB2API_REAL_HUMAN_AUTHORIZED` 在**本 Go 仓库根本不存在**（全仓 grep 0 处；若有也只在独立 Hono 仓 `apps/hono-api`）；`trial_mode=tiny_real` 确实存在于 Go 侧 HTTP 路径（[video_handler.go:39,267](backend/internal/handler/video_handler.go)、[video_gateway_service.go:466](backend/internal/service/video_gateway_service.go)）。**两者形态 B（直调适配器结构体）都不经过。** | 任务书按**真实闸门**列（见 §4）。多开那两道无害但无效，别误以为它们在保护形态 B。 |
| 失真② | 前置要「DB/redis 能起」 | 形态 B **不碰 DB、不碰 redis、不需要起服务**——它在内存里构造 `VideoProviderAccount`/`VideoTask`，直调适配器结构体。DB/redis/daily-limit/credits 全在 service 层，形态 B 绕开（这正是化解 CB3 的原因）。 | 前置只需：代理在线 + Go 进程出站走代理 + 真实 key。 |
| 失真③ | （隐含）审计日志 0600 受保护 | `appendRedactedVideoEvent` 的 0600 用 `f.Chmod(0o600)` 兜底，但 **Windows 上 `Chmod` 不给 POSIX 权限**（测试也在 `runtime.GOOS=="windows"` 时跳过 0600 断言，[security_test.go:60](backend/internal/service/video_gateway_security_test.go)）。 | 见 §3 残余风险。日志内容已脱敏（无 key），裸 Windows 一次性冒烟可接受，**事后删日志**；要真 0600 走 WSL/Docker。 |

---

## 1. 一句话 + 红线（钉死）

形态 B = 一个 **env-gated、默认三层关断** 的 Go 测试，直调
`seedanceVideoAdapter{}.CreateTask` + 手控、硬上限的 `PollTask` 循环，走完
smoke gate / SSRF / duration / 审计 全路径，但**绕开 worker（化解 VA2）、绕开 DB/credits/日限（化解 CB3）**。

文件：[backend/internal/service/video_gateway_realsmoke_test.go](backend/internal/service/video_gateway_realsmoke_test.go)（`//go:build realsmoke`）

**红线（不可逾越）：**
- 外部硬止损：火山账户封顶约 **200 RMB**。
- 只发 **1 次 Create** + 手控 **N 次 Poll（硬上限 30，env 只能调低不能调高）**。
- 真实 key **学者本人填进环境变量**，Claude 不碰 / 不读 / 不打印（key 是 `json:"-"`，String/LogValue 全 `[REDACTED]`）。
- 代理链全程在线（家服务器别掉）；冒烟前验 Go 测试进程出站走代理。
- **不 push**；除这一条盯着的冒烟外，不做任何真实调用。
- 失败**不重试**（测试无任何重试逻辑，Create 只一次）。
- 冒烟后**立即复位**所有 env（见 §7），并验普通 `go test ./...` 恢复无真实调用能力。

---

## 2. 默认为什么绝不会发真实调用（已验证，物证在本会话）

三层关断，缺任意一层即 inert（不开 socket、不读 key）：

1. **build tag `realsmoke`**：普通 `go test ./...` 根本**不编译**此文件。
   - 物证：`go test -C backend -run TestSeedanceSingleRealSmokeFormB ./internal/service/` → `no tests to run`。
2. **run-flag**：`SUB2API_SEEDANCE_REAL_SMOKE_RUN=1` 未置 → `t.Skip`，不读 key、不发包。
   - 物证：`go test -C backend -tags realsmoke -run ... -v ./internal/service/` → `--- SKIP`（flag 未置时）。
3. **适配器自身 fail-closed 闸门**：`seedanceSmokeGateBlockedReasons` 缺任一条件即返回 `VIDEO_PROVIDER_DISABLED`，`real_call_executed:false`，不发 HTTP。
4. **开 socket 前的 key 脱敏自检**（应对评审发现 `redact-gap-opaque-token`）：开任何 socket 前，证明 `redactVideoUpstreamSecrets` 能脱敏本次真实 key（裸 / `Bearer ` / JSON 三形态）；不能则在 HTTP 前 `t.Fatal`，且**绝不打印 key**。已实测：危险形态 key 在 HTTP 前 ABORT；安全形态放行后仍在闸门处止步。

`go vet -tags realsmoke` 干净；既有 13 个 security 测试 + 4 个 gateway 测试非缓存全绿。

---

## 3. 前置硬条件（形态 B 专用，已简化）

| 前置 | 怎么做 | 怎么验 |
|------|--------|--------|
| 代理链在线 | 学者家服务器规则代理在线（cn-beijing 火山仅经此可达，fake-ip） | 略 |
| Go 测试进程出站走代理 | 在**启动 `go test` 之前**于同一 shell `export`/`$env:` 设 `HTTPS_PROXY`（和必要的 `HTTP_PROXY`/`NO_PROXY`）。Go 默认 `http.DefaultTransport` 走 `ProxyFromEnvironment`，适配器 `&http.Client{Timeout:30s}` 继承之。**注意：`http.ProxyFromEnvironment` 进程内只读一次环境变量，故必须在进程启动前设好，不能在测试里设。** | ① 非计费预检：经同一代理对 Ark 发一个**不计费**请求（GET 一个不存在的 task）确认 401/404 而非超时（见 §5 预检命令）。② 冒烟瞬间在家服务器代理连接日志看到去 `ark.cn-beijing.volces.com:443` 的 CONNECT（进程级现场确认）。 |
| 真实 key | 学者把真实火山 Ark key `export` 到 `SUB2API_SEEDANCE_SMOKE_API_KEY`。**绝不写进文件、不进 git、不贴给 Claude。** | `[bool]$env:SUB2API_SEEDANCE_SMOKE_API_KEY` 为 True 即可（不打印值）。 |
| 审计日志路径可写 | 指向 git-ignored 路径，推荐 `backend/data/seedance-smoke-redacted.log`（`.gitignore:106` 已覆盖）。 | 目录存在可写。 |

**残余风险（已知，学者拍板）：**
- **Windows 0600 不完整**：日志内容已脱敏（无 key），裸 Windows 一次性冒烟可接受，**事后 `Remove-Item` 删日志**。要真正 0600 → 在 WSL/Docker 跑 `go test`。
- **DNS-rebinding 不做**：适配器注释明确不在校验器里做 live DNS（TOCTOU 陷阱）；result_url/reference_url 不被本进程抓取（Ark 抓 reference、前端播 result），故可接受。
- **Ark 字段名 UNVERIFIED**：见 §6，首条真实响应后逐一核对回填。

---

## 4. 武装步骤（按顺序；全部在同一个 shell）

> **执行位置（全篇统一）：本任务书所有命令一律在 sub2api 仓库根目录执行。** 开工前先 `cd` 到根并验证 `$PWD`：
>
> ```powershell
> cd "D:\Codex创业任务\企业 API 管理后台项目\02_source\sub2api"   # ← 仓库根
> echo $PWD   # 必须显示 ...\02_source\sub2api（不能是 ...\sub2api\backend，更不能是包目录）
> ```
>
> 钉死 CWD=根之后，下方 `$PWD\backend\data\...`（展开成绝对路径）与 §5 的 `go test -C backend`（相对根）才同时成立。

> 形态 B 真实闸门（适配器实际检查的全集）：
> `SUB2API_VIDEO_REAL_SMOKE_ENABLED=1` + metadata `single_smoke_authorized`（测试已在内存账户里置 true）
> + `SUB2API_VIDEO_REDACTED_EVENT_LOG` 非空 + `SUB2API_VIDEO_URL_ALLOWLIST` 非空
> + `task.Model` 含 "seedance" + `duration ∈ [1,5]`。
> 加上 build tag + run-flag + 真实 key，共同构成武装条件。

PowerShell（裸 Windows）：

```powershell
# 1) 代理（务必在 go test 之前设好，进程启动前生效）
$env:HTTPS_PROXY = "http://<家服务器代理>:<port>"
$env:HTTP_PROXY  = $env:HTTPS_PROXY
# 2) 真实 key（学者本人，值不外泄）
$env:SUB2API_SEEDANCE_SMOKE_API_KEY = "<真实火山 Ark key>"
# 3) 适配器闸门
$env:SUB2API_VIDEO_REAL_SMOKE_ENABLED = "1"
# 审计日志固定落在 <仓库根>\backend\data\（命中 .gitignore 第 106 行 backend/data/）。
# 从根执行时 $PWD=根，下行展开成绝对路径——必须是绝对路径：go test 会把测试二进制的工作目录
# 设为包目录 backend\internal\service，故裸相对值（如 "backend\data\..."）会错位到包目录下，切勿改成相对。
$env:SUB2API_VIDEO_REDACTED_EVENT_LOG = "$PWD\backend\data\seedance-smoke-redacted.log"
$env:SUB2API_VIDEO_URL_ALLOWLIST      = "volces.com"   # Ark 结果 CDN 后缀；如文档另有专用域名后缀，回填更精确的
# 4) 人工授权位（form B 的 run-flag = 人工授权）
$env:SUB2API_SEEDANCE_REAL_SMOKE_RUN  = "1"
# 5)（可选）压更小：duration / 轮询
$env:SUB2API_SEEDANCE_SMOKE_DURATION  = "5"   # 1..5；越小越省
# $env:SUB2API_SEEDANCE_SMOKE_MAX_POLLS = "20"  # 只能 ≤30
```

> `single_smoke_authorized` 无需在环境里设——形态 B 在内存账户里直接置 `true`（账户是 ephemeral，
> 冒烟后无持久授权残留可复位，这点比 service 路径更干净）。

---

## 5. 执行命令

**先做非计费预检**（确认代理 + DNS(fake-ip) + TLS 到真实 Ark 通，不创建任务、不计费）：

```powershell
# GET 一个不存在的 task → 期望 401/404（鉴权/不存在），不是超时/连接错误。不计费。
try {
  Invoke-WebRequest -Proxy $env:HTTPS_PROXY -Method GET `
    -Uri "https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks/nonexistent-preflight" `
    -Headers @{ Authorization = "Bearer $env:SUB2API_SEEDANCE_SMOKE_API_KEY" } -TimeoutSec 30
} catch { $_.Exception.Response.StatusCode.value__ }   # 打印状态码即代表网络通
```

**再发单条真实冒烟**（学者拍板后）：

```powershell
go test -C backend -tags realsmoke -count=1 -timeout 300s -v `
  -run '^TestSeedanceSingleRealSmokeFormB$' ./internal/service/
```

- `-count=1`：禁缓存（真实调用绝不能命中缓存）。
- `-timeout 300s`：覆盖最坏 30 poll × 5s = 150s + 余量。
- 期间盯家服务器代理日志的 `ark...:443` CONNECT（进程级出站确认）。

---

## 6. 现场 Go / No-Go 核对清单（逐条映射测试输出）

| 闸门项 | 看哪里 | GO 标准 |
|--------|--------|---------|
| result_url 可播 | 测试日志 `RESULT URL (validated, playable): "..."` | 该 URL 在浏览器/播放器能播；为空且 status=succeeded → No-Go（疑字段名，见 §6.1） |
| 实账单 ≤ 预估且 ≤ 封顶 | 火山控制台计费 | ≤ 学者 §3 成本模型预估，且远 < 200 RMB |
| 审计两行落地 | 测试日志末尾 `REDACTED AUDIT LOG (N line(s))` | ≥ 2 行（1 create + ≥1 poll）；行内 `phase`/`status_code` 正确；**绝无 key 明文** |
| 失败不重试 | 测试日志 `creates=1 polls=K` | creates 恒为 1；无任何重试痕迹 |
| Ark 字段名逐一核 | 见 §6.1 | id/status/content.video_url/duration/resolution/aspect_ratio 全部确认 |
| 无 SSRF 放行 | 若 result_url 被拒会出现 `result_url_rejected` / status=failed | 合法 Ark CDN URL 应通过 allowlist |

任一项 No-Go → **停**，记录证据，复位 env（§7），不重试、不 push，回报学者。

### 6.1 Ark 字段名核对程序（UNVERIFIED → 首响后回填）

适配器对响应用**写死的 json tag** 解析；若 Ark 实际字段名不同，`json.Unmarshal`
**不报错**只留零值，表现为 `upstream_id` 空或 succeeded 但 `result_url` 空。

| 位置 | 适配器假定字段名（[adapter.go](backend/internal/service/video_gateway_adapter.go)） | 怎么核 |
|------|------------------|--------|
| 请求(create) | `duration` / `resolution` / `aspect_ratio`（adapter.go:181-189，注释明标 UNVERIFIED） | 看实账单时长是否=5s（证明 `duration` 被接受）；分辨率/比例是否生效 |
| create 响应 | `id` / `status`（adapter.go:230-236） | 日志 `upstream_id` 非空即 `id` 对。**`status` 不能靠 `normalized_status` 判定**——`NormalizeStatus("")` 默认返回 "running" 永不空（评审 `status-fieldname-mismatch-silent`），必须在审计 dump 里读到真实 `status` 键才算确认 |
| poll 响应 | `id` / `status` / `content.video_url`(嵌套) / `error.message`（adapter.go:314-323） | 审计 dump 里能看到真实 JSON **键名**（值可能被脱敏，但键名保留）；`result_url` 非空即 `content.video_url` 路径对 |

> **两个隐藏陷阱（评审坐实，harness 已加告警）：**
> - **未映射的终态 token**：若 Ark 返回适配器 case 表没有的终态值（如 `done`/`finished`/`complete`——注意只 `completed` 被映射），`NormalizeStatus` 默认归 "running"，poll 会烧满 30 次并报 "running"。封顶时 harness 会提示去审计 dump 读**原始 `status`** 值；若确是未映射终态 → 任务其实已完成，扩 `NormalizeStatus`，**别再计费重发**。
> - **审计 body 截断**：body 被 `truncate(redact(...),1000)`，超 1000 字符后的键会被 `...(N more)` 切掉；harness 检测到该标记会告警"该行字段名核对无效"。若需要的键被切，用更短 prompt 重跑（不影响安全，仅影响核对完整性）。

**字段名不匹配的处置**：停、不重试、把审计 dump 的真实键名贴回，改 adapter.go 的 json tag，重跑 localhost 测试（不真实调用），下一窗口再核。

---

## 7. 冒烟后立即复位（不可省）

```powershell
Remove-Item Env:SUB2API_SEEDANCE_REAL_SMOKE_RUN -ErrorAction SilentlyContinue
Remove-Item Env:SUB2API_SEEDANCE_SMOKE_API_KEY  -ErrorAction SilentlyContinue
Remove-Item Env:SUB2API_VIDEO_REAL_SMOKE_ENABLED -ErrorAction SilentlyContinue
Remove-Item Env:SUB2API_VIDEO_REDACTED_EVENT_LOG -ErrorAction SilentlyContinue
Remove-Item Env:SUB2API_VIDEO_URL_ALLOWLIST     -ErrorAction SilentlyContinue
# 裸 Windows：脱敏日志事后删除（已知 0600 不完整）
Remove-Item "$PWD\backend\data\seedance-smoke-redacted.log" -ErrorAction SilentlyContinue
```

复位后验：`go test -C backend -run TestSeedanceSingleRealSmokeFormB ./internal/service/` → `no tests to run`（恢复无真实调用能力）。**不 push。**

---

## 8. 错误码速查（适配器返回，全部已脱敏）

| 错误码 | 含义 | 处置 |
|--------|------|------|
| `VIDEO_PROVIDER_DISABLED` (`real_call_executed:false`) | 闸门未开全 / model 不含 seedance / duration 越界 | 补齐闸门，未计费，可再武装 |
| `SEEDANCE_CREATE_HTTP_ERROR` / `SEEDANCE_POLL_HTTP_ERROR` | 网络层错（多半代理/连通） | 查代理链，未必计费，**不盲目重试** |
| `SEEDANCE_CREATE_UPSTREAM_ERROR` (auth/quota/rate_limit/...) | Ark 非 2xx | 看 type；auth→key 错；quota→额度；**停，不重试** |
| `SEEDANCE_CREATE_BUSINESS_ERROR` | 200 但 body 带 error.message | 已脱敏，停 |
| `SEEDANCE_AUDIT_LOG_FAILED` | 审计写不进（fail-closed） | 审计是真实调用前置；修日志路径再来 |

---

## 9. 下一步（VA1，团队可用前的硬前提，非本次冒烟范围）

冒烟成功 ≠ 团队可用。C 阶段（Hono `seedance tiny-real` 网关那条，VA1）上线给多人用前，
**必须补 per-call 预算门 + 计费/credits 扣减**——那条路径绕过 team-credits 且无单次预算门。
本次形态 B 首枪**刻意不走那条**，正是为了规避 VA1。
