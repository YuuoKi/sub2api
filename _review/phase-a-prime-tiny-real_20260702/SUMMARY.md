# Phase A' Tiny Real 三证同屏审查包

状态：已阻塞
日期：2026-07-02
执行目录：D:\sub2api-trunk
分支：wujie/video-capture-moat-20260702

## 目标

按 Sub2API -> QCanvas -> 三证对账 -> 清理 顺序执行 1 次 tiny real 视频闭环，获取三证：

1. QCanvas /studio-v2 画布节点出现真片 URL，realChainReady=true。
2. Sub2API `ai_generation_content` 新增 1 行 video 记录，task_id 对上 HTTP task id。
3. Admin generation-content stats/samples 返回 is_live=true。

## 结论

本轮停在 Sub2API provider preflight，未进入 QCanvas，未发起真实 tiny video POST，未产生 task id，未产生计费。

停止原因：独立 compose project `sub2api_phasea_prime` 可构建、可启动，`/health` 曾短暂返回 `{"status":"ok"}`，但服务持续重启。provider preflight 期间 Admin 登录请求无法稳定连接，随后 `docker compose ps` 显示服务又回到 `Up 5 seconds (health: starting)`，日志显示 `Server started` 后约 10 秒出现 `Shutting down server...` 和 `Server exited`。

三证结果：

| 证据 | 结果 | 说明 |
| --- | --- | --- |
| 画布有真片 | 未执行 | 因 Sub2API preflight 阻塞，未启动 QCanvas，未截图 |
| 数据库有记录 | 未执行 | 未创建 HTTP task id，未写入 `ai_generation_content` |
| 看板 is_live=true | 未执行 | 未产生内容记录，未调用 stats/samples 验收 |

## 已完成动作

- 复核 Git 三件套：root 为 `D:/sub2api-trunk`，分支为 `wujie/video-capture-moat-20260702`。
- 仅提交 G3 白名单文件，本地 commit：`98b4ea8e docs(review): close G3 capture dev validation`。
- 使用本地 Go cache 验证采集 flag 测试：

```powershell
cd D:\sub2api-trunk\backend
$env:GOCACHE='D:\sub2api-trunk\.phasea-go-build-cache'
go test ./internal/config -run TestLoadContentCaptureFlagsFromEnv
```

结果：`ok github.com/Wei-Shaw/sub2api/internal/config 1.428s`

- 创建临时 `deploy/.env`，只写本轮运行变量，未打印 secret 值。
- 使用独立 project 启动：

```powershell
cd D:\sub2api-trunk\deploy
docker compose -p sub2api_phasea_prime -f docker-compose.dev.yml config --quiet
docker compose -p sub2api_phasea_prime -f docker-compose.dev.yml up --build -d
```

结果：compose config 通过，up/build 成功。compose warning：`REDIS_PASSWORD` 未设置，默认空值。

- `/health` 曾返回：

```json
{"status":"ok"}
```

- provider preflight 未通过：Admin 登录/读取 provider 时服务连接失败，随后确认 compose project 反复重启。

## 证据文件

- `blocked_result.json`：结构化阻塞结论。
- `compose_ps_blocked.txt`：compose 状态摘录，显示服务反复处于 `Up 5 seconds (health: starting)`。
- `sub2api_logs_tail_blocked.txt`：Sub2API 日志摘录，显示 server started 后自行 shutdown。

## 红线执行情况

- 真实 tiny video POST：未发起。
- retry storm：未发生。
- provider key/token/cookie：未打印。
- push：未执行。
- QCanvas：未触碰，因 Sub2API 前置阻塞。
- 成本：0，本轮未调用真实视频供应商。

## 风险

- 当前独立 compose project 的服务生命周期不稳定，不能作为 tiny real 验证基线。
- 启动日志中 PricingService 尝试拉取公开 pricing 文件并超时，这不是视频供应商调用，但说明本地 dev 启动仍可能触发外部网络尝试。
- compose 文件中的 container_name 仍是通用 dev 名称，虽使用了独立 project 和独立 volumes，后续仍建议复核是否会与其他本机 dev 栈产生名称层面的冲突。

## 回滚与清理

- G3 commit 可回滚：`git revert 98b4ea8e`
- Phase A' 审查证据目录：`D:\sub2api-trunk\_review\phase-a-prime-tiny-real_20260702`
- 本轮 compose 清理命令：

```powershell
cd D:\sub2api-trunk\deploy
docker compose -p sub2api_phasea_prime -f docker-compose.dev.yml down -v
```

- 临时文件清理：删除 `D:\sub2api-trunk\deploy\.env` 与本轮 Go cache `D:\sub2api-trunk\.phasea-go-build-cache`。

## 后续提示词

继续 Phase A' 前，请先只读排查 `sub2api_phasea_prime` dev 栈为何启动约 10 秒后自行 shutdown，确认 `/health` 和 Admin provider preflight 可连续稳定通过，再继续 QCanvas 真链。不要补发 tiny real POST，不要重试出片，仍保持 1 次真实 POST 上限。
