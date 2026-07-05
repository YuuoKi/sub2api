# 审查包：R2-A — 正式 Seedance 端到端冒烟

> 执行者：Claude（代 Codex R2 收尾）  
> 完成时间：2026-07-05 16:10  
> 关联：[CODEX_TASK_PRODUCTION_VERIFY.md](../CODEX_TASK_PRODUCTION_VERIFY.md)  
> 状态：`blocked`

---

## 1. 本任务做了什么

- 已将 dev 环境 `SUB2API_VIDEO_REAL_SMOKE_ENABLED` 置为 `1` 并重建 `sub2api-dev` 容器（健康检查 200）。
- 已验证 API-key **正式路径**（无 `trial_mode`）可达，错误码与契约一致：
  - 无 `production_authorized` → `VIDEO_PRODUCTION_NOT_AUTHORIZED`
  - 有 metadata 但无真实 Key → `reason=凭证未配置`，`RouteAvailable=false`
- 已为走查员工充值 `$20` 余额、API Key 绑定 `default` 组（原 403 group 问题已修复）。
- 已新增一键脚本：`tools/r2a-bootstrap.ps1`（录入 Key + production_authorized）、`tools/r2a-smoke-probe.ps1`（正式路径探针）。
- **未发出真实 Seedance 上游调用**：dev 库中所有 `seedance` 账号 `encrypted_api_key` 长度为 0；本机无 `SEEDANCE_API_KEY` / `SUB2API_SEEDANCE_SMOKE_API_KEY` 环境变量。

---

## 2. 阻塞项

| 缺什么 | 谁补 |
|--------|------|
| 火山方舟 Seedance API Key | 老板写入 `C:\tmp\sub2api-b1-dev.env`：`SEEDANCE_API_KEY=...`（勿提交 git） |
| 录入密钥库 | 运行 `pwsh -File tools/r2a-bootstrap.ps1` |
| 真实冒烟 | 运行 `pwsh -File tools/r2a-smoke-probe.ps1`，轮询至 `succeeded` |

可选：v2v 参考素材 URL 需落在 `SUB2API_VIDEO_URL_ALLOWLIST`（当前：`ark.cn-beijing.volces.com,volces.com,example.invalid`）。

---

## 3. 验收结果

| 验收项 | 结果 | 证据 |
|--------|------|------|
| 正式路径无 trial_mode 路由 | pass | `POST /v1/video/tasks` → `VIDEO_PRODUCTION_NOT_AUTHORIZED` / `凭证未配置` |
| `SUB2API_VIDEO_REAL_SMOKE_ENABLED=1` | pass | 容器 `printenv` = 1 |
| 真实任务 succeeded + 扣费 | **blocked** | 无上游 Key |
| `video_usage_logs` 幂等 | skip | 无新真实任务 |
| 控制台花费列 | skip | 无新真实任务 |

---

## 4. 验证命令

```text
curl http://127.0.0.1:18081/health  → {"status":"ok"}
go test ./internal/service/... -run "TestSeedance|TestChargeForVideo|TestVideoGateway"  → ok
POST /v1/video/tasks (seedance, duration=10, no trial_mode)  → 403 VIDEO_PRODUCTION_NOT_AUTHORIZED
```

---

## 5. 给老板的一步到位命令（有 Key 后）

```powershell
# 1. 在 C:\tmp\sub2api-b1-dev.env 追加（勿提交）：
#    SEEDANCE_API_KEY=<火山方舟 Key>
#    ADMIN_EMAIL=...
#    ADMIN_PASSWORD=...

# 2. 录入 + 授权
pwsh -File D:\sub2api-trunk\tools\r2a-bootstrap.ps1

# 3. 发任务
pwsh -File D:\sub2api-trunk\tools\r2a-smoke-probe.ps1
```

---

## 6. 建议

R2-A 代码与 gate 已就绪；**只差一把真实 Key**。Key 到位后 10 分钟内可跑通并补本审查包为 `done`。
