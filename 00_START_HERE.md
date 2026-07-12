# Sub2API 当前入口

更新时间：2026-07-12
当前状态：**待复核**

边界：Gemini 单图与 Seedance 5 秒视频已各完成 1 次真实上游验证；资产、账务与三角色浏览器闭环仍待复核，生产可用性未验证。

## 当前目标

在不 push、不部署、不使用生产用户数据的前提下，复核真实图片、Seedance 视频与三角色前端用户闭环。Windows 用户变量、WSL 2.7.10 与 Docker 29.1.3 已恢复；真实计数为图片 1/4、视频 1/4、累计预留 ¥12.5。

## 阅读顺序

1. [当前现实](02_CURRENT_REALITY_STATUS.md)
2. [当前目标](docs/goals/03_CURRENT_GOAL.md)
3. [产品不变量](PRODUCT_INVARIANTS.md)
4. [架构护栏](ARCHITECTURE_GUARDRAILS.md)
5. [质量门禁](CODE_QUALITY_GATE.md)
6. [视频网关契约](docs/api/video-gateway-contract.md)
7. [最新审查包](docs/reviews/LATEST_REVIEW_PACKAGE.html)
8. [Cleanup Merge 审查](docs/superpowers/codex-handoff/deliverables/2026-07-11-GROK-CLEANUP-MERGE-review.md)

## 决策边界

mock 通过只证明内部试运行链路。2026-07-12 Gemini Batch 真实任务成功并通过 Get/OpenResult/图片解码恢复验证；Seedance Form A 真实任务以 1 次 create、46 次 poll 达到 succeeded。Gemini 真实响应暴露并修复了 operation metadata 状态解析缺陷（`0b277da5`）。但 reliability-core Form A 尚未驱动 outbox 资产归档，Provider usage/系统账本/用户余额也未三方对账，浏览器三角色截图未完成，因此整体仍为待复核。
