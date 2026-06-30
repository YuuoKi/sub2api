# 00 · START HERE — Sub2API 收编主线 五件套导航

> `[DRAFT 待学者校准]` ｜ 生成于 M0-A 只读侦察（2026-06-18）

## 现在站在哪
**M0（分支收编阶段）· 已完成 M0-A 只读侦察 → 待学者拍板。**
- 已查清：所谓"10+ 未并分支"实为**同一条 25-commit 线性链**的检查点；`night-run/D(40e83bf4)` 是超集；零互冲突；零上游包袱。
- 已坐实：Seedance 真适配器、竖屏 `aspect_ratio→ratio`、SSRF/脱敏/预算护城河、采集口现状——均带 file:line 证据。
- 凭据扫描：**CLEAN**（25 commit 无明文密钥入库）。

## 下一步是什么
1. **学者拍板**：收编策略（ff 全收 D / squash 精修）、前端去留、品牌文案二选一。
2. **M0-B（授权后另写任务）**：按 [C_trunk_plan](../C_trunk_plan.md) 拉 `wujie/trunk`、收编、跑全绿测试。**不 push、不开真门。**
3. **M1**：开采集口（命门 = `backend/ent/schema/usage_log.go`，当前只存计费元数据、无 prompt/结果）。

## 五件套导航
| 文件 | 内容 |
|---|---|
| [00_START_HERE](./00_START_HERE.md) | 本页：站位 + 下一步 + 导航 |
| [01_PROJECT_BASELINE](./01_PROJECT_BASELINE.md) | 底座=开源 sub2api；白送/半成品/缺失三分；护城河层 |
| [02_CURRENT_REALITY_STATUS](./02_CURRENT_REALITY_STATUS.md) | main 落差、分支线性真相、采集/Seedance 禁用状态（带证据） |
| [03_CURRENT_GOAL](./03_CURRENT_GOAL.md) | M0-A→拍板→M0-B→M1 路线 |
| [LATEST_REVIEW_PACKAGE](./LATEST_REVIEW_PACKAGE.md) | 指向本次 M0-A 审查包 |
| [QCanvas 五件套占位](./QCanvas_五件套_占位说明.md) | 另一仓，本次不深挖（占位） |

## 状态词纪律（全员遵守）
> 只用这几档，**不轻易说"完成"**：
> **能演示** / **局部 READY** / **待复核** / **已阻塞** / **已冻结** / **产品 READY**
- 例：Seedance 真链路 = **局部 READY**（单条已验证，未产品化）；竖屏修复 = **局部 READY**（在链上、未并 main）；采集口 = **未开**。
