# Sub2API 内部真实可用收口任务包

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` task-by-task. 每个任务先写失败测试，再做最小实现，再由独立 reviewer 复核。持续执行到 `READY_FOR_USER_REAL_TEST`，不要在阶段之间询问用户是否继续。

**Goal:** 在不新增真实 Provider 调用的开发阶段，补齐员工受控真实创建入口、验收后的日常内部真实模式、Gemini 产品数据库与持久资产链、Provider 账单导入对账和三角色浏览器闭环；最后只把真实付费调用、真实账单上传和生成内容人眼验收留给用户。

**Architecture:** 产品正常路径与真实复核路径共享同一套任务、结算和资产服务，但真实复核必须经过显式 `execution_mode=review_real`、后端固定路由和跨进程原子预算硬门。Gemini 结果首次读取后归档为本地受控资产，预览/下载优先读取本地副本。Provider 对账通过账单批次、规范化账单行和内外匹配结果三层模型实现，不允许直接修改用户余额。

**Tech Stack:** Go、Gin、Ent/PostgreSQL、Redis、Vue 3、TypeScript、pnpm 9、Vitest、Testcontainers、Playwright/Edge、Docker/WSL。

## Global Constraints

- 工作目录只允许 `D:\sub2api-trunk`；当前分支应为 `wujie/video-capture-moat-20260702`。
- 接手基线提交为 `b6045db6317141a68c6adcda0fe22dada5413c48`。若 HEAD 已前进，必须先证明新增提交来源与任务相关，不得回退。
- 当前已有未提交改动：`.dockerignore`、`backend/internal/service/video_gateway_types.go`、`backend/internal/service/video_task_finalization.go`、`backend/internal/service/video_task_finalization_test.go`。它们属于用户或并行执行者；先只读审查并分类，禁止覆盖、restore、stash、reset 或顺手暂存。
- 允许本地精确暂存和分批 commit；禁止 push、部署、公开服务、生产用户数据、`git add .`、`git add -A`、`git commit -a`、reset、clean、rebase。
- 开发与自动验证阶段禁止调用真实 Gemini、Seedance 或其他付费 Provider。不得读取、打印、复制或写入任何密钥值；只允许 presence-check。
- 真实复核会话历史计数必须按现有状态继续：图片 1/4、视频 2/4、累计预留 ¥20；全局上限图片 4、视频 4、累计 ¥60。不得重置或新建计数文件规避限制。该额度仅用于验收，不得冒充团队日常额度。
- 产品执行模式固定为 `mock | review_real | internal_real`：`mock` 永不调用付费 Provider；`review_real` 受 4+4/¥60 会话硬门；`internal_real` 只在验收通过且管理员显式启用后开放，并受成员/团队日限额、月限额、余额、Provider 状态和全局 kill switch 约束。
- 只留下最后的真实测试环节给用户：真实图片 create、真实视频 create、真实账单文件上传、生成内容人眼确认、测试结束后废弃临时密钥。
- mock、fake、恢复既有 upstream task、静态截图、缓存镜像或测试 skip 都不得包装成真实用户闭环。
- 所有员工真实创建必须只产生一次上游 create；幂等重放、worker 重试和页面重复点击不得新增 create。
- 所有真实结果必须成为持久资产，可预览、下载、再次引用；`result_url` 存在不算交付完成。
- 内部 usage/价目表/账本/余额一致不等于 Provider 正式账单一致。没有真实账单样本前只能标记“待用户真实账单复核”。
- 最终重要证据必须进入单一自包含审查包 `docs/reviews/LATEST_REVIEW_PACKAGE.html`。
- 开发完成但用户尚未执行真实测试时，状态只能是 `待复核`；只有真实图片、真实视频、资产交付和真实账单样本复核均通过，才可判定 `内部可用`。

---

## 直接粘贴到新 Codex 对话的开场提示词

```text
继续完成 Sub2API 内部真实可用收口。

