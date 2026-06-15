<!-- 单一审查包 / 2026-06-15 / sub2api 分支 phase-3.8.2-overnight-readiness / 零真实调用·零真实凭据·未 push -->

# SUB2API · Seedance 2.0 Blocker 修复 · 单一审查包

- **日期**：2026-06-15
- **仓库 / 分支**：`02_source/sub2api` @ `phase-3.8.2-overnight-readiness`（未新建分支，未 push，未 `git add`）
- **范围**：把"真实Seedance2.0方案_对抗评审.md"的三个 blocker 中 **Go 后端可独立修的部分**修掉；`go test ./...` 全绿；**codex + Claude 双家族评审均已签字 GO**；产出 QCanvas 侧契约核对清单。
- **结论(一句话)**：Go 侧三 blocker 可独立修部分**全部完成并经双家族交叉评审通过**(codex 第一轮抓 3 个真问题已全修),测试全绿,未 push;**唯一留给冒烟**的是 Ark 真实响应字段名核对。
- **铁律遵守声明**：
  - ✅ 全程**零真实调用** Ark / 任何 provider（新增测试用 `httptest` 本地服务器 + dummy key `test-key-not-real`）。
  - ✅ **未接入任何真实 API key**，未读取/打印任何真实凭据 / JWT / secret。
  - ✅ **未 `git push`**，未 `git add .` / `add -A`。
  - ✅ **未触碰 QCanvas 仓库**（QCanvas 侧落点全部转为下方第 7 节核对清单）。
  - ✅ **未写 `/wujie`、未部署**。
  - ✅ **未为绿而绿**：没有改动任何既有测试去迁就实现；新增测试断言的是真实安全行为。
  - ✅ codex 仅做**只读评审**（`codex -s read-only -a never exec`，内容内联、零写盘、零真实调用）；你本人完成 OAuth 登录后由我执行（见第 2、5 节）。

---

## 1. 改了什么（总览）

| 文件 | 类型 | Blocker | 摘要 |
|---|---|---|---|
| `backend/internal/service/video_gateway_adapter.go` | 改 | B1a / B2 / B3a | 上游错误体脱敏（非2xx **+ 200-OK 业务错误**）；create payload 下发 duration/resolution/aspect_ratio；reference_image_url + result_url SSRF 校验；写脱敏审计日志（**写失败 fail-closed `SEEDANCE_AUDIT_LOG_FAILED`**）；smoke gate 强制 URL allowlist |
| `backend/internal/service/video_gateway_redact.go` | 新 | B1a / B1b | `redactVideoUpstreamSecrets`（复用同包 pattern 脱敏器 + 叠加 `AKLT` pattern）+ `appendRedactedVideoEvent`（脱敏审计日志：**既有文件强制 `chmod 0600`、写/降权失败返回 error**） |
| `backend/internal/service/video_gateway_ssrf.go` | 新 | B3a | `validateExternalVideoURL`（**拒绝反斜杠/空白/控制字符 + 拒绝 userinfo 防解析器差异**；先归一化尾点再委托既有 `internal/util/urlvalidator`；https-only + 内网/CGNAT/混淆IP 封锁 + allowlist + localhost/`.local`/`.internal`） |
| `backend/internal/service/video_gateway_types.go` | 改 | B1c | `VideoProviderAccount` 增加 `String()` / `GoString()`（含 `%#v`）/ `LogValue()` 屏蔽 `PlainAPIKey`，并给 `PlainAPIKey` 加 `json:"-"` |
| `backend/internal/service/video_gateway_service.go` | 改 | B3a | 状态层 blocked_reasons 同步加入 URL allowlist 必填项 |
| `backend/internal/repository/video_key_encryptor.go` | 改 | B3b | `video_gateway.encryption_key` 为空时从 totp key fallback 改为**硬失败 return error** |
| `backend/internal/service/video_gateway_security_test.go` | 新 | B1/B2/B3 | 13 个安全行为测试（脱敏含 sk-/AKLT、审计日志、SSRF 16 子用例、key 不外泄、duration 上线、result_url 拒绝/接受、业务错误脱敏、参考图拒绝） |
| `backend/internal/server/routes/api_key_video_gateway_test.go` | 改 | B3a | 成功用例 fixture 补 allowlist env（新必需前提，非弱化断言） |
| `deploy/config.example.yaml` | 改 | B3b | `video_gateway.encryption_key` 文案改"必填/启动即失败"（删除已移除的 totp fallback 描述） |
| `.gitignore` | 改 | B1b | 明确忽略脱敏审计日志路径 |

`git --no-pager diff --stat`（最终，Claude 两轮 + codex 一轮修复后）：**7 个被跟踪文件 +120 / -15**；另有 3 个新文件（含 1 个测试文件）。

---

## 2. Phase 0 — codex 安装与 MCP 接入结果（**含你需要本人登录的那一步**）

