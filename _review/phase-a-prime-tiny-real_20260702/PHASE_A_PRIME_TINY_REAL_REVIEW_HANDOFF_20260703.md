# Phase A' Tiny Real 三证同屏审查交付

生成时间：2026-07-03
状态：内部可用 / 可演示
范围：Sub2API + QCanvas `/studio-v2` 真链路 tiny real 验证
边界：不宣称产品 READY；不包含 API key、JWT、cookie、数据库密码、完整签名 URL。

## 结论

Phase A' tiny real 三证同屏已跑通。

- Seedance preflight：`ready`
- QCanvas task id：`1`
- final_status：`succeeded`
- has_result_url：`true`
- realChainReady：`true`
- `ai_generation_content` 对 task `1` 的行数：`1`
- Admin stats `is_live`：`true`
- Admin stats `captured_today`：`1`

## 你需要重点审查什么

1. `qcanvas_three_proofs_masked.png`
   - 同屏包含 QCanvas 节点、SQL capture 行数、Admin stats live。
   - result URL 只保留脱敏后的主干，不含签名查询参数。

2. `qcanvas_studio_v2_node_masked.png`
   - 单看 QCanvas `/studio-v2` 节点状态。
   - 重点看 `task_id=1`、`succeeded`、`has_result_url=true`、`realChainReady=true`。

3. `QCanvas_REVIEW_PACKAGE.html`
   - 自包含 HTML 审查包。
   - 包含目标、执行目录、证据、截图、风险、回滚、后续提示词。

4. 本 Markdown
   - 作为过程、边界、结果和复核点索引。

## 过程摘要

1. WSL / Docker
   - 使用 WSL `Ubuntu-24.04` 内的 Docker CLI。
   - 最终核验：`docker_cli=/usr/bin/docker`，`Docker Compose version 2.37.1+ds1-0ubuntu2~24.04.1`。
   - 未使用 Docker Desktop 桌面端作为执行路径。

2. Sub2API 环境
   - 写入临时 `deploy/.env`。
   - `.env` 中密钥均为临时运行用途，结束后已删除。
   - 使用 compose project：`sub2api_phasea_prime`。
   - 起服务后通过 10 次 health gate。

3. Seedance provider
   - 执行 provider bootstrap。
   - 结果：`seedance_preflight=ready`。

4. Worker 等待修正
   - 早先直连 Sub2API 任务因 worker poll cap 过短，Seedance 正常慢返回时没有等够，失败已单独记录。
   - 本轮未做 retry 风暴。
   - 后续用临时 compose override 把 tiny real 等待窗口加长：
     - `VIDEO_GATEWAY_POLL_INTERVAL_SECONDS=5`
     - `VIDEO_GATEWAY_TASK_TIMEOUT_MINUTES=30`
     - `VIDEO_GATEWAY_MAX_POLL_ATTEMPTS=300`

5. QCanvas 真链
   - QCanvas 分支：`work/night-hardening-20260702`。
   - Hono 指向 Sub2API：`http://127.0.0.1:8080`。
   - `/studio-v2` 浏览器 localStorage：
     - `studioV2RealChainReady=true`
     - `studioV2RealSeedanceArmed=true`
   - 通过 QCanvas `/studio-v2` 发起 1 次有效 tiny real。
   - 最终任务：`task_id=1`，`succeeded`。

6. 三证
   - 真片：QCanvas 节点有 result URL，`realChainReady=true`。
   - 入库：`SELECT COUNT(*) FROM ai_generation_content WHERE task_id='1';` 返回 `1`。
   - 看板：Admin stats 返回 `is_live=true`。

7. 收尾
   - QCanvas Hono / Web dev 进程已停止。
   - Sub2API compose project 已执行 `down -v`。
   - `deploy/.env` 已删除。
   - WSL `/tmp` 中临时 Sub2API API key 已删除。
   - Windows `C:\tmp\phasea*` 临时文件已删除。

## 验证记录

- QCanvas 安全摘要解析：
  - `summary_task_id=1`
  - `summary_status=succeeded`
  - `summary_has_result_url=true`
  - `summary_real_chain_ready=true`
  - `summary_sql_capture_rows=1`
  - `summary_admin_is_live=true`

- Seedance / capture / live 摘要：
  - `seedance_preflight=ready`
  - `task_id=1`
  - `final_status=succeeded`
  - `capture_rows_for_task=1`
  - `is_live=true`

- 清理核验：
  - `deploy_env_absent=true`
  - `sub2api_phasea_container_count=0`
  - `wsl_tmp_api_key_absent=true`
  - `phasea_tmp_remaining=0`

- 定向敏感信息扫描：
  - 最终证据 Markdown、QCanvas HTML 审查包、LATEST HTML：`targeted_secret_scan=clean`
  - 另一次宽松扫描命中了图片 base64 中的随机 `eyJ...` 片段，以及旧脚本中的 env 字段名/占位写法；未命中明文火山 key 或 TOS 签名参数。

## 文件索引

桌面包内文件：

- `PHASE_A_PRIME_TINY_REAL_REVIEW_HANDOFF_20260703.md`：本文件。
- `qcanvas_three_proofs_masked.png`：三证同屏截图。
- `qcanvas_studio_v2_node_masked.png`：QCanvas 节点截图。
- `QCanvas_REVIEW_PACKAGE.html`：QCanvas 自包含 HTML 审查包。
- `QCanvas_SUMMARY_SAFE.json`：无密钥安全摘要。
- `Sub2API_FINAL_EVIDENCE_20260702.md`：Sub2API 侧最终证据索引。

源路径：

- Sub2API：`D:\sub2api-trunk\_review\phase-a-prime-tiny-real_20260702\`
- QCanvas：`D:\Codex创业任务\QCanvas（无界版）\QCanvas\docs\reviews\PhaseA-prime-tiny-real_20260702\`
- QCanvas latest：`D:\Codex创业任务\QCanvas（无界版）\QCanvas\docs\reviews\LATEST_REVIEW_PACKAGE.html`

## 风险 / 待复核

- 早先 Sub2API 直连尝试失败原因是等待窗口不足，失败记录保留；成功证明以 QCanvas 真链路任务 `1` 为准。
- QCanvas 仓库运行前已有大量脏树，本轮未整理、未回滚、未提交。
- 真实 result URL 的完整签名查询参数未进入本审查包；截图和 HTML 均只保留脱敏显示。
- 本轮仅证明 Phase A' 内部可用 / 可演示，不代表产品整体 READY。

## 回滚 / 清理状态

已经执行：

- Sub2API compose：`down -v`
- Sub2API 临时 `.env`：已删除
- QCanvas dev 进程：已停止
- 本轮临时 token/key/tmp 文件：已删除

如需进一步清掉证据，只删除本桌面审查包和两个仓库中的本轮 review 目录即可；不要执行 `git reset` / `git clean`，因为两边仓库均存在非本轮脏树。

## 可复制后续提示词

请只读审查桌面 `PhaseA-prime-tiny-real_审查包_20260703`：

1. 打开 `PHASE_A_PRIME_TINY_REAL_REVIEW_HANDOFF_20260703.md`。
2. 对照 `qcanvas_three_proofs_masked.png` 与 `QCanvas_REVIEW_PACKAGE.html`。
3. 核验三证是否同时成立：QCanvas 真片节点、SQL 行数 1、Admin stats `is_live=true`。
4. 检查包内是否存在明文 key/token/完整签名 URL。
5. 结论只允许写：内部可用 / 可演示 / 待复核 / 已阻塞，不得写产品 READY。
