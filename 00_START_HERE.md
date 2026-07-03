# Sub2API 当前入口

本文件是新 chat、新同事、后续执行 agent 进入本仓库时的第一入口。

## 当前定位

Sub2API 当前应按“无界内部 AI API / 模型调度 / chat 战略决策底座”理解。

它的价值不是先对外宣称生产公开平台 ready，而是为内部 chat 对话提供可复核的后台证据：账号、模型路由、用量、成本、任务编排、结果回收和审计留痕。

## 当前状态

- Phase A' tiny real：内部可用 / 可演示；仅覆盖受控单次三证链路。
- Phase B1 账本日常化：内部可用 / 待复核；已落地 generation-content adoption 反馈、weekly report、Admin ContentWall 样本反馈入口与复查修复。
- 后续真实供应商调用：已冻结，需单独授权。
- 已验证链路：登录、管理后台、视频 mock 任务创建、后台处理、状态回传、结果资产 HTTP 200；Phase A' tiny real 受控单次三证链路；B1 adoption 保存与 weekly report 管理端路径。
- Phase A' tiny real 三证：QCanvas task `1` 为 `succeeded`，`ai_generation_content` 对 task `1` 有 1 行，Admin stats 返回 `is_live=true`。
- B1 账本证据：`docs/reviews/LATEST_REVIEW_PACKAGE.html` 已指向 Phase B1 复查后修复包；HEAD 为 `9ebbacb9 fix(admin): harden generation content ledger review flow`。
- 不可声明能力：真实 Seedance/Kling/S3 日常生产交付闭环、后续真实付费供应商调用自动解冻、公开商业平台可用。

## 当前真相源

- 当前审查包：`docs/reviews/LATEST_REVIEW_PACKAGE.html`
- Chat 战略上下文：`docs/CHAT_STRATEGY_CONTEXT.md`
- Phase A' 成功结果：`_review/phase-a-prime-tiny-real_20260702/success_result.json`
- Phase B1 账本入口：`docs/reviews/LATEST_REVIEW_PACKAGE.html`
- 本次早晨结果：`MORNING_RESULT_2026_06_28.md`

## 历史资料边界

- `_archive/` 是历史归档证据，不代表当前状态。
- `_review/` 是阶段性审查和试跑记录，不代表当前状态。
- 旧 Seedance、旧夜跑、旧审查包只能作为背景材料，不能直接作为当前状态依据。

判断当前状态时，先读本文件，再读最新审查包；不要从历史目录反推当前结论。
