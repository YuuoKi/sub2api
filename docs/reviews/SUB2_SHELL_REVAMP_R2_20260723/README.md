# Sub2 管理台二轮壳改 — Playwright 真实链路验收（2026-07-23）

基线：trunk 源码现编后端（v0.1.151 + 本轮改动 + 本轮验收中修的 2 个 guard bug）。
上一轮参考：`docs/reviews/SUB2_SHELL_REVAMP_20260723/README.md`。

## 本轮改动清单

后端：
1. `qcanvas-key-pair` 允许 `video_group_id == media_group_id`（同组双 Key）。
2. `GET /admin/video/contract` 新增 `platforms` 数组（seedance `adapter_ready=true`；jimeng/即梦、veo/Veo 3.1、kling/快乐小马 `adapter_ready=false`）。
3. 创建视频通道支持自定义 `base_url`（中转）与 `default_model` 透传。
4. 新增 `DELETE /admin/video/providers/:id`。

前端：
5. 密钥库视频 Tab 页内完整管理（列表 + 录入弹窗：平台下拉[非 ready 置灰「即将接入」]→ 名称 → 分组 → API Key → 可选接口地址；启停用/删除二次确认；保存后启用 + 自动授权默认勾选）。
6. 员工卡改「员工与开卡」，新增员工单弹窗（姓名/邮箱/所在分组/充值金额 → 同组双 Key 明文一次）。
7. 侧栏「用量」变父级含「AI 调用记录 + 系统健康」，系统下只剩「设置与备份/高级」。
8. 总览零数据时中部换「三步上手」引导卡。

## 环境

- 后端构建：宿主机无 go1.26（本地 1.24.1，go.mod 要求 1.26.5），用本地镜像 `golang:1.26.5-alpine` 容器内构建：
  `docker run --rm -v backend:/src -v ~/go/pkg/mod:/go/pkg/mod -v backend/.gocache:/gocache -v /tmp/sub2api-r2-bin:/out -e GOCACHE=/gocache -e CGO_ENABLED=0 -w /src golang:1.26.5-alpine go build -ldflags="-s -w -X main.Version=$V" -o /out/sub2api ./cmd/server/`
- 后端运行：新容器 `sub2api-r2`（镜像 golang:1.26.5-alpine，二进制 bind-mount `/tmp/sub2api-r2-bin/sub2api`），
  网络 `wujie-lan-network`，映射 `127.0.0.1:8089->8088`，`AUTO_SETUP=true` + `ADMIN_EMAIL/ADMIN_PASSWORD` 自动 bootstrap，
  `DEPLOYMENT_PROFILE=lan_admin`、`VIDEO_GATEWAY_WORKER_ENABLED=true`。
- 数据：全新 scratch 库 `sub2api_r2`（容器 wujie-lan-postgres）+ redis db14（wujie-lan-redis）；seed 分组 media(id=2)/video(id=3)（脚本沿用上一轮 seed_k3acc.py 改端口）。
- 前端：`cd frontend && VITE_DEV_PROXY_TARGET=http://127.0.0.1:8089 npx vite --port 3000`（挤掉了上一轮残留的同端口 vite）。
- 截图脚本：`scripts/r2-0*.mjs`（playwright，1440×900，真登录真点击真提交；管理员 admin@wujie.local）。
- 合规：admin 首次登录后经 `POST /admin/compliance/accept` 完成合规确认（zh 文案）。

## 验收中修的 bug（均为 trunk 源码 bug，已修已测）

lan_admin 档案下 `BackendModeProductSurfaceGuard` 启用，但白名单漏了本轮/上轮新增的三个管理端点，导致真实链路 403：

| bug | 现象 | 修复 |
|---|---|---|
| `DELETE /admin/video/providers/:id` 未入白名单 | UI 删除视频通道恒 403「Backend mode is active…」（r2-05 前置 API 清理时发现） | `backend/internal/server/middleware/backend_mode_guard.go` 视频段补 DELETE；`backend_mode_guard_test.go` 方法级用例同步；`admin_video_routes_test.go` 注册用例补 DELETE |
| `POST /admin/users/:id/qcanvas-key-pair` 未入白名单 | 员工开卡弹窗双 Key 签发 403（r2-06 首跑复现） | 同上 users 段补 POST |
| `POST /admin/users/:id/balance` 未入白名单 | 开卡充值/行内充值 403（同链路下一跳必踩） | 同上 users 段补 POST；path 级 blocked 用例移除 balance（path 级探针固定 GET，POST-only 端点的精确契约由方法级用例承载） |

测试回归（容器内 go1.26.5，`-tags unit`）：
`internal/server/middleware`、`internal/server/routes`、`internal/handler/admin`（Video）、`internal/service`（VideoAdmin、QCanvas/APIKey）全部 ok。
修复后重新构建二进制、重启 `sub2api-r2`，对应场景重跑通过。未执行任何 git 命令。

