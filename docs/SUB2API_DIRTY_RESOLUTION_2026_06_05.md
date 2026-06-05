# SUB2API_DIRTY_RESOLUTION_2026_06_05

最终状态：**SUB2API_SAFE_PARTIAL_DIRTY**

一句话结论：Sub2API 的 Day0 / 视频网关 / 产品模式 / UI 收口改动已形成本地 checkpoint，数据库备份已隔离；仍剩部署风险文件和两份文档需人工确认，因此可以作为 QCanvas P0.9E 功能救火的相对安全基线，但不能宣布工作区完全 clean。

## 1. 当前 branch / head

- current branch: $branch
- head: $head
- safety branch: safety-sub2api-before-dirty-resolution-20260605
- checkpoint commit hash: $checkpoint
- checkpoint commit message: wip: checkpoint sub2api day0 video gateway and white-label worktree

## 2. 文件分类表

### A. DAY0_RELATED_CAN_COMMIT

已提交：

- .dockerignore
- deploy/.gitignore
- ackend/internal/service/setting_service.go
- ackend/internal/web/embed_on.go
- ackend/internal/web/embed_test.go
- ackend/migrations/138_day0_wujie_video_copy.sql

### B. PRODUCT_MODE_OR_UI_CAN_COMMIT

已提交：

- ackend/internal/service/video_gateway_service.go
- rontend/index.html
- rontend/package.json
- rontend/src/components/layout/AppHeader.vue
- rontend/src/components/layout/AppLayout.vue
- rontend/src/components/layout/AppSidebar.vue
- rontend/src/components/layout/AuthLayout.vue
- rontend/src/i18n/index.ts
- rontend/src/i18n/locales/en.ts
- rontend/src/i18n/locales/zh.ts
- rontend/src/main.ts
- rontend/src/router/__tests__/title.spec.ts
- rontend/src/router/index.ts
- rontend/src/router/title.ts
- rontend/src/stores/app.ts
- rontend/src/utils/productMode.ts
- rontend/src/views/HomeView.vue
- rontend/src/views/InternalPilotView.vue
- rontend/src/views/KeyUsageView.vue
- rontend/src/views/admin/SettingsView.vue
- rontend/src/views/admin/video/VideoCreateTaskView.vue
- rontend/src/views/admin/video/VideoDashboardView.vue
- rontend/src/views/admin/video/VideoProvidersView.vue
- rontend/src/views/admin/video/VideoSystemCheckView.vue
- rontend/src/views/admin/video/VideoTaskDetailView.vue
- rontend/src/views/admin/video/VideoTasksView.vue
- rontend/src/views/admin/video/videoUtils.ts
- rontend/src/views/auth/EmailVerifyView.vue
- rontend/src/views/auth/LoginView.vue
- rontend/src/views/auth/RegisterView.vue
- rontend/src/views/public/LegalDocumentView.vue
- rontend/src/views/setup/SetupWizardView.vue
- rontend/src/views/user/KeysView.vue

### C. DEPLOY_RISK_NEEDS_CONFIRM

未提交：

- Dockerfile
- deploy/Caddyfile
- deploy/Dockerfile
- deploy/config.example.yaml
- deploy/backup.sh
- deploy/day0/backup.sh
- deploy/day0/check.sh
- deploy/day0/start.sh
- deploy/day0/stop.sh
- deploy/day0/windows_disable_lan_access.ps1
- deploy/day0/windows_enable_lan_access.ps1
- deploy/docker-compose.wsl.prod.yml
- deploy/docker-compose.wsl.yml

原因：这些文件会影响 Docker 构建、Caddy 反代、compose、备份脚本或 LAN 访问面，默认不进入 checkpoint。

### D. LOCAL_ARTIFACT_QUARANTINE

已隔离：

- deploy/backups/sub2api_20260530_015821.sql.gz
- deploy/backups/wujie_api_20260530_035938.sql.gz
- deploy/backups/wujie_api_20260530_142707.sql.gz
- deploy/backups/wujie_api_20260530_143430.sql.gz
- deploy/backups/wujie_api_20260530_144330.sql.gz
- deploy/backups/wujie_api_20260530_145043.sql.gz
- deploy/backups/wujie_api_20260530_152635.sql.gz
- deploy/backups/wujie_api_20260530_160125.sql.gz

隔离目录：D:\Codex创业任务_PROJECT_REGISTRY\SUB2API_QUARANTINE_2026_06_05\

manifest：D:\Codex创业任务_PROJECT_REGISTRY\SUB2API_QUARANTINE_MANIFEST_2026_06_05.md

### E. NEEDS_USER_CONFIRM

未提交：

- AUDIT_REPORT.md
- BOSS_DEPLOY_GUIDE.md

原因：文档名显示可能包含审计结论、操作员登录说明、部署细节或临时凭据，本轮未纳入 checkpoint。

## 3. 已提交文件

