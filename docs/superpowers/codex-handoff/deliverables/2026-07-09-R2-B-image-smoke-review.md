# 审查包：R2-B — NB2 图片真实冒烟

> 执行者：Grok  
> 完成时间：2026-07-09 Asia/Shanghai  
> 状态：**`done`**（上游 Google + Sub2API 产品链均已签字）

---

## 1. 本任务做了什么

1. **上游直连**（较早）：`generateContent` 512 档成功（responseId `zEFPat3DN6_sz7IP-NjMoQI`，$0.045）。
2. **产品链**（本轮）：修复本机 Docker 网络后，录入临时 Gemini Key → `gemini-default` 分组 + `nb2-temp-smoke` 账号 → 员工 API Key → `POST /v1/messages` NB2 512 → `usage_logs` / 余额扣减。

Key 仅运行时注入；审查包与 git **不含** Key / 图片 base64。

---

## 2. 验收结果（产品链）

| 验收项 | 结果 | 证据 |
|--------|------|------|
| 路径 | pass | `POST /v1/messages` + `anthropic-version`；model=`gemini-3.1-flash-image-preview` |
| HTTP | pass | **200**；`msg_d018298a80eaa88b30bf4c15`；`stop_reason=end_turn` |
| 供应商账号 | pass | accounts.id=**1** `nb2-temp-smoke` / platform=gemini / type=apikey |
| 分组 | pass | groups.id=**2** `gemini-default` / `allow_image_generation=true` |
| `usage_logs` | pass | id=**1**；`media_type=image`；`total_cost=actual_cost=0.045` |
| tokens | pass | input=26 / output=1133 |
| 余额扣减 | pass | user_id=3：`18.60850000` → **`18.56350000`**（Δ **$0.045**） |
| 上游直连（对照） | pass | 见历史 §上游 |

> 注：Claude messages 兼容响应里 `content` 图片块形态需前端按契约解析；本轮以 **计费/usage/余额** 为产品链签字主证据。

---

## 3. 任务摘要（脱敏）

| 字段 | 值 |
|------|-----|
| path | Sub2API `/v1/messages` → Gemini AI Studio |
| msg id | `msg_d018298a80eaa88b30bf4c15` |
| imageSize | 512（0.5K 档） |
| usage_log_id | 1 |
| cost | **$0.045** ≈ ¥0.32 @7.2 |
| 发起人 | zoucha-test@wujie.local（user_id=3） |

---

## 4. 安全提醒

**请老板立即在 Google AI Studio 废弃本次临时 Gemini API Key。**  
建议同时在 admin 停用/删除 `nb2-temp-smoke` 账号与 `nb2-product-chain-temp` API Key。

---

## 5. 运维附注（本轮顺带）

- 本机 `deploy_sub2api-network` 容器间 TCP 超时导致登录/调度失败；已重建为 `deploy_sub2api-network_fix` 并恢复 `18081`。
- 数据卷保留；未 wipe DB。
