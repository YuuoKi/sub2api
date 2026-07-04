# Sub2API 内部试点指引（本人试用 → 小团队）

更新：2026-07-04 Asia/Shanghai
适用版本：`wujie/video-capture-moat-20260702` 分支，bug 修复波次提交后（`17112f8b`..`b4897f77`）。

## 当前部署状态（dev）

- 容器：`sub2api-dev`（镜像已用修复后代码重建），入口 `http://127.0.0.1:18081`
- 数据库：`sub2api-postgres-dev`（migration 148 已应用并登记）
- Redis：`sub2api-redis-dev`
- 已冒烟通过：`/health`、admin 登录、`POST /api/v1/admin/ops/ws/ticket`（64 位 hex，60s TTL）、`/api/v1/admin/dashboard/stats`

## 阶段一：本人试用（当天可开始）

前置：DB 中目前只有 admin 用户，0 个上游账号、0 个 API Key。以下步骤需要你本人操作（涉及真实凭据）：

1. 浏览器打开 `http://127.0.0.1:18081`，用 admin 账号登录。
2. 管理后台 → 账号管理：添加你的上游 provider 账号（Claude/OpenAI/Gemini 任一）。
3. 管理后台 → 分组：确认默认分组的平台/模型配置与账号匹配。
4. 用户端 → API Keys：创建一个自己的 Key，设置配额。
5. 用任意客户端走一次网关调用（如 `POST /v1/messages` 或 `/v1/chat/completions`）。
6. 回看：用户端用量页、管理端 Dashboard、`/admin/ops` 运维大盘（QPS WebSocket 现在走 ticket 鉴权，页面会自动先取 ticket）。
7. 视频链路只用 mock provider；真实 Seedance 保持冻结（P1-019/020 未修）。

## 阶段二：小团队开放（本人稳定使用 3-5 天后）

1. 保持自助注册关闭（管理后台 → 设置），首批成员由 admin 手工创建账号。
2. 每人独立 API Key，设置每日/总配额；不共享 Key。
3. 分组按平台拆分，避免跨平台批量修改模型映射（见 DEV_GUIDE 坑 10）。
4. 已知竞态（P1-027）：高并发下配额可能短暂超额，用下面的监控 SQL 人工兜底：

```sql
-- 每日跑一次：检查配额使用与上限
SELECT id, name, quota_used, quota, (quota_used - quota) AS overage
FROM api_keys WHERE quota > 0 AND quota_used > quota * 0.9
ORDER BY overage DESC;
```

5. 每周导出用量对账一次（管理端 Usage 页或 `usage_logs` 表），与上游账单核对。

## 已知行为变化（升级公告用）

- Admin Ops WebSocket：旧的 `Sec-WebSocket-Protocol: jwt.*` 鉴权已移除，改为 `POST /admin/ops/ws/ticket` + `?ticket=`。旧脚本/旧前端需更新。
- OAuth：URL fragment 里的 `access_token` 一律拒绝，走 pending-session 交换流程。
- 计费 fail-closed：Redis/DB 故障时，配置了限流的 Key 会被拒绝（503），属安全换可用性。
- 支付：卡在 RECHARGING 超过 2 分钟的订单会触发 provider 重试并写审计，运维需关注。

## 红线（继续有效）

- 不接真实 Seedance/Kling 付费 provider（计费刹车 P1-019/020 未修）。
- 不对公网开放（安全评估结论：内部"谨慎可用"，对外"不建议"）。
- Admin 面板与支付 webhook 仅内网/受控网络可达。
- 不读取/打印 `.env`、key、token。

## 未修项清单（对外前必须处理，见 Phase 5）

| 项 | 内容 |
|----|------|
| P1-019/020 | 视频计费刹车默认无 budget guard、Charge 为 no-op |
| P1-027 | 请求前原子配额预扣未实现 |
| B2 | 用户 API Key 明文存库、列表 API 返回完整 key |
| B3 | refresh token 存 localStorage |
| B5 | Email OAuth 仍通过 fragment 下发 token（后端遗留路径） |
