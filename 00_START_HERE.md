# Sub2API 当前入口

更新时间：2026-07-12
当前状态：**可演示**

边界：Gemini 单图完成 1 次真实上游验证；Seedance 5 秒视频完成 2 次真实上游验证（16:9 与 9:16）。既有 9:16 真实任务已通过生产 repository/worker/finalizer/outbox 恢复链写入临时 Postgres，完成 usage 定价、reservation、单一账本、余额扣减、内容采集和 MP4 归档。当前 HEAD 的三角色浏览器路径与员工 mock 创建闭环已验证；真实 UI create、Provider 正式账单和图片多规格场景仍待复核，生产可用性未验证。

## 当前目标

在不 push、不部署、不使用生产用户数据的前提下，复核真实图片、Seedance 视频与三角色前端用户闭环。Windows 用户变量、WSL 2.7.10 与 Docker 29.1.3 已恢复；真实计数为图片 1/4、视频 2/4、累计预留 ¥20。

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

mock 通过只证明内部试运行链路。Gemini Batch 已通过真实结果解码；Seedance 两次真实生成均 succeeded。第二条任务随后以 submitted + 既有 upstream ID 接入生产数据链，只 Poll、不 Create，并完成 usage 108900、账本/余额/outbox/内容/MP4 一体闭环。`da22a229` 还将视频 USD 账本纳入老板总览与成员排行，并在视频总览明确显示 CNY/USD。最新镜像健康运行于 `127.0.0.1:18080`，三角色 7 张截图、59 个 2xx API 响应已留证。整体从“待复核”提升为“可演示”，但尚不能判定内部可用。