进入 D:\sub2api-trunk，完整读取：
docs/superpowers/codex-handoff/CODEX_TASK_REAL_PRODUCT_READINESS_20260715.md

严格从 G0 开始按顺序执行。使用 subagents 做只读侦察、任务实现后的规格复核和最终审查。持续自动执行到任务包定义的 READY_FOR_USER_REAL_TEST，不要在阶段之间问我是否继续。

开发阶段禁止调用任何真实付费 Provider，禁止读取或打印密钥，禁止 push、部署和公网暴露。允许本地精确 commit。当前工作区已有 4 个未提交文件，先审查归属，不得覆盖或清理。

你的目标不是再写计划，而是完成代码、测试、Testcontainers、Docker、本地浏览器三角色验证、审查包和最终用户真实测试卡。最后只把真实图片/视频点击、真实账单上传、人眼验收和密钥废弃留给我。
```

## 1. 强制阅读顺序

- [ ] `00_START_HERE.md`
- [ ] `01_PROJECT_BASELINE.md`
- [ ] `02_CURRENT_REALITY_STATUS.md`
- [ ] `docs/goals/03_CURRENT_GOAL.md`
- [ ] `PRODUCT_INVARIANTS.md`
- [ ] `ARCHITECTURE_GUARDRAILS.md`
- [ ] `CODE_QUALITY_GATE.md`
- [ ] `docs/reviews/LATEST_REVIEW_PACKAGE.html`
- [ ] 本任务包
- [ ] `backend/internal/reviewguard/session_guard_realsmoke.go`
- [ ] `backend/internal/service/video_gateway_worker.go`
- [ ] `backend/internal/service/video_gateway_adapter.go`
- [ ] `backend/internal/service/batch_image_public.go`
- [ ] `backend/internal/service/batch_image_processor.go`
- [ ] `backend/internal/service/batch_image_settlement.go`
- [ ] `backend/internal/service/batch_image_download.go`
- [ ] `backend/internal/service/batch_image_gemini_forma_realsmoke_test.go`
- [ ] `backend/internal/repository/video_seedance_recovery_realsmoke_integration_test.go`

不得读取真实 `.env`、交付包、数据库备份或密钥文件补充上下文。

## 2. 当前已确认事实

| 能力 | 当前证据 | 当前判定 |
|---|---|---|
| Gemini Provider | 1 次真实 Batch succeeded；同一 job Get→OpenResult→图片解码通过，0 次新增 create | 局部通过 |
| Gemini 产品链 | 尚未进入真实产品 DB、结算、服务端持久资产、员工预览下载 | 未通过 |
| Seedance Provider | 两条 5 秒真实任务 succeeded；分别 16:9、9:16，均 usage 108900 | 通过 |
| Seedance 产品恢复 | 既有 9:16 task 经 Postgres→worker Poll→finalizer→outbox→MP4 归档，审计 0 create | 通过 |
| 员工 UI create | UI 已连接产品 create API，但 mock 偏好不会锁定 mock，后端仍可能自动选真实账号 | 不安全 |
| 共享硬门 | 4+4/¥60 guard 仅在 `realsmoke` build tag 测试代码使用，未进入产品 Composition Root | 未通过 |
| 内部账务 | Provider usage→内部价目表→内部 ledger→用户余额→老板总览已对齐 | 通过 |
| Provider 正式账单 | 没有账单导入、发票归档、账单行或内外匹配模型 | 未实现 |
| 浏览器 | 老板/管理员/员工 mock 路径 59 个业务 API 2xx，7 张截图 | 可演示 |

## 3. 最终交付物

- 产品路径可启用但默认关闭的真实复核硬门。
- 图片与视频员工页面明确区分“免费试跑”“一次真实复核”和验收后的“内部真实生成”，普通员工不能裸选 Provider 账号。
- 应用进程可从 Windows 用户环境安全装配一次性复核账号；Agent/命令/日志不显示密钥。测试完成后账号可禁用并清除临时凭证。
- Gemini 既有 job 的 0-create 产品恢复 integration 证据。
- Gemini 图片服务端持久资产、预览、下载和再次引用能力。
- Provider 账单 CSV/XLSX 导入、规范化、匹配和差异审核能力。
- 老板、管理员、员工三角色无付费浏览器闭环与截图。
- 用户专用真实测试卡：只包含必要点击、预期结果、停止条件和密钥废弃动作。
- 最新真相源和 `docs/reviews/LATEST_REVIEW_PACKAGE.html`。
- 固定 closeout：`docs/superpowers/codex-handoff/deliverables/2026-07-15-REAL-PRODUCT-READINESS-closeout.md`。

---

## G0：冻结基线并保护并行改动

- [ ] 运行并保存证据：

```powershell
git status --short
git branch --show-current
git rev-parse --show-toplevel
git rev-parse HEAD
git rev-parse --git-dir
git rev-parse --git-common-dir
git log -5 --oneline
```

预期：root 为 `D:/sub2api-trunk`，branch 为 `wujie/video-capture-moat-20260702`；当前是 linked worktree。

- [ ] 对 4 个既有未提交文件执行只读 `git diff -- <exact-path>`，确认它们是否属于正在进行的视频 finalization 修复。
- [ ] 若这些改动通过测试且与本任务 G1/G2 接口重叠，则把它们作为独立前置提交收口；若来源或意图无法证明，保持未暂存并在后续实现中绕开，禁止覆盖。
- [ ] 建立 `.delivery-tools/real-product-readiness-20260715/` 作为忽略的本机证据目录，不放密钥、Authorization、签名 URL 或生产数据。

**G0 gate：** 未分类既有改动、repo/branch 不符、暂存区混入无关文件时停止实施并标记 `已阻塞`。

## G1：把共享真实预算硬门接入产品 create chokepoint

**Files:**

- Modify or split: `backend/internal/reviewguard/session_guard_realsmoke.go`
- Create: `backend/internal/reviewguard/session_guard.go`
- Create: `backend/internal/reviewguard/session_guard_test.go`
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/config/config_test.go`
- Modify: `backend/internal/service/video_gateway_worker.go`
- Modify: `backend/internal/service/video_gateway_worker_test.go`
- Modify: `backend/internal/service/batch_image_public.go`
- Test: `backend/internal/service/batch_image_public_test.go`
- Modify: `backend/internal/service/wire.go`
- Modify: `backend/cmd/server/wire.go`

