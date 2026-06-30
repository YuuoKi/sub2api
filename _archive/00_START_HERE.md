# Sub2API · START HERE（单一真相入口）

更新时间：2026-06-18 Asia/Shanghai · 维护：夜间无人值守 Claude Code（opus-4-8）

## 这是什么

Sub2API = 企业 AI 视频 API 调度中台 + 员工 Skill 引擎（表面是「多 provider API 中转站」，内核是统一数据采集口）。
一期范围：企业 API 管理中台 + 视频任务网关，首批 provider = Seedance 2.0 +（待接）Kling。
北极星见 `D:/Codex创业任务/QCanvas（无界版）/北极星/无界AI生产系统_总规划兼方向矫正书_v2_20260618.html`。

## 当前状态（一句话）

**真实 Seedance 2.0 链路在单仓 harness 层已真打通（真出片、真计费、token 计费已校准）= 局部 READY；但端到端（QCanvas→Sub2API→Seedance→回画布）一根线没织起来 = 整网未通。** 局部 READY ≠ 产品 READY。

## 下一步 = M1 / C1（唯一焦点）

打穿第一条端到端真实链路：QCanvas 点一下 → Sub2API（从 harness 升级为真实可调服务）→ Seedance → 结果回 QCanvas 候选区。
在此之前：不碰 Kling、不碰图片/文本、不美化 UI、不抠 Seedance 边角字段。

## 五件套锚点（打开就知道在哪）

1. `00_START_HERE.md` —— 本文件（入口 + 当前状态 + 下一步）
2. `01_PROJECT_BASELINE.md` —— 基线版本与 commit
3. `02_CURRENT_REALITY_STATUS.md` —— 真实链路状态矩阵（含 B1/B2/B3 接口契约缺陷定位与证据）
4. `03_CURRENT_GOAL.md` —— 当前唯一目标 = C1 端到端
5. `最新审查包/` —— 指向最近一次审查包（当前 = `_NIGHT_RUN_20260618/`）

## 散落探索包

历史发射前检查 / 多发勘察 / 多发实打 / 各 phase 审查包已归档至 `_review_packages/_archive_20260618/`，索引见该目录 `INDEX.md`。根级 Seedance 方案/对账文档保留原位（属现行参考，非散落）。

## 硬边界（长期生效，违反即停）

- 真实付费 API 调用：需在场授权 + 预算上限 + 失败退出，绝不无人值守自动执行。
- 密钥 / token / 凭据：绝不读取、打印、落库、落日志；只经 env 注入。
- `push` 到 origin：origin 指向开源上游，push = 泄露公司代码，一律本地 git。
- 真实部署 / 生产 / DB 迁移 / 不可逆删除或覆盖：须显式授权。
- `git reset --hard` / `git clean` / `git add .` 全量提交：禁止。
- 真实 provider 闸门保持【关闭】，一律走 mock/stub。
