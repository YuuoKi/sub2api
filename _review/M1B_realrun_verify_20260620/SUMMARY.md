# M1-B 主路径补线 · 全栈真跑收口 · SUMMARY

> 任务:G_M1B_REALRUN_ENV_AND_VERIFY ｜ 日期:2026-06-20 ｜ 分支:`wujie/trunk` @ `eca1b65c`
> 工作区:`D:\sub2api-trunk` ｜ 全本地 throwaway、¥0、未连真库、未真调上游、未用真 key、未 push、未碰 main/旧路径/worktree、**未改任何已提交代码**
> 执行:Claude Code(Opus 4.8, ultracode)｜ 模式:学者人在、边做边确认

---

## 结论(BC 红线放最前)

**★ BC = 实测逐字节一致 = PASS。** 用补线前二进制(`server_baseline.exe`,b919650f)与补线后二进制(`server_patched.exe`,b919650f+补丁)对**同一固定请求集**抓客户端原始字节,非流式与流式**两路 body+header 全部逐字节相同**(sha256 完全相等)。补丁对客户端字节零影响,此前"强证据"现已转为**全栈真跑实证**。

**主路径补线 = 产品 READY。** 四项验收(ENV / DB / BC / BC-OFF)全部真跑 PASS。昨晚因 Docker 不可用而 BLOCKED 的三项(DB/BC/BC-OFF),今日用**本地原生 Postgres 17.5(替代坏掉的 Docker)**全部补齐实测。

无停止条件触发(BC 未现任何字节差异;未疑似打到真 Anthropic;未需连真库/真付费/真 key/push)。

---

## 逐条验收(实测)

| # | 项 | 期望 | 实测 | 判定 |
|---|---|---|---|---|
| ENV | 本地真跑环境 | PG 起、二进制连上、mock 在位、P0 未打真 Anthropic | PG17.5@5433 + redis@6380 + mock@9099 + server@8090 全起;P0 探活经网关 200 拿到 mock 原样,**mock 进程日志确证收包**(`POST /v1/messages?beta=true`),全库唯一账号 `type=apikey`→`base_url=127.0.0.1:9099` | ✅ PASS |
| DB | 主路径真落库 | 真请求→`ai_generation_content` 出行,prompt+response 同行+脱敏+全归因 | patched 真请求后落 1 行:`id=1 request_id=mock-req-1`,prompt+response **同行**,双脱敏(电话→`[PHONE]`、UUID→`[已脱敏]`),`api_key_id/user_id/group_id/account_id` 全非空,`response_bytes=300 == 客户端 Content-Length`,`response_truncated=false`,`redaction_version=1` | ✅ PASS |
| **BC ★** | 客户端字节一致 | baseline vs patched 同请求集逐字节/逐字段一致 | 非流式 body+header **IDENTICAL**(sha256 `94fd0553…` 相等);流式 SSE body+header **IDENTICAL**(sha256 `da163e26…` 相等);连 `Date`/`X-Request-Id` 都因 mock 固定而无差异 → **零字节差异** | ✅ PASS |
| **BC-OFF ★** | flag 关回归 | 关时字节==补线前 且 0 落库 | flagoff(patched + `enabled:false`)body+header 与 golden **IDENTICAL**(同上 sha256);`ai_generation_content` **0 行** = 关闭即纯 no-op | ✅ PASS |

---

## 原文证据

### BC / BC-OFF —— sha256 三方相等(`evidence/sha256.txt`)
```
94fd0553bad09c3c2428d344f5462aa2090996b1953329b7032bd8972cc0847a  golden/ns.body
94fd0553bad09c3c2428d344f5462aa2090996b1953329b7032bd8972cc0847a  after/ns.body
94fd0553bad09c3c2428d344f5462aa2090996b1953329b7032bd8972cc0847a  flagoff/ns.body
da163e260a23ac017a702805a8bc6897f6766253c04355d2de9384bd40f573bd  golden/st.body
da163e260a23ac017a702805a8bc6897f6766253c04355d2de9384bd40f573bd  after/st.body
da163e260a23ac017a702805a8bc6897f6766253c04355d2de9384bd40f573bd  flagoff/st.body
```
`diff golden/ after/`、`diff golden/ flagoff/`(body + headers,非流式与流式)均**空输出**。

