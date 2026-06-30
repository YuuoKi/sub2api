# M1-B 采集口本地点火测试 · SUMMARY

> 日期:2026-06-19/20 ｜ 基线代码:`b919650f`(B.1+B.2)｜ 全本地 Docker、¥0、mock 上游、无真 key、未 push、未碰 main/wujie 基线
> 执行端实情:本机 = Windows + Docker Desktop(Linux 引擎,从 Windows 直接 `docker`),无需切 WSL。

## 0. 一句话结论
**采集口本地真实运行 = 成功落库**;并且点火额外**揪出一个真实接线缺口**:主 Anthropic `/v1/messages` 转发路径(生产核心路径)在 `b919650f` **漏接了内容采集**——`go test` 单测与静态链路审查都没发现,只有「真请求真运行」暴露出来(正是本步的价值)。

成功取到两类真实落库证据:
- **阶段1(直采)**:真 PG + 真 collector + flag=on + 真 INSERT/SELECT → 库里出现脱敏记录(id=1)。
- **阶段2(端到端)**:真 `curl` → 网关 → mock 上游 → 客户端拿到原样响应(透明性成立)→ **补上漏接的那一行后**,库里出现端到端采集记录(id=2,全归因 + prompt/response 双脱敏)。

## 1. 关键发现(BUG,需在 wujie/trunk 单独修复)
- **现象**:开 flag 后,真 `curl POST /v1/messages`(anthropic 账号)成功转发并写了 `usage_logs`,但 `ai_generation_content` **没有**新行,也无任何采集错误日志。
- **根因(file:line 实证)**:`backend/internal/handler/gateway_handler.go` 有**两个**转发成功闭包:
  - `441 geminiCompatService.Forward` → 闭包 **508** → **有** `CollectGenerationContent`(line 535)。
  - `769 h.gatewayService.Forward`(**主 Anthropic/通用路径**,bedrock/passthrough 也走它)→ 闭包 **908** → **无** 采集调用。
  全仓 `CollectGenerationContent` 调用点仅 2 处:`gateway_handler.go:535`(gemini)与 `gateway_handler_chat_completions.go:282`(CC)。**主 Anthropic messages 路径(908)缺失**。
- **影响**:flag 打开后,`/v1/chat/completions` 与 gemini-compat messages 会采集,但**最核心的 Anthropic `/v1/messages` 主路径不采集**。B.1 的预期范围本就含 messages,属接线遗漏。
- **修复(一行级,已在 throwaway 沙盒验证有效)**:在 `gateway_handler.go:908` 闭包内 `RecordUsage` 之后,镜像 535 增加一段 `CollectGenerationContent`(用该闭包的 `currentAPIKey`/`subject`/`account`/`reqModel`/`requestPayloadHash`/`parsedReq.Body`/`result`)。补上后端到端立即采集成功(见 id=2)。
- **建议**:在 `wujie/trunk` 单独提一个小修复(±11 行)+ 回归;并顺带核查 `openai_gateway_handler.go`(OpenAI 上游路径)是否也需采集(B.3 议)。**本 smoke 任务不改 mainline 代码、不提交**。

## 2. 阶段1 证据(直采,id=1)
手工起一次性 PG(`docker run --rm postgres:18-alpine` @127.0.0.1:5433),`//go:build integration` 白盒测试(`backend/internal/service/smoke_real_pg_integration_test.go`,external `service_test` 包):`repository.ApplyMigrations` → flag=on cfg → 真 `repository.NewGenerationContentRepository` + `service.NewGenerationContentCollector` → `Collect` → 真 SELECT。`go test -tags integration ./internal/service/ -run TestSmokeRealPGContentCapture` = PASS。
```
id=1 model="claude-opus-4-8"
prompt_redacted   = {"messages":[{"content":"帮我给客户回个电话 [PHONE],工单号 [已脱敏],再写一句问候语","role":"user"}],"model":"claude-opus-4-8"}
response_redacted = {"content":[{"text":"好的,已记下电话 [PHONE],稍后联系。","type":"text"}],...}
prompt_bytes=177 response_bytes=189 redaction_version=1
```
证明:flag 配置真生效、脱敏管线真跑(电话→[PHONE]、UUID→[已脱敏](被密钥脱敏整体脱掉,**非**被电话正则切碎)、模型名/普通中文不误伤)、真 PG 落盘。这两环正是单测 mock 掉的。