### 2.1 已完成（我做的）— codex CLI 现已可运行 ✅
- `npm i -g @openai/codex` 已执行并**已验证可运行**：`codex --version` → **`codex-cli 0.139.0`**。
- 全局 shim 已生成（`C:\Users\浩臣移动工作站\AppData\Roaming\npm\` 下有 `codex` / `codex.cmd` / `codex.ps1`），且该目录**在交互式 shell 的 PATH 内**，`codex` 可直接调用。
- 已用 `codex --help` 核对子命令（0.139.0）：`login`（登录，OAuth）、`mcp-server`（以 stdio 启动 codex 为 MCP server，正是接入所需）、`-s/--sandbox read-only`（只读沙箱）均存在。
- 安装小插曲（已解决，供你了解）：首装后 `...vendor\...\codex.exe` 一度被一个挂起的后台 `npm install` 持有句柄锁住、shim 没生成；我停掉该挂起进程后重装，shim 与 `--version` 均恢复正常。本机另装有 Codex 桌面应用（WindowsApp v26.609），与 npm CLI 互不影响。

### 2.2 🛑 **HARD STOP：请你本人执行（OAuth/账号登录，我不能代劳）**

在你自己的终端里：

```powershell
codex login          # 【只有你能做】登录 OpenAI 账号（OAuth）；我无法代你完成账号授权
```

登录成功后告诉我，我再执行（**只读评审**）：

```bash
claude mcp add codex -- codex mcp-server -c sandbox_mode="read-only" -c approval_policy="never"
# 然后我以只读方式让 codex 对本次 diff 做【跨家族】评审，补齐第 5 节，完成双签字
```

- codex 在本任务中**只做只读评审**：只让它读 `git diff` / 新文件做裁决，绝不让它写代码、绝不碰真实调用 / 真实凭据。
- 子命令名已核实无误（`login` / `mcp-server`）。

---

## 3. Phase 1 — 三个 Blocker 的修复（各自 diff + 理由）

> 完整统一 diff 见下；理由逐条说明为什么这样改、为什么安全、留了什么给冒烟。

### B1 — 密钥泄露（最高优先）

**B1a 上游错误体脱敏**（`video_gateway_adapter.go` create:213-225 / poll:294-305）
- **理由**：原代码把 Ark 非 2xx 响应体 `truncate(string(respBody),500)` 原样塞进 error，经 `failTask` 落 `task.ErrorMessage`（DB）+ `video_task_events.Message`（DB）并随响应回前端。401/403 体可能回显 `Authorization`/`Bearer`/key 前缀 → 真实泄露通道。现在在嵌入前先 `redactVideoUpstreamSecrets(...)`。
- **为什么用 `redactContentModerationSecrets` 而非 `internal/util/logredact`**：`logredact` 是**按 key 脱敏**（只对 `access_token` 等已知键名的值打码），对"错误消息正文里内联出现的 `Bearer sk-xxx`"会**漏网**；而 `content_moderation_redact.go` 是**按模式脱敏**（含 `Bearer\s+<token>`、`sk-`/key/JWT、长 hex/base64、URL），正是自由文本正文该用的。脱敏偏激进（连 URL 都打码）——对"即将落库并回前端的冒烟期错误体"，过度脱敏是更安全的失败方向。

**B1b 脱敏审计日志真正落地**（新 `video_gateway_redact.go` + `.gitignore`）
- **关键事实**：原仓库**根本没有** `SUB2API_VIDEO_REDACTED_EVENT_LOG` 的写入实现——env 只被 smoke gate 当"非空即放行"的绊线，`redacted_event:true` 只是 payload 里一个标志位。即"脱敏日志"是个**幽灵**。
- **修法**：实现 `appendRedactedVideoEvent(phase,statusCode,rawBody)`：写前 `redactVideoUpstreamSecrets`；文件 `O_CREATE|O_WRONLY|O_APPEND` + **0600**；env 为空则 no-op（真实路径上 gate 已保证非空）。在 create/poll 读到响应体后都记一行（成功+失败都记）。
- **路径治理**：`.gitignore` 增加 `backend/video-redacted-events.log` 与 `*.redacted-events.log`；并在代码注释要求运维把 env 指向 `*.log` 或 `backend/data/`（两者本就被忽略）。

**B1c `VideoProviderAccount` 屏蔽明文 key**（`video_gateway_types.go:79-101`）
- **理由**：`PlainAPIKey` 是解密后的真实上游凭据；任何 `%+v`/`slog`/panic dump 都可能打印。新增 `String()`（fmt.Stringer）+ `LogValue()`（slog.LogValuer），值接收者 → 指针也走同一屏蔽。JSON 序列化走 struct tag，不受影响。

### B2 — 成本/契约（Go 侧能做的部分）

**create payload 显式下发 duration（+resolution/aspect_ratio）**（`video_gateway_adapter.go:171-189`）
- **理由**：原 payload 只有 `model`+`content`+`negative_prompt`，smoke gate 校验的 `Duration∈[1,5]` 因此是**纸面约束**，真实时长由 Ark 默认值决定，与 §3 成本估算脱钩。现按需注入。
- **明确留给冒烟（未假设）**：Ark 真实 JSON 字段名（`id`/`status`/`content.video_url`，以及 duration/resolution 的确切字段名）**本任务不假设**，代码注释已标注"UNVERIFIED，冒烟第一步核对"。这是 B2 中**必须等真实响应**的部分，不在本仓库强行断言。

### B3 — 鉴权/SSRF（Go 侧能做的部分）

**B3a 两个外部可控 URL 的 SSRF 校验**（新 `video_gateway_ssrf.go`；接入 create:159-163、poll:331-344）
- **理由**：`reference_image_url`（转发给 Ark 当 image_url）与 `result_url`（信任 Ark 返回、前端播放 + 写 Day0）是两个信任边界。
- **修法**：`validateExternalVideoURL` **委托给既有 `internal/util/urlvalidator.ValidateHTTPSURL`**（不重复造 SSRF 轮子）——https-only + 封锁 localhost/私网/loopback/link-local/unspecified + 通配 allowlist + 端口校验；再叠加非空校验与 `.local/.internal` 内网 TLD 封锁。`reference_image_url` 不合法 → `BadRequest` 直接挡掉真实调用；`result_url` 不合法 → 任务置 failed 且**不存 URL**。
- **DNS rebinding 故意不在此做**：这两个 URL 都不由本进程拉取（Ark 拉 reference，前端播 result），且校验函数里做实时 DNS 是 TOCTOU/不确定性陷阱——转入冒烟清单。

**B3b 加密 key 的 dev fallback 改硬失败**（`video_key_encryptor.go:18-23`）
- **理由**：原逻辑 `video_gateway.encryption_key` 为空时静默 fallback 到 `totp.encryption_key` 仅 Warn → 密钥域混淆。现直接 `return error`。
- **爆炸半径已确认安全**：grep 确认**无任何测试/wire 在测试期构造真实 encryptor**（`go test ./...` 不触发它），config 校验把该 key 视为"非空才校验"。所以硬失败**只影响运行时启动**（正是评审要的），不破坏 `go test` / `go build`。运维需在 `config.yaml` 显式配置该 key（见第 8 节运维须知）。

### 3.x 完整统一 diff（被跟踪文件）

```diff
diff --git a/.gitignore b/.gitignore
@@ -112,6 +112,11 @@ backend/config.yaml
 deploy/config.yaml
 backend/.installed
 
+# 视频网关脱敏事件日志（SUB2API_VIDEO_REDACTED_EVENT_LOG 指向的审计文件）。
+# 即便已脱敏也绝不入库；务必指向 *.log 或 backend/data/ 下的受限文件。
+backend/video-redacted-events.log
+*.redacted-events.log
+
 # ===================
 # 其他
 # ===================

