# 审查包：P0-3 后端 — 外部工具归属修正

> 执行者：Codex  
> 完成时间：2026-07-05 00:19  
> 执行目录：`D:\sub2api-trunk`  
> 关联规划：[2026-07-04-console-v2-roadmap.md](../../plans/2026-07-04-console-v2-roadmap.md)  
> 状态：`done`；当前状态：`内部可用 / 待复核`

---

## 1. 本任务做了什么（给 Claude / 老板看）

- 目标：完成路线图 P0-3【Codex 后端】外部工具归属修正，避免 n8n、剪辑脚本、批量出图器等工具消费被算到管理员本人。
- 背景：现有 `users.notes` 字段已贯通 admin 创建/更新/列表，本轮选择 notes 前缀方案，不加表、不迁移。
- 完成 P0-3 后端最小闭环：外部工具仍是普通 `users` 成员，但可用 `member_type=tool` 标记为工具账号。
- 未新增数据库表或迁移；工具归属复用现有 `users.notes`，后端统一维护 `[工具]` 前缀。
- `POST /api/v1/admin/users` 和 `PUT /api/v1/admin/users/:id` 支持 `member_type: "human" | "tool"`。
- 管理员用户响应新增 `member_type`，后端按 notes 前缀派生 `tool/human`，前端不需要自己猜。
- 总览员工消费排行 `GET /api/v1/admin/dashboard/users-ranking` 的每行新增 `username`、`user_notes`、`member_type`，可直接把工具和员工分开展示。
- 承接当前工作区已有的管理员代用户开卡接口：`POST /api/v1/admin/users/:id/api-keys`；并把 API Key 状态契约对齐为 `active | disabled`。
- 未修改 `frontend/src/views/admin/console/` 下的新页面，未读取 `.env`、真实 Key、token 或 cookie，未做真实 provider 调用。

---

## 2. 改了哪些文件

| 文件 | 变更摘要 |
|------|----------|
| `backend/internal/service/user_member_type.go` | 新增工具/成员类型派生与 notes 前缀规范化逻辑。 |
| `backend/internal/service/admin_service.go` | `CreateUserInput` / `UpdateUserInput` 增加 `MemberType`，创建/更新用户时维护 `[工具]` 前缀。 |
| `backend/internal/handler/admin/user_handler.go` | Admin 用户创建/更新请求支持 `member_type`，并做 `human/tool` 入参校验。 |
| `backend/internal/handler/dto/types.go` | `AdminUser` 响应新增 `member_type`。 |
| `backend/internal/handler/dto/mappers.go` | Admin 用户 DTO 从 `notes` 派生 `member_type`。 |
| `backend/internal/pkg/usagestats/usage_log_types.go` | 用户消费排行 item 增加 `username`、`user_notes`、`member_type`。 |
| `backend/internal/repository/usage_log_repo.go` | 用户消费排行 SQL 连出 notes/username，并派生 `member_type`。 |
| `backend/internal/handler/admin/apikey_handler.go` | API Key 更新状态入参对齐后端既有 `active/disabled`。 |
| `backend/internal/server/routes/admin.go`、`backend/cmd/server/wire_gen.go`、`backend/internal/handler/admin/apikey_handler*.go` | 当前工作区已有的管理员代用户开卡、改卡、删卡配套后端改动，已纳入本次验证范围。 |
| `backend/internal/**/**_test.go` | 增加/调整 P0-3 行为测试、排行字段测试、API Key 状态测试。 |

未纳入本次后端交付：当前工作区已有 `frontend/**` 变更和 `frontend/src/views/admin/console/` 未跟踪目录，保持原样未处理。

---

## 3. 验收结果（必须可核对）

| 验收项 | 结果 | 证据 |
|--------|------|------|
| 工具账号与员工区分，复用 notes 前缀 `[工具]` | pass | `TestAdminService_CreateUser_ToolMemberPrefixesNotes`、`TestAdminService_UpdateUser_ToolMemberTypePrefixesExistingNotes`、`TestAdminService_UpdateUser_HumanMemberTypeStripsToolPrefix` |
| 创建/更新成员接口支持 `member_type` | pass | `TestUserHandlerCreateAcceptsToolMemberType`、`TestUserHandlerUpdateAcceptsMemberType` |
| 总览员工排行可区分工具和人 | pass | `TestUsageLogRepositoryGetUserSpendingRanking` 断言 `member_type=tool/human` |
| 管理员代用户开卡接口仍可编译验证 | pass | `go test ./...` 覆盖 `backend/internal/handler/admin`、`backend/internal/server/routes` |
| 不修改 console 新页面 | pass | 本轮未编辑 `frontend/src/views/admin/console/` |
| 不读密钥、不真实调用 provider | pass | 未读 `.env`；运行测试时设置 `SUB2API_VIDEO_REAL_SMOKE_ENABLED=0`；secret scan 无发现 |

---

## 4. 验证命令与结果

