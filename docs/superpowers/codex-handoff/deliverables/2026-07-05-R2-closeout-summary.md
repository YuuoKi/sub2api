# 审查包：R2 收尾总览 — 控制台商用签字

> 完成时间：2026-07-05  
> PR：[#3726](https://github.com/Wei-Shaw/sub2api/pull/3726)  
> 状态：内部可上线试跑；**商用签字待 R2-A Key**

---

## 已交付（可收工）

| 模块 | 状态 |
|------|------|
| 控制台 v2（总览/密钥库/成员与开卡/任务记录） | ✅ 浏览器走查 |
| R1 后端（扣费、content[]、幂等、allowlist） | ✅ 单测 + 代码 |
| P0-4 总览通道红条 | ✅ |
| API 契约文档 | ✅ `docs/api/*` |
| PR #3726 | ✅ OPEN |

---

## R2 执行结果

| 任务 | 状态 | 审查包 |
|------|------|--------|
| R2-A 正式 Seedance | **blocked**（无 Key） | [R2-A](./2026-07-05-R2-A-production-smoke-review.md) |
| R2-B 图片 NB2 | skip（无调用样本） | — |
| R2-C 计费对账 | partial | [R2-C](./2026-07-05-R2-C-billing-reconciliation-review.md) |

---

## 老板只需再做一件事

在 `C:\tmp\sub2api-b1-dev.env` 加 `SEEDANCE_API_KEY`，然后：

```powershell
pwsh -File D:\sub2api-trunk\tools\r2a-bootstrap.ps1
pwsh -File D:\sub2api-trunk\tools\r2a-smoke-probe.ps1
```

成功后把 R2-A 审查包状态改为 `done` 即可商用签字。

---

## P1  backlog（不挡收工）

- 卡额度 80%/100% 告警（R2-D1）
- 任务资产归档（R2-D2）
- 月度总预算进度条（需老板设数字）
