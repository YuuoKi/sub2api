# Sub2API Project Baseline

更新时间：2026-06-12 16:16 Asia/Shanghai

## 仓库基线

- 执行目录：`D:/Codex创业任务/企业 API 管理后台项目/02_source/sub2api`
- Git root：`D:/Codex创业任务/企业 API 管理后台项目/02_source/sub2api`
- 分支：`phase-3.8.2-overnight-readiness`
- HEAD：`4dd599af`

## Phase 基线

- Phase 4B.7：mock-only 后端闭环已形成审查包。
- Phase 4B.8：补 deploy hygiene、前端路径证据口径和最小真相源。
- 本轮没有部署、没有 push、没有接真实上游。

## 当前脏改动边界

已有 dirty 文件包括 Docker、deploy 配置、本地 Day0 候选脚本、两份过期报告和一个前端状态文案工具文件。

本阶段只允许：

- 对 deploy 相关脏改动做脱敏审查和归类。
- 创建最小真相源文件。
- 更新 Phase 4B.8 审查包和 LATEST 指针。
- 保留截图后补路径，但不把截图缺失作为硬阻塞。

本阶段不允许：

- 回滚或删除未知脏文件。
- 运行部署脚本。
- 启动公网或生产形态。
- 接真实上游。

## 产品状态边界

- mock-only 后端验证：通过。
- 前端状态文案验证：通过。
- 人工浏览器截图：后补 / 已跳过硬验收。
- 内部产品路径：可复核。
- 真实上游：NOT_READY。
- Production：NOT_READY。
