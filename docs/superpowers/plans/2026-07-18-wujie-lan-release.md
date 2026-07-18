# 无界公司内网 Release Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 交付一个管理员专用 Web、公司内网 HTTPS 唯一入口，以及供 QCanvas 持续调用大模型、图片和视频的正式 release candidate。

**Architecture:** 以 `70d296f09` 的统一控制台为集成主线，按行为移植 K3 `830581592` 的真实图片资产和视频可靠性能力；不整分支合并、不复用 K3 迁移编号。应用保持 loopback，由 Windows Caddy 暴露内网 TLS；生产数据使用全新卷。

**Tech Stack:** Go/Gin/Ent/PostgreSQL/Redis、Vue 3/TypeScript/Vitest、Docker Compose、PowerShell、Caddy。

## Global Constraints

- Web 仅管理员可登录；注册、OAuth、自助用户、支付、订阅、返利、员工工作台和 mock/review 入口必须同时从 UI 与服务端阻断。
- QCanvas 只使用 API Key；所有视频创建、查询、取消和下载必须校验 UserID、APIKeyID、GroupID。
- 正式调用无内部日/月预算上限；reservation 只保证原子记账，未知价格必须 fail-closed。
- 视频 production 不依赖一次性 smoke；保留持久化 production authorization、紧急停止、429 退避、dispatch unknown 和持久资产。
- 应用仅绑定 `127.0.0.1:8080`；PostgreSQL/Redis 不发布端口；LAN 只开放 Caddy 443。
- 不读取或提交真实密钥，不 push，不复用现有测试卷，不更新 main，直到全部放行门槛通过。
- 所有行为变更按 RED → GREEN → REFACTOR；每个任务独立提交并经任务审查。

---

### Task 0: 基线与真相源

**Files:**
- Modify: `backend/internal/handler/page_handler_test.go`
- Modify: `backend/internal/handler/page_handler.go`
- Modify: `docs/goals/03_CURRENT_GOAL.md`

**Interfaces:**
- Produces: Windows 与 POSIX 一致的页面图片路径解析；release 目标与状态真相源。

- [x] 复现 `go test ./internal/handler -run TestResolvePageImagePath -count=1` 的 Windows 失败并记录根因。
- [x] 用最小回归用例锁定直接图片路径与目录逃逸边界，确认 RED。
- [x] 只修路径规范化根因，确认 focused test 与 `go test ./internal/handler -count=1` GREEN。
- [x] 更新当前目标为 release 分支、明确现有 head 不可上线与后续任务。
- [x] 提交 `fix(handler): normalize page image paths on windows`（`8286402fb`）。

### Task 1: 管理员专用控制台

**Files:**
- Modify: `frontend/src/components/layout/roleAwareNavigation.ts`
- Modify: `frontend/src/router/index.ts`
- Modify: `backend/internal/server/middleware/backend_mode_guard.go`
- Test: `frontend/src/components/layout/__tests__/roleAwareNavigation.spec.ts`
- Test: `backend/internal/server/middleware/backend_mode_guard_test.go`

**Interfaces:**
- Produces: 五个管理员顶层入口；`lan_admin` 模式下注册/OAuth/用户/支付服务端拒绝；API-Key gateway 不受影响。

- [ ] 先写失败测试：管理员只有五个入口，员工/支付/Studio 路由不可达，注册和普通登录返回管理员专用错误。
- [ ] 实现 `lan_admin` 部署档案与路由守卫；前端删除旧入口和跳转。
- [ ] 运行 focused frontend/backend tests、typecheck，确认 GREEN。
- [ ] 提交 `feat(console): enforce lan admin only surface`。

### Task 2: 持续图片与正式账号隔离

**Files:**
- Create: `backend/migrations/187_batch_image_assets.sql`
- Create: `backend/internal/repository/batch_image_asset_repo.go`
- Create: `backend/internal/service/batch_image_asset_archive.go`
- Modify: `backend/internal/service/batch_image_public.go`
- Modify: `backend/internal/service/batch_image_processor.go`
- Modify: `backend/internal/service/batch_image_download.go`

**Interfaces:**
- Produces: 批图本地归档、下载优先本地资产、资产归属复用；正式调度永久排除 `review_only`/mock/test 账号。

- [ ] 从 K3 行为契约重写归档/复用测试并确认在统一控制台基线 RED。
- [ ] 以 187 新迁移重新实现表和 repository，不复制 K3 的 175 迁移。
- [ ] 接入 processor、download 和 provider selection，失败时保留真实 delivery 状态。
- [ ] 启用 batch API/queue 的 LAN 配置契约，运行 service/repository/migration tests。
- [ ] 提交 `feat(image): persist production batch assets`。

### Task 3: 持续视频可靠性与 QCanvas 契约