**Interfaces:**

```go
type RealCreateKind string

const (
    RealCreateImage RealCreateKind = "image"
    RealCreateVideo RealCreateKind = "video"
)

type RealCreateReservation struct {
    OperationID  string
    Kind         RealCreateKind
    ReservedCNY  decimal.Decimal
    PricingSource string
    PricingVersion string
}

type RealCreateSnapshot struct {
    ImageUsed       int
    ImageRemaining  int
    VideoUsed       int
    VideoRemaining  int
    ReservedCNY     decimal.Decimal
    RemainingCNY    decimal.Decimal
    PricingVersion  string
}

type RealCreateGuard interface {
    Reserve(ctx context.Context, reservation RealCreateReservation) error
    Snapshot(ctx context.Context) (RealCreateSnapshot, error)
}
```

- [ ] 先写失败测试，证明普通构建中 guard 默认关闭、启用时必须使用绝对 state path、损坏 state fail-closed、图片第 5 次/视频第 5 次/累计超过 ¥60 被拒绝、跨进程并发不会超额。
- [ ] `OperationID` 使用本地稳定任务 ID/幂等键；同一 OperationID 和相同参数重复 Reserve 返回成功但不重复计数，参数不一致则 fail-closed。
- [ ] 图片当前 hold 以 USD 保存，禁止把 `HoldAmount` 直接当 CNY。使用创建任务时冻结的定价快照和审查会话固定汇率转换为 `ReservedCNY`；视频同样记录 pricing source/version。所有运算使用 decimal，不使用 float。
- [ ] 运行：

