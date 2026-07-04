# Sub2API 内网部署前配置检查表

> 更新：2026-07-04。适用场景：5–10 人内部小团队、内网/VPN 访问、非公开 SaaS。
> 配置来源：`backend/internal/config/config.go`（Viper 加载）、`deploy/config.example.yaml`、启动与 migration 代码。
> 环境变量规则：Viper `AutomaticEnv` + `.` → `_`（如 `jwt.secret` → `JWT_SECRET`）。

## 0. 部署前总览（先勾再启动）

| # | 检查项 | 通过标准 |
|---|--------|----------|
| 0.1 | PostgreSQL 可达 | 能连接，库已创建或可自动创建 |
| 0.2 | Redis 可达 | `PING` 成功 |
| 0.3 | `config.yaml` 或 `AUTO_SETUP` | 首次安装二选一 |
| 0.4 | 密钥类配置已固定 | JWT / TOTP / 视频网关加密密钥不依赖"每次重启随机生成" |
| 0.5 | 注册策略已决定 | 内网建议关闭自助注册 |
| 0.6 | 无需手动 `psql` 跑 migration | 启动时自动执行 `backend/migrations/`（见 §9） |
| 0.7 | 前端访问方式明确 | 官方 Docker 镜像为 embed 单二进制；源码直编需 `-tags embed` |

配置文件搜索顺序（`config.Load`）：`$DATA_DIR/config.yaml` → `/app/data/config.yaml` → `./config.yaml` → `./config/config.yaml` → `/etc/sub2api/config.yaml`。

## 1. 运行模式 `run_mode`

- 键名：`run_mode` / `RUN_MODE`；默认 `standard`；非法值回退 `standard`。
- `simple` 模式跳过：API Key 计费/配额、余额/订阅检查、分组隔离调度；仍写 usage log 但不扣费；启动打印 `Running in SIMPLE mode - billing and quota checks are DISABLED`。
- 内网推荐：纯内部网关用 `simple`；需要余额/订阅/RPM 治理用 `standard`。

## 2. JWT Secret

- 键名：`jwt.secret` / `JWT_SECRET`；相关：`jwt.expire_hour`（默认 24，最大 168）、`jwt.refresh_token_expire_days`（默认 30）。
- 启动校验：`cfg.Validate()` 要求非空且 ≥32 字节；弱口令仅 warn。
- 配置为空时：`ensureBootstrapSecrets()` 自动生成 32 字节 hex 并持久化到 DB `security_secrets`（key=`jwt_secret`），跨重启一致；配置与 DB 不一致时以 DB 为准并 warn。
- 内网推荐：显式设置 `openssl rand -hex 32`；多实例必须一致。

## 3. TOTP / 支付 / 视频加密密钥

| 密钥 | 键名 | 缺失行为 | 内网推荐 |
|------|------|----------|----------|
| TOTP | `totp.encryption_key` / `TOTP_ENCRYPTION_KEY` | 自动生成随机值 + `EncryptionKeyConfigured=false`；重启后变化，已绑 2FA 用户失效；管理后台开 TOTP 会报 400 | 固定 64 hex |
| 支付加密/resume 签名 | 派生自 TOTP 密钥；可用 `PAYMENT_RESUME_SIGNING_KEY` 覆盖 | 未配置 → nil + warn，resume token 跨重启失效 | 用支付时必须固定 TOTP 密钥 |
| 视频网关 | `video_gateway.encryption_key` / `VIDEO_GATEWAY_ENCRYPTION_KEY` | DI 层（`video_key_encryptor.go`）空密钥启动失败；AUTO_SETUP 可自动生成写入 config.yaml | 必填 64 hex，与 TOTP 密钥不同 |

## 4. 数据库 / Redis

- PostgreSQL：`DATABASE_HOST/PORT/USER/PASSWORD/DBNAME/SSLMODE`；连接池 `DATABASE_MAX_OPEN_CONNS`（默认 256，小团队 32–64 够用）。
- Redis：`REDIS_HOST/PORT/PASSWORD/DB`；池 `REDIS_POOL_SIZE`（默认 1024，小团队 64–128）。
- 内网无 TLS：`DATABASE_SSLMODE=disable` 可接受；建议给 Redis 设密码。

## 5. Admin API Key

- 存储在 DB `settings` 表键 `admin_api_key`，格式 `admin-` + hex。
- 设置方式：管理后台 设置 → Admin API Key → 生成；认证用请求头 `x-api-key`。
- 未配置时 Admin API Key 路径统一 401（JWT 管理员登录不受影响）。

