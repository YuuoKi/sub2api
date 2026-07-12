# Sub2API 当前现实状态

更新时间：2026-07-12
状态：**待复核**

边界：真实 Gemini 单图与 Seedance 5 秒视频上游链已局部通过；资产、账务、浏览器闭环和生产可用性未验证。

## 已有证据

- 2026-07-11 本机构建机完成 Docker mock 彩排：健康检查、admin 登录、首次合规确认、创建绑定分组的 API Key、mock create/poll、`succeeded + result_url`。
- mock 状态机观测为 `queued → submitted → running → succeeded`。
- 交付包存在于 `sub2api-delivery/`；密钥、备份、镜像与安装包不进入 Git，也不作为审查内容读取。
- 2026-07-11 Grok cleanup：脏工作树已分批本地提交并收口；证据见 `docs/superpowers/codex-handoff/deliverables/2026-07-11-GROK-CLEANUP-MERGE-review.md`。
- 2026-07-11 pass gates：WSL Ubuntu 目标 repository integration **PASS**；缓存 `sub2api:local` 合成 env mock `/health` + create→poll **PASS**（未重建镜像、未 push）。
- 2026-07-12 新鲜门禁：Go 最低门禁通过；前端 lint/typecheck、视频 8 tests 与 production build 通过；repository integration 35 cases 实际执行并通过、无 skip。
- Windows 页面图片路径在 `EvalSymlinks` 被拒绝时已增加逐段 `Lstat` fail-closed fallback，并有定向与完整 handler 回归。
- 普通员工侧栏已有“视频试跑 / 任务记录”；管理员视频导航中文业务化；系统检查改为 admin-only。
- `realsmoke` 专用测试 harness 会话硬门默认关闭，限制图片 4 次、视频 4 次、累计预留 ¥60。12 子进程竞争测试证明跨进程次数与预算上限；`ReserveBefore` 测试及 Form A 代码位置分别证明拒绝发生在 callback/socket/create 前。Seedance 与 Gemini/Nano Banana Form A 测试 harness 均已接入。
- Gemini harness 固定每次预留 ¥5，模型保持仓内契约 `gemini-3.1-flash-image-preview`，严格单 item。首个真实 Batch 在上游约 107 秒成功；真实 operation 返回 `metadata.state=BATCH_STATE_SUCCEEDED`，暴露旧实现持续误判 running。`0b277da5` 增加 operation envelope/`BATCH_STATE_*` 解析，随后对既有任务执行 Get→OpenResult→真实图片解码 PASS，未新增 create。
- Seedance Form A 首个真实 5 秒任务 PASS：恰好 1 次 create、46 次 poll、约 234 秒 succeeded，720p、16:9、24fps、usage 108900；审计 47 行且两项临时 Key 均未出现。当前会话计数为图片 1/4、视频 1/4、累计预留 ¥12.5。

## 本轮已收口

- Cancel provenance：取消响应保留 trial/production events；lookup 失败不再默认 production。
- Integration isolation：删除全表 DELETE 与批量取消其他任务；按用例 ID cleanup/assert；billable-fake 按 owned task 计数。
- 视频契约：UTF-8 完整契约恢复 admin/QCanvas 端点、字段、计费与三种 boundary。
- 配置 defaults / Validate / example / compose 透传、专用视频加密密钥、worker 关闭可观察。
- 前端视频 console 缺失模块（productMode / display currency / lifecycle 测试）已提交。
- 本地 Git clean；`sub2api-delivery` / `.worktrees` / `.delivery-tools` 仅 ignore，未删除。

## 尚未证明 / 边界

- 剩余 Gemini/Nano Banana 场景、剩余 Seedance 规格与其他付费 Provider。
- 实际 Provider 账单、系统账本、用户余额三方对账，以及真实资产下载/复用。
- 真实支付、生产数据、生产部署、公网暴露。
- 2026-07-12 本轮 presence-check 未检测到 `GEMINI_API_KEY` / `SUB2API_SEEDANCE_SMOKE_API_KEY`（未读取值）；聊天中的临时密钥未写入命令、文件或日志。恢复后需重新检查。
- 2026-07-12 已在真实 Windows 用户上下文恢复 Ubuntu-24.04、WSL 2.7.10、内核 6.18.33.2 和 Docker Server 29.1.3；沙箱上下文隔离仍存在，因此 Docker/WSL 命令需在真实用户上下文执行。
- 官方根 Dockerfile 被 `docker/dockerfile:1.7` 镜像代理 HTTP 429 阻断，未产出当前 HEAD 镜像。
- 冒烟镜像为缓存 `sub2api:local`（可能早于 tip）；未强制重建。
- 浏览器三角色截图与真实点击链：未补，保持待复核。
- Windows symlink/junction 安全用例因当前账号无创建权限而 skip；不得写成本机已实证。

## 事实边界

mock 成功不等于生产可用；静态入口/路由测试不等于浏览器用户闭环；`result_url` 存在不等于资产已持久交付。真实调用必须在凭证仅通过安全环境注入、会话硬门已启用且运行环境恢复后执行。
