# Sub2API Phase 4B.8 Start Here

更新时间：2026-06-12 16:16 Asia/Shanghai

## 当前结论

当前仓库处于 Phase 4B.8 受控收口状态：

- mock-only 后端路径仍成立。
- deploy 相关脏改动已完成脱敏审查和逐项归类。
- 最小真相源与 Phase 4B.8 审查包已生成。
- 三张人工前端截图按用户最新口径改为后补 / 跳过，不再作为硬阻塞 gate。
- 当前状态：mock-only 内部产品路径可复核。
- 不得声明 Production READY、Commercial READY、Real Provider READY。

## 最新入口

- 最新审查包：`docs/reviews/LATEST_REVIEW_PACKAGE.html`
- Phase 4B.8 审查包：`docs/reviews/PHASE_4B_8_DEPLOY_HYGIENE_INTERNAL_PATH_REVIEW_20260612.html`
- 当前目标：`docs/goals/03_CURRENT_GOAL.md`
- 当前真实状态：`02_CURRENT_REALITY_STATUS.md`

## 硬边界

本阶段禁止：

- 接入 Seedance 或 Kling 真实上游。
- 公网部署、生产部署、push。
- 读取或打印项目本地敏感凭据。
- 清理未知脏文件、reset、clean、rebase。
- 绕过 Browser 或企业安全策略。

## 截图口径

目标截图路径保留为后补位置：

- `docs/reviews/phase_4b_8_manual_screenshots/01_boss_path.png`
- `docs/reviews/phase_4b_8_manual_screenshots/02_employee_path.png`
- `docs/reviews/phase_4b_8_manual_screenshots/03_admin_path.png`

当前未伪造、未复制旧图冒充新图；审查包记录为“后补 / 已跳过硬验收”。
