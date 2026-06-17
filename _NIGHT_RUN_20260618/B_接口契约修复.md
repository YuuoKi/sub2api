# 阶段 B 审查包 · Seedance 接口契约修复（Sub2API）

> 夜间无人值守 · 2026-06-18 · 分支 `night-run/20260618-B-contract`（off 阶段A）
> 原则：改码 + 测试证明 payload 正确，**全程不真实付费调用**（无 `-tags=realsmoke`，无真实 key，无网络）。

## 0. 结论

- **B1【必修】完成**：请求侧比例字段 `aspect_ratio` → `ratio` + 竖屏/横屏/方形→取值映射，契约测试证明。
- **B2【必修】完成**：轮询窗口 + wall-clock 兜底随分辨率上调（480p 短 / 720p 不变 / 1080p 长），单测证明。
- **B3【降级】完成（构造+测试，字段名未坐实）**：v2v 参考视频接入 `content` 数组（镜像 image 模式 + SSRF 门）+ 草案；Ark 视频 content 字段名 `video_url` 为推断、**UNVERIFIED**，真实确认入待授权。
- 验证：`go build ./...` ✅；`go vet ./...` ✅；`go test ./internal/service/ -count=1` ✅（43.9s 全绿，含既有 FormA/security/VA2/ratio gold-sample）。
- **未真实调用 / 未碰密钥 / 未 push / 未部署。**

## 1. 改动清单（仅 `internal/service/`，不碰主路由/Kling/drama）

```
backend/internal/service/video_gateway_adapter.go   | 56 +++++--   (B1 ratio + 映射helper; B3 v2v content)
backend/internal/service/video_gateway_worker.go    | 95 +++++++--  (B2 分辨率分层poll预算 + wall-clock兜底)
backend/internal/service/video_gateway_security_test.go | 13 +-    (既有断言 aspect_ratio→ratio)
backend/internal/service/video_gateway_b1b2b3_test.go   | 新增       (B1/B2/B3 契约+单测)
```

> 未改：handler 入站 DTO（`aspect_ratio` JSON 字段是前端入站契约，保持）、DB 列 `aspect_ratio`（migration/repo，保持）、mock adapter、kling adapter、drama_gateway（独立路由，且其 `aspect_ratio` 经 `VideoTask.AspectRatio` 流入本 adapter，已被本修复在出站处翻译）。

## 2. B1 —— 请求侧 `ratio` + 朝向映射

**根因**：`CreateTask` 请求体发 `aspect_ratio`（Ark 不识别→落默认 16:9 横屏→竖屏做不出）。响应侧早已解析 `ratio`（commit 7b78f9ca），请求侧未改——这就是「ratio 字段对齐」与「缺陷仍在」并存的真相。

**改法**（`video_gateway_adapter.go`）：
- `CreateTask` 请求体：`payload["aspect_ratio"]=task.AspectRatio` → `if r:=normalizeSeedanceRatio(task.AspectRatio); r!=""{ payload["ratio"]=r }`。
- 新增纯函数 `normalizeSeedanceRatio`：竖屏/portrait/vertical/9:16→`9:16`；横屏/landscape/16:9→`16:9`；方形/square/1:1→`1:1`；其余合法 `W:H` 直通；空→省略。
- `BuildCreatePayload`（契约文档 payload）同步改 `ratio`。
- `resolution` 字段不动（已验证生效）。

**测试**：
- `TestNormalizeSeedanceRatio`：映射表全覆盖（中英关键词 + 直通 + trim）。
- `TestSeedanceCreateSendsRatioWithOrientationMapping`：8 子例，**捕获真实请求体**（localhost mock-Ark，非真 Ark），断言 `ratio` 取值正确且**不再发** `aspect_ratio`。
- `TestSeedanceCreateSendsDurationAndAuditsRedacted`（既有，已更新）：断言 `ratio==16:9` 在线上、`aspect_ratio` 不在。

## 3. B2 —— 轮询/超时随分辨率上调

**根因**：`videoDefaultMaxPollAttempts=72`×5s=6min 窗，非分辨率相关；wall-clock `task_timeout_minutes` 默认 15min。1080p/5s 实测 ~19min → 两道兜底都会提前杀任务。

**改法**（`video_gateway_worker.go`）：
- 分层默认 poll 预算常量：`480p=48`(4min) / `720p=72`(6min,不变) / `1080p=300`(25min)（5s 间隔）。`normalizeResolutionTier` 把 480/720/1080/2k/4k 折叠到三层，未知→720p 安全中位。
- `maxPollAttempts()` → `maxPollAttemptsForTask(task)`：config `max_poll_attempts>0` 时按配置钉死所有分辨率（向后兼容）；未配置时按分辨率分层。
- `effectiveTaskTimeout(task, base)`：wall-clock = `max(配置task_timeout, pollWindow + 5min margin)`，保证 wall-clock 始终在 poll 窗之外（自动满足 config 注释里「task_timeout ≥ poll窗 + margin」不变式）。1080p：25min 窗 → wall-clock 抬到 30min（不再被 15min 杀）。