## 3. 阶段2 证据(端到端,id=2)
- 起栈:host 二进制(从 `b919650f` 构建)指向 smoke-pg(5433)+ smoke-redis(6380),`config.yaml` 开 `gateway.content_capture.enabled=true`(env 不能覆盖该无默认嵌套键,故写进 config.yaml);SSRF 默认就放行 private/http。:8090(避开 :8080 day0)。
- seed(裸 SQL,凭据明文 jsonb、key 明文——已核查无加密/无哈希):user(active,balance=100)+ group(anthropic,active)+ account(**type=`apikey`**,credentials.base_url=`http://127.0.0.1:9099`)+ account_group + api_key(`sk-smoke-localtest-0001`)。**安全**:全库仅此一个账号且指向 mock,`GetBaseURL()` 在 type=apikey+base_url 下返回 mock,不可能触达真实 provider。
- mock 上游:`127.0.0.1:9099` 固定返回带 `usage` 的 Anthropic JSON,正文含唯一标记 `MOCK_REPLY` + 一个电话。
- `curl POST :8090/v1/messages`(Bearer key,非流式,body 含电话+UUID)→ **HTTP 200**,客户端正文 == mock 原样(含 `MOCK_REPLY`,Content-Length 294)→ **客户端透明性成立(真实运行下未被采集改动)**。
- 补上 §1 漏接行后,1s 内落库:
```
id=2 request_id=mock-req-1 model=claude-3-5-sonnet-20241022  (api_key/user/group/account 全部非空=归因正确)
prompt_redacted   = {...,"content":"...回拨客户电话 [PHONE],工单 [已脱敏]...",...}
response_redacted = {"content":[{"text":"MOCK_REPLY 工单已收到,我们会尽快回电 [PHONE] 处理。",...}],...}
prompt_bytes=179 response_bytes=294(== 客户端 Content-Length,tee 抓到的正是客户端可见字节) response_truncated=f
```
证明:真 HTTP → 主路径转发 → handler 闭包(补线后)→ 采集器 → 真库;prompt+response 双脱敏;客户端字节不变。

## 4. 环境说明 / 踩的坑
- 迁移在启动 `setup.AutoSetupFromEnv → repository.ApplyMigrations` 自动跑(幂等);首启因缺 `video_gateway.encryption_key`(拒绝复用 totp key)退出,补进 config.yaml 即过。
- `content_capture.enabled` 无 viper 默认 → `GATEWAY_*` env 经 Unmarshal 不可靠生效,必须写 config.yaml(已用 throwaway `cmd/checkcfg` 实测确认 flag 真为 true)。
- `account.type` 真值是 `"apikey"`(无下划线;域常量,早期侦察误记为 `api_key`)——错值会让 `GetBaseURL` 回退到真 Anthropic,**安全攸关**,已核对。
- psql 控制台对 prompt 字段中文偶现乱码 = Windows 终端编码显示问题,库内为正确 UTF-8(同行 response 字段中文正常 + [PHONE]/[已脱敏] 标记正确为证)。

## 5. git 不动主线证明
- 点火全在独立 detached worktree `D:/sub2api-smoke @ b919650f`;所有临时产物(集成测试、mock、checkcfg、throwaway 补线、config/seed)只在该 worktree,**未 commit、随 worktree 删除**。
- `main` 仍 `69f648e2`、`wujie/trunk` 仍 `b919650f`,主工作树未碰。(清理与校验见执行日志)

## 6. 下一步建议(按价值)
1. **【优先·小修复】** 在 `wujie/trunk` 给 `gateway_handler.go:908`(主 Anthropic 闭包)补上 `CollectGenerationContent`(镜像 535),回归 + Codex 复核。这是「让生产核心路径真采集」的关键一行。
2. 核查 `openai_gateway_handler.go` 是否也需采集(OpenAI 上游路径)。
3. B.3:responses 路径接入(M1-A 既定缓做)。
4. 保留期/清理任务;灰度开 flag。
