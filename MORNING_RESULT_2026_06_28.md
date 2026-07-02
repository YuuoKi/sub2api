# Sub2API 早晨结果入口 - 2026-06-28

## 结论

当前状态：**内部可用 / 可演示 / 真实供应商待授权复核**。

我完成了全项目扫库、复查、小规模修复、再扫描、最佳实践小修和最终验证。本机服务已启动在 `http://127.0.0.1:18081`，健康检查 200；浏览器已完成“登录 -> 新建视频试跑任务 -> 后台处理 -> 状态回传 -> 结果资产打开”的闭环。真实 Seedance/Kling/S3、生产数据、部署和公网暴露未获授权，所以不宣称生产 READY。

## 战略结论

这个项目当前最有价值的定位，不是先宣称“生产公开平台 ready”，也不是把重点放在真实视频供应商大闭环上；它更适合作为无界内部的 AI API 与生产控制面，服务新的 chat 对话里的战略决策。

也就是说，新 chat 对话应该把 Sub2API 当成可复核的后台底座：账号、模型路由、用量、成本、任务编排、结果回收、审计留痕都由这里提供证据。老板、运营和技术管理员可以在 chat 里讨论模型选择、成本边界、生产排期、供应商接入优先级和风险控制，而不是只看一个孤立的接口管理页面。

当前已经能支撑“内部战略决策验证”的部分：登录、管理后台、视频任务 mock 闭环、任务状态回传、结果资产 HTTP 200、用量与审计相关修复、测试与构建闸门。当前不应对外承诺的部分：真实 Seedance/Kling/S3 资产交付、真实付费供应商生产闭环、公开商业平台 ready。

新的 chat 对话建议采用这个判断：Sub2API 现在可以作为“内部可用的决策与调度底座”继续推进；下一阶段应先设计并验证 chat 主路径：新建对话 -> 读取业务上下文 -> 查询 Sub2API 能力和用量 -> 给出模型/成本/排期建议 -> 触发受控任务 -> 回写状态和结果证据。
## 最重要证据

- 审查包：`docs/reviews/LATEST_REVIEW_PACKAGE.html`
- 浏览器闭环结果：`output/playwright/local-smoke-task-flow-after-mock-asset-fix-verified.json`
- 任务证据：任务 `#2`，最终 `succeeded`，结果地址 `/api/v1/video/mock-assets/2.svg`，资产 HTTP 200。
- 截图：
  - `output/playwright/local-smoke-task-create-after-asset-fix-before-submit.png`
  - `output/playwright/local-smoke-task-detail-after-asset-fix.png`
  - `output/playwright/local-smoke-task-list-after-asset-fix.png`
  - `output/playwright/local-smoke-result-asset-after-fix.png`

## 已修复重点

- Windows 单体二进制缺时区数据导致 `Asia/Shanghai` 启动失败。
- AUTO_SETUP 未写入 `video_gateway.encryption_key` 导致启动校验失败。
- 未用 `-tags embed` 构建时 `/login` 404；最终 `backend/bin/server.exe` 已按 embed 重建。
- mock 视频任务以前返回不可打开的外部假域名；现在返回本服务可打开的 `/api/v1/video/mock-assets/:id.svg`。
- 视频真实派发计数、worker 并发领取、usage log 幂等、每日试用并发、S3 流式上传、前端 cost 空值、OAuth/Stripe/CSV 导出等问题已修。
- `pnpm-workspace.yaml` 的 build-script allow list 已从占位文字修正为布尔值。

## 验证结果

- `go test ./...`：通过。
- `go build -tags embed -ldflags="-s -w" -trimpath -o bin\server.exe .\cmd\server`：通过，并已用最终二进制重启本地服务。
- `pnpm.cmd install --frozen-lockfile`：通过。
- `pnpm.cmd test:run`：97 个测试文件、581 个测试通过。
- `pnpm.cmd typecheck`：通过。
- `pnpm.cmd lint:check`：通过。
- `pnpm.cmd build`：通过。
- `pnpm.cmd audit --prod --audit-level high`：通过；无 high/critical，剩余 3 low / 21 moderate。
- `python tools\secret_scan.py --include-untracked`：通过，无高置信泄露命中。
- `git diff --check` / `git diff --check --cached`：通过，仅有 LF/CRLF 提示。

## 当前运行

- 本地服务：`127.0.0.1:18081`，PID `30872`。
- 临时 Postgres：`sub2api-goal-pg-20260628goal`，本机端口 `15434`。
- 临时 Redis：`sub2api-goal-redis-20260628goal`，本机端口 `16381`。
- 本轮使用临时本机管理员账号验证，报告不明文输出账号密码。

## 风险 / 阻塞

- 真实供应商、生产库、部署、公网访问都未授权，仍保持阻塞。
- 当前工作树进入任务前已有大量 staged 归档重命名，提交前必须确认是否保留。
- `docs/reviews/` 和 `output/` 可能被 Git 忽略；正式交付前需决定是否 force-add 或迁移证据。
- Vite 仍有 chunk 体积、动态/静态 import 混用、Browserslist 过期提示；不阻断当前内部使用。

## 下一步

1. 确认是否保留进入任务前已有的 staged 归档重命名。
2. 决定是否把审查包和截图证据纳入 Git 追踪。
3. 如要跑真实 Seedance/Kling/S3 smoke，先给出明确授权、预算上限、失败退出条件和允许使用的测试账号。
