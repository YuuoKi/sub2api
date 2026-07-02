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

WSL 续跑已把 Sub2API 稳住：Ubuntu-24.04 WSL2 内 Docker 29.1.3 / Compose 2.37.1 可用，`sub2api_phasea_prime` 在 keepalive 后连续 10 次 `/health` 返回 `{"status":"ok"}`，compose ps 显示 4 个服务 `Up 5 minutes (healthy)`。

本轮最终停在 Sub2API Seedance provider preflight，未进入 QCanvas，未发起真实 tiny video POST，未产生 task id，未产生计费。

停止原因：Admin provider 脱敏摘要显示没有任何 Seedance provider 同时满足 `enabled=true`、`api_key_configured=true`、`route_available=true`。当前 Seedance 行分别处于 disabled、missing key、auth_failed 状态。

续跑补充取证：WSL Docker 中另有一个旧的 `s2a-mock-pg`，只读 SQL 发现 `seedance-smoke` 行满足 `enabled=true`、`api_key_configured=true`，并带 smoke 授权 metadata。但它没有 compose labels，来源不明；唯一发现的 `wujie-api-day0:VIDEO_GATEWAY_ENCRYPTION_KEY` 对该行密文解密失败（`InvalidTag`），Windows/WSL/Docker env 中也没有发现 Seedance/Ark upstream key 变量。因此不能把该行当作可用 provider，也不能复制到 Phase A' dev DB 后宣称 ready。

再次续跑补充取证：对命名配置来源做了更广的只读扫描，只匹配 `VIDEO_GATEWAY_ENCRYPTION_KEY` / `video_gateway.encryption_key` / `encryption_key` 这类明确命名项，不扫描泛化 64 hex 字符串。共找到 3 个候选 key 来源，全部对 `s2a-mock-pg.seedance-smoke.encrypted_api_key` 解密失败：

```text
candidate_count=3
docker:wujie-api-day0:VIDEO_GATEWAY_ENCRYPTION_KEY:decrypt_ok=false:InvalidTag
/mnt/d/sub2api-trunk/_review/M1B_mainpath_patch_20260620/harness/config.empirical.yaml:31:decrypt_ok=false:InvalidTag
/mnt/d/Codex创业任务/企业 API 管理后台项目/02_source/sub2api/backend/config.yaml:33:decrypt_ok=false:InvalidTag
matched=false
```

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

## WSL 续跑

- 初始状态：当前 Windows 用户下没有已注册 WSL distro；已安装/注册 `Ubuntu-24.04` 后继续。
- WSL Docker：`docker version` client/server 均为 `29.1.3`；`docker compose version` 为 `2.37.1+ds1-0ubuntu2~24.04.1`。
- 第一次 10 次 health 门禁通过，但下一次 WSL 命令发现容器回到 `Up 6 seconds`，判断为 WSL distro 生命周期回收导致 Docker state 重启。
- 启动临时 keepalive 后重新跑 10 次 health 门禁，通过：

```text
21:11:18 health[01]={"status":"ok"}
21:11:47 health[02]={"status":"ok"}
21:12:17 health[03]={"status":"ok"}
21:12:46 health[04]={"status":"ok"}
21:13:15 health[05]={"status":"ok"}
21:13:45 health[06]={"status":"ok"}
21:14:15 health[07]={"status":"ok"}
21:14:44 health[08]={"status":"ok"}
21:15:14 health[09]={"status":"ok"}
21:15:43 health[10]={"status":"ok"}
```

- compose ps 复核：`sub2api-dev`、postgres、redis、anthropic-mock 均 `Up 5 minutes (healthy)`。
- 采集 flag 复核：`GATEWAY_CONTENT_CAPTURE_ENABLED=true`、`GATEWAY_CONTENT_RETENTION_ENABLED=true`。
- Seedance provider 预检：已阻塞。没有 Seedance 行同时满足 `enabled=true`、`api_key_configured=true`、`route_available=true`。

