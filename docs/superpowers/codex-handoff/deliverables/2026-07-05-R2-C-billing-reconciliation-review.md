# 审查包：R2-C — 计费对账抽检

> 执行者：Claude（代 Codex R2 收尾）  
> 完成时间：2026-07-05 16:12  
> 状态：`partial`

---

## 1. 结论

- **`usd_cny_rate`**：dev 库 `settings` 表 = **7.20**（视频 CNY → USD 折算口径）。
- **`usage_logs`**：0 条（本期 dev 无 LLM/图片真实调用，R2-B 未跑）。
- **`video_usage_logs`**：2 条（历史试跑/测试数据，非本次 R2-A 产生）。
- **统一总花费**：控制台总览在 mock 数据下为 $0；后端 `total_combined_actual_cost` 字段已接线（R1-G 前端已消费）。

真实 Seedance 扣费对账需 R2-A 跑通后补抽 3 视频 + 3 图片样本。

---

## 2. 验收表

| 验收项 | 结果 | 说明 |
|--------|------|------|
| `usd_cny_rate` 文档化 | pass | 默认 7.20，可在管理设置调整 |
| unified 与明细一致 | skip | 无足够真实样本 |
| Seedance CNY ±1% | skip | R2-A blocked |

---

## 3. 下一动作

R2-A `done` 后：对 succeeded 任务查 `video_usage_logs.actual_cost_cny`、`users.balance` 前后差、`settings.usd_cny_rate` 折算，与火山账单核对。