diff --git a/backend/internal/repository/video_key_encryptor.go b/backend/internal/repository/video_key_encryptor.go
@@ import (
 	"encoding/hex"
 	"fmt"
-	"log/slog"
 	"strings"
@@
 // NewVideoKeyEncryptor creates the dedicated reversible encryptor for video
-// upstream API keys. Production deployments should set video_gateway.encryption_key.
+// upstream API keys. video_gateway.encryption_key is REQUIRED.
+//
+// Previously an empty key silently fell back to totp.encryption_key with only a
+// Warn. That is a key-domain-confusion risk: a production deployment that forgot
+// to configure the dedicated key would encrypt real Seedance credentials under
+// the TOTP key, and the Warn is easy to miss. We now hard-fail instead.
 func NewVideoKeyEncryptor(cfg *config.Config) (service.VideoKeyEncryptor, error) {
 	keyHex := strings.TrimSpace(cfg.VideoGateway.EncryptionKey)
 	if keyHex == "" {
-		keyHex = strings.TrimSpace(cfg.Totp.EncryptionKey)
-		slog.Warn("video_gateway.encryption_key is empty; using dev-only fallback from totp.encryption_key")
+		return nil, fmt.Errorf("video_gateway.encryption_key is required: refusing to fall back to totp.encryption_key (key-domain confusion); configure a dedicated 32-byte (64 hex char) key")
 	}

diff --git a/backend/internal/service/video_gateway_adapter.go b/backend/internal/service/video_gateway_adapter.go
@@ CreateTask: reference_image_url SSRF guard
 	content := []map[string]any{{"type": "text", "text": task.Prompt}}
 	if task.ReferenceImageURL != "" {
+		if err := validateExternalVideoURL(task.ReferenceImageURL); err != nil {
+			return nil, infraerrors.BadRequest("VIDEO_UNSAFE_REFERENCE_URL",
+				"reference_image_url failed SSRF/allowlist validation: "+err.Error())
+		}
 		content = append(content, map[string]any{"type": "image_url", "image_url": map[string]string{"url": task.ReferenceImageURL}})
 	}
@@ CreateTask: B2 send generation params
 	if task.NegativePrompt != "" {
 		payload["negative_prompt"] = task.NegativePrompt
 	}
+	// Explicitly send generation parameters so the smoke-gate duration cap
+	// (1-5s) is applied upstream instead of relying on Ark's default duration.
+	// NOTE: exact Ark field names UNVERIFIED — confirm at first real smoke (B2).
+	if task.Duration > 0 {
+		payload["duration"] = task.Duration
+	}
+	if task.Resolution != "" {
+		payload["resolution"] = task.Resolution
+	}
+	if task.AspectRatio != "" {
+		payload["aspect_ratio"] = task.AspectRatio
+	}
@@ CreateTask: B1a/B1b audit + redact (poll path identical with phase "poll")
 	respBody, err := io.ReadAll(resp.Body)
 	...
+	appendRedactedVideoEvent("create", resp.StatusCode, string(respBody))
+
 	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
+		// Redact upstream body before it reaches DB / events / API response.
 		return nil, infraerrorsUnavailable("SEEDANCE_CREATE_UPSTREAM_ERROR",
-			fmt.Sprintf("... %s", ..., truncate(string(respBody), 500)))
+			fmt.Sprintf("... %s", ..., truncate(redactVideoUpstreamSecrets(string(respBody)), 500)))
 	}
@@ PollTask: B3a result_url SSRF guard (ResultURL removed from literal, set after validation)
 	result := &VideoAdapterResult{
 		UpstreamTaskID: parsed.ID,
 		Status:         a.NormalizeStatus(parsed.Status),
-		ResultURL:      parsed.Content.VideoURL,
 		Payload: map[string]any{ ... },
 	}
+	if parsed.Content.VideoURL != "" {
+		if err := validateExternalVideoURL(parsed.Content.VideoURL); err != nil {
+			result.Status = VideoStatusFailed
+			result.ErrorMessage = "upstream result_url failed validation: " + err.Error()
+			result.Payload["result_url_rejected"] = true
+			return result, nil
+		}
+		result.ResultURL = parsed.Content.VideoURL
+	}

diff --git a/backend/internal/service/video_gateway_types.go b/backend/internal/service/video_gateway_types.go
@@ import ( "context"
+	"fmt"
+	"log/slog"
 	"time" ... )
@@ after VideoProviderAccount struct
+func (a VideoProviderAccount) String() string {
+	return fmt.Sprintf("VideoProviderAccount{ID:%d, Provider:%q, DisplayName:%q, Enabled:%t, APIKeyConfigured:%t, MaskedKey:%q, PlainAPIKey:[REDACTED]}",
+		a.ID, a.Provider, a.DisplayName, a.Enabled, a.APIKeyConfigured, a.MaskedKey)
+}
+func (a VideoProviderAccount) LogValue() slog.Value {
+	return slog.GroupValue(
+		slog.Int64("id", a.ID), slog.String("provider", a.Provider),
+		slog.String("display_name", a.DisplayName), slog.Bool("enabled", a.Enabled),
+		slog.Bool("api_key_configured", a.APIKeyConfigured),
+		slog.String("masked_key", a.MaskedKey), slog.String("plain_api_key", "[REDACTED]"),
+	)
+}
```

> 上面是阅读友好的精简 diff。**逐行权威源**为工作树本身：`git --no-pager diff -- backend/ .gitignore` 加 3 个新文件（评审 agent 已对照真实工作树逐行核对）。

> 三个新文件（`video_gateway_redact.go`、`video_gateway_ssrf.go`、`video_gateway_security_test.go`）为新增，未在上面的 `git diff` 中显示；其完整内容已在工作树中，评审 agent 已逐行读过（见第 6 节）。
>
> ⚠️ 上面的精简 diff 为**第一轮**形态；经 **Claude 两轮 + codex 一轮**对抗评审后,工作树**还包含**以下加固(见第 5、6 节):
> **Claude 轮**:① create+poll 的 **200-OK 业务错误体也脱敏**;② smoke gate + service 状态层**强制 `SUB2API_VIDEO_URL_ALLOWLIST`**(fail-closed);③ `validateExternalVideoURL` 先剥尾点再委托 urlvalidator + 阻断混淆 IP/CGNAT/0.0.0.0/8 + `localhost` 名称判定;④ `redactVideoUpstreamSecrets` 叠加 `AKLT` pattern;⑤ `deploy/config.example.yaml` 改"必填/启动即失败";⑥ `routes` 成功用例 fixture 补 allowlist env。
> **codex 轮(跨家族新发现并已修)**:⑦ `validateExternalVideoURL` **拒绝反斜杠/空白/控制字符 + 拒绝 userinfo**(防 `\@` 解析器差异 SSRF);⑧ `appendRedactedVideoEvent` 对既有文件 **`chmod 0600`、写/降权失败返回 error**,create/poll 审计失败时返回 `SEEDANCE_AUDIT_LOG_FAILED`(fail-closed);⑨ `VideoProviderAccount` 加 **`GoString()`(`%#v`)+ `PlainAPIKey json:"-"`**。
> 最终被跟踪改动:**7 个文件 +120/-15**(新增 `video_gateway_service.go`、`api_key_video_gateway_test.go`、`deploy/config.example.yaml`)+ 3 个新文件;**逐行权威源始终是工作树 `git --no-pager diff` + 3 个新文件**。

