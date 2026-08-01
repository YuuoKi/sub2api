# 无界 · 企业 AI 中台 · 本地镜像入口

本文件只描述当前收口仓的本机受控入口。完整操作与失败边界见 [`../docs/00_START_HERE.md`](../docs/00_START_HERE.md) 和 [`WUJIE_SINGLE_ENTRY_SOP.md`](WUJIE_SINGLE_ENTRY_SOP.md)。

## Quick Start

```powershell
# cwd: repository root
./deploy/wujie-delivery-preflight.ps1 Check `
  -RepoRoot (Get-Location).Path `
  -ReleaseRoot (Join-Path (Get-Location) 'release') `
  -ExpectedBranch 'codex/sub2api-production-readiness-20260802' `
  -RequiredProductCommit (git rev-parse HEAD)
./deploy/wujie-delivery-preflight.ps1 Build `
  -RepoRoot (Get-Location).Path `
  -ReleaseRoot (Join-Path (Get-Location) 'release') `
  -ExpectedBranch 'codex/sub2api-production-readiness-20260802' `
  -RequiredProductCommit (git rev-parse HEAD) `
  -Commit (git rev-parse HEAD)
./deploy/wujie-local-entry.ps1 Start
./deploy/wujie-local-entry.ps1 Status
```

`Check` 只读核对固定工作树、指定分支、gap-fill 产品提交祖先、tracked 干净状态、`.dockerignore`、最新审查包、端口与 Docker 引擎；`Build` 只构建并核对 `wujie-sub2api:local` 镜像，不启动服务。两者均不读取环境文件、不展开 compose、不调用真实 Provider。若 WSL/Docker 未恢复，必须在此停止，不得用旧镜像冒充当前 HEAD。

唯一浏览器入口是 `http://127.0.0.1:8080`。禁止把 Vite `:3000`、外部地址、`weishaw/sub2api:latest` 或 one-off 诊断容器作为验收入口。

## Docker Compose

仓内 `deploy/docker-compose.yml` 的当前硬约束是：

```yaml
services:
  sub2api:
    image: wujie-sub2api:local
    build:
      context: ..
    ports:
      - "127.0.0.1:8080:8080"
    environment:
      - SERVER_PORT=8080
      - RUN_MODE=standard
```

数据库、Redis 与 TOTP 等本机配置由 `deploy` 目录的本地环境文件提供；禁止用文档示例覆盖真实值，也禁止打印或展开配置。配置导致启动失败时必须原样记录失败并停止，不得读取、回显或临时改写秘密。

## 前端与导航口径

- 唯一前端是“完整管理后台 + 无界品牌”，不是 `video_gateway_demo` 或 video-only 页面。
- 管理员必须能看到用户、账号、分组、兑换码、设置等管理入口；员工个人 Key 与 usage 入口保留。
- 2C 订阅购买、支付、分销、促销等销售导航从实际侧栏隐藏，但对应历史路由不等于删除管理页。
- 当前后端角色仅 `admin` / `user`，其中 `user` 对应员工；`admin` 兼任管理者开卡，不扩第三角色。

## 当前状态

当前 gap-fill 产品提交为 `4c6111502`，审查包刷新提交为 `a14670879`；仓库实际 HEAD 始终以 `git rev-parse HEAD` 为准。Task 6 在旧 HEAD `7f4c15ca1` 的 Docker/8080 通过仅为历史证据。当前 HEAD fresh Docker build 因本机 WSL `CreateVm/E_INVALIDARG` 仍为 **NOT VERIFIED**，状态保持**待复核 / 部分门禁通过**。不得用历史镜像、旧 digest、Vite `:3000` 或源码单测冒充当前运行证据。

## 回滚

只允许停止本轮应用容器并按 SOP 恢复旧应用容器。禁止删除 volume、数据库或备份目录。

## 证据入口

- 当前入口：[`../docs/00_START_HERE.md`](../docs/00_START_HERE.md)
- 单入口 SOP：[`WUJIE_SINGLE_ENTRY_SOP.md`](WUJIE_SINGLE_ENTRY_SOP.md)
- 当前 compose：[`docker-compose.yml`](docker-compose.yml)
