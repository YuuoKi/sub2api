# Sub2API 当前入口

更新时间：2026-07-12
当前状态：**待复核**

边界：mock 可演示；真实图片、真实视频和生产可用性未验证。

## 当前目标

在不 push、不部署、不使用生产用户数据的前提下，复核真实图片、Seedance 视频与三角色前端用户闭环。当前已完成无付费门禁、员工视频入口与真实复核会话硬门；真实调用仍等待安全环境变量和本机运行环境恢复。

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

mock 通过只证明内部试运行链路。当前真实复核已获最多图片 4 次、视频 4 次、累计 ¥60 授权，但 2026-07-12 本轮 presence-check 未检测到凭证变量（未读取值），WSL 本轮无发行版，官方镜像又被基础镜像代理 429 阻断；因此尚未产生真实 Provider、真实扣费、资产交付或浏览器截图证据。恢复环境后必须重新检查。