```powershell
cd backend
go test ./internal/reviewguard -count=1
```

预期：新测试在实现前 FAIL。

- [ ] 将锁、原子状态写入和 4/4/60 规则移入无 build tag 的核心文件；`realsmoke` 文件只保留测试适配，不复制规则。
- [ ] 配置仅在 `SUB2API_REAL_REVIEW_SESSION_ENABLED=1` 且 state path 为绝对路径时注入真实 guard；其他环境注入拒绝真实复核的 fail-closed guard，而不是无限放行。
- [ ] 视频在 worker CAS 成功、真正调用 `adapter.CreateTask` 之前执行一次 Reserve；guard 失败时 adapter create 调用数必须为 0，任务进入可解释失败终态并释放 billing reservation。
- [ ] 图片在 `provider.Submit/CreateBatch` 前执行一次 Reserve；参数校验失败、幂等重放和恢复既有 job 不消耗次数。若 job/hold 已创建，guard 拒绝必须调用既有 pre-upstream failure 清理，释放 frozen balance 并记录人话错误。
- [ ] 增加并发、重复 worker、幂等重放和 guard 错误测试。
- [ ] 增加 admin-only capability/status API，返回 enabled、图片/视频剩余次数、剩余 CNY、定价版本；不得返回 state path、凭证或 Provider secret。
- [ ] 运行：

```powershell
cd backend
go test ./internal/reviewguard ./internal/service ./internal/config ./cmd/server -run 'RealCreate|RealReview|Budget|Idempot|Worker|BatchImage' -count=1
```

- [ ] 精确提交：`feat(review): gate product real creates atomically`。

**G1 gate：** 任意真实 create 能绕过 guard、同一任务重复计次或 guard 依赖前端自觉时不得进入 G2。

## G2：图片与视频增加明确且不可绕过的三种执行模式

**Files:**

- Modify: `backend/internal/handler/video_handler.go`
- Modify: `backend/internal/service/video_gateway_types.go`
- Modify: `backend/internal/service/video_gateway_service.go`
- Modify: `backend/internal/handler/batch_image_handler.go`
- Modify: `backend/internal/service/batch_image_public.go`
- Modify: `frontend/src/api/admin/video.ts`
- Modify: `frontend/src/views/admin/video/VideoCreateTaskView.vue`
- Modify: `frontend/src/api/batchImage.ts`
- Modify: `frontend/src/views/user/BatchImageGuideView.vue`
- Create: product-level mock batch image provider and deterministic PNG/JSONL fixture under existing service test patterns.
- Create or modify tests under corresponding `__tests__` and backend handler/service tests.

**Contract:**

```text
execution_mode = mock | review_real | internal_real
```

