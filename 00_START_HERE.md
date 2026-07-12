# Sub2API 当前入口

更新时间：2026-07-12
当前状态：**待复核**

边界：Gemini 单图完成 1 次真实上游验证；Seedance 5 秒视频完成 2 次真实上游验证（16:9 与 9:16），两条视频均已恢复并归档为本地 MP4。当前 HEAD 的老板、管理员、员工浏览器路径与员工 mock 创建闭环已验证；真实账务三方对账、真实任务进入本地产品数据库和图片多规格场景仍待复核，生产可用性未验证。

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

mock 通过只证明内部试运行链路。2026-07-12 Gemini Batch 真实任务成功并通过 Get/OpenResult/图片解码恢复验证；Seedance 两次 Form A 真实任务分别以 1 次 create 达到 succeeded，第二次 9:16 为 31 次 poll，随后只读恢复并归档 1,761,009 字节 MP4。当前 HEAD `c2566a2b` 镜像健康运行于 `127.0.0.1:18080`，员工 mock 任务真实经历 queued→submitted→running→succeeded，三角色 7 张浏览器截图、58 个 2xx API 响应已留证。Provider 账单/系统账本/用户余额仍未三方对账，真实付费任务也未写入该本地产品数据库，因此整体仍为待复核。
