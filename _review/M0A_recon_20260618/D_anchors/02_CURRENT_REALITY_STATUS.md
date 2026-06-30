# 02 · CURRENT REALITY STATUS — Sub2API（带证据）

> `[DRAFT 待学者校准]` ｜ 全部 file:line / commit 来自 M0-A 实扫（2026-06-18）

## 状态词纪律
**能演示 / 局部 READY / 待复核 / 已阻塞 / 已冻结 / 产品 READY** —— 不轻易说"完成"。

## 1. main 与上游落差
- `main(69f648e2)` 落后 `origin/main` **590**、自有 **5**（`git rev-list --left-right --count origin/main...main` → `590 5`）。
- origin = `https://github.com/Wei-Shaw/sub2api.git`（fetch=push 同一上游）→ **push 红线**。

## 2. 分支乱象——真相（重要修正）
- 表面"10+ 未并分支"，实为**一条 25-commit 线性链**的检查点标签；`night-run/D(40e83bf4)` 超集，零互冲突，零上游包袱。
- `safety-…20260605` = `p0-9b`（同 `7ac5335b`）**纯重复**；`claude/*` 两条 = main（空）。
- 详见 [A_branch_map](../A_branch_map.md) DAG 图。**状态：待收编（待学者拍板）。**

## 3. Seedance 真调用状态
- **链上**：真适配器 `1b8865ff`（真 POST/GET `ark.cn-beijing.volces.com/api/v3`）+ smoke 门 `7ac5335b` + 形态A harness `c35049a4`。
- **main 上**：**禁用**——`video_gateway_adapter.go:134` 返回 `SEEDANCE_REAL_CALL_DISABLED`（poll 禁用 `:144`）。`git grep` main adapter 对 `http.NewRequest/ark.cn-beijing` 零命中。
- **状态：局部 READY**（记忆载单条真实出片已验证，未产品化；真门默认关闭 `¥0`）。

## 4. 竖屏 `aspect_ratio→ratio` 状态
- **响应侧**对齐：`7b78f9ca`（poll 解析 `ratio`）。**请求侧**修复：`1be53de3`（`BuildCreatePayload`/`CreateTask` 改发 `ratio`+`normalizeSeedanceRatio`）+ `831e9c98`（B2 收紧）。
- **main 上**：仍发 `aspect_ratio`——`video_gateway_adapter.go:110`(mock)/`:180`(seedance)/`:238`(kling)。
- **状态：局部 READY**（修复在链上、未并 main；main 仍 bug 态但因真调用被禁而潜伏）。

## 5. Kling 真调用
- **全程禁用**：`adapter.go:198` `KLING_REAL_CALL_DISABLED`（poll `:208`）。**状态：缺失/未做。**

## 6. 采集口状态（命门，M1 锚点）
- `backend/ent/schema/usage_log.go`：字段仅 user/api_key/account/request_id、model 三件套、billing_*、token 计数、cost、ip_address、user_agent、duration_ms、image_count/size、created_at……
- **无 prompt 字段、无结果/`video_url`/content 字段、无 hash/sha 字段**（`grep prompt|result|content|hash|sha` 零命中；`request_id` 是普通 `MaxLen(64)` 字符串，非内容哈希）。
- **状态：采集口未开**（M1 才动；本任务只记录不改）。

## 7. 安全/护城河现状
- SSRF 门 + 上游密钥脱敏 + 0600 fail-closed 审计 + key 加密必填：`1d5badd8`（`video_gateway_ssrf.go` / `video_gateway_redact.go` / `video_key_encryptor.go`）。
- 预算门 DI + 回显凭证 fail-closed + worker 反双发：`c35049a4`。
- **凭据扫描 CLEAN**：25 commit 无明文密钥入库（仅 `*.example` 占位 + `*_test.go` 合成假夹具）。详见 [F·附](../F_recheck.md)。
- **状态：局部 READY**（在链上，待收编 + 收编后回归测试坐实）。

## 8. 已知阻塞 / 待复核
- **已阻塞**：Kling 真调用（未做）；多实例（当前单实例约束，`deploy/README` 钉死）；v2v 换皮字段名 UNVERIFIED（`1be53de3` 标 B3 草案）。
- **待复核（交学者）**：真实出片物证是否以**未提交改动**存于主工作树/WSL `p0-9b-clean`——本侦察未进入查看。详见 [C·§8](../C_trunk_plan.md)。
