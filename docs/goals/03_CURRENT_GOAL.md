# 当前目标：真实图片、视频与前端用户闭环复核

更新时间：2026-07-12
状态：**待复核**

## 目标

在不 push、不部署、不使用生产用户数据的前提下，以最多 4 次 Gemini/Nano Banana 图片、4 次 Seedance 2.0 视频、累计 ¥60 的硬上限，验证真实任务、计费、资产交付和老板/管理员/员工三角色用户路径。

## 已完成

- 当前分支本地提交：`1145f850` Windows 页面路径安全 fallback；`70d9d516` 员工视频入口与权限语义；`06020077` realsmoke 会话次数/预算硬门。
- Go 最低门禁、前端 lint/typecheck/视频测试/build 通过。
- repository integration 35 cases 实际执行并通过、无 skip。
- Seedance Form A 专用测试 harness 在读取 Key、构造客户端和 create 前先占用会话次数与预算。

## 当前阻断

- 2026-07-12 本轮 presence-check 未检测到 `GEMINI_API_KEY` 与 `SUB2API_SEEDANCE_SMOKE_API_KEY`（未读取值）；不得把聊天中的密钥拼入命令，恢复后需重新检查。
- Gemini/Nano Banana 缺少与 Seedance Form A 同等级的安全真实产品链 harness。
- WSL 当前无发行版；官方 Dockerfile 又被基础镜像代理 HTTP 429 阻断。
- 因本地服务未能启动，mock/真实三角色浏览器截图和资产下载复用尚未执行。

## 停止条件

- 鉴权异常、密钥回显、未知终态、重复 create、URL 不安全或账务不一致。
- 图片或视频任一达到 4 次，或累计预留达到 ¥60。
- 任何测试 skip、缓存镜像或静态源码证据不得包装成真实用户闭环。

## 证据入口

- 最新审查包：`docs/reviews/LATEST_REVIEW_PACKAGE.html`
- 北极星更新草案：`docs/reviews/NORTH_STAR_V5_REAL_REVIEW_UPDATE_DRAFT_20260712.md`
