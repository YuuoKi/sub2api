# 审查包：R2-C — 计费对账抽检

> 执行者：Claude  
> 完成时间：2026-07-05 16:30  
> 状态：`partial`（1 条真实视频样本）

---

## 真实样本：video task #4

| 字段 | 值 |
|------|-----|
| cost_estimate (CNY) | 5.009400 |
| currency (usage_log) | CNY |
| pricing_source | provider_usage |
| tokens | 108900 |
| usd_cny_rate | 7.20 |
| 折算 USD | 5.0094 / 7.2 = **0.69575** |
| 用户余额 Δ | 20.00000000 → 19.30425000 (**-0.69575**) ✅ 一致 |

---

## 验收

| 项 | 结果 |
|----|------|
| CNY→USD 折算与余额 | pass |
| usage_log 单行幂等 | pass |
| 3 视频 + 3 图片全量抽检 | skip（仅 1 条真实视频） |
| 与火山账单 ±1% | 待老板在方舟控制台核对 task cgt-20260705162538-bd5cl |

---

## usd_cny_rate

dev `settings.usd_cny_rate = 7.20`；管理后台 settings 可调整。