**测试**：
- `TestVideoPollBudgetScalesWithResolution`：480<720<1080；720p==旧默认；未知→720p；1080p 窗 ≥19min。
- `TestMaxPollAttemptsForTaskResolutionAndOverride`：分辨率分层 + config 钉死覆盖。
- `TestEffectiveTaskTimeoutWrapsPollWindow`：1080p wall-clock>窗且>15min base；720p 保持 15min。
- 既有 `TestVideoWorkerPollCapTerminatesNeverSucceeding` / `...PollsPatientlyUntilSucceeded` / `TestVideoPollWindowMeetsMinimum` 仍全绿（向后兼容）。

## 4. B3 —— 换皮 v2v（构造+测试；字段名未坐实）

**现状**：`task.ReferenceVideoURL` 已在数据模型+handler，但 adapter 忽略。

**改法**（`video_gateway_adapter.go`，镜像 image_url 模式）：
```go
if task.ReferenceVideoURL != "" {
    if err := validateExternalVideoURL(task.ReferenceVideoURL); err != nil { /* SSRF拒绝 */ }
    content = append(content, map[string]any{"type":"video_url","video_url":map[string]string{"url":task.ReferenceVideoURL}})
}
```
- **UNVERIFIED**：Ark 视频参考的确切 content 字段名（`video_url`）由 image_url 模式推断，**真实确认入待授权**。测试只断言「我们构造的 payload 形状」，不断言「Ark 接受」。

**草案：v2v 请求体结构**
```json
{
  "model": "doubao-seedance-2-0-260128",
  "content": [
    {"type": "text", "text": "<描述目标变换的 prompt，可 @Video 引用>"},
    {"type": "image_url", "image_url": {"url": "<可选风格参考图>"}},
    {"type": "video_url", "video_url": {"url": "<参考视频URL，做风格/运动迁移>"}}
  ],
  "duration": 5, "resolution": "1080p", "ratio": "16:9"
}
```

**测试**：
- `TestSeedanceCreateAttachesReferenceVideoForV2V`：捕获请求体，断言 content 同时含 text + `video_url` 项且 url 正确。
- `TestSeedanceCreateRejectsUnsafeReferenceVideoURL`：内网 ref URL 被 SSRF 门在发请求前拒绝。

## 5. 脱敏 payload 示例（构造产物，无密钥）

竖屏 1080p 5s + v2v 参考（**真实 key 仅在 Authorization 头，不在 body**；以下为契约测试断言的实际请求体形状）：
```json
{
  "model": "doubao-seedance-2-0-260128",
  "content": [
    {"type": "text", "text": "<prompt>"},
    {"type": "video_url", "video_url": {"url": "https://<allowlisted-host>/clip.mp4"}}
  ],
  "duration": 5,
  "resolution": "1080p",
  "ratio": "9:16"
}
```
> `ratio:"9:16"`（竖屏映射，B1）；`video_url` 项（B3，字段名待坐实）；无 `aspect_ratio`；无 key。

## 6. 验证命令（全部不触发真实网络）

```
cd backend
go build ./...                                  # ✅ exit 0
go vet ./...                                    # ✅ exit 0
go test ./internal/service/ -count=1            # ✅ ok 43.9s（全绿）
go test ./internal/service/ -run 'Ratio|Seedance|VideoPoll|MaxPollAttempts|EffectiveTaskTimeout' -v   # ✅ 逐条 PASS
```
（真实 smoke 路径 `-tags=realsmoke` 默认 inert，本阶段**未**编译/未运行。）

## 7. 停止条件 & 待授权

- B1、B2 修完且测试证明 payload 正确 → **可进阶段 C**。
- **待授权清单（B 相关）**：
  1. **B1 真实确认**：发 `ratio:"9:16"` 是否真产出竖屏视频——需 1 次真实付费调用（建议 720p/5s 竖屏，~个位数￥）。
  2. **B3 字段名坐实**：Ark 视频参考 content 的确切字段名（`video_url`？）——需查 Ark 官方 v2v 文档或 1 次真实 v2v 调用。
  3. **请求侧 duration/resolution 字段名**：仍为响应回显推断，未经真实 create 请求坐实（低风险，可随 B1 同枪验证）。

## 8. 提交记录

分支 `night-run/20260618-B-contract`；`git add` 仅 4 个 backend 文件 + 本审查包（显式逐一，绝不 `git add .`）。**未 push。** 确切 hash 见 `git log` 与 `00_黎明总结.md`。
