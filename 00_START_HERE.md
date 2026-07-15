# Sub2API 当前入口

更新时间：2026-07-15
当前状态：**READY_FOR_USER_REAL_TEST / 待复核**

边界：开发与无付费验证已完成（共享硬门、三种执行模式、Gemini 0-create 恢复 fixture、图片持久资产、Provider 账单对账、三角色 mock 浏览器闭环）。历史真实 Gemini 1 次与 Seedance 2 次上游证据仍有效。用户尚未执行本轮真实图片/视频点击、真实账单上传、人眼验收和密钥废弃，因此不能判定“内部可用”。

## 当前目标

在不 push、不部署、不使用生产用户数据的前提下，由用户完成最后的真实复核动作。Windows 用户变量与本地 Docker/WSL 可用；真实计数延续为图片 1/4、视频 2/4、累计预留 ¥20。

## 阅读顺序

1. [当前现实](02_CURRENT_REALITY_STATUS.md)
2. [当前目标](docs/goals/03_CURRENT_GOAL.md)
3. [产品不变量](PRODUCT_INVARIANTS.md)
4. [架构护栏](ARCHITECTURE_GUARDRAILS.md)
5. [质量门禁](CODE_QUALITY_GATE.md)
6. [最新审查包](docs/reviews/LATEST_REVIEW_PACKAGE.html)
7. [本轮 closeout](docs/superpowers/codex-handoff/deliverables/2026-07-15-REAL-PRODUCT-READINESS-closeout.md)

## 决策边界

mock 与无付费浏览器闭环只证明内部试运行链路。真实 Provider 历史证据证明上游与部分产品恢复链，不等于员工 UI 新真实 create 已验收。`result_url` 存在不等于资产持久交付。Provider usage 与内部价目表一致不等于正式账单一致。只有用户真实测试卡全部通过后，才可提升为“内部可用”。