---

## 4. `go test ./...` 原文（全绿）

构建：`go build ./...` → **exit 0**（无输出即成功）。

`go test ./...`（节选关键包，**全绿，exit 0，零 FAIL / 零 panic**）：

```
ok  	github.com/Wei-Shaw/sub2api/cmd/server	0.516s
ok  	github.com/Wei-Shaw/sub2api/internal/config	(cached)
ok  	github.com/Wei-Shaw/sub2api/internal/handler	21.186s
ok  	github.com/Wei-Shaw/sub2api/internal/handler/admin	0.813s
ok  	github.com/Wei-Shaw/sub2api/internal/repository	1.817s
ok  	github.com/Wei-Shaw/sub2api/internal/server/routes	2.093s
ok  	github.com/Wei-Shaw/sub2api/internal/service	43.647s
ok  	github.com/Wei-Shaw/sub2api/internal/setup	0.553s
ok  	github.com/Wei-Shaw/sub2api/internal/util/urlvalidator	(cached)
ok  	github.com/Wei-Shaw/sub2api/migrations	(cached)
... (其余包全部 ok / [no test files]，无一 FAIL)
=== FULL SUITE EXIT 0 ===  ===  any FAIL/panic? NONE — all green
```

定向门禁命令（指定的验证命令）：

```
$ go test ./internal/service -run TestVideoGateway -count=1
--- PASS: TestVideoGatewayMockWorkerSuccessAndFailure
--- PASS: TestVideoGatewayAutoRouteSkipsUnavailableAccounts
--- PASS: TestVideoGatewayTaskOwnershipBoundary
--- PASS: TestVideoGatewayTaskListFiltersByEmployeeUnlessAdmin
ok  	github.com/Wei-Shaw/sub2api/internal/service

# 新增安全测试（同包，共 13 个 func / 含表驱动 18 子用例）：
--- PASS: TestRedactVideoUpstreamSecrets            (含 sk- 与 AKLT 两类脱敏断言)
--- PASS: TestAppendRedactedVideoEvent              (含 fail-closed 返回值 + 非 Windows 0600 权限断言)
--- PASS: TestValidateExternalVideoURL (18 子用例全 PASS：十进制/十六/八进制 IP、尾点 loopback/localhost、CGNAT、前导0合法主机放行、反斜杠解析器差异、userinfo)
--- PASS: TestValidateExternalVideoURLAllowlist     (含合法 FQDN 带尾点放行)
--- PASS: TestVideoProviderAccountRedactsPlainKey   (含 %v/%+v/%s/%#v/json/slog 全不泄露)
--- PASS: TestSeedanceCreateSendsDurationAndAuditsRedacted
--- PASS: TestSeedanceCreateRedactsUpstreamErrorBody
--- PASS: TestSeedancePollRejectsUnsafeResultURL
--- PASS: TestSeedancePollAcceptsSafeResultURL
--- PASS: TestSeedanceCreateRedactsBusinessErrorMessage   (200-OK error.message 脱敏)
--- PASS: TestSeedancePollRedactsBusinessErrorMessage     (200-OK error.message 脱敏)
--- PASS: TestSeedancePollRedactsUpstreamErrorBody        (poll 非 2xx 脱敏)
--- PASS: TestSeedanceCreateRejectsUnsafeReferenceURL     (拒绝且不发起上游调用)
```

> 注：以上为最终（两轮修复后）全绿状态——`go build ./...` / `go vet ./internal/service/...` / `go test ./...` 均 exit 0、零 FAIL/panic。`routes` 包的 `TestAPIKeyVideoGatewaySeedanceTrialSuccess` 因新增 allowlist 前提补了一行 `setEnv(...URL_ALLOWLIST...)`（提供新的必需前提，**非弱化断言**）。

---

## 5. codex（GPT 家族）只读评审裁决全文

已完成 ✅。用户本人用 `codex login --device-auth` 登录(`Logged in using ChatGPT`)。
执行方式说明:本机 Claude 是桌面应用、shell 无 `claude` 命令,且 codex 的 Windows 只读沙箱有初始化 bug(`windows sandbox orchestrator_helper_incomplete`),无法让 codex 自己跑 `git diff`/读文件。故改为把**真实 diff + 3 个新文件**全部喂给 `codex -s read-only -a never exec`(只读、不批准任何写、内容内联),codex 仅做分析、零写盘、零真实调用。codex 版本 `codex-cli 0.139.0`。

### 5.1 codex 第一轮:**NO-GO** —— 抓到 3 个 Claude 两轮都漏掉的真问题

> **codex 裁决(第一轮,原文要点):** Found material issues. I would not clear the Go-side security scope yet.
> 1. **HIGH — `video_gateway_ssrf.go` + `adapter.go` URL 解析器差异绕过**:示例 `https://169.254.169.254\@x.cn-beijing.volces.com/a.mp4` —— Go 的 `url.Parse` 把 host 解析成白名单内的 `x.cn-beijing.volces.com`,但浏览器/WHATWG 把 `\` 当 `/`,真实目标是 `169.254.169.254`;而 adapter 存/转发的是**原始 raw URL**。Fix: reject `\`, reject `u.User != nil`, 使用规范化后的 URL。
> 2. **MED — `video_gateway_redact.go` 审计日志不保证 0600**:`os.OpenFile(...,0o600)` 只在**创建**时套权限,文件已存在(如 0644)则不变;且写失败只 warn,smoke 仍会继续(没有审计证据)。Fix: 对既有文件 chmod 0600,写失败让真实 smoke 失败。
> 3. **MED — `video_gateway_types.go` PlainAPIKey 仍可经 `%#v`/JSON 泄露**:`String()/LogValue()` 没覆盖 `fmt.Sprintf("%#v", acc)`,且 `PlainAPIKey` 是导出字段无 `json:"-"`。Fix: 加 `GoString()` + `json:"-"`。
> **FINAL VERDICT: NO-GO** —— B1/B3 improved, but the URL parser-differential gap and audit-log guarantees need closing.

