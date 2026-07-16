# 无界 AI 管理中台 · 本地镜像入口

本文件只描述当前收口仓的本机受控入口。完整操作与失败边界见 [`../docs/00_START_HERE.md`](../docs/00_START_HERE.md) 和 [`WUJIE_SINGLE_ENTRY_SOP.md`](WUJIE_SINGLE_ENTRY_SOP.md)。

## Quick Start

```powershell
# cwd: repository root
docker build -t wujie-sub2api:local .
./deploy/wujie-local-entry.ps1 Start
./deploy/wujie-local-entry.ps1 Status
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

数据库、Redis 与 TOTP 等本机配置由 `deploy` 目录的本地环境文件提供；禁止用文档示例覆盖真实值，也禁止打印或展开配置。配置导致启动失败时必须原样记录失败并停止，不得读取、回显或临时改写秘密。

## 前端与导航口径

- 唯一前端是“完整管理后台 + 无界品牌”，不是 `video_gateway_demo` 或 video-only 页面。
- 管理员必须能看到用户、账号、分组、兑换码、设置等管理入口；员工个人 Key 与 usage 入口保留。
- 2C 订阅购买、支付、分销、促销等销售导航从实际侧栏隐藏，但对应历史路由不等于删除管理页。
- 当前后端角色仅 `admin` / `user`，其中 `user` 对应员工；`admin` 兼任管理者开卡，不扩第三角色。

## 当前状态

Sub2API 功能实现提交为 `7cf7404f`，QCanvas 功能实现提交为 `e7c8af3`；仓库 HEAD 通过 `git rev-parse HEAD` 实时读取。运行证据基线为 `d6687e89`。canonical `127.0.0.1:8080` 已有 HTTP 200、无界 title、完整管理五段与两次 tiny_real 证据；`blocked=[]`、`open_p0_count=0`、`realCallExecuted=2`，整体为**待复核**。不得外推为生产上线、已 push 或商业交付完成；QCanvas 既有资产回填、W4.2 fake-gateway pilot 与历史 provider key 轮换仍未执行。

当前 `wujie-sub2api:local` 已由功能实现提交 `7cf7404f` 对应工作树重建为 `sha256:72b91368ff03b620e430a8cfe6ae4bdaff49716414cf7d8a7bd1dcdf8fb40380`，并通过入口脚本 Stop/Start/Status 门禁；Status 核对不可变 image ID，Start 使用 `--no-deps`，端口检查不要求提升权限。canonical Docker-NAT 只通过显式 compose 开关信任私有 bridge peer + 精确 loopback Host:8080，不读取 XFF。首次强制改密、UX 与资产接力代码完成不代表受保护浏览器走查完成。

## 回滚

只允许停止本轮应用容器并按 SOP 恢复旧应用容器。禁止删除 volume、数据库或备份目录。

## 证据入口

- 当前入口：[`../docs/00_START_HERE.md`](../docs/00_START_HERE.md)
- 单入口 SOP：[`WUJIE_SINGLE_ENTRY_SOP.md`](WUJIE_SINGLE_ENTRY_SOP.md)
- 当前 compose：[`docker-compose.yml`](docker-compose.yml)