- [ ] 先写图片和视频失败测试：非管理员请求传裸 `provider_account_id` 必须被忽略或拒绝；`mock` 只能路由 mock；`review_real` 只能路由管理员预先授权的单个复核账号；未启用 session 时返回人话错误且 create=0。
- [ ] Handler 不再允许普通员工通过构造 JSON 枚举 Provider 账号。
- [ ] 图片和视频后端均根据 `execution_mode` 选择路由，不能继续让 UI 的“mock 偏好”落成 `provider_account_id=0` 后自动路由。
- [ ] 增加产品级 mock 图片 Provider：只在本地/复核配置中注册，生成确定性的本地 PNG 和 JSONL，完整经过 batch job/items、结算 not_required、持久资产、预览和下载；普通 production 配置默认不注册。
- [ ] 增加 `ReviewCredentialBootstrap`：只有 review session 开启时，应用进程直接消费 `GEMINI_API_KEY` 和 `SUB2API_SEEDANCE_SMOKE_API_KEY`，使用现有加密存储创建/更新两个 `review_only` Provider 账号；禁止日志、响应、审查包或命令输出值。Bootstrap 只返回 presence/account ID/status，缺失时 fail-closed。
- [ ] 临时复核账号不得被 `internal_real` 使用；用户测试后提供 admin-only 禁用/清除动作。日常 `internal_real` 使用管理员正常配置并加密保存的正式内部账号。
- [ ] 增加最小 `provider_real_access_policies`（或遵循现有 schema 的等价模型）：成员/分组 allow、图片/视频日 CNY、月 CNY、全局 kill switch、enabled_at/disabled_at/audit actor。`internal_real` 同时受用户余额和该策略约束。
- [ ] `internal_real` 额度不能只在 create 前查询累计值。任务创建与成员/团队/媒体类型日月 CNY reservation 必须在同一数据库事务中原子预留；以稳定 OperationID 幂等，重复请求不重复占用；终态按实际金额结算差额，失败/取消释放。增加跨进程并发 Testcontainers，证明临界额度下不会超额。
- [ ] `internal_real` capability 只有在至少一个 enabled、非 `review_only`、凭证可解密且健康的正式 Provider 账号存在时才允许开启；否则策略保存失败并显示“请先配置正式内部通道”，不得回退使用临时复核账号。
- [ ] UI 默认选“免费试跑”；“一次真实复核”只有后端 capability 明确开启时显示；`internal_real` 只有验收后管理员策略允许时显示。
- [ ] 真实确认弹窗必须显示模型、时长、分辨率、比例、预计最高费用、会计入次数与费用，并要求二次确认。
- [ ] 图片确认弹窗同样显示模型、张数、规格、预计最高费用和次数；图片请求使用稳定 Idempotency-Key。
- [ ] 提交按钮在请求期间禁用；重复点击复用同一 Idempotency-Key。
- [ ] 余额不足、凭证失效、限流、预算硬门、Provider 不可用、轮询超时都显示下一步动作，不显示底层 Key、Authorization 或签名 URL。
- [ ] 用 fake Seedance adapter 和产品 mock 图片 Provider 分别跑员工 UI→create→poll→succeeded→预览→下载→再次引用，断言每个任务 upstream/create=1。
- [ ] 运行：

```powershell
cd backend
go test ./internal/handler ./internal/service -run 'Video.*Create|BatchImage|ExecutionMode|ProviderAccount|RealReview|InternalReal' -count=1
cd ..\frontend
pnpm.cmd exec vitest run src/views/admin/video --reporter=basic
pnpm.cmd exec vue-tsc --noEmit
```

- [ ] 精确提交：`feat(video): add guarded employee real review mode`。

## G3：Gemini 既有任务 0-create 产品数据库恢复

**Files:**

- Create: `backend/internal/repository/batch_image_gemini_recovery_integration_test.go`
- Modify only if test exposes a product defect: batch image processor/repository/settlement files.

- [ ] 先创建 Testcontainers Postgres/Redis 中的用户、Gemini account、任务、items、余额和冻结记录；任务状态设为 submitted 并绑定既有 upstream job name。
- [ ] 包装 Provider，记录 `Submit/Get/OpenResult` 调用次数；恢复必须断言 `Submit=0`、`Get>=1`、`OpenResult>=1`。
- [ ] 使用生产 `BatchImagePipelineProcessor` 完成 Poll、JSONL 索引、success items、结算和 usage log。
- [ ] 验证余额、冻结余额、actual cost、成功张数、usage log、终态不可逆和重复恢复幂等。
- [ ] 通过生产下载服务解码至少一张真实图片，但不得把图片内容、签名 URL 或密钥写入 Git。
- [ ] 开发阶段只使用既有真实响应的脱敏录制 fixture，证明生产 processor 的 Get→OpenResult→Index→Settle 路径和 `Submit=0`；不进行任何真实 Provider 读取。已有 2026-07-12 的真实 Get/OpenResult/解码证据作为历史边界引用，不重复调用。
- [ ] 真实 Provider 下的完整恢复不作为 `READY_FOR_USER_REAL_TEST` 前置条件；用户最后执行的新真实图片 create 将同时验证真实产品 DB/结算/资产链。
- [ ] 精确提交：`test(image): prove zero-create Gemini product recovery`。

