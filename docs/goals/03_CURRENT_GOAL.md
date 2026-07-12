# 当前目标：真实图片、视频与前端用户闭环复核

更新时间：2026-07-12
状态：**待复核**

## 目标

在不 push、不部署、不使用生产用户数据的前提下，以最多 4 次 Gemini/Nano Banana 图片、4 次 Seedance 2.0 视频、累计 ¥60 的硬上限，验证真实任务、计费、资产交付和老板/管理员/员工三角色用户路径。

## 已完成

- 当前分支本地提交：`1145f850` Windows 页面路径安全 fallback；`70d9d516` 员工视频入口与权限语义；`06020077` realsmoke 会话次数/预算硬门；`be346bb9` Gemini/Nano Banana 安全产品流 harness；`0b277da5` 真实 Gemini operation 状态解析；`798546cf` Seedance 资产恢复与 capture 真相修复；`165d2823` 员工任务页权限/移动端；`ca6829e0` 真实测试签名 URL 脱敏；`c2566a2b` mock SVG 结果正确预览。
- Go 最低门禁、前端 lint/typecheck/视频测试/build 通过。
- repository integration 35 cases 实际执行并通过、无 skip。
- Seedance Form A 专用测试 harness 在读取 Key、构造客户端和 create 前先占用会话次数与预算。
- Gemini Form A 专用测试 harness 固定每次预留 ¥5，并在读 Key/建客户端前占用图片次数与共享预算；fake 覆盖 Submit→Get→OpenResult、取消和图片解码。
- Seedance 第二次真实 Form A：5 秒、720p、9:16、24fps，1 次 create、31 次 poll、usage 108900、succeeded；同一任务只读恢复并归档为 1,761,009 字节 MP4，未新增 create。
- 当前 HEAD `c2566a2b` 本地镜像 ID `sha256:0343f327b95bbd6cfecdc2c7dcc77f4f74bc86b4e6c6c39a63d428b8018edf5f`，健康运行且实际进程 UID 1000；服务只绑定 `127.0.0.1:18080`。
- 老板、管理员、员工共 7 张真实浏览器截图；58 个业务 API 响应全部 2xx。员工 mock 任务 #1 经 queued→submitted→running→succeeded，结果证据可打开，桌面与移动端均已渲染验证。

## 当前缺口

- Gemini/Nano Banana 仅完成首张真实图片；常用规格、参考图和真实图片资产进入产品数据库尚未验证。
- Seedance 已完成两次真实生成及本地文件归档，但真实任务由受控 Form A memory repo 驱动，未进入当前本地产品数据库；尚不能证明 UI 创建→真实 Provider→系统账本→用户余额→资产复用的一体链。
- Provider 账单、系统账本、用户余额和老板总览仍缺真实三方对账，因此不能判定“内部可用”。
- 浏览器业务页面 58 个 API 均为 2xx；无头 Edge 在员工任务列表产生 1 条 CSP inline-script 告警。同一响应的 header/body nonce 已核对一致，暂按自动化注入噪声记录，但仍保留风险。
- 当前会话计数图片 1/4、视频 2/4、累计预留 ¥20；剩余额度不代表应继续调用，只有补齐明确缺失证据时才可使用。

## 停止条件

- 鉴权异常、密钥回显、未知终态、重复 create、URL 不安全或账务不一致。
- 图片或视频任一达到 4 次，或累计预留达到 ¥60。
- 任何测试 skip、缓存镜像或静态源码证据不得包装成真实用户闭环。

## 证据入口

- 最新审查包：`docs/reviews/LATEST_REVIEW_PACKAGE.html`
- 北极星更新草案：`docs/reviews/NORTH_STAR_V5_REAL_REVIEW_UPDATE_DRAFT_20260712.md`
