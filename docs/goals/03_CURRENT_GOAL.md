# 当前目标：用户真实复核收口

更新时间：2026-07-15
状态：**READY_FOR_USER_REAL_TEST / 待复核**

## 目标

在会话硬上限（图片 4、视频 4、累计 ¥60；当前已用图片 1、视频 2、¥20）内，由用户完成真实图片、真实视频、真实账单与人眼验收；通过后才可判定内部可用。

## 已完成（无付费）

- G0–G7 开发与自动验证按任务包执行到用户真实测试卡。
- 最终 HEAD：`b1756b0f`；镜像代码证据：`8296c2a6`。
- 审查包：`docs/reviews/LATEST_REVIEW_PACKAGE.html`。
- Closeout：`docs/superpowers/codex-handoff/deliverables/2026-07-15-REAL-PRODUCT-READINESS-closeout.md`。

## 当前缺口（仅用户动作）

1. 一次真实 Gemini 低规格图片 create 与资产交付确认。
2. 一次真实 Seedance 5 秒 9:16 视频 create=1 与播放/下载确认。
3. 真实账单上传或明确保留“待账单复核”。
4. 禁用 review_only 并废弃临时密钥。
5. 可选：正式通道齐备后启用 `internal_real`。

## 停止条件

鉴权异常、密钥回显、重复 create、未知终态、费用超过剩余 ¥40、URL 不安全、账务不一致、资产不可下载。
