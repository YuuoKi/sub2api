# 本地注册—模拟任务 Gap-fill 验收报告

## 状态

**待复核 / 部分门禁通过**。本报告不是生产 READY 声明。

## 范围与执行环境

- 工作树：`D:\sub2api-trunk\.worktrees\console-unification`
- 分支：`codex/wujie-console-unification-20260717`
- 本次代码提交：
  - `aec67e549 fix(frontend): guide ungrouped users to channels`
  - `de9678ed0 fix(video): run mock simulation worker independently`
- 本地验收地址：`http://127.0.0.1:8080`；Docker health 与 HTTP `/health` 均为 `200`。

## 浏览器闭环证据（本地一次性测试用户）

1. 管理端将本地注册开关打开，公开配置回读为 `registration_enabled=true`。
2. 新用户在 `/register` 注册后自动进入 `/dashboard`。
3. 在“我的密钥”选择现有 `default` 分组，创建仅用于本地模拟的 API 密钥；没有记录或展示该密钥值。
4. 在 `/video/create` 使用该本地密钥创建任务 #1。页面和接口均标识 `mock / mock-video-v1`、`0 USD`、`无外网`。
5. 任务记录观察到 `排队中 → 生成中 → 已完成`，任务详情展示模拟图像预览；“下载结果”按钮可用并已触发下载动作。

截图（均为本地 mock 证据，无真实 Provider 密钥）：

- `screenshots/local-registration.png`
- `screenshots/new-user-dashboard.png`
- `screenshots/simulation-task-form.png`
- `screenshots/simulation-task-terminal.png`
- `screenshots/simulation-result-detail.png`

## 查缺与修复

### 已修复：mock 任务永久排队

复现：本地用户创建模拟任务后，HTTP `POST /api/v1/user/video/simulation/tasks` 返回 `202`，任务持续停在排队。

根因：`VideoSimulationRuntime.Start()` 错误依赖 `VideoGateway.WorkerEnabled`。该开关默认关闭，且属于真实网关的安全/付费运行门；因此 mock worker 从不启动。开启真实网关开关不是正确解法。

修复：模拟 worker 仅要求自身 worker 和配置对象存在，独立于真实网关开关。新增 `TestSimulationRuntimeRunsWhenRealVideoWorkerIsDisabled`：修复前在 3 秒内失败，修复后通过。

说明：当前 mock worker 为了让“生成中”状态可观察，使用 30 秒领取租约；本次浏览器实测在租约后完成并产出预览。它不调用 Seedance、Gemini、字节或任何外部服务。

### 防呆改进：无分组时的 API 密钥入口

页面原先在没有可用分组时仍可打开一个必然因“请选择订阅分组”而不能提交的创建表单。现改为跳转“查看可用渠道”。`KeysView` 回归用例覆盖该条件。

本轮真实验收库后来确认实际存在 `default` 分组；此前文本快照漏掉了视觉下拉项，**并非本地库无分组**。该改动是防呆，不应被表述为该库的根因修复。

## 验证命令与结果

| 命令/证据 | 结果 |
| --- | --- |
| `go test ./internal/service -run TestSimulationRuntimeRunsWhenRealVideoWorkerIsDisabled -count=1` | 修复前失败，修复后通过 |
| `go test ./internal/service` | 通过 |
| `go test ./internal/server/routes -run TestUserSimulationVideoCreateRouteRegisteredBehindJWT -count=1` | 通过 |
| `go test ./...` | 通过（exit 0） |
| `npm run test:run` | 通过（exit 0） |
| `npm run lint:check`、`npm run typecheck`、`npm run build` | 通过；构建有既存 bundle/Browserlist 警告 |
| Docker 构建 `sub2api:wujie-delivery-de9678ed0` | 成功 |
| 本机 Docker `/health` | `200` |
| 浏览器本地 mock 闭环 | 注册、登录、密钥、任务、状态、预览、下载动作均已实测 |

## 严格未验证项

- 未调用、未保存、未打印任何用户提供的外部 API 密钥；未调用真实或付费 Provider。
- Seedance、字节系图像、Gemini 的真实账号配置、真实请求、结果资产交付：**未验证**。
- 管理员/员工/老板三角色完整浏览器回归：**未验证**（本轮实测为新注册的普通用户路径）。
- PostgreSQL 182–186 的真实迁移验收：**未验证**；本轮仅通过隔离 Docker 8080 的已有本地数据库运行 mock 数据路径。
- Linux dirfd：**未验证**。
- 生产环境、生产数据、部署：**未触碰**。

## 风险、回滚与下一步

- 本地 mock worker 当前可见“生成中”约 30 秒；若产品要求秒级演示体验，应单独评审领取租约与状态可见性的取舍。
- 回滚代码使用非破坏性 `git revert de9678ed0`；如需一并撤销入口防呆，再 `git revert aec67e549`。本机旧镜像仍保留。
- 若要验证真实 Provider，必须由用户提供经过授权的隔离测试账号与预算，并先获得明确的真实调用许可；在此之前保持 mock 验收边界。