```text
# RED/GREEN 目标测试
$env:GOCACHE = (Join-Path (Get-Location) '.gocache')
go test -tags=unit ./internal/service -run 'TestAdminService_(CreateUser_ToolMemberPrefixesNotes|UpdateUser_ToolMemberTypePrefixesExistingNotes|UpdateUser_HumanMemberTypeStripsToolPrefix)' -count=1
ok github.com/Wei-Shaw/sub2api/internal/service 4.796s

go test -tags=unit ./internal/handler/admin -run 'TestUserHandler(CreateAcceptsToolMemberType|UpdateAcceptsMemberType)|TestAdminAPIKeyHandler_Update_StatusAndName' -count=1
ok github.com/Wei-Shaw/sub2api/internal/handler/admin 3.580s

go test -tags=unit ./internal/repository -run 'TestUsageLogRepositoryGetUserSpendingRanking' -count=1
ok github.com/Wei-Shaw/sub2api/internal/repository 4.266s

# 本次触达包回归
go test -tags=unit ./internal/service ./internal/handler/admin ./internal/repository -count=1
ok github.com/Wei-Shaw/sub2api/internal/service 94.823s
ok github.com/Wei-Shaw/sub2api/internal/handler/admin 3.609s
ok github.com/Wei-Shaw/sub2api/internal/repository 5.514s

# 后端全量测试，显式关闭真实视频冒烟
$env:SUB2API_VIDEO_REAL_SMOKE_ENABLED='0'
go test ./...
PASS / exit 0

# 后端 lint
golangci-lint run ./...
0 issues / exit 0
warning: golangci-lint facts cache 写入 AppData 被拒绝；不影响 lint 结果。

# 格式与秘密扫描
git diff --check
exit 0

python tools/secret_scan.py --include-untracked
secret-scan: no high-confidence tracked-plus-untracked findings
```

---

## 5. 给 Claude 的前端接口说明（如有）

- **创建成员**：`POST /api/v1/admin/users`
  - 请求新增可选字段：`member_type?: "human" | "tool"`
  - 当 `member_type="tool"` 时，后端会把 `notes` 规范为 `[工具] xxx`。
  - 当 `member_type` 为空时保持原 notes；如果 notes 已经手写 `[工具]` 前缀，响应仍会派生为工具。

- **更新成员**：`PUT /api/v1/admin/users/:id`
  - 请求新增可选字段：`member_type?: "human" | "tool"`
  - `tool` 会加 `[工具]` 前缀；`human` 会移除已有 `[工具]` 前缀。

- **AdminUser 响应**：
  - 新增字段：`member_type: "human" | "tool"`
  - 建议前端优先读 `member_type`，不要自己解析 `notes`。

- **总览员工消费排行**：`GET /api/v1/admin/dashboard/users-ranking`
  - `ranking[]` 新增：`username`、`user_notes`、`member_type`
  - 可用 `member_type` 分组展示“员工 / 工具”消费，避免工具消费污染管理员本人。

- **开卡 / 管卡接口**：
  - `POST /api/v1/admin/users/:id/api-keys`
  - `GET /api/v1/admin/users/:id/api-keys`
  - `PUT /api/v1/admin/api-keys/:id`
  - `DELETE /api/v1/admin/api-keys/:id`
  - API Key 状态请使用 `active | disabled`；不要再发 `inactive`。

---

## 6. 风险与遗留

- 未解决问题：P0-2 视频计费对账未在本文件完成。只读侦察结论是当前缺 Seedance 官方价格表版本、实际账单字段和对账字段；不应硬编码假价格。
- 需要老板决策：如继续 P0-2，需要提供 Seedance 官方单价表、币种、版本/生效日期，以及是否允许加 `actual_cost/currency/pricing_source/pricing_version` 等迁移字段。
- 需要 Claude 继续：前端“成员与开卡”页应使用 `member_type` 做类型切换和展示；“外部工具接入”导航合并策略由前端执行。
- 风险：`member_type` 目前是 notes 前缀约定，不是数据库枚举；管理员手工把 notes 改成 `[工具] xxx` 也会被识别为工具，这是本轮按 roadmap 选择的低风险最小方案。
- 安全自查：本轮触达管理员写接口、成员归属和 API Key 管理；已限制 `member_type` 只能为 `human/tool`，工具账号不获得 admin role，不新增 secret 返回面，secret scan 无高置信发现。
- 回滚方案：回退本次 `member_type` 相关 helper、DTO/handler/service/排行 SQL/test 改动即可；无 DB 迁移，无数据结构回滚。
- 建议下一任务：P0-1 若老板提供真实 Key 和预算授权，可做受控冒烟；否则继续把 P0-2 作为 blocked 价格表/字段设计任务处理。

---

## 7. 阻塞项（若 status=blocked）

- 本任务 `P0-3 后端` 未阻塞。
- P0-2 阻塞原因：缺 Seedance 官方单价表、币种/版本/生效日期、实际账单口径；当前 `video_tasks.cost_estimate` / `video_usage_logs.cost_estimate` 仍是估算字段。
- P0-1 阻塞原因：未获得真实 provider Key、预算授权和真实冒烟 stop condition。

---

## 8. 文件索引与可复制后续提示词

文件索引：

- 后端成员类型契约：`backend/internal/service/user_member_type.go`
- Admin 用户创建/更新入口：`backend/internal/handler/admin/user_handler.go`
- Admin 用户 DTO：`backend/internal/handler/dto/types.go`、`backend/internal/handler/dto/mappers.go`
- 总览排行 SQL：`backend/internal/repository/usage_log_repo.go`
- 管理员开卡/管卡入口：`backend/internal/handler/admin/apikey_handler.go`、`backend/internal/server/routes/admin.go`
- 关键测试：`backend/internal/service/admin_service_create_user_test.go`、`backend/internal/service/admin_service_update_user_rpm_test.go`、`backend/internal/handler/admin/admin_basic_handlers_test.go`、`backend/internal/repository/usage_log_repo_request_type_test.go`

可复制给 Claude 的后续提示词：

```text
请接手 Sub2API 前端 P0-3：不要重新设计后端契约。后端已支持 POST/PUT /api/v1/admin/users 的 member_type: "human" | "tool"，AdminUser 响应新增 member_type；GET /api/v1/admin/dashboard/users-ranking 的 ranking[] 新增 username/user_notes/member_type。请把“员工与开卡”页改为“成员与开卡”，新增“新增工具”入口并使用 member_type 展示/筛选工具账号。API Key 状态请使用 active/disabled，不要发 inactive。
```