**3 项全部已修复**(如下),`go build`/`go vet`/`go test ./...` 复测全绿:
- HIGH 解析器差异:`validateExternalVideoURL` 拒绝反斜杠/空白/控制字符 + 拒绝 `u.User != nil`。
- MED 0600/fail-closed:`appendRedactedVideoEvent` 对既有文件 `f.Chmod(0o600)`,并改为**返回 error**;create/poll 在审计写失败时返回 `SEEDANCE_AUDIT_LOG_FAILED`(没审计就不放行真实调用)。
- MED `%#v`/JSON:新增 `GoString()` + 给 `PlainAPIKey` 加 `json:"-"`。

### 5.2 codex 第二轮(复查):**GO-WITH-FOLLOWUPS** ✅

> **codex 裁决(第二轮,原文要点):** 结论:3 个前次问题基本已闭合;我没有发现新的 Go 侧阻断级真实缺陷。
> 1. HIGH URL parser-differential:**已闭合**(拒绝反斜杠/空白/控制字符 + 拒绝 userinfo)。
> 2. MED audit log 0600 + fail-closed:**已闭合**(`f.Chmod(0o600)` + `appendRedactedVideoEvent` 返回 error,审计失败时 create/poll 返回 `SEEDANCE_AUDIT_LOG_FAILED`)。
> 3. MED PlainAPIKey via `%#v`/JSON:**已闭合**(`json:"-"` + `String()`/`GoString()`/`LogValue()` 覆盖普通 fmt、`%#v`、slog)。
> **剩余非阻断 follow-up:** (a) `duration`/`resolution`/`aspect_ratio` 已下发,但 Ark 字段名仍需首次真实 smoke 确认(本轮允许 UNVERIFIED,非 NO-GO);(b) `f.Chmod` 失败原只 warn,从严格 fail-closed 看最好也返回 error(风险低)。
> **FINAL VERDICT: GO-WITH-FOLLOWUPS** —— 前次 3 项已实质闭合,Go 侧可进入受控 smoke。

**codex 第二轮的 follow-up (b) 也已立即闭合**:`f.Chmod(0o600)` 失败现在**返回 error**(不再只 warn),审计彻底 fail-closed;复测仍全绿。follow-up (a) 是 Ark 字段名,属冒烟环节(非本仓库代码任务)。

### 5.3 codex 签字

> **codex(GPT 家族)只读跨家族评审:GO-WITH-FOLLOWUPS** —— 前一轮 3 项 must-fix 已实质闭合,未发现新的 Go 侧阻断级缺陷;唯一剩余为"首次真实 smoke 核对 Ark 字段名"。

---

## 6. Claude 对抗评审裁决全文（多镜头 → 对抗复核 → 综合）

> 评审方式：4 个独立镜头 reviewer（B1 密钥泄露 / B3 SSRF+encryptor / B2+回归 / 测试诚信）各自对照**真实 diff**找问题 → 每条 finding 由独立 skeptic agent **试图反驳**（只有扛住复核才算真问题）→ 综合裁决。

### 6.1 第一轮：4 镜头 → 20 raw findings → 对抗复核 → **6 项 confirmed**

镜头：B1 密钥泄露 / B3 SSRF+encryptor / B2+回归 / 测试诚信。每条 finding 由独立 skeptic agent 试图反驳，只有扛住的才计入。confirmed 6 项（均已修复，见第 3 节及下）：

| # | Sev | Finding | 修复 |
|---|-----|---------|------|
| F1 | **HIGH** | 200-OK 业务错误体 `error.message` 未脱敏直落 DB/API（原修复只覆盖非 2xx） | create+poll 两处 `redactVideoUpstreamSecrets` |
| F2 | LOW | 脱敏器漏掉无分隔符的 AKLT 式 key | 叠加 `volcengineAccessKeyPattern` |
| F3 | **HIGH** | 默认无 allowlist 时 SSRF 可绕（十进制/十六/八进制 IP、尾点、CGNAT 100.64/10） | 闸门强制 allowlist（fail-closed）+ 主机加固 |
| F4 | LOW | `config.example.yaml` 仍写已删除的 totp fallback | 改为"必填/启动即失败" |
| F5 | MED | poll 非 2xx 脱敏无测试 | 新增 `TestSeedancePollRedactsUpstreamErrorBody` |
| F6 | LOW | reference_image_url 拒绝无 adapter 级测试 | 新增 `TestSeedanceCreateRejectsUnsafeReferenceURL` |

### 6.2 第二轮：re-verify 6 项 + 对新加固做新鲜对抗扫描

结果：**6 项中 5 项 NONE**；F2 残留 NIT（代码已修，缺一条专门断言 AKLT 的测试）。新鲜扫描确认 **3 项新 finding（全 LOW/NIT）**，其中 N2 是我引入的**可用性回归**（合法 allowlist FQDN 带尾点被误拒）。第二轮 Claude 裁决全文如下：

> #### Final Claude-Family Verdict — Round 2（GO-SIDE / sub2api, B1·B2·B3）
> **裁决：GO-WITH-FOLLOWUPS** — 6 项 round-1 blocker 代码层面全部解决，生产（受闸门）路径 fail-closed 且 SSRF 加固，未引入新的 must-fix；follow-up 为 1 条测试 NIT + 3 条 `validateExternalVideoURL` 边界 LOW/NIT，在受闸门生产路径上均不可利用。
>
> **6 项 re-verify**：F1 ✅NONE（两个 200-OK 出口都过脱敏；`qcanvasVideoContractStatus` 不二次嵌入原文，非泄露通道）。F2 ✅代码修复，残留 NIT（无测试喂 AKLT token，现有测试用 sk- 前缀被旧 pattern 命中，新 pattern 无回归护栏）。F3 ✅NONE（闸门 fail-closed + 加固双机制；`TestValidateExternalVideoURL` 在**不设 allowlist** 下跑松分支全过）。F4 ✅NONE（stale 文案已删，与运行时硬失败一致）。F5/F6 ✅NONE（真实驱动 adapter，dummy key 无真实调用，去掉守卫即挂）。
>
> **3 项新 finding**：N1(LOW) 尾点命名主机 `localhost.` 在无 allowlist 松分支可绕（加固在 urlvalidator 调用之后才 strip，未复跑 localhost 名称规则）——生产强制 allowlist 下 `localhost.` 不匹配后缀故被挡。N2(LOW) 合法 allowlist FQDN 带尾点（`x.volces.com.`）被误拒（可用性，非安全洞——尾点只会在后缀内放松匹配，不会让非白名单主机匹配）。N3(NIT) `isObfuscatedNumericHost` 误伤 RFC-1123 合法的前导 0 全数字 label（如 `007.cdn.example.com`），fail-closed 不是绕过。**N1+N2 同根**：尾点归一化发生在委托 urlvalidator **之后**而非之前。
>
> **强制前提**：① 这是**单家族（Claude）**裁决，codex/GPT 跨家族复核**待你 OAuth 登录**，未做；② **Ark 响应字段名契约（`id`/`status`/`content.video_url`/duration 字段名）本轮有意未校验，必须在首条真实冒烟核对**——代码已显式注释标注 UNVERIFIED；全程 httptest+dummy key，零真实调用零真实 key。
>
> **follow-ups（不阻断 GO，须跟踪）**：#1 加 AKLT 脱敏测试；#2 把尾点归一化移到 urlvalidator **之前** 并复跑 localhost 名称规则（同时解决 N1+N2）；#3 收紧八进制 label 启发式（仅当整主机 `onlyDigitsAndDots` 才判八进制）；#4 冒烟核对 Ark 字段名与真实 AKLT key 形状。
>
> **SIGN-OFF：GO-WITH-FOLLOWUPS** — 6 项 round-1 blocker 全解决，无新 must-fix；受闸门路径可发，跟踪 #1–#4，Ark 字段名冒烟核对，待 codex 跨家族复核。

