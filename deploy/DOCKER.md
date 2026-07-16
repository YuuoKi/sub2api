# 无界 AI 管理中台 · 本地镜像入口

本文件只描述当前收口仓的本机受控入口。完整操作与失败边界见 [`../docs/00_START_HERE.md`](../docs/00_START_HERE.md) 和 [`WUJIE_SINGLE_ENTRY_SOP.md`](WUJIE_SINGLE_ENTRY_SOP.md)。

## Quick Start

```powershell
# cwd: repository root
docker build -t wujie-sub2api:local .

Push-Location deploy
docker compose -f docker-compose.yml up -d
Pop-Location
```

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

数据库、Redis 与 TOTP 等本机配置由 `deploy` 目录的本地环境文件提供；禁止用文档示例覆盖真实值，也禁止打印或展开配置。若 TOTP key 无效，记录 `BLOCKED:totp-encryption-key` 并停止，不得读取、回显或临时改写。

## 前端与导航口径

- 唯一前端是“完整管理后台 + 无界品牌”，不是 `video_gateway_demo` 或 video-only 页面。
- 管理员必须能看到用户、账号、分组、兑换码、设置等管理入口；员工个人 Key 与 usage 入口保留。
- 2C 订阅购买、支付、分销、促销等销售导航从实际侧栏隐藏，但对应历史路由不等于删除管理页。
- 当前后端角色仅 `admin` / `user`，其中 `user` 对应员工；`admin` 兼任管理者开卡，不扩第三角色。

## 当前状态

代码、镜像、部署契约与隔离开卡/usage smoke 为**内部可用**；canonical `:8080` 当前 exit 1、无 HTTP，既有 `BLOCKED:totp-encryption-key` 未解除，整体仍为**已阻塞**，不得标记为**可演示**。

## 回滚

只允许停止本轮应用容器并按 SOP 恢复旧应用容器。禁止删除 volume、数据库或备份目录。

## 证据入口

- 当前入口：[`../docs/00_START_HERE.md`](../docs/00_START_HERE.md)
- 单入口 SOP：[`WUJIE_SINGLE_ENTRY_SOP.md`](WUJIE_SINGLE_ENTRY_SOP.md)
- 当前 compose：[`docker-compose.yml`](docker-compose.yml)