## G4：Gemini 图片持久资产、预览、下载和再次引用

**Files:**

- Create migration: next available `backend/migrations/*_batch_image_assets.sql`
- Create repository/service files following existing video asset archive patterns.
- Modify: `backend/internal/service/batch_image_processor.go`
- Modify: `backend/internal/service/batch_image_download.go`
- Modify: `backend/internal/service/batch_image_provider_gemini.go`
- Modify: `frontend/src/views/user/BatchImageGuideView.vue`
- Modify: `frontend/src/api/batchImage.ts`
- Add repository integration, service and Vitest tests.

**Asset record minimum fields:**

```text
batch_id, item_id, image_index, storage_key, mime_type, byte_size,
sha256, archived_at, source_provider, source_ref
```

- [ ] 先写失败测试：Provider 文件过期后，本地资产仍可预览和下载；重复归档不产生第二条资产；下载验证 MIME、大小上限和 SHA-256。
- [ ] 首次索引成功后将图片流式归档到受控本地资产目录，禁止把整张大图无限制读入内存。
- [ ] 归档路径必须防 traversal/symlink；只允许配置的资产根目录。
- [ ] 下载优先本地资产；只有本地缺失时才回源 Provider 并在成功后补归档。
- [ ] 增加“再次引用”操作：后端只接受当前用户拥有的 asset ID，读取本地资产并作为 Gemini request inline image part；禁止前端传任意服务器文件路径或不受控外部 URL。
- [ ] 修复 Gemini 请求 TODO：真实下发 `response_mime_type`、`aspect_ratio`、`image_size`；数据库和 usage log 记录真实规格与真实 Provider endpoint，不再固定写 `vertex:batchPredictionJobs`/`1K`。
- [ ] UI 结果卡必须有预览、下载、再次引用三个真实动作；刷新页面后资产仍存在，不依赖浏览器 IndexedDB。
- [ ] 运行 repository integration、service tests、Vitest、typecheck。
- [ ] 精确提交：`feat(image): persist and reuse Gemini assets`。

## G5：Provider 正式账单导入与内外对账

**Files:**

- Create migrations for `provider_billing_imports`, `provider_billing_lines`, `provider_reconciliation_matches`.
- Create repository/service/handler/routes under existing admin patterns.
- Create frontend admin API/view/components and tests.
- Backend dependency: `github.com/xuri/excelize/v2`（固定当前兼容版本并记录 license）；CSV 使用 Go `encoding/csv`。后端直接接收原始 multipart 文件、计算 SHA-256、解析并规范化，禁止信任前端生成的规范化账单行。

**Normalized line contract:**

```json
{
  "external_line_id": "string",
  "upstream_task_id": "string",
  "model": "string",
  "sku": "string",
  "usage_quantity": "decimal string",
  "usage_unit": "string",
  "net_amount": "decimal string",
  "tax_amount": "decimal string",
  "gross_amount": "decimal string",
  "currency": "CNY|USD",
  "occurred_at": "RFC3339"
}
```

- [ ] 导入 header 必须包含 provider、provider_account_id、billing_period_start/end、timezone、original_currency、source_type 和可选 invoice_number；账期时间统一归一化为 UTC，但保留原时区。
- [ ] 先写失败测试：同一原始文件 SHA-256 不可重复导入；同一 provider/external_line_id 唯一；金额和 usage 使用 decimal，不用 float；非法币种、负数、超大文件、公式单元格、隐藏外链和 zip bomb 被拒绝。
- [ ] 后端对原始 CSV/XLSX 实施大小、行数、sheet 数、解压后大小和单元格长度上限；原始文件只写受控对象存储/本地账单根目录，数据库保存 hash 和 storage key，不写日志/Git。
- [ ] Seedance 优先按 `video_tasks.upstream_task_id` 匹配；Gemini 优先按 `batch_image_jobs.provider_job_name` 匹配。
- [ ] 无 task ID 时只能按 account+day+model/SKU 聚合，结果必须标 `aggregate_only`，不得声称逐任务一致。
- [ ] 匹配状态至少包含 `matched`、`amount_mismatch`、`usage_mismatch`、`internal_only`、`provider_only`、`adjustment`。
- [ ] 发票只记录受控对象引用、文件 hash、号码、账期、subtotal/tax/total；不得把发票原件写进日志或 Git。
- [ ] 对账差异不得自动修改用户余额；需要调整时写独立 adjustment ledger，并要求管理员确认。
- [ ] 管理员页面提供账期汇总、导入预览、重复拦截、差异队列、匹配详情和 CSV 导出；老板首页只显示“已对账/有差异/未上传”结论。
- [ ] 用合成 Gemini/Seedance CSV 和 XLSX fixtures 覆盖完全一致、税费、折扣、舍入、重复行、漏单和 Provider-only。
- [ ] 精确提交：`feat(billing): reconcile provider statements`。

