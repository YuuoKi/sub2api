# G_D3_REDACTION_RETENTION — 审查总览(SUMMARY)

> 任务:采集口安全前置 = 脱敏加固(D3-a) + 保留期清理(D3-b)
> 工作区 `D:\sub2api-trunk`(linked worktree)｜分支 `wujie/trunk` @ HEAD `eca1b65c`(未提交,未 push)
> 执行:Claude Code(Opus 4.8, ultracode)｜跨家族 Codex 复核由学者另派(不在本任务内)
> 日期:2026-06-21｜环境:本地零成本(zonky 嵌入式 PG,不连真库/真 key/真付费,未采任何真数据)

---

## 0. 结论先行

两道安全网均**就绪并经验证**,可作为 D1(采真数据,需学者授权)的前提。全程未碰红线。

- **D3-a 脱敏加固**:在既有 `email/phone` 之外补齐 **身份证号 / 银行卡号 / 高熵 opaque token** 三类漏网,且**精准**——正常中文业务 prompt、模型名、UUID、订单号、日期、snowflake 等**零误抹**(22 条单测全绿 + 前后对比证据)。规则版本 `generationRedactionVersion` 1→2。
- **D3-b 保留期清理**:建成**可配置、独立可调用、可 dry-run** 的 NULL-OUT 清理机制(超期把内容字段置空、**保留计数行**),经**真 PG 实跑**验证「只清超期、未超期不动、计数/字节行保留、dry-run 零副作用、已清空行不被重复处理」。机制**就绪但默认 dark**(daemon 不自动启动),D1 授权时**单行武装**。
- **范围**:改动只落在脱敏/清理相关文件;主路径 `/v1/messages`、tee/转发/路由、采集补线 `wire_gen.go:188` **逐字未动**;未跑 wire codegen;`git diff --name-only | grep gateway` 为空。
- **停止点**:符合任务停止条件 → **停,等学者拍 D1**。未自行进入采真数据。

---

## 1. 改动清单(含归属,区分本任务 vs 既有未提交看板work)

### 1a. 本任务(D3)新增文件
| 文件 | 作用 |
|---|---|
| `backend/migrations/141_ai_generation_content_retention_index.sql` | partial index(谓词=尚有内容),保清理扫描有界 |
| `backend/internal/service/generation_content_retention_service.go` | 保留期清理服务(Start/Stop/runLoop/cleanupOnce + 导出 RunOnce) |
| `backend/internal/service/generation_content_retention_service_test.go` | 服务单测(fake repo:默认/夹紧/dry-run零副作用/只清超期/多批排空/fail-open/nil-safe) |
| `backend/internal/service/generation_content_redact_structured_test.go` | 脱敏加固单测(身份证/卡/opaque 命中 + 一批标识符不误抹) |
| `backend/internal/service/generation_content_redact_demo_test.go` | 前后对比证据(§3.1),`-v` 输出可直接贴 |
| `backend/internal/repository/generation_content_retention_repo_integration_test.go` | 真 PG 集成测(`//go:build integration`):只清超期/批上限/已清空不复算 |

### 1b. 本任务(D3)对既有文件的增量(均脱敏/清理相关)
| 文件 | 本任务增量 | 备注 |
|---|---|---|
| `backend/internal/config/config.go` (+21/-0) | 新增 `ContentRetentionConfig` + `GatewayConfig.ContentRetention` 字段 | 文件在 HEAD 处干净→该 diff **全为本任务** |
| `backend/internal/service/generation_content_redact.go` (+132/-3) | 新增 `redactGenerationStructuredPII`(身份证/卡/opaque)+ 校验函数;管线插层;版本 1→2 | 文件在 HEAD 处干净→该 diff **全为本任务** |
| `backend/internal/service/generation_content.go` | 接口加 `PurgeExpiredContent(...)` | 同文件另含**既有看板**的结构体/读接口(非本任务) |
| `backend/internal/repository/generation_content_repo.go` | 实现 `PurgeExpiredContent`(dry-run COUNT + CTE UPDATE)+ `time` 引入 | 同文件另含**既有看板**的 `GetCaptureStats/GetRecent`(非本任务) |
| `backend/internal/service/generation_content_collector_test.go` / `_panic_test.go` | 给两处现有 fake 补 `PurgeExpiredContent` 以保编译 | — |

### 1c. **非本任务**——会话开始前就已存在的未提交「C 看板」work(切勿误记为 D3)
`backend/cmd/server/wire_gen.go`(看板 handler 接线 @~244,**非** :188)、`internal/handler/handler.go`、`internal/handler/wire.go`、`internal/repository/wire.go`、`internal/server/routes/admin.go`、`internal/handler/admin/generation_content_handler.go`、`frontend/**`(api/sidebar/i18n/router/view/components)。本任务**一字未改**这些(它们在会话开始的 `git status` 即为 M/??)。