### 6.3 第三轮：机械修复 round-2 follow-ups #1–#3（已加测试验证）

第二轮的 follow-up #1–#3 我**已立即修复并加测试**（#4 属冒烟、非代码）：

| follow-up | 修复 | 验证测试 |
|---|---|---|
| #1 AKLT 测试 NIT | `TestRedactVideoUpstreamSecrets` 增加无分隔符 `AKLTabc123DEF456ghi789Zxy` 经 `redactVideoUpstreamSecrets` 必被脱敏的断言 | ✅ PASS |
| #2 N1+N2 尾点（同根） | `validateExternalVideoURL` 改为**先**剥尾点并重建 URL **再**委托 `urlvalidator`，且自有防御层加 `localhost`/`.localhost` 名称判定 | ✅ `trailing dot localhost` 拒绝 + 合法 `x.volces.com.` 带 allowlist 放行 |
| #3 N3 八进制启发式 | `isObfuscatedNumericHost` 去掉单 label 前导 0 规则，仅当整主机 `onlyDigitsAndDots` 才判八进制溢出 | ✅ `007.cdn.example.com` 放行；`0177.0.0.1` 仍拒绝 |

修复后 `go build ./...`、`go vet`、`go test ./...` **再次全绿**。**剩余仅 follow-up #4（Ark 字段名留待冒烟，非本仓库代码任务）** 与 **codex 跨家族复核（待你登录）**。

### 6.4 Claude 最终签字

> **Claude 对抗评审（两轮 + 机械修复）：sub2api Go 侧 B1/B2/B3 范围 —— GO。**
> 6 项 round-1 confirmed + 3 项 round-2 新 finding + F2 测试缺口**全部已修并由确定性测试覆盖**；`go build`/`go vet`/`go test ./...` 全绿；零真实调用、零真实凭据、未 push。
> **codex/GPT 跨家族复核现已完成**（第 5 节：第一轮 NO-GO 抓 3 个真问题 → 全修 → 第二轮 GO-WITH-FOLLOWUPS）。**双家族签字达成。**
> **唯一固有前提**：**Ark 响应字段名必须在首条真实冒烟核对**（代码已注释 UNVERIFIED）—— 这是冒烟环节,非本仓库代码任务。

---

## 7. QCanvas 侧契约核对清单（拿去 QCanvas 仓库单独核）

> 本任务在 `sub2api` 目录**看不到** QCanvas/Hono/React 代码，故方案里所有"A 侧"落点都无法在此验证，全部转成下面这张清单。每条标注"需在 QCanvas 仓库验证什么 + 不验证的风险"。

**共 23 条**，按落点分组。每条：`[严重度] ID — 落点` ＋ 在 QCanvas 仓库要验证什么 ＋ 不验证的风险。
> 说明：本仓库 Go 侧已对 `result_url` / `reference_image_url` 做了 SSRF 校验（B3），但 **A 侧仍应独立校验**（纵深防御）——见 SEC1/SEC2。

### 一、Hono / 前端契约落点（Q1–Q7）

- **[MED] Q1 — dry-run→real 切换（`studioV2RealTaskAdapter.ts:359-363`）**：确认翻到 `real` 的**唯一**输入是浏览器本地 `studioV2RealChainReady`（query/localStorage）+ `studioV2MockRealEnabled`；无 env/服务端/构建期路径；分享的 URL query 或共享机 localStorage 不能静默为他人开启真实调用。grep 全仓所有 `studioV2RealChainReady` 读写以枚举每个 setter。**风险**：任何可持久化/可分享的开关把"真实调用边界"前移到前端，前端开关不是闸门。
- **[HIGH] Q2 — provider+trialMode allowlist（`studioV2RealTaskAdapter.ts:83-100`,`assertProviderAllowed:98`）**：确认客户端只发 `provider:'seedance',trialMode:'tiny_real',allowRealCalls:true`，且 `assertProviderAllowed` 只放行 seedance+tiny_real；其余 provider/trialMode 抛错；不与 Q7 的 `z.literal(false)` 硬契约冲突。**风险**：放行过宽会让其他 full-real provider 走 trial 路绕过最小爆炸半径设计。
- **[HIGH] Q3 — Hono 路由 trial gate（`sub2api.routes.ts:184-224`）**：确认强制 `SUB2API_VIDEO_REAL_SMOKE_ENABLED==='1'`、`SUB2API_REAL_HUMAN_AUTHORIZED==='1'`、`trialMode==='tiny_real'`，否则 403 `SUB2API_VIDEO_PROVIDER_BLOCKED`；挂在 `/sub2api`（`app.ts:332`）authMiddleware 后并捕获 `userId`（→ created_by 链路候选）；**每请求实时读 env**（非启动缓存），以便把 env 置 0 能即时停真实调用（Hono 侧 kill-switch）。**风险**：任一条件缺失/松散真值 → 闸门失效开放；env 启动缓存 → 关不掉。
- **[HIGH] Q4 — Hono→Go 代理 base_url + 员工 key 源（`sub2api.video-mock-gateway.service.ts:34-36,72-78,143-155`）**：确认 `base_url` 只来自 env/Worker binding `SUB2API_BASE_URL`，URL 恒为 `${SUB2API_BASE_URL}/v1/video/tasks` 无客户端可控段；`SUB2API_API_KEY` 仅服务端读且**不入日志**；`SUB2API_VIDEO_MOCK_GATEWAY_ENABLED` 真正门控。**风险**：URL 任一段被客户端影响 → 带员工 key 的 open/SSRF 代理；key 进 access log/APM → 共享凭据泄露。
- **[HIGH] Q5 — auth-contract split（`apps/web/src/api/server.ts:6249-6273`,`withAuth:35-41`）**：确认前端 JWT 只认证 浏览器→Hono，不透传给 Go 上游；并追踪 Hono 到底向 Go 传了什么身份（仅共享 `SUB2API_API_KEY`？还是带 JWT 派生的终端用户 id？）——决定真实终端用户身份是否到达 Go（→ CB1）。**风险**：只有共享员工 key 到 Go → Go 无法区分终端用户（见 CB1）。
- **[MED] Q6 — Day0 回写（`studioV2BusinessBridge.ts:260-331`→`day0CandidateStore.ts:136-169`）**：确认真实成功快照写 Day0 时 `resultSource==='sub2api-real-trial'`、`persistable:false`、`reuseAllowed:false`，key 正确，80 条上限淘汰可接受；尤其核对 Go poll 响应（status/result_url/id）字段到快照字段的映射（见 XF 系列）。**风险**：映射错 → 真实成功不落 Day0 或落空 URL；persistable 未钉 false → 被误当 /wujie 暂存。
- **[HIGH] Q7 — `allowRealCalls z.literal(false)` 硬契约不可动（`StudioV2Workbench.tsx:200-201`,`studioV2Lifecycle.ts:24`,`studio-v2-mock-real.service.ts:23-24,35-37`）**：确认这些 `literal(false)` 只约束通用 full-real 链（Seedream/Kling），seedance tiny-real **不走**这些 schema；开启冒烟不需要改这些字面量。**风险**：若 seedance 走这些 schema，开冒烟就被迫改硬安全契约，连带解锁 Seedream/Kling full-real。