## G6：无付费全量回归与三角色浏览器闭环

- [ ] 后端最低门禁：

```powershell
cd backend
go vet ./internal/config ./internal/service ./internal/handler ./internal/repository ./cmd/server
go test ./internal/config ./internal/service ./internal/handler ./internal/server/routes ./cmd/server -count=1
```

- [ ] WSL/Testcontainers repository integration 必须实际执行；本机固定使用 `Ubuntu-24.04`，报告 executed/passed/failed/skipped，任何 skip 不算通过：

```powershell
wsl.exe -d Ubuntu-24.04 --exec bash -lc "cd /mnt/d/sub2api-trunk/backend && go test -tags=integration ./internal/repository -count=1 -timeout 25m -v"
```
- [ ] 前端门禁：

```powershell
cd frontend
pnpm.cmd run lint
pnpm.cmd exec vue-tsc --noEmit
pnpm.cmd exec vitest run --reporter=basic
pnpm.cmd run build
```

- [ ] 使用当前 HEAD 重建本地镜像；只绑定 `127.0.0.1`，验证非 root、`/health`、迁移和静态资源。
- [ ] 浏览器验证老板、管理员、员工三角色；所有创建使用 mock/fake：
  - 老板第一屏看到总花费、调用量、成员排行、通道异常和 Provider 对账状态。
  - 管理员能配置授权复核通道、查看 guard Snapshot 剩余额度、配置 internal_real 日/月限额与 kill switch、上传合成账单、处理差异。
  - 员工能完成 mock 图片/视频创建、状态轮询、失败提示、持久预览、下载、再次引用。
- [ ] 保存截图、网络状态、任务 ID、console 错误；截图和 evidence 扫描 Key、Authorization、password、签名 URL query，命中必须为 0。
- [ ] 使用两个并行 reviewer：一个审安全/账务/幂等，一个审三角色 UX/移动端/错误文案。Critical/Important 必须修复并复验。

## G7：真相源、审查包和用户真实测试卡

- [ ] 更新 `00_START_HERE.md`、`02_CURRENT_REALITY_STATUS.md`、`docs/goals/03_CURRENT_GOAL.md`。
- [ ] 更新 `docs/reviews/LATEST_REVIEW_PACKAGE.html`，必须包含：目标、HEAD、镜像 ID、变更、测试统计、截图、0-create Gemini 恢复、fake UI create=1、账单 fixtures、风险、回滚和文件索引。
- [ ] 更新北极星草案，不直接修改仓外北极星 HTML。
- [ ] 写 closeout：`docs/superpowers/codex-handoff/deliverables/2026-07-15-REAL-PRODUCT-READINESS-closeout.md`。
- [ ] 生成单页用户测试卡，放入审查包，动作只能包括：
  1. 确认临时凭证已在 Windows 用户环境中存在；应用 bootstrap 页面显示两个 review_only 账号已装配，但不显示值。
  2. 登录员工账号，选择“一次真实复核”。
  3. 创建 1 张 Gemini 低规格图片，确认 task ID、终态、预览、下载、再次引用。
  4. 创建 1 条 Seedance 5 秒 9:16 视频，确认 task ID、create=1、终态、播放、下载、再次引用。
  5. 管理员上传 Gemini 和 Seedance 的真实账单明细，确认匹配/差异结论；没有账单明细时明确保留“待账单复核”。
  6. 老板确认总览、成员排行、余额和对账状态符合实际。
  7. 管理员先确认至少一个非 review_only 的正式内部 Provider 账号已配置并通过无费用连接检查，再决定是否启用 `internal_real`：只为指定成员/分组设置日/月上限；保持全局 kill switch 可见。没有正式账号时保持 internal_real 关闭。
  8. 立即禁用 review_only 账号并废弃临时 Gemini/Seedance 密钥。