## 逐项验收表（全部真点真填；断言 JSON 在 after/_r2-0*-report.json）

| 项 | 证据 | 结果 |
|---|---|---|
| r2-01 侧栏新结构 | `after/r2-01-sidebar-structure.png`（9 断言） | 用量展开 = AI 调用记录 + 系统健康（DOM 归属断言 opsGroup=用量）；系统展开 = 设置与备份 + 高级（高级再展开含视频通道等 9 项）；系统下无 /admin/ops |
| r2-02 总览空态引导卡 | `after/r2-02-overview-empty-guide.png`（7 断言） | scratch 库零数据 → 「三步上手」卡（录入 AI 账号 / 给员工开卡 / 回到这里看消费），步骤 1/2 为可用链接；无花费趋势图区块 |
| r2-03 平台下拉置灰 | `after/r2-03-platform-dropdown-open.png`、`r2-03-provider-modal.png`（5 断言） | 契约 platforms=4（seedance:true，其余 false）；下拉打开截图可见 Seedance 2.0 高亮可选，即梦/Veo 3.1/快乐小马 置灰「（即将接入）」（option disabled DOM 断言全过） |
| r2-04 页内创建通道 | `after/r2-04a-provider-form-filled.png`、`r2-04b-provider-created-list.png`（14 断言） | 填名称+video 组+假 Key+`http://relay.example.com/v3`，**取消勾选自动授权**后保存：POST 201，tiny-real-authorization 请求数=0，列表出现「R2探针-中转通道」启用中；base_url/default_model 透传经 DB 直查证实（见已知限制 2）；API 探针（media 组，建后即删）证实自定义 default_model 落库 |
| r2-05 删除通道（二次确认） | `after/r2-05a-delete-confirm-dialog.png`、`r2-05b-provider-deleted-list.png`（4 断言） | 危险确认弹窗「确定删除通道…」→ 确认 → DELETE 200 + toast「通道已删除」→ 列表空态 |
| r2-06 员工开卡双 Key | `after/r2-06a-staff-form-filled.png`、`after/r2-06b-dual-keys-plaintext.png`（9 断言） | 单弹窗填姓名/邮箱/video 组（单组）/充值 $1 → qcanvas-key-pair 请求体 `video_group_id=3, media_group_id=3`（同组）返回 200；同屏展示两把不同 sk- 明文 Key；「已充值 $1.00，当前余额 $1.00」；API 复核 balance=1 |
| r2-07 探针清理 | `after/r2-07-staff-list-clean.png`（4 断言） | 探针员工 DELETE 200 且列表/DB 不再出现（软删除 tombstone 留在 scratch 库）；探针通道 0 残留（库内仅剩系统内建 Internal Mock Video 行，管理列表不展示）；UI 员工列表空态 |

合计 52 条断言全绿。

## 已知限制

1. **即梦 / Veo 3.1 / 快乐小马仅注册表预留**：`video_provider_registry.go` 宣告 `adapter_ready=false`，契约 platforms 数组可见、前端置灰「即将接入」，无真实 dispatch adapter，不能建通道。
2. **base_url 不回显**：`VideoProviderAccount.BaseURL` 的 json tag 为 `"-"`（与密钥同级的保密设计），创建/列表响应均不含 base_url；透传持久化以 DB 直查为证（`video_provider_accounts.base_url`）。
3. **Update 透传语义**：`VideoAdminService.UpdateProvider` 中 `base_url`/`default_model` 传 nil 或空串会回填为平台官方默认值（不是保持不变）；要保留自定义值必须每次 PUT 都带上。
4. **同组 seedance 通道唯一性槽位**：`validateVideoProviderTarget` 仍以内置 `SeedanceModel` 为槽位键——同组已存在默认模型通道时，再建任何 seedance 通道（无论自定义 model 与否）都会 409「video provider model already exists」。该 legacy 语义未随 default_model 透传改动，多中转通道需放不同分组。
5. **tiny-real 授权/生产冒烟在 lan_admin 仍被封禁**（guard 设计如此），r2-04 用「取消自动授权 + 请求监听 0 次」证明未触发真实上游调用。
6. 探针数据清理为软删除（users/api_keys 带 deleted_at tombstone），scratch 库 `sub2api_r2` 可整库弃置；容器 `sub2api-r2`、vite:3000 验收后保留运行中。

## 复跑

```bash
cd docs/reviews/SUB2_SHELL_REVAMP_R2_20260723/scripts
# 依赖经符号链接复用上一轮 scripts/node_modules（drvfs 可用）
for s in r2-01-sidebar r2-02-overview-empty r2-03-keyvault-video-platforms r2-04-provider-create r2-05-provider-delete r2-06-staff-dualkey r2-07-cleanup; do
  node $s.mjs ../after
done
```
