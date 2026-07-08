# CODEX_TASK_NIGHT_PHASEB — 彻夜任务：Phase B 运营强化两件套 + adoption 401 修复（2026-07-09 夜）

> **执行者：Composer 2.5 或 Codex（Sub2API 仓，单会话，与 QCanvas 夜任务不同仓可并行）。老板睡觉，无人值守。**
> 背景：北极星 Phase B 运营强化未开始；另 S2-D 走查发现 **adoption 回流 401**（QCanvas hono 代理 `POST /v1/generation-content/:task_id/adoption` 时，员工 Key 被 generation-content 鉴权拒绝，上游 `401 INVALID_TOKEN`）——这是画布经验回流的断点，优先修。
>
> **无人值守硬边界**：禁止 push、deploy、真实 provider 调用、读/写 `.env`/密钥、reset/clean/rebase、跨仓写 QCanvas 仓、动支付/webhook 代码。遵守 depguard 分层（handler→service→repository）。每 Phase 单 commit；门禁红最多修 3 轮否则 BLOCKED 停止。交付物：`docs/superpowers/codex-handoff/deliverables/2026-07-09-NIGHT-PHASEB-review.md`。

## Phase B0 · 基线

`git log -3 --oneline` + `git status`（应干净）；快速门禁：`go test ./...` 相关包 + frontend typecheck。记录到交付物。

## Phase B1 · adoption 员工 Key 鉴权修复（P0，来自 S2-D 缺口）

1. RECON：复现 S2-D 场景——员工 API Key（`sk-` 前缀）调 `POST /v1/generation-content/:task_id/adoption` 返回 401。定位 generation-content 路由的鉴权中间件：是否只接受 admin/JWT 而不接受员工 API Key？对照同 Key 可通过的 `GET /v1/video/providers` 的鉴权链。
2. FIX（最小）：允许员工 API Key 提交**自己任务**的 adoption（校验 task 归属：task 的 api_key/account 与请求 Key 一致，否则 403）。不放开其他 generation-content 端点。
3. TEST：红测先行（401→200 归属正确 / 403 归属错误）；`go test ./...` 相关包 + golangci-lint。
4. commit：`fix(generation-content): allow employee key adoption on own tasks`

## Phase B2 · 卡额度 80%/100% 告警（Phase B 第 1 件）

- 后端：额度用量比例计算（service 层），API 暴露到总览/密钥列表既有接口（加字段，不破坏契约）。
- 前端：总览与密钥库对应卡片/行 80% 标黄、100% 标红（沿用现有 UI token，不新增设计）。
- TEST：service 单测（边界 79/80/100）+ 前端组件测试。门禁：go test + lint + vue-tsc + eslint + 相关 vitest。
- commit：`feat(console): quota usage warnings at 80/100 percent`

## Phase B3 · 任务成功素材归档（Phase B 第 2 件，最小版）

- 目标：成功视频任务的 `result_url` 过期前可再取。最小实现：任务记录页提供「下载/复制链接」动作 + 后端在任务详情接口返回过期时间提示字段（若上游提供）。**不做**对象存储转存（那是大件，留日间拍板）。
- RECON 先确认上游 URL 过期语义（Ark CDN），若详情接口已有全部所需字段则仅做前端。
- TEST + 门禁同上。commit：`feat(console): expose result download before expiry`

## Phase B4 · 全量门禁 + 收工

`go test ./...`、`golangci-lint run ./...`、`pnpm --dir frontend run lint:check`、`typecheck`、critical vitest、`make secret-scan`。交付物写 Final Gate 计数 + commit 列表 + Still Open + 一句话 verdict（状态词纪律）。更新 `CODEX_START_HERE.md`。commit：`docs: close out night phase b loop`。树必须干净。

## 停损

B1 是唯一 P0：若 B1 BLOCKED，仍可继续 B2/B3；B2/B3 任一 BLOCKED 则跳过进下一 Phase。任何时刻发现在做范围外的事 → 回退本 Phase 未提交改动，收工。
