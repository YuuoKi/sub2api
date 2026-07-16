# 无界 AI 管理中台本地单入口 SOP

本 SOP 仅用于本机受控环境。唯一浏览器验收入口是 `http://127.0.0.1:8080`；禁止使用 Vite `:3000` 作为验收入口，禁止用 `weishaw/sub2api:latest` 冒充无界构建。

## 1. 端口与容器预检

在仓库根目录执行：

```powershell
Get-NetTCPConnection -State Listen -LocalPort 3000,8080 -ErrorAction SilentlyContinue |
  Select-Object LocalAddress,LocalPort,OwningProcess,State

docker ps --format '{{.Names}}`t{{.Image}}`t{{.Ports}}`t{{.Status}}'
```

如 `:3000` 有监听，先核对进程后停止该开发服务。只停止确认占用 host `:8080` 的旧 Sub2API 容器；不要停止 `:18081` 审查容器、QCanvas 容器或不冲突的数据库/Redis 容器。

当前旧容器的精确停止命令：

```powershell
docker stop wujie-final-sub2
```

若预检发现 `weishaw/sub2api:latest` 容器占用 host `:8080`，先从上述安全字段输出确认其容器名，再对该精确名称执行 `docker stop <container-name>`。不得按模糊名称批量停止容器。

## 2. 重建 frontend 与 embed 二进制

仓库 Dockerfile 固定使用 pnpm 9；Windows 本地构建使用同一主版本：

```powershell
Push-Location frontend
$env:CI = 'true'
corepack pnpm@9.15.9 install --frozen-lockfile --prefer-offline
corepack pnpm@9.15.9 run build
Pop-Location

Select-String -LiteralPath .\backend\internal\web\dist\index.html -Pattern '<title>'
```

标题必须是 `无界 · 企业 AI 管理中台`。随后构建带 embed tag 的本地二进制，输出保留在系统临时目录，不写入仓库：

```powershell
Push-Location backend
$embedBinary = Join-Path $env:TEMP ('wujie-sub2api-embed-' + [guid]::NewGuid().ToString('N') + '.exe')
go build -tags embed -o $embedBinary ./cmd/server
Get-Item -LiteralPath $embedBinary | Select-Object FullName,Length,LastWriteTime
Pop-Location
```

## 3. 构建本地镜像并启动 compose

```powershell
docker build -t wujie-sub2api:local .

Push-Location deploy
docker compose -f docker-compose.yml up -d
Pop-Location
```

compose 会正常加载 `deploy` 目录的本地环境文件。host 入口已硬绑定为 `127.0.0.1:8080`，本地环境文件只能影响容器内业务配置，不能改写 host 端口。禁止用 `Get-Content`、`cat` 或 `docker compose config` 打印环境文件、密钥或展开后的配置。

仓内 compose 已把 host 绑定硬性固定为 `127.0.0.1:8080`，并把 `RUN_MODE` 固定为 `standard`；不得通过 `.env` 改为对外地址、其他 host 端口或 simple/demo 隐藏模式。如本地环境文件中的 TOTP key 无效，记录 `BLOCKED:totp-encryption-key` 并停止验收，禁止读取、回显或临时改写该 key。`docker compose run -e` 产生的 one-off 容器只可用于脱敏诊断，必须精确停止，不能作为本 SOP 的 `up` / `stop` 生命周期或品牌验收证据。

## 4. 只验收 `127.0.0.1:8080`

```powershell
curl.exe -sS -o NUL -w "%{http_code}`n" http://127.0.0.1:8080/
curl.exe -sS http://127.0.0.1:8080/ | Select-String '无界 · 企业 AI 管理中台'

docker ps --filter 'name=^/wujie-single-entry-sub2api$' --format '{{.Names}}`t{{.Image}}`t{{.Ports}}`t{{.Status}}'
Get-NetTCPConnection -State Listen -LocalPort 3000 -ErrorAction SilentlyContinue
```

验收条件：HTTP 状态码为 `200`；页面 title 命中无界品牌；运行镜像为 `wujie-sub2api:local`；host `:3000` 无监听。不得打开 `http://127.0.0.1:3000` 进行验收。

## 5. 非破坏性回滚

如需恢复本轮启动前的本机入口，仅停止新的应用容器并启动旧应用容器；不要删除 volume、备份目录或数据库：

```powershell
Push-Location deploy
docker compose -f docker-compose.yml stop sub2api
Pop-Location
docker start wujie-final-sub2
```

## 6. Video gateway operator check (no dispatch)

Before starting the approved paid window, configure the dedicated `VIDEO_GATEWAY_ENCRYPTION_KEY`, worker switch, pricing/exchange-rate fields and both CNY estimate/maximum fields in the private `deploy/.env`. Keep `VIDEO_SINGLE_SMOKE_AUTHORIZED=false` until the provider, employee group and budget are reviewed. The canonical compose now forwards these variables explicitly; every paid-runtime control defaults inert. Never reuse the JWT or TOTP key and never print the expanded compose configuration.

1. Open `/admin/video/providers`, bind Seedance to the controlled employee group, save the secret, and verify that only a masked value is rendered.
2. Keep the provider disabled until the group and model are reviewed. Enabling it alone must not dispatch a request.
3. Grant the one-time `tiny_real` authorization only inside the approved paid-test window. This records the gate and still performs zero upstream calls.
   Dispatch requires both the process kill switch and the unconsumed database provider grant. The worker atomically consumes the provider grant, the global gate, and the task dispatch claim before any upstream request.
   The canonical contract is read-only in the UI: model `doubao-seedance-2-0-260128`, endpoint `https://ark.cn-beijing.volces.com/api/v3`, 4 seconds, 720p.
4. Open `/admin/video/system-check` and record provider, authorization, task, global gate, and real dispatch counters before the paid smoke.
5. After an employee API-key request, use `/admin/video/tasks` and its detail page to capture local/upstream task ids, terminal status, asset URL, explicit error, cost, and `real_dispatch_count`.

Never print provider secrets, `.env`, expanded compose configuration, cookies, or employee API keys. UI save/authorization is not real-generation evidence; the paid smoke additionally requires a real upstream id, terminal result, delivered asset, usage/cost delta, and incremented dispatch count.
