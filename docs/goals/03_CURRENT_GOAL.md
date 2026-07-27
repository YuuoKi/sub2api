# 当前目标：HC-ATOM V3 视频与图片接入 Sub2API

日期：2026-07-27
分支：`codex/hc-atom-v3-integration-20260727`
基线：`cc6a150c1644915c1576ca8e1263071a5a54e16f`
状态：**待真实链路复核（非生产 READY）**

## 目标

将 HC-ATOM 作为独立供应商接入 Sub2API：

- Seedance 2.0 通过固定的 `https://api-aigc.fzyinghe.com/v3/video/tasks` 异步链路创建、轮询和确认取消。
- 图片通过固定的 `https://api-aigc.fzyinghe.com/image/generation/tasks` 异步链路创建、轮询、归档和结算。
- 管理员只能配置固定供应商、固定域名、固定模型目录和加密密钥，不允许任意中转地址或静默跨供应商 fallback。

## 当前已实现

### 视频

- 新增独立 provider `hc_atom_seedance_v3` 和 provider-neutral `VideoProviderClient`。
- 公共模型别名 `doubao-seedance-2.0` 固定映射为上游 `doubao-seedance-2-0-260128`。
- 支持 `content[]`、`ratio`、`generate_audio`、`return_last_frame`、`watermark`，旧 `prompt` 自动转换为文本 content。
- Base64、私网 URL、混合进制/非规范数值 IP、非法 `asset://` 均失败关闭。
- 远程媒体在提交上游前执行可访问性探测；每个 DNS 解析地址、实际拨号目标与 HTTPS 重定向均按公网边界校验，探测失败不会创建上游任务。
- 创建结果不确定时进入 `review_required`，保留已取得的上游 ID，不自动重放、不释放预留。
- 已提交任务只有在上游 DELETE 确认成功后才取消并释放预留。
- HC 视频真实调度默认关闭，必须显式开启配置门禁。

### 图片

- 新增独立平台/provider `hc_atom`，固定 POST/GET/DELETE 路径，单批单 item。
- HC 图片仅在请求显式选择 `provider: "hc_atom"` 时参与调度；省略 provider 时维持原 Gemini/Vertex 顺序，不会静默切到中转站。
- 启用 `seedream-5.0`、`doubao-seedream-5.0-pro`；`dola-seedream-5.0-pro` 仅保留受限映射，不进入可用模型目录。
- HC 图片 Key 使用独立 AES-256-GCM 域；管理 API、Redis 调度快照和日志不返回明文。
- HC 凭证禁止批量更新，必须走单账号加密更新路径，避免通用 JSON 合并绕过密文保存边界。
- 结果 URL 每次重定向均执行 HTTPS/443/DNS/私网安全校验，并校验图片容器、MIME、字节数和像素上限。
- 结果先归档到 Sub2 控制的 owned store；只有归档、索引和结算完成后才对用户完成。
- item/ZIP 下载和清理由 provider-neutral owned store 提供，不依赖供应商账号继续可用。

### 管理与本地证据

- 管理台已增加 HC-ATOM V3 视频通道与 HC 图片账号配置，固定域名、固定模型目录、masked key、启用状态和最近错误可见。
- 视频通道最近错误由对应 provider 的后端精确查询字段返回，不再从全局最近任务列表推断。
- video group 与 V3 provider、media group 与 HC 图片账号分别绑定，不静默跨供应商 fallback。
- 后端全量 `go test ./... -count=1`、后端构建、前端全量 1181 项测试、typecheck、聚焦 ESLint 和生产构建均通过。
- 已使用本地 fake API 完成真实浏览器渲染，并将视频/图片管理台截图纳入唯一审查包。
- 独立只读收口复核确认 5/5 边界问题 CLEAR：显式 HC 选择、批量凭证失败关闭、视频媒体 SSRF/可访问性、最近错误精确归属、审查包回滚与 JPEG MIME。
- 唯一审查包为 `docs/reviews/LATEST_REVIEW_PACKAGE.html`。

## 当前剩余动作

1. 保持分支本地，不 push、不部署、不合入 main。
2. 获得用户再次授权后，先确认当前 Key 已在 `https://api-aigc.fzyinghe.com` 开通 V3。
3. 分别执行 1 个最小真实视频任务和 1 个已授权图片任务。
4. 取得真实 taskId、状态回传、Sub2 归档资产和一次正确结算后，再复核是否可提升状态。

## 真实验收门槛

真实验收必须同时取得：

- Sub2 内部任务 ID 与真实上游 taskId。
- 上游状态回传到 Sub2 标准状态。
- 可预览、下载和复用的 Sub2 本地归档资产。
- 一次且仅一次的正确结算。
- 数据库、Redis、管理 API、日志和审查包中均无明文 Key。

在真实视频与图片任务均完成前，本目标只能标记为“待真实链路复核”，不能标记生产 READY。

## 硬边界

- 不读取、打印、提交或截图真实 provider Key、token、cookie。
- 不发起真实付费调用，不自行确认 V3 域名已为当前 Key 开通。
- 不开放任意 `base_url`，不允许静默跨供应商 fallback。
- 不自动按折扣调整员工售价、补扣或退款；供应商折扣仅作为后续成本对账依据。
- 不 push、部署、merge、reset、clean、rebase，不改动原脏工作区。
