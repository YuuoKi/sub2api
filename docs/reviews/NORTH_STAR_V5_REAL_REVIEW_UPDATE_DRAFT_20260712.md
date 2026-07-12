# 北极星 V5.0 状态更新草案（2026-07-12）

> 本文件仅为 Sub2API 仓内草案，不直接修改仓外北极星总规划。

## 建议状态修订

| 能力 | 建议状态 | 当前证据 |
|---|---|---|
| Sub2API mock 视频链 | 可演示 | 历史 mock create→poll→`succeeded + result_url`；当前 HEAD 未重建镜像 |
| Sub2API 工程门禁 | 内部可用 | Go、前端、35 cases repository integration 新鲜通过 |
| 老板/管理员/员工视频入口 | 待复核 | 路由与侧栏已修复并测试；无浏览器三角色截图 |
| Seedance 2.0 正式链 | 待复核 | 1 次 5 秒 Form A 真实任务 succeeded（1 create、46 poll）；outbox 资产归档未由 harness 驱动 |
| Gemini/Nano Banana 正式链 | 待复核 | 1 次真实 Batch succeeded；修复 operation 状态解析后，既有任务 Get/OpenResult/图片解码通过；账单和资产未闭环 |
| 真实计费与对账 | 待复核 | 当前硬门预留 ¥12.5；Seedance usage=108900，Provider 账单、用户余额和总览尚未三方对账 |
| 结果资产预览/下载/复用 | 待复核 | 静态接线存在；未验证真实资产归档与长期可用 |

## 不应继续沿用的旧断言

- 不应仅凭 2026-07-05 历史材料继续写“真实 Seedance 当前内部可用”。
- 不应把 `result_url`、缓存镜像或静态页面入口写成资产已交付或用户闭环已通过。
- 在本轮真实调用、扣费对账和浏览器证据完成前，整体状态保持“待复核”；生产可用性未验证。
