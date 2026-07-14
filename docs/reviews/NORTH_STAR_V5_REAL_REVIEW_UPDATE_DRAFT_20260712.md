# 北极星 V5.0 状态更新草案（2026-07-13）

> 本文件仅为 Sub2API 仓内草案，不直接修改仓外北极星总规划。

## 建议状态修订

| 能力 | 建议状态 | 当前证据 |
|---|---|---|
| Sub2API mock 视频链 | 可演示 | 当前 HEAD 的员工 API 创建、轮询和浏览器详情闭环通过；任务经历 queued→submitted→running→succeeded，SVG 试跑证据可打开且正确渲染 |
| Sub2API 工程门禁 | 内部可用 | `da22a229` 当前 HEAD 镜像重建、健康检查、实际非 root 进程、Go/前端测试、typecheck、production build 与 Testcontainers integration 通过 |
| 老板/管理员/员工入口 | 可演示 | 7 张真实浏览器截图、59 个业务 API 全部 2xx；员工不接触 Provider 明文密钥，桌面与移动端路径通过 |
| Seedance 2.0 正式链 | 可演示 | 两次 5 秒真实任务 succeeded；既有 9:16 任务已通过真实 Postgres/repository/worker/finalizer/outbox 完成内部计费、余额、内容采集与 1,761,009 字节 MP4 归档；恢复链 0 create |
| Gemini/Nano Banana 正式链 | 待复核 | 1 次真实 Batch succeeded；修复 operation 状态解析后，既有任务 Get/OpenResult/图片解码通过；常用规格和参考图未验证 |
| 真实计费与对账 | 待复核 | 共享硬门图片 1/4、视频 2/4、预留 ¥20；usage 108900 已按 CNY catalog 写入单一 USD charge ledger并扣余额，老板总览/成员排行已合并视频账本；仍缺 Provider 正式账单/发票 |
| 结果资产预览/下载/复用 | 待复核 | 真实 Seedance 恢复链的 archive/capture/outbox 全完成，MP4 已持久化；mock UI 可预览。真实资产在浏览器中的下载与再次引用仍缺同一链证据 |

## 已确认的产品方向

- 先服务无界 AI 中短剧/漫剧内部团队，不先做泛用大平台。
- 老板首页优先总花费、调用量、成员表现和通道异常；技术细节下沉。
- 管理员负责成员、额度、通道与任务审计；员工只创建任务、查看状态、预览和复用结果，不接触 Provider 密钥。
- mock 仅用于验证接收、调度和记录能力，必须明确标注“试跑”，不得冒充真实生成。
- `succeeded`、`result_url` 和“资产已交付”必须分开判断；只有本地资产或仍可用的受控远程资产才能标记可交付。

## 当前不能升级为“内部可用”的原因

1. Seedance 已进入真实产品 repository/worker/outbox，但采用既有 upstream task 恢复，未证明员工 UI create→上游 create 的同一请求链；Gemini 仍未进入产品数据库。
2. 内部 usage/账本/余额/老板总览已对齐，Provider 实际账单/发票仍未接入。
3. Gemini 常用规格/参考图与 Seedance 首帧/尾帧输入尚未验证。
4. 浏览器仍记录 1 条无头 Edge CSP inline-script 告警；header/body nonce 已一致，倾向自动化注入噪声，但应保留风险说明。

## 北极星建议结论

Sub2API 已从“仅 mock/静态页面”推进到“可演示”：真实图片与视频上游能力已有可复核证据，三角色本地用户路径可操作，既有 Seedance 真实任务已进入生产数据链并完成内部计费与 MP4 交付。仍不能升级为“内部可用”，直到真实 UI create、Provider 正式账单对账、Gemini 产品资产链和真实资产复用全部通过。