## 6. 注册开关

- DB 键 `registration_enabled`；查询逻辑 fail-closed（缺失/出错 → 关闭）。
- 辅助键：`invitation_code_enabled`、`email_verify_enabled`、`registration_email_suffix_whitelist`。
- 内网推荐：保持关闭，管理员手工建号。

## 7. 视频网关安全开关

- `video_gateway.worker_enabled`（默认 true）、`poll_interval_seconds`×`max_poll_attempts` 必须 ≥360s，否则拒绝启动。
- Budget guard：仅当 `per_call_budget > 0` 且 `cost_per_second > 0` 才启用 `StaticBudgetGuard`，默认都为 0 = 不启用（P1-019 残余，接真 provider 前必须配置）。
- 真实 provider 三闸：`SUB2API_VIDEO_REAL_SMOKE_ENABLED=1` + `SUB2API_VIDEO_REDACTED_EVENT_LOG` + `SUB2API_VIDEO_URL_ALLOWLIST`，另需账号 metadata `single_smoke_authorized`。
- API Key 路径默认 mock-only；drama 演示 `SafeDemoOnly` 只路由 mock。

## 8. 日志与敏感信息

- `LOG_LEVEL`（默认 info）、`LOG_FORMAT`（容器建议 json）。
- `GATEWAY_LOG_UPSTREAM_ERROR_BODY` 默认 true，生产建议 false（排障时临时开）。
- `GATEWAY_CONTENT_CAPTURE_ENABLED` / `GATEWAY_CONTENT_RETENTION_ENABLED` 默认 false；开启即采集脱敏 prompt/response 进 `ai_generation_content`（B1 护城河功能，按需开启）。
- `SERVER_MODE=release`（团队使用时）；`security.response_headers.enabled=true` 保持。

## 9. Migration 执行方式

- 自动：每次启动 `InitEnt()` → `applyMigrationsFS(migrations.FS)`（嵌入二进制的 `backend/migrations/*.sql`）。
- 记录表 `schema_migrations`（filename + checksum）；多实例用 PostgreSQL Advisory Lock 防并发。
- 正常部署不需要手动 psql。

## 10. 监听与前端

- `SERVER_HOST`/`SERVER_PORT`（默认 0.0.0.0:8080）；`server.trusted_proxies` 有反代时必填。
- 官方 Dockerfile `-tags embed` 嵌入前端；无 embed 构建需独立 nginx 反代 `frontend/dist`。

## 11. 密钥生成

```bash
openssl rand -hex 32   # JWT / TOTP / VIDEO_GATEWAY 各一次，勿复用
```

## 12. 启动后必做检查（5 分钟）

1. 日志无 config/migration 错误
2. 确认 run_mode 与预期一致（simple 会打印 billing disabled 警告）
3. 管理员登录 → 确认注册关闭
4. 需要脚本调 admin API 时生成 Admin API Key
5. 创建用户/API Key，打一条网关请求验证
6. 用视频时确认 `video_gateway.encryption_key` 已固定

---

## 附：当前 dev 实例核对结果（2026-07-04）

对 `sub2api-dev`（127.0.0.1:18081）的实际核对：

| 项 | 状态 | 说明 |
|----|------|------|
| RUN_MODE | `standard` ✅ | 计费/配额启用 |
| JWT_SECRET | env 为空，DB `security_secrets` 已持久化 ✅ | 跨重启一致；正式部署建议显式配置 |
| TOTP_ENCRYPTION_KEY | env 为空 ⚠️ | 2FA/支付加密暂不可用；启用支付或 2FA 前必须固定 |
| VIDEO_GATEWAY_ENCRYPTION_KEY | env 为空，AUTO_SETUP 已生成写入 `/app/data/config.yaml` ✅ | 卷持久化，勿删卷 |
| SUB2API_VIDEO_REAL_SMOKE_ENABLED | `0` ✅ | 真实视频冻结 |
| GATEWAY_CONTENT_CAPTURE_ENABLED | `true`（有意开启，B1 账本） | retention=false |
| SERVER_MODE | `debug` ⚠️ | 团队使用前改 `release` |
| registration_enabled | 关闭 ✅ | `/settings/public` 确认 |
| admin_api_key | 未配置 | 不用脚本时可暂不配 |
| migration 148 | 已应用并登记 ✅ | 唯一索引已建 |
