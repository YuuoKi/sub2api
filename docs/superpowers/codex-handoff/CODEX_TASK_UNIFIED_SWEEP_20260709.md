# CODEX_TASK_UNIFIED_SWEEP — Sub2API 统合冲刺（P0–P2 · 付费最后）

> **执行者：Grok（本仓单会话）**  
> **日期：2026-07-09**  
> **基线 HEAD：** `8e401f42`（Night Phase B closeout）· 分支 `wujie/video-capture-moat-20260702`  
> **交付：** [deliverables/2026-07-09-UNIFIED-SWEEP-review.md](./deliverables/2026-07-09-UNIFIED-SWEEP-review.md)

## 目标

一次会话把北极星 V5.0 仍开放的 Sub2API 侧运营强化做完：零成本先绿，成品归档与月预算次之，真实付费冒烟最后；无老板授权则 G6=`blocked`。

## Phase 顺序

| Phase | 内容 | Commit 建议 |
|-------|------|-------------|
| G0 | 本任务书 + 审查包骨架 + 入口指针 | `docs: start unified sweep g0 baseline` |
| G1 | 总览下钻 + AiRecords 采纳/周报补洞 | `feat(console): overview drill-down and prompt adoption` |
| G2 | 月度总预算 settings + 总览进度条 | `feat(billing): company monthly budget cny progress` |
| G3 | 视频成品本地归档 | `feat(video): archive succeeded results to local assets` |
| G4 | 任务预览 + 备份超期黄条 | `feat(console): video preview and backup stale alert` |
| G5 | 全量门禁 + 文档收口 | `docs: close out unified sweep gates` |
| G6 | R2-B/C/竖屏（有授权才跑） | 审查包章节；无授权 `blocked` |

## 硬边界

- 禁止 push / deploy / reset / rebase / 读 `.env` 密钥明文
- 禁止改支付/webhook；禁止跨仓写 QCanvas
- 不做：多租户、组长权限、SkillEngine 自动总结、QCanvas S2
- 每 Phase 门禁红最多修 3 轮；G3 失败不阻塞 G4

## 已完成勿重做

- P0-4 通道红条、Night B1 adoption Key、B2 额度告警、B3 URL 过期提示
- 独立页 `/admin/generation-content`（采纳+周报已有）