### 二、跨字段契约（XF1–XF6）

- **[HIGH] XF1 — taskId int64↔string（`...gateway.service.ts:119-126`）**：确认 Go 返回的数字 id（int64）在 JS 端**强制字符串化**且 >2^53 无精度丢失，用于 poll URL。**风险**：当作 JS number → 大 id 精度丢失 → poll 命中错任务 → 挂起超时。
- **[HIGH] XF2 — `id` vs `task_id` 字段名**：确认 A 侧读 Go 创建响应里的 `id`（非 `task_id`）作为任务标识并复用于 poll；并确认该字段名假设对**真实 Ark 响应**也成立（Go adapter 是 skeleton/未验证真实 Ark JSON）。**风险**：读错字段 → poll URL `/tasks/undefined` 永不解析。
- **[HIGH] XF3 — 丢弃 upstream_task_id**：确认 A 侧 poll 用的是 **Go 自己的任务 id**（非 Ark upstream id），从而丢 upstream_task_id 不破坏 A 的轮询；同时 Go 侧若 `UpstreamTaskID` 为空（Ark 字段名不符）其 poll URL 会变 `/tasks/`。**风险**：轮询目标为空 → 真实结果取不回。
- **[HIGH] XF4 — status/error 枚举对齐（`agents-tool-bridge.generate-video-to-canvas.ts:243-310`(P2) 与 `...gateway.service.ts:160-171`(P1)）**：确认 A 期望的 `queued/running/succeeded/failed` 与 Go 返回**逐字匹配**；succeeded-但无 url 当独立错误（502）；failed 首 poll 即终止（break+单次 throw 不重试）。**风险**：枚举漂移 → 死循环 / 误判 / 接受无 URL 的"成功"写坏 Day0。
- **[HIGH] XF5 — 创建 payload 形状 A→B（对 `video_handler.go:37-50` `apiKeyVideoTaskCreateRequest`）**：核对字段名/大小写/枚举（provider、trial_mode、task_type 必填、prompt 必填、duration/resolution/aspect_ratio/reference_image_url）；**确认 A 是否真把 duration 发给 Go**（Go 侧已下发给 Ark，但要确认全链从 A 起就带 duration）。**风险**：缺/错必填 → 400 冒烟跑不起；缺 duration → 成本上限纸面化。
- **[MED] XF6 — base_url 来源（Hono `SUB2API_BASE_URL` vs Go Ark fallback）**：确认有两个不同 base_url，A 只控制 Hono→Go 的那个（指向 Go 后端，**非** Ark）；A 绝不发 Ark URL 或 base 覆盖。**风险**：误指 Ark → 绕过 Go 闸门/worker/加密。

### 三、wujie 边界（WB1–WB2）

- **[HIGH] WB1 — real-success 路径 `wujieWriteCount:0`（`studioToWujieAdapter.ts:102-150`,`computeWujieBlockHints:419-436`）**：用**真实成功**快照（`resultSource='sub2api-real-trial'`、真实 https result_url、objectKey 无 mock/dry-run 前缀、mock/dryRun 标记为 false）驱动一条端到端，断言 `wujieWriteCount:0` / `persistStatus:'not-persistable'`。**风险**：边界保证目前只对 mock/dry-run 证明过；真实成功时标记翻 false，若 not-persistable 依赖这些标记 → 真实结果可能溜进 /wujie。
- **[HIGH] WB2 — `wujieWriteEnabled/Count` 钉死 false/0（`studio-v2-mock-real.service.ts:69-70,152-153`,`sub2api.video-mock-gateway.service.ts:64`,`wujie.staging.service.ts:135,145,13`）**：确认这些在 real-trial 响应路径上是**硬编码** false/0（非 config 派生），wujie staging 仅内存。**风险**：若 config 派生 → env 一翻就开启真实持久化写。

### 四、created_by 链路（CB1–CB3）