---

## 2. 脱敏加固:前后对比 + 误抹检查

证据原文:`evidence/redaction_before_after.txt`(`go test -run TestRedactGenerationHardening_BeforeAfterDemo -v`)。

### 2a. 漏网→堵住(连续形态:加固前 9 条正则全漏,加固后精准标记)
| 类别 | IN | BEFORE(加固前) | AFTER(加固后) |
|---|---|---|---|
| 身份证 | `…身份证 11010519491231002X 麻烦核对` | `…11010519491231002X…`(漏) | `…[ID]…` |
| 银行卡(连续) | `卡号 4111111111111111 转账` | `…4111111111111111…`(漏) | `卡号 [CARD] 转账` |
| 高熵 token | `临时令牌 Ab3Cd6Ef9Gh2Jk5Lm8Np1Qr4 已发` | `…Ab3Cd…Qr4…`(漏) | `临时令牌 [已脱敏] 已发` |

### 2b. 不误抹(正常业务 prompt / 分析字段,AFTER == IN)
- `用户反馈登录后首页加载很慢,希望排查接口超时问题并给出优化建议。` → 原样
- `请帮我把这段产品介绍润色得更专业一些,目标受众是企业采购决策者。` → 原样
- `工单号 ORD-20240115-0042,客户要求本周内回复处理进度与预计完成时间。` → 原样
- 模型名 `claude-opus-4-8` → 保留

### 2c. 精准性设计(为何不误抹)
- **身份证**:候选 `\b\d{17}[0-9Xx]\b` + 回调校验「出生日期窗(年1900..当年/月/日)+ ISO 7064 mod-11-2 校验位」。→ 18-19 位 snowflake/订单号(日期或校验位不过)**不抹**;校验位错的伪号**不抹**(`TestRedactGenerationStructuredPII_InvalidIDNotRedacted`)。
- **银行卡**:候选 `\b(?:\d[ -]?){12,18}\d\b` + 回调「去分隔→13-19 位→Luhn」。→ 16 位非 Luhn 订单号(夹具 `1234567890123456`,Luhn 和=64)**不抹**,旧测试 `TestRedactGenerationPII_IdentifiersUntouched` 保持绿。
- **opaque token**:**复用** `video_gateway_redact.go` 同包 `videoOpaqueTokenCandidate` + `looksLikeOpaqueVideoSecret`(20+ 位、同时含数字与字母才算密钥)。→ 纯字母词(`antidisestablishmentarianism`)、纯数字计数器(20 位)**不抹**。未改 video 文件、未抽公共函数(避免触碰 video blindspot 套件)。
- **RE2 安全**:三条均「\b 锚定候选 + ReplaceAllStringFunc 回调」,不会像早期过宽电话正则那样把 UUID 切碎(`TestRedactGenerationPrompt_UUIDNotMangledByPhone` 仍绿)。
- **管线顺序**:`RedactJSON/Text → email/phone → structured(ID→卡→opaque) → content_moderation`。

### 2d. 已知可接受残留(已正其名,非泄露)
**分隔形态银行卡**(`4111 1111 1111 1111` / `4111-1111-1111-1111`):因管线 phone 先于 card,卡号主体先被电话分组正则吃成 `[PHONE]`,**仅尾 4 位残留**(PCI 允许展示尾 4)。卡号主体**不泄露**;marker 为 `[PHONE]` 而非 `[CARD]`。证据见 `redaction_before_after.txt` (A2) 段。判断:可接受(连续卡号→`[CARD]` 干净;分隔卡号主体已脱)。

---

## 3. 单测 / dry-run / 真跑证据

### 3a. 单测(默认 `go test`,无 DB)— 全绿
`evidence/unit_tests.txt`:脱敏 14 + 保留服务 8 = **22 条全 PASS**。覆盖:身份证/卡/opaque 命中、标识符不误抹、配置默认/夹紧(RetentionDays<7→7)、RunOnce dry-run 零副作用、只清超期、多批排空、fail-open、nil-safe。

### 3b. 保留期清理 真 PG 实跑(§3.3)— 全部符合预期
`evidence/retention_sql_proof.txt`(zonky 嵌入式 PG 17.5 @127.0.0.1:5434,跑**仓库实际 SQL**):
1. 迁移 140+141 应用成功,partial index `idx_ai_generation_content_unpurged_created_at` 已建。
2. 造数据:1 过期(now-100d)+ 1 未过期(now-1d),均有内容。
3. **dry-run**:`would_purge=1`;清理后仍 `with_content=2` → **零副作用**。
4. **真清**:`total_rows=2`(**行保留**);过期行 `prompt/response_redacted` 置空、字节列 6/8 **保留**;未过期行内容 `PROMPT/RESPONSE` **原样**。→ 只清超期、未超期不动、计数/字节不缩水。
5. **再 dry-run**:`would_purge_again=0` → 已清空行不被重复处理(partial index 谓词生效,扫描有界)。

