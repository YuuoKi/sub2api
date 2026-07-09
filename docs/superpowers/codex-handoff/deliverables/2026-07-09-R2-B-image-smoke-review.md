# 审查包：R2-B — NB2 图片真实冒烟

> 执行者：Grok  
> 完成时间：2026-07-09 Asia/Shanghai  
> 状态：**`partial`**（上游 Google `generateContent` 签字 **done**；Sub2API 产品链入库/计费未跑）

---

## 1. 本任务做了什么

- 老板提供临时 Gemini API Key + 沿用 ¥50 预算硬顶。
- 对 **Nano Banana 2**（`gemini-3.1-flash-image-preview`）发起 **1 次**最低档真实作图：`imageSize=512`、`aspectRatio=1:1`、`responseModalities=["TEXT","IMAGE"]`。
- Key 仅运行时注入；审查包与 git **不含** Key / 图片 base64。
- 本仓 `image_gateway_realsmoke_scaffold_test.go` 仍为 skip-only 脚手架，**未**走 Sub2API `/v1/messages` 产品链（无本机员工 API Key / 已授权 Gemini 供应商账号接线）。

---

## 2. 验收结果

| 验收项 | 结果 | 证据 |
|--------|------|------|
| 上游 HTTP | pass | `200`，~9.65s |
| 模型 | pass | `modelVersion=gemini-3.1-flash-image-preview` |
| `imageConfig` 透传意图 | pass | 请求含 `imageSize=512`、`aspectRatio=1:1` |
| `responseModalities` | pass | 请求 `["TEXT","IMAGE"]`；响应含 `inlineData` |
| 出图 | pass | 1 张 `image/jpeg`，约 110916 bytes；`finishReason=STOP` |
| usage tokens | pass | prompt=23 / candidates=1185 / total=1208 |
| 计费估算 | pass | NB2 0.5K 官方档 **$0.045**（≈¥0.32 @7.2） |
| Sub2API `usage_logs.media_type=image` | skip | 未走产品链，无本仓 usage 行 |
| 余额扣减 | skip | 同上 |

---

## 3. 任务摘要（脱敏）

| 字段 | 值 |
|------|-----|
| path | `POST .../v1beta/models/gemini-3.1-flash-image-preview:generateContent` |
| responseId | `zEFPat3DN6_sz7IP-NjMoQI` |
| imageSize / ratio | 512 / 1:1 |
| mime | image/jpeg |
| tokens | 1208 total |
| est cost | $0.045 ≈ ¥0.32 |
| 预算余量（相对 ¥50，含此前 Seedance ≈¥5） | ≈¥44.7 |

---

## 4. 安全提醒

**请老板立即在 Google AI Studio / Cloud 控制台废弃本次临时 Gemini API Key。**  
审查包不含 Key 明文；完整响应（含 base64）仅在本机 `.cache/g6-realsmoke/`（gitignored）。

---

## 5. 遗留

- 若需「产品链签字」：在 dev 录入 `production_authorized` Gemini 供应商账号 + 员工 API Key，再经 `/v1/messages` 或 native 路径跑 1 次，核对 `usage_logs` / 余额。
- 四档扩样（1K/2K/4K）未跑；控预算仅 512。
- v2v 仍 skip。
