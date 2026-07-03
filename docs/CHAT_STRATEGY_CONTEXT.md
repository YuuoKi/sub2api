# Sub2API Chat 战略上下文

本文件服务新的 chat 对话，用于让老板、运营、技术管理员或后续 agent 快速获得当前真实状态，并据此做战略决策。

## 一句话判断

Sub2API 现在可以作为无界内部“AI API 与生产调度控制面”继续推进；它已经能支撑内部决策验证和演示，Phase A' tiny real 受控单次三证链路已验证，Phase B1 generation-content 账本日常化已进入内部可用 / 待复核，但后续真实付费供应商调用仍处于已冻结状态，需单独授权。

## 它是什么

- 内部 API 管理与模型调度底座。
- 用量、成本、任务、结果和审计证据的集中控制面。
- generation-content 账本、adoption 反馈、weekly report 和管理员样本复核入口。
- 给 chat 对话提供决策依据的后台系统。
- 后续可承接“模型选择、成本边界、生产排期、供应商接入优先级、风险控制”的决策流。

## 它不是什么

- 现在不是公开商业平台可用。
- 现在不是已验证的真实 Seedance/Kling 生产交付系统。
- 现在不是可以直接外放给真实客户使用的生产入口。
- 现在不是只为单个视频 mock demo 存在的临时页面。

## 已验证能力

- 本地受控环境可启动并通过健康检查。
- 管理后台可登录。
- 视频 mock 任务可创建。
- 后台 worker 可处理任务。
- 任务状态可回传。
- mock 结果资产可通过 HTTP 200 打开。
- Phase A' tiny real 受控单次三证：QCanvas task `1` 为 `succeeded`，Sub2API 入库 1 行，Admin stats 返回 `is_live=true`。
- Phase B1 账本日常化：generation-content adoption 反馈 API、weekly report、Admin ContentWall 样本反馈入口已落地。
- B1 复查修复：adoption `saved:false` 不再伪装成功，weekly report 局部失败不清空 stats/samples，管理端 i18n 与专项测试已补。
- Go 测试、前端测试、typecheck、lint、build、安全扫描门禁已在本轮扫库中通过。

## 待授权复核能力

- 后续真实 Seedance/Kling 等供应商调用。
- 真实付费 API 任务的生产闭环。
- 真实 S3/对象存储资产交付闭环。
- 对外公开部署和公网访问。
- 真实用户或客户数据路径。

## 下一轮 chat 的建议主路径

新的 chat 对话应把 Sub2API 当成“内部可用的决策与调度底座”：

1. 读取本文件和 `00_START_HERE.md`。
2. 读取 `docs/reviews/LATEST_REVIEW_PACKAGE.html` 获取最新验证证据。
3. 明确本轮决策问题：模型选择、成本判断、任务调度、供应商接入或风险复核。
4. 只在受控环境里讨论和验证，不触发真实付费供应商调用。
5. 输出结论时区分“已验证”“待授权复核”“已阻塞”。

## 禁止误判

- 不要把 `_archive/` 或 `_review/` 里的旧阶段结论当作当前状态。
- 不要把 mock 任务成功等同于真实供应商生产日常可用。
- 不要把 `resultUrlPresent=true` 等同于真实资产已交付。
- 不要读取、打印或传播密钥、token、cookie、账号或供应商凭证。
- 不要私自 push、部署、清理历史文件或触发真实付费 API。
