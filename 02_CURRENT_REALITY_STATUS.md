# Sub2API 当前现实状态

更新时间：2026-07-12
状态：**待复核**

边界：mock 可演示；真实图片、真实视频和生产可用性未验证。

## 已有证据

- 2026-07-11 本机构建机完成 Docker mock 彩排：健康检查、admin 登录、首次合规确认、创建绑定分组的 API Key、mock create/poll、`succeeded + result_url`。
- mock 状态机观测为 `queued → submitted → running → succeeded`。
- 交付包存在于 `sub2api-delivery/`；密钥、备份、镜像与安装包不进入 Git，也不作为审查内容读取。
- 2026-07-11 Grok cleanup：脏工作树已分批本地提交并收口；证据见 `docs/superpowers/codex-handoff/deliverables/2026-07-11-GROK-CLEANUP-MERGE-review.md`。
- 2026-07-11 pass gates：WSL Ubuntu 目标 repository integration **PASS**；缓存 `sub2api:local` 合成 env mock `/health` + create→poll **PASS**（未重建镜像、未 push）。
- 2026-07-12 新鲜门禁：Go 最低门禁通过；前端 lint/typecheck、视频 8 tests 与 production build 通过；repository integration 35 cases 实际执行并通过、无 skip。
- Windows 页面图片路径在 `EvalSymlinks` 被拒绝时已增加逐段 `Lstat` fail-closed fallback，并有定向与完整 handler 回归。
- 普通员工侧栏已有“视频试跑 / 任务记录”；管理员视频导航中文业务化；系统检查改为 admin-only。
- `realsmoke` 专用测试 harness 会话硬门默认关闭，限制图片 4 次、视频 4 次、累计预留 ¥60；12 子进程竞争测试证明次数与预算均在 socket/create 前 fail-closed。Seedance Form A 测试 harness 已接入；Gemini/Nano Banana 尚无同等级安全真实 harness。

## 本轮已收口

- Cancel provenance：取消响应保留 trial/production events；lookup 失败不再默认 production。
- Integration isolation：删除全表 DELETE 与批量取消其他任务；按用例 ID cleanup/assert；billable-fake 按 owned task 计数。
- 视频契约：UTF-8 完整契约恢复 admin/QCanvas 端点、字段、计费与三种 boundary。
- 配置 defaults / Validate / example / compose 透传、专用视频加密密钥、worker 关闭可观察。
- 前端视频 console 缺失模块（productMode / display currency / lifecycle 测试）已提交。
- 本地 Git clean；`sub2api-delivery` / `.worktrees` / `.delivery-tools` 仅 ignore，未删除。

## 尚未证明 / 边界

- 真实 Seedance / 其他付费 Provider。
- Gemini/Nano Banana 真实图片生成、计费与资产下载。
- 真实支付、生产数据、生产部署、公网暴露。
- 2026-07-12 本轮 presence-check 未检测到 `GEMINI_API_KEY` / `SUB2API_SEEDANCE_SMOKE_API_KEY`（未读取值）；聊天中的临时密钥未写入命令、文件或日志。恢复后需重新检查。
- 2026-07-12 本轮 `wsl.exe --list --quiet` 无发行版；无法重跑 Docker/mock 浏览器环境。恢复后需重新检查。
- 官方根 Dockerfile 被 `docker/dockerfile:1.7` 镜像代理 HTTP 429 阻断，未产出当前 HEAD 镜像。
- 冒烟镜像为缓存 `sub2api:local`（可能早于 tip）；未强制重建。
- 浏览器三角色截图与真实点击链：未补，保持待复核。
- Windows symlink/junction 安全用例因当前账号无创建权限而 skip；不得写成本机已实证。

## 事实边界

mock 成功不等于生产可用；静态入口/路由测试不等于浏览器用户闭环；`result_url` 存在不等于资产已持久交付。真实调用必须在凭证仅通过安全环境注入、会话硬门已启用且运行环境恢复后执行。