`	ext
.dockerignore
backend/internal/service/setting_service.go
backend/internal/service/video_gateway_service.go
backend/internal/web/embed_on.go
backend/internal/web/embed_test.go
backend/migrations/138_day0_wujie_video_copy.sql
deploy/.gitignore
frontend/index.html
frontend/package.json
frontend/src/components/layout/AppHeader.vue
frontend/src/components/layout/AppLayout.vue
frontend/src/components/layout/AppSidebar.vue
frontend/src/components/layout/AuthLayout.vue
frontend/src/i18n/index.ts
frontend/src/i18n/locales/en.ts
frontend/src/i18n/locales/zh.ts
frontend/src/main.ts
frontend/src/router/__tests__/title.spec.ts
frontend/src/router/index.ts
frontend/src/router/title.ts
frontend/src/stores/app.ts
frontend/src/utils/productMode.ts
frontend/src/views/HomeView.vue
frontend/src/views/InternalPilotView.vue
frontend/src/views/KeyUsageView.vue
frontend/src/views/admin/SettingsView.vue
frontend/src/views/admin/video/VideoCreateTaskView.vue
frontend/src/views/admin/video/VideoDashboardView.vue
frontend/src/views/admin/video/VideoProvidersView.vue
frontend/src/views/admin/video/VideoSystemCheckView.vue
frontend/src/views/admin/video/VideoTaskDetailView.vue
frontend/src/views/admin/video/VideoTasksView.vue
frontend/src/views/admin/video/videoUtils.ts
frontend/src/views/auth/EmailVerifyView.vue
frontend/src/views/auth/LoginView.vue
frontend/src/views/auth/RegisterView.vue
frontend/src/views/public/LegalDocumentView.vue
frontend/src/views/setup/SetupWizardView.vue
frontend/src/views/user/KeysView.vue
`

## 4. 安全扫描结果

- 是否读取 .env/key/token/cookie 原文：否。
- 是否打印密钥：否。
- 是否发现疑似真实 secret：否。
- 是否 push/deploy：否。
- 是否真实 provider 调用：否。
- 高置信形态扫描：已提交内容仅命中 rontend/src/i18n/locales/en.ts、rontend/src/i18n/locales/zh.ts 中的占位 sk-... 与代理 URL 示例格式；已用掩码确认，不是实值。
- 数据库备份：8 个 .sql.gz 已移出 repo 并写入 quarantine manifest。

## 5. 测试结果

- go service tests: PASS
  - go test ./internal/service -run "TestVideoAdapterContractSafeProviderBehavior|TestOpenAIGatewayServiceParseOpenAIImagesRequest_JSON|TestOpenAIGatewayServiceForwardImages_APIKeyGenerationUsesConfiguredV1BaseURL"
- go routes tests: PASS
  - go test ./internal/server/routes -run TestGatewayRoutesOpenAIImagesPathsAreRegistered
- frontend route title test: PASS
  - .\node_modules\.bin\vitest.CMD run src/router/__tests__/title.spec.ts
- frontend build: PASS
  - .\node_modules\.bin\vite.CMD build
  - 备注：构建存在既有 chunk size / dynamic import / Browserslist 过期 warning，未作为本轮阻塞。
- diff check: PASS with CRLF warnings only

## 6. git diff --check 结果

`	ext
warning: in the working copy of 'deploy/Caddyfile', LF will be replaced by CRLF the next time Git touches it
`

## 7. 当前未提交 diff --stat

`	ext
warning: in the working copy of 'deploy/Caddyfile', LF will be replaced by CRLF the next time Git touches it
 Dockerfile                 |  9 +++++----
 deploy/Caddyfile           | 45 +++++++++++++++++++--------------------------
 deploy/Dockerfile          |  9 +++++----
 deploy/config.example.yaml |  2 +-
 4 files changed, 30 insertions(+), 35 deletions(-)
`

## 8. git status --short 完整结果

`	ext
 M Dockerfile
 M deploy/Caddyfile
 M deploy/Dockerfile
 M deploy/config.example.yaml
?? AUDIT_REPORT.md
?? BOSS_DEPLOY_GUIDE.md
?? deploy/backup.sh
?? deploy/day0/backup.sh
?? deploy/day0/check.sh
?? deploy/day0/start.sh
?? deploy/day0/stop.sh
?? deploy/day0/windows_disable_lan_access.ps1
?? deploy/day0/windows_enable_lan_access.ps1
?? deploy/docker-compose.wsl.prod.yml
?? deploy/docker-compose.wsl.yml
`

## 9. 回滚方式

- 回到整理前安全指针：git switch phase-3.8.2-overnight-readiness && git reset --hard safety-sub2api-before-dirty-resolution-20260605。注意：这是破坏性回滚，必须人工确认后执行。
- 仅回滚 checkpoint commit：git revert c4e2337d。
- 恢复隔离备份：执行 D:\Codex创业任务_PROJECT_REGISTRY\SUB2API_QUARANTINE_MANIFEST_2026_06_05.md 中每条 rollbackCommand。

## 10. 是否允许继续 P0.9E QCanvas 功能救火

允许继续，但条件是：

- 基于 checkpoint $checkpoint 之后继续；
- 不把剩余 C 类部署风险文件带入功能调试；
- 不提交或读取 AUDIT_REPORT.md / BOSS_DEPLOY_GUIDE.md 中可能存在的账号、凭据或部署细节；
- 不把 dry-run 写成 real，不真实调用 provider，不 push/deploy。