### 客户端响应头(golden = after = flagoff,逐行相同,`golden/ns.headers`)
```
HTTP/1.1 200 OK
Content-Type: application/json
Date: Fri, 20 Jun 2026 00:00:00 GMT          <- mock 固定值,网关透传,无 wall-clock 漂移
Referrer-Policy: strict-origin-when-cross-origin
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-Request-Id: bc-ns-001                       <- 客户端 pin 值,中间件回显(消除随机 UUID)
X-Request-Id: mock-req-1                       <- 上游 mock 透传值
Content-Length: 300
```
> 注:补线唯一可能引入的易变头是中间件随机 `X-Request-ID`;本验证按计划用固定 `X-Request-ID` 请求头钉死,且 mock 回固定 `Date`/`x-request-id`,故**客户端响应无任何易变字段**,BC 为纯字节 diff,无需归一化。

### DB 落库(patched 真请求后,`ai_generation_content`)
```
id=1  request_id=mock-req-1  model=claude-3-5-sonnet-20241022
api_key_id=1  user_id=1  group_id=2  account_id=1   (全归因)
prompt_bytes=178  response_bytes=300 (== 客户端 Content-Length)  response_truncated=false  redaction_version=1
prompt_redacted   = {"max_tokens":256,"messages":[{"content":"回拨客户电话 [PHONE],工单号 [已脱敏]","role":"user"}],"model":"claude-3-5-sonnet-20241022"}
response_redacted = {"content":[{"text":"MOCK_REPLY 工单已收到,我们会尽快回电 [PHONE] 处理。","type":"text"}],"id":"msg_mock_0001","model":"claude-3-5-sonnet-20241022","role":"assistant","stop_reason":"end_turn","stop_sequence":null,"type":"message","usage":{"input_tokens":12,"output_tokens":18}}
```
- baseline(补线前)同请求 → `ai_generation_content` **0 行**(主路径漏接 bug 复现,佐证补线确为"补缺")。
- 非流式与流式两笔共享 mock 固定 `request_id=mock-req-1`,经 `(api_key_id,request_id)` 唯一索引去重 → 只落 1 行(预期行为,非缺陷)。

### P0 安全(未打真 Anthropic,`evidence/p0_probe.http` + `evidence/mock.log`)
- 网关 200 + 正文含 `MOCK_REPLY`;mock 进程日志:`[mock] #3 POST /v1/messages?beta=true stream=false bodylen=178` = 网关把请求转发到了 mock@9099。
- 结构性保证:全库唯一账号 `type=apikey`+`base_url=http://127.0.0.1:9099`,`GetBaseURL()` 只能解析到 loopback mock,不可能触达真实 provider。

---

## 本地原生真跑环境 · 起法记录(供采集口后续验证复用)

