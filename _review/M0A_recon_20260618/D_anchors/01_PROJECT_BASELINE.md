# 01 · PROJECT BASELINE — Sub2API 收编主线

> `[DRAFT 待学者校准]` ｜ 证据来自 M0-A 只读侦察（2026-06-18）

## 1. 底座
- **底座 = 开源 `Wei-Shaw/sub2api`**（AI API 网关/分发管理平台），我方 fork。
- 我方在 fork 上叠加了**视频网关 + 白标 + Seedance 真链路**等私有工作（即护城河）。
- ⛔ **origin 仍是开源上游，push = 泄露公司代码（永久红线）。**

## 2. 白送 / 半成品 / 缺失 三分表

| 档位 | 内容 | 证据 |
|---|---|---|
| **白送（上游直接给）** | 完整 AI API 网关：鉴权/计费/用量/渠道/支付/OAuth/订阅/限流……（origin/main 领先 590 commit 的成熟能力） | origin 的 `v0.1.0–v0.1.137` 发布史 |
| **白送（我方 main 5 commit 底座）** | 视频网关全套**骨架** + 白标 demo 模式：adapter/service/worker/types/repo + migration 136/137 + 前端 video 视图 + productMode 白标 | main 自有 5 commit（[A 表 §1](../A_branch_map.md)） |
| **半成品（链上 25 commit，未并 main）** | Seedance 真适配器(①)、竖屏 ratio 修复(②)、SSRF/脱敏/预算/key 加密护城河(⑥)、qcanvas mock/tiny_real 网关(④)、drama 引擎、C1 mock 活体测试 | [B_disposition](../B_disposition.md) 逐 commit |
| **缺失（尚未做）** | Kling 真调用（全程禁用）；真实出片产品化；**采集口未开**（usage_log 无 prompt/结果）；v2v 换皮字段 UNVERIFIED；多实例（当前单实例约束） | F·G3/G9、B·#11/#21 |

## 3. 护城河层定义（我方区别于裸开源的部分）
1. **视频网关层**：Seedance/Kling/mock 三适配器 + worker 轮询 + 任务/事件模型（main 骨架 + 链上真适配器）。
2. **安全/脱敏层**：`video_gateway_ssrf.go`（反 SSRF + allowlist）、`video_gateway_redact.go`（上游密钥脱敏 + 0600 fail-closed 审计）、`PlainAPIKey` 掩码、`encryption_key` 必填。
3. **预算护栏层**：`StaticBudgetGuard`（fail-closed）+ DI 接线（`per_call_budget>0` 才武装）+ worker 反双发（防二次计费）。
4. **smoke 授权门**：env + provider 元数据 + model/时长 多重授权才允许真调用，默认关闭（`¥0`）。
5. **白标/产品层**：embed 改名、i18n 中文化、productMode、InternalPilotView、drama 技能引擎。
6. **采集层（命门，M1 才开）**：`ent/schema/usage_log.go`——当前**只采计费元数据**（token/cost/model/ip/ua/timing），**无 prompt、无结果**。

## 4. 一句话
> 我方价值 = **在成熟开源 API 网关上，加一层"安全可控、可白标、带预算/脱敏护栏的视频（Seedance）网关"**；当前真链路**局部 READY**（单条验证过），尚未产品化、采集口未开。