## 证据文件

- `blocked_result.json`：结构化阻塞结论。
- `compose_ps_blocked.txt`：compose 状态摘录，显示服务反复处于 `Up 5 seconds (health: starting)`。
- `sub2api_logs_tail_blocked.txt`：Sub2API 日志摘录，显示 server started 后自行 shutdown。
- `wsl_docker_baseline.txt`：WSL Docker / Compose 基线。
- `wsl_health_gate_10x.txt`：两轮 10 次 health 门禁及 keepalive 观察。
- `wsl_content_flags.txt`：容器内采集双闸。
- `wsl_provider_preflight.json`：Admin provider 脱敏摘要。
- `wsl_seedance_blocker.txt`：Seedance 未 ready 的非敏感结论。
- `wsl_candidate_provider_inventory.txt`：现有 WSL Postgres 容器的 provider 非敏感盘点。
- `wsl_candidate_decrypt_probe.txt`：候选 `seedance-smoke` 密文与候选 encryption key 的不泄密匹配结果。
- `wsl_named_key_decrypt_scan.txt`：命名配置来源中 3 个候选 key 的不泄密匹配结果。
- `wsl_key_inventory.txt`：Windows/WSL/Docker env 中 key 变量名和值长度盘点，不含值。
- `wsl_cleanup_result.txt`：WSL 清理结果。

## 红线执行情况

- 真实 tiny video POST：未发起。
- retry storm：未发生。
- provider key/token/cookie：未打印。
- push：未执行。
- QCanvas：未触碰，因 Sub2API 前置阻塞。
- 成本：0，本轮未调用真实视频供应商。

## 风险

- Windows 侧直接调用 WSL 时，distro 可能在命令结束后回收，导致 Docker state 重启；本轮通过临时 keepalive 后健康门禁通过。
- 当前 dev DB 内 Seedance provider 未配置到可路由状态，不能作为 tiny real 验证基线。
- `s2a-mock-pg` 中存在一个配置状 seedance-smoke 行，但 3 个已发现命名 encryption key 候选均无法解密；强行复用会把“密文存在”误判为“真实可路由”。
- 启动日志中 PricingService 尝试拉取公开 pricing 文件并超时，这不是视频供应商调用，但说明本地 dev 启动仍可能触发外部网络尝试。
- compose 文件中的 container_name 仍是通用 dev 名称，虽使用了独立 project 和独立 volumes，后续仍建议复核是否会与其他本机 dev 栈产生名称层面的冲突。

## 回滚与清理

- G3 commit 可回滚：`git revert 98b4ea8e`
- Phase A' 审查证据目录：`D:\sub2api-trunk\_review\phase-a-prime-tiny-real_20260702`
- 本轮 compose 清理命令已执行：

```powershell
cd D:\sub2api-trunk\deploy
docker compose -p sub2api_phasea_prime -f docker-compose.dev.yml down -v
```

- 临时文件清理：删除 `D:\sub2api-trunk\deploy\.env` 与本轮 Go cache `D:\sub2api-trunk\.phasea-go-build-cache`。
- WSL 续跑清理：`deploy/.env` 已删除，`sub2api_phasea_prime` 容器/network/volumes 已删除，临时 keepalive 已停止。

## 后续提示词

继续 Phase A' 前，请先提供或恢复与 `seedance-smoke` 密文匹配的 `VIDEO_GATEWAY_ENCRYPTION_KEY`，或在管理后台重新保存一个 Seedance provider，使其在 Phase A' dev DB 中同时满足 `enabled=true`、`api_key_configured=true`、`route_available=true` 且可解密。当前已发现的 3 个命名 key 候选均不匹配。然后在 WSL 中保持 keepalive，重新跑 10 次 health 门禁和 provider preflight。不要补发 tiny real POST，不要重试出片，仍保持 1 次真实 POST 上限。