- **[BLOCKER] CB1 — created_by 是否真实终端用户**：Go 侧已核实 `created_by = subject.UserID = apiKey.User.ID`（上游调用所用 key 的拥有者），daily limit 按此计。auth-split 下 Hono 用**共享员工 key** 调 Go → 所有终端用户塌缩成同一 created_by。**必须在 QCanvas 仓库验证**：Hono 是否 (a) 为每用户签发独立 sub2api key，或 (b) 在 header/body 透传 JWT 派生 user id 让 Go 覆盖 `subject.UserID`（查 `sub2api.routes.ts:184-224`、`...gateway.service.ts:72-78,143-155`、`server.ts:6249-6273`）。**风险**：若都没有 → `1/user/day` 退化成**全公司 1/day** 或形同虚设，批量阶段成本护栏被架空。
- **[HIGH] CB2 — 员工 key 的 group 归属（Go `gateway.go:41 requireGroupAnthropic`）**：在 QCanvas 配置/密钥库确认注入的 `SUB2API_API_KEY` 所属用户/key 有 Go 侧 `/v1/video/*` 前的 group 归属。**风险**：缺归属 → 每次真实调用在 Go group 闸门被拒，冒烟静默到不了 Ark，易被误读为 Ark/契约问题。
- **[LOW] CB3 — daily limit 计入失败任务（Go `service.go:428-444`）**：在运维手册写明失败的冒烟也占当天额度；A 侧应展示所用 `created_by`(CreatedByLabel) 并写明重试需换用户/清当天记录。**风险**：首次失败静默耗尽额度，运维对已耗尽用户重试得到困惑的限速拒绝。

### 五、vendor auto 路由（VA1–VA2）

- **[HIGH] VA1 — `runPublicTask` vendor:'auto' 必须钉死 seedance（`generate-video-to-canvas.ts:416-420`）**：确认 seedance trial 是否会经 `runPublicTask` 的 `vendor:'auto'+vendorCandidates`；若会，强制单一 `provider='seedance'`、candidates 空，失败不级联。**风险**：失败后自动切换到别的 vendor → 违反"无 fallback"，导致**第二次付费调用**。
- **[HIGH] VA2 — 走 P1 直连而非 P2 内部管线**：确认 trial 走 P1（`...gateway.service.ts` 直连 Go 真 HTTP）而非 P2（`generate-video-to-canvas.ts` 内部管线，不到 Go）；并记录 P1 是否有服务端轮询循环、每次生成对 Ark 的真实 poll GET 次数。**风险**：误走 P2 → 根本不到 Go 闸门；走 P1 但轮询无上限 → "1 次生成"= 1 create + N 次真实 poll，poll 是否计费未知（§3 ASSUMPTION），¥10 预算可能被轮询击穿。

### 六、A 侧信任边界 / 凭据（SEC1–SEC3）

- **[HIGH] SEC1 — result_url 域名 allowlist（A 侧最后防线）**：A 在播放/写 Day0 前校验 result_url 为 https + 预期 CDN/Ark 域名 allowlist（Go `adapter.go` 已透传未过滤，**本仓库 B3 已加 Go 侧校验，但 A 侧应独立再校验**）。**风险**：被篡改/MITM 的 URL 被浏览器加载并持久化 → XSS/SSRF/恶意媒体。
- **[MED] SEC2 — reference_image_url allowlist（A 侧）**：A 在发送前校验 reference_image_url（https、禁 file://、禁 localhost/内网 IP）。stage-0 用 text_to_video 无参考图，优先级较低，但启用 image/reference 任务前必须就位（Go 侧 B3 已对其做校验）。**风险**：file:// / 内网 URL → 经 Ark/Go 的 SSRF 探针。
- **[HIGH] SEC3 — A 侧不记 key / 不回显 Ark 错误体（`...gateway.service.ts:160-171`）**：确认 (a) `SUB2API_API_KEY` 经 readRuntimeText 读取且绝不写 console/access log/APM；(b) 非 2xx 合成的 failed error_message（可能含截断 Ark 错误体回显 Authorization/key 片段）**不**未脱敏直出浏览器/Day0。**风险**：共享员工 key 或 Ark 回显的 key 片段进 A 侧日志或用户可见响应——本仓库 D-HIGH 同源泄露通道在 A 侧的镜像。

---

## 8. 运维须知（因 B3b 硬失败而新增的启动前提）

- **B3b 后果**：`video_gateway.encryption_key` 现在是**必填**。若 `config.yaml` 未配置该 key，服务**启动即报错**（这是有意为之，防止用 totp key 误加密真实凭据）。
- **行动**：冒烟/部署前，在 `config.yaml` 的 `video_gateway:` 下显式配置一个 32 字节（64 hex 字符）的专用 key（与 totp.encryption_key **不同**）。生成示例：`openssl rand -hex 32`。
- **脱敏审计日志**：把 `SUB2API_VIDEO_REDACTED_EVENT_LOG` 指向 `*.log` 或 `backend/data/` 下的文件（已被 `.gitignore` 忽略），文件由代码以 0600 创建。
- **新增必填（因 B3 fail-closed）**：真实 seedance 路径现要求 `SUB2API_VIDEO_URL_ALLOWLIST` 非空（逗号分隔的可信媒体域名后缀，如 `volces.com,volccdn.com`，会同时匹配 apex 与子域）。未设置则 smoke gate 直接拦截真实调用并返回 `blocked_reasons: media url allowlist ... is missing`。冒烟前请把 Ark 结果 CDN 域名 + 允许的参考图域名填进去。
- **冒烟后收口**：冒烟完立即把 `single_smoke_authorized` 与 `SUB2API_VIDEO_REAL_SMOKE_ENABLED` 复位（方案 §8 强制项）。

---

## 9. 停止条件与双签字

| 门禁 | 状态 |
|---|---|
| `go build ./...` 通过 | ✅ exit 0 |
| `go test ./...` 全绿 | ✅ exit 0，零 FAIL/panic |
| 零真实调用 / 零真实凭据 | ✅ |
| 未 push / 未 add | ✅ |
| Claude 对抗评审签字 | ✅ **GO**（两轮 + 机械修复：6 项 round-1 + 3 项 round-2 + F2 全修，确定性测试覆盖；见第 6 节） |
| codex 跨家族评审签字 | ✅ **GO-WITH-FOLLOWUPS**（第一轮 NO-GO 抓 3 个真问题 → 全修 → 第二轮复查闭合 + 第三方 follow-up 也闭合；见第 5 节） |
| QCanvas 契约清单成文 | ✅ 23 条已成文（第 7 节） |

**双家族签字达成**：codex(GPT) 与 Claude **均已 GO**(codex 为 GO-WITH-FOLLOWUPS，唯一剩余 follow-up 是冒烟核对 Ark 字段名)。

**当前停止点**：Go 侧三 blocker 可独立修部分已修、`go test ./...` 全绿、codex + Claude **双家族评审签字完成**、QCanvas 清单成文。我在此**停下，不 push，等你 review**。唯一留给冒烟的是 Ark 真实响应字段名核对(代码已注释 UNVERIFIED)。绝不真实调用。