> 这套"本地原生 PG + 便携 redis + 自建 mock + Go SQL runner"环境替代了坏掉的 Docker,作为常备基础设施留存。产物全部在 `_review\M1B_realrun_verify_20260620\` 与 repo 外 throwaway 目录。

**1. Postgres(throwaway 簇 @5433)** —— 本机无可复用的原生 Windows PG(:5432 是 WSL 托管的真 qcanvas 库,红线禁碰),改用 zonky 嵌入式 PG 二进制(Maven Central,~23MB):
- 下载 `embedded-postgres-binaries-windows-amd64-17.5.0.jar`(EDB 官方 zip 被限速 14KB/s 弃用),`tar -xf` 解 jar→`postgres-windows-x86_64.txz`→`tar -xf` 解出 `bin/`(initdb/pg_ctl/postgres)。位置:`%TEMP%\m1b_pg\bin`。
- `initdb -D D:\m1b_pg5433\data -U postgres -A trust -E SQL_ASCII --locale=C`(**注意**:`-E UTF8` 在中文 Windows 上必失败于 `pg_import_system_collations`——GBK locale 名非法 UTF8;`SQL_ASCII` 字节透传,对本验证无碍,脱敏在 Go 侧完成,DB 只存/返原始 UTF8 字节)。
- `pg_ctl -D D:\m1b_pg5433\data -l pg.log -o "-p 5433" start`。

**2. redis @6380** —— 便携 redis-windows 8.8.0(GitHub release zip):`redis-server.exe --port 6380 --save "" --appendonly no`。位置:`%TEMP%\m1b_redis\Redis-8.8.0-Windows-x64-msys2`。

**3. mock 上游 @9099** —— 自建确定性 mock(`mock\main.go`,Go stdlib,固定 `Date`/`x-request-id`、`MOCK_REPLY`+电话、含 `usage`、流式固定 SSE)。`go build` 后后台跑 `mock.exe`。

**4. Go SQL runner** —— zonky 不带 psql,自建 `sqlrunner\main.go`(`lib/pq`,离线 `GOPROXY=off` 从模块缓存编)。`sqlrunner.exe exec|query`,SQL 走 stdin(**用 bash 喂,勿用 PowerShell 管道——会加 BOM**)。DSN 默认 `host=127.0.0.1 port=5433 user=postgres password=postgres dbname=postgres sslmode=disable`。

**5. 服务器启动关键**:
- 配置文件必须名为 **`config.yaml`**(见 `run\{baseline,patched,flagoff}\config.yaml`)。
- 必须设环境变量 **`DATA_DIR=<config.yaml 所在目录>`**——否则 `setup.NeedsSetup()` 的 `GetDataDir()` 不认 CWD 里的 config.yaml(viper 认,但安装检查不认),会进**安装向导**而非网关。设了 DATA_DIR → `NeedsSetup()=false` → 走主服务器 → `InitEnt` 自动迁移空库(建全表含 `ai_generation_content`)+ 自动生成持久化 JWT 密钥。
- 启动后确认 stdout 出现 `Server started on 0.0.0.0:8090`(非安装向导地址)。
- 每个 run 之间 `DROP SCHEMA public CASCADE; CREATE SCHEMA public;` 重置,再启二进制(重跑迁移)再种 `seed.sql`。

**固定请求集**:`req-nonstream.json` / `req-stream.json`(model=claude-3-5-sonnet-20241022,max_tokens=256,content 含电话+UUID 触发脱敏)。
**种子**:`seed.sql`(单 apikey 账号→mock,api_key=`sk-smoke-localtest-0001`)。
**抓字节**:`curl -sS -D headers -o body -H "X-Request-ID: <pin>" --data @req ...`(流式加 `-N`,curl 自动去 chunked → 比对 SSE 序列+data+终止)。

---

## 与计划/recipe 的偏差登记(均不影响结论)

1. **PG = zonky 17.5 + SQL_ASCII**(非 recipe 的 Docker postgres:18-alpine/UTF8):EDB 官方包被限速、zonky 不带 psql、中文 Windows UTF8 collation 导入失败——三处均已绕过;SQL_ASCII 对字节级验证无碍(脱敏在 Go 侧;DB 仅透传 UTF8 字节,落库字段中文+`[PHONE]`/`[已脱敏]` 显示正确)。
2. **mock@9099 为自建**(recipe 未提供):属 harness 基建,**未改任何已提交产品码**,与 `server_*.exe`/`config.empirical.yaml` 同性质。
3. **`DATA_DIR` 必设**:补充了 recipe 未明示的启动关键(否则进安装向导)。
4. **去重**:两笔共享 mock 固定 request_id → 唯一索引去重为 1 行(预期);如需流式单独落库,可让 mock 对流式回不同 `x-request-id`(但对 BC 无影响,golden/after 同 mock 即可比)。

---

## 异常 / 停止登记
- **未触发任何停止条件**:BC 零字节差异(非"现差异");P0 实证未打真 Anthropic;未需连真库/真付费/真 key/push;未碰旧路径/worktree;未改已提交代码。
- 唯一"卡点"是环境层(EDB 限速、中文 Windows collation),均在本地范围内绕过,无需提权/系统级改动。

## 下一步建议
1. **主路径补线 `eca1b65c` 判定为产品 READY**——DB/BC/BC-OFF 全栈真跑实证齐备,可进入灰度(分阶段开 flag)。
2. **B.3**:把 B.2 响应 tee 延伸到 responses 出口,再补 responses 采集;openai_gateway 需独立适配(结果类型不同)。可直接复用本环境真跑验证。
3. 保留期/清理任务;灰度 flag 的运营口径。
4. 仍勿 push(origin=开源上游)。

---

## 配套产物(本目录)
- `golden/` `after/` `flagoff/` —— 三方客户端字节捕获(ns/st 各 body+headers)
- `evidence/sha256.txt` —— sha256 清单;`evidence/p0_probe.http` —— P0 探活;`evidence/mock.log` —— mock 收包日志;`evidence/server_*.{out,err}` —— 三个服务器启动日志
- `mock\main.go`(+`mock.exe`)、`sqlrunner\main.go`(+`sqlrunner.exe`)、`run\*\config.yaml`、`req-*.json`、`seed.sql`