### 3c. 集成测(真 PG,CI/Docker)
`generation_content_retention_repo_integration_test.go`(`//go:build integration`)已写并 `go vet -tags integration` 通过。本机 Docker 不可用(集成 harness 走 testcontainers),故未在本机跑该 harness;等价行为已由 3b 的 zonky 真跑直接证明(同一套 SQL)。

### 3d. 全栈复验(§3.4)说明
脱敏函数即采集器 `Collect` 实际调用的 `redactGenerationPrompt/Response`,已用「PII + 中文 prose」直测(2a/2b),等价于样本墙落库内容;看板读路径(`GetCaptureStats/GetRecent`/handler)本任务**未改动**,空·实两态结构上不受影响。完整 M1B 起栈(server.exe+mock+redis)未重跑——属低增量高成本确认,如学者要求可另跑。

### 3e. 构建/质量
`go build ./...` OK;`go vet ./internal/service ./internal/config ./internal/repository` OK(含 `-tags integration`);`go test ./internal/service`(全包,48s)/`./internal/config`/`./internal/repository`(非集成)全 OK。

---

## 4. NULL-OUT 选型理由(任务 §2 D3-b 二选一)

**选 NULL-OUT(清空内容字段、保留计数行)**,而非删行。唯一裁决标准 = 老板/超管看板使用体验优先(用户拍板)。
- **看板体验**:NULL-OUT 下全时段护城河指标(`Total` / 去重员工·团队·模型 / `TotalBytes`——字节列不清空)**持续累计、不缩水**;删行会让这些缩成「保留窗口内」,削弱护城河展示。已真跑证明字节/计数行保留(3b.4)。
- **安全达标**:承载 PII 的是内容文本(`prompt_redacted`/`response_redacted`),NULL-OUT 在保留期后**精准抹除内容文本**;残留行只剩归因(user/model/时间/字节,与 usage_logs 同级,非敏感内容)。
- **样本墙**:`GetRecent` 取近样本(保留期内,仍有内容),不受影响。
- **效率**:partial index(141)使已清空行离开索引,清理扫描恒定有界,化解 NULL-OUT「重扫已清空行」隐患(3b.5 已证 re-dry-run=0)。

---

## 5. D1 单行武装说明(机制就绪·不现在启动)

后台 daemon **默认不启动**(`ContentRetentionConfig.Enabled` dark by default;且服务未进生成的 DI 图)。`RunOnce(ctx,dryRun)` 已可独立调用/测试。D1 授权采真数据时,在 `cmd/server/wire_gen.go` 采集器注入行(:188)**之后**手工补一行点火(与 :188 同源、单行可复核):
```go
service.NewGenerationContentRetentionService(
    repository.NewGenerationContentRepository(db), configConfig,
).Start()
```
并在 config 设 `gateway.content_retention.{enabled: true, retention_days: 90}`。本任务**不做**此武装(D1 范畴)。

---

## 6. git 范围声明

- **`wire_gen.go:188`(采集补线 eca1b65c)逐字未动**——已 `sed -n '188p'` 核验原文一致。
- `git diff --name-only | grep -i gateway` = **空**(主路径零触碰)。
- 未运行 `wire` codegen。
- 本任务改动仅落脱敏/清理相关文件(§1a/1b);`git diff --stat` 中出现的 `wire_gen.go`(@~244 看板 handler 接线)、`handler*.go`、`routes/admin.go`、`frontend/**`、`generation_content_handler.go` 等均为**会话开始前既存的未提交 C 看板 work**,本任务一字未改(§1c)。
- `_review/`(含本 SUMMARY、evidence、SQL 夹具)**不入库、不 push**。

---

## 7. 偏差 / 停止登记

- **偏差(已正其名)**:分隔形态银行卡的 marker 是 `[PHONE]`+尾4 而非 `[CARD]`(管线 phone 先行),卡号主体仍脱敏——可接受残留,见 §2d。
- **未做(范畴/成本)**:完整 M1B 全栈起栈复验(§3d 已说明等价证明);集成 harness 本机未跑(Docker 不可用,§3c,zonky 真跑等价)。
- **停止**:符合任务正常停止条件(脱敏加固 + 保留清理就绪、证据齐)→ **停,等学者拍 D1**(采真数据的环境/入口/授权范围)。未翻 flag、未跑真流量、未采真数据、未连真库/真 key/真付费、未碰主路由/采集补线/旧路径/main。
