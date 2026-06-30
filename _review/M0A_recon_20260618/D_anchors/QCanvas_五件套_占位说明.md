# QCanvas 仓 五件套 — 占位说明（不深挖）

> `[DRAFT 待学者校准]` ｜ **本占位不作证据**

## 边界声明
- QCanvas 是**另一个仓**，**M0-A 不访问、不假设可访问**。
- 本文件仅按"北极星已知现状"做占位，**不含任何 file:line 复核**。
- **QCanvas 仓的 file:line 复核留到该仓单独只读任务**（建议命名 `M0A-QCanvas-recon`）。

## 已知现状（来自北极星，二手，待该仓任务坐实）
- 商品 / 设计师入口存在。
- 套用开源画布（canvas）。
- 已接到 **mock 契约**（与 Sub2API api-key 视频 mock 网关对接：`provider=mock`、`mock_only`、`provider_boundary=api-key-video-mock-only`——此契约在 Sub2API 侧由 `3351338d`/`47cf1146`/`40e83bf4` 定义并测试）。
- **真链路未通**（QCanvas → Sub2API 真实出片尚未打通）。

## 待该仓单独任务复核的问题（占位清单）
1. QCanvas 自身的分支/main 落差与未并工作。
2. QCanvas → Sub2API 的调用代码位置与契约字段（`normalizeTaskRecord` 消费的 `id/status/result_url/error_message/provider`）。
3. QCanvas 侧是否有凭据/key 风险。
4. QCanvas 五件套（00–03 + LATEST）正式版。

## 五件套占位骨架（待填）
- `00_START_HERE.md` — 待该仓任务
- `01_PROJECT_BASELINE.md` — 待该仓任务
- `02_CURRENT_REALITY_STATUS.md` — 待该仓任务
- `03_CURRENT_GOAL.md` — 待该仓任务
- `LATEST_REVIEW_PACKAGE.md` — 待该仓任务

> 与 Sub2API 的衔接点：mock 契约已在 Sub2API 侧 READY（局部）；真链路打通是跨两仓的后续目标。