**Files:**
- Create: `backend/migrations/188_video_production_reliability.sql`
- Modify: `backend/internal/service/video_gateway_service.go`
- Modify: `backend/internal/service/video_gateway_worker.go`
- Modify: `backend/internal/repository/video_gateway_repo.go`
- Modify: `backend/internal/handler/video_gateway_handler.go`
- Modify: `backend/internal/server/routes/gateway.go`

**Interfaces:**
- Produces: 完整视频请求契约；幂等、reservation、dispatch CAS/unknown、退避、原子终态、outbox、持久资产和 API-Key 下载。

- [ ] 写失败契约测试：文/图/参考视频、横竖比、时长、480/720/1080、尾帧；production 连续创建不消费 single-smoke。
- [ ] 写失败范围测试：创建/查询/取消/下载均需 UserID+APIKeyID+GroupID；幂等键按 API Key 范围隔离。
- [ ] 写失败可靠性测试：429 Retry-After、网络/5xx 退避、dispatch unknown 不重发、归档未完成不返回 deliverable。
- [ ] 以 188+ 迁移新增可靠性字段/账本/outbox；旧 tiny 表只读 legacy。
- [ ] 按 K3 行为移植可靠性模块并适配统一控制台接口，不复制冲突迁移。
- [ ] 增加 `GET /v1/video/tasks/:id/local-asset` API-Key 路由和稳定任务响应字段。
- [ ] 运行 video service/repository/routes tests 与 race-sensitive focused suites。
- [ ] 提交 `feat(video): enable continuous qcanvas production`。

### Task 4: 可靠 usage 与价格门禁

**Files:**
- Modify: `backend/internal/handler/gateway_handler.go`
- Modify: `backend/internal/handler/openai_gateway_handler.go`
- Modify: `backend/internal/service/usage_record_worker_pool.go`
- Modify: `backend/internal/service/openai_gateway_usage.go`

**Interfaces:**
- Produces: 文本 usage 提交失败时同步兜底或持久 outbox；正式模式未知价格拒绝调用前置检查。

- [ ] 写失败测试：worker queue full/stopped 时 usage 不丢；未知价格在真实调用前返回稳定错误。
- [ ] 复用图片 mandatory fallback 或持久 outbox，保持 API 响应兼容。
- [ ] 增加 release preflight 的启用模型价格覆盖检查。
- [ ] 运行 handler/service/billing tests。
- [ ] 提交 `fix(billing): fail closed and persist gateway usage`。

### Task 5: Windows LAN 部署与恢复

**Files:**
- Create: `deploy/Caddyfile.lan`
- Create: `deploy/wujie-lan-bootstrap.ps1`
- Create: `deploy/wujie-lan-start.ps1`
- Create: `deploy/wujie-lan-backup.ps1`
- Create: `deploy/wujie-lan-restore-drill.ps1`
- Modify: `deploy/docker-compose.yml`

**Interfaces:**
- Produces: 固定 compose project/卷、loopback app、Caddy internal TLS、CIDR 防火墙、分层 bootstrap、备份/恢复与 release manifest。

- [ ] 先写 Pester/契约测试，确认脚本不再硬编码 worktree/branch，且拒绝空 secrets、非 loopback app、未授权 CIDR。
- [ ] 实现 DB/Redis healthy → migration → readiness → admin/TOTP → Caddy 443 的 bootstrap。
- [ ] 实现镜像 digest/SHA-256 manifest、离机 PostgreSQL+资产备份和隔离恢复。
- [ ] 验证 compose config 不暴露 5432/6379/8080，Caddy admin 仅 loopback。
- [ ] 提交 `feat(deploy): add windows lan release bundle`。

### Task 6: 全量验证、安全审查与审查包

**Files:**
- Modify: `docs/00_START_HERE.md`
- Modify: `docs/reviews/LATEST_REVIEW_PACKAGE.html`

**Interfaces:**
- Produces: 单一 release commit、完整证据与明确 READY/阻塞状态。

- [ ] 运行 Go full test/vet、frontend lint/typecheck/test/build、git diff check、空库迁移和 Docker isolated smoke。
- [ ] 浏览器验证唯一登录、五个管理员入口、旧入口阻断、1440/390、LAN TLS；截图不含密钥。
- [ ] 运行 release candidate 全库安全扫描并关闭所有 P0/P1，或把未关闭项明确标记已阻塞。
- [ ] 真实付费调用只在目标机录入密钥并再次现场确认后执行；记录真实 provider/task/asset/cost 证据。
- [ ] 完成冷重启与备份恢复；更新审查包、release manifest、风险和回滚。
- [ ] 仅当全部门槛通过，快进本地 main；否则保持 release 分支，不 push。