- [ ] 用户测试卡必须列出立即停止条件：鉴权异常、密钥回显、重复 create、未知状态、费用超过剩余 ¥40、URL 不安全、账务不一致、资产不可下载。
- [ ] 用户真实测试前最终状态写为：`READY_FOR_USER_REAL_TEST / 待复核`，不得提前写“内部可用”。
- [ ] 精确提交 docs/evidence；不 push。

---

## 4. 最终验收矩阵

| 能力 | 自动证据 | 用户最终真实证据 |
|---|---|---|
| 预算硬门 | 单元/并发/跨进程/fail-closed 测试 | 真实页面显示剩余额度且不超限 |
| 员工视频 create | fake 浏览器链 create=1 | 1 次真实 Seedance create=1 |
| 员工图片 create | 产品 mock 图片浏览器链 create=1 | 1 次真实 Gemini create=1 |
| Gemini 产品链 | 既有 job 0-create 恢复、DB、结算、资产测试 | 1 次真实图片创建与资产交付 |
| 图片资产 | Provider 过期 fixture 后仍可预览下载复用 | 刷新后仍可下载和再次引用 |
| 内部账务 | Testcontainers 精确一次 ledger/余额/总览 | 真实任务金额和余额一致 |
| Provider 对账 | 合成 CSV/XLSX 完整匹配与差异 fixtures | 真实账单明细上传后的结果 |
| 三角色 UX | 本地浏览器与移动端截图 | 用户对生成内容和业务文案确认 |

## 5. 完成条件

新对话只有同时满足以下条件，才可以停止自动执行并把控制权交给用户：

- [ ] 所有开发和无付费测试完成；无 Critical/Important reviewer finding。
- [ ] shared guard 已进入真实产品 create chokepoint，而非仅 realsmoke 测试。
- [ ] 图片和视频的 mock、review_real、internal_real 路由不可混淆、不可绕过。
- [ ] Gemini 脱敏录制 fixture 的产品 DB/结算/资产恢复证明为 0 create；历史真实结果证据被准确引用但未重复调用。
- [ ] review_only 临时凭证由应用安全装配；internal_real 有独立正式账号与成员/团队额度策略。
- [ ] 图片和视频资产均可持久预览、下载、再次引用。
- [ ] Provider 账单导入与差异审核对合成 fixtures 可用。
- [ ] 当前 HEAD 镜像健康，三角色无付费浏览器闭环通过。
- [ ] 审查包与 closeout 指向当前代码证据 HEAD。
- [ ] 用户真实测试卡已生成，状态为 `READY_FOR_USER_REAL_TEST / 待复核`。

如果任何一项未满足，继续自动修复；只有需要真实 Provider、真实账单文件或用户人眼判断时才停下交给用户。

## 6. 回滚规则

- 每个 G 阶段独立本地提交；回滚使用 `git revert <commit>`，禁止 reset/clean。
- 真实复核功能默认关闭；紧急回滚先关闭 session capability，再 revert UI/worker/asset/账单提交。
- Provider 对账差异不得自动回写用户余额，因此关闭对账模块不会改变既有用户账务。
- 资产迁移只新增表和读取优先级；回滚应用代码时保留资产表与文件，禁止破坏性删除。
