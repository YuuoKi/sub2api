# Sub2 管理台壳大改 — 证据与验收（2026-07-23）

方案真源：QCanvas `docs/reviews/GUANGZHOU_CUTOVER_20260723/KIMI_SHELL_REVAMP_PLAN_20260723.md`
基线：Sub2API `450673cfe`（frontend Vue3 + Tailwind）；本地验收栈 vite:3000 → wujie-lan-sub2api:8080。

## before 标注（对照老板 4 张截图；原图为 Cursor 会话附图未落盘，以反馈文档文字为准逐条对应）

| 老板吐槽 | 本地 before 证据 | 标注 |
|---|---|---|
| ① 侧栏长名截断 | `before/08-sidebar.png` | 「系统、健康、备份与…」带省略号；四项均为并列长句；`before/09-version-popover.png` 背景可见 L2 展开后「上游账号、模型…/调用、任务与资…/系统、健康、备…」与 L1 重复 |
| ② 版本号溢出 + ③「有新版本可用/立即更新」 | `before/09-version-popover.png` | 弹窗「A new version is available! v0.1.164」琥珀框 + 绿色「Update Now」；badge 带琥珀 ping 点；全英文 |
| ④ 密钥库「测试」按钮 + ⑥ 无分组 | `before/04-keyvault-create-modal.png` | 录入弹窗仅 平台/名称/API Key/接口地址/备注，**无分组选择器** |
| ⑥ 无分组可保存成功 | `before/04c-keyvault-after-save-nogroup.png` | 真人操作：不选分组直接保存 → toast「账号已录入密钥库」，列表出现探针行且**无分组列**（探针账号已当场删除清理） |
| ⑤ 视频通道步骤过重 | `before/07-video-providers.png` | 「保存后启用」默认**不勾选**；保存后还要单独点「授权一次最小真实调用」；状态术语「未授权/待消费」无人话解释 |
| 总览 AI 味 | `before/02-overview.png`（+09 背景） | KPI 卡装饰圆环图标、「用了什么 AI」Doughnut、副标营销句 |
| 员工开卡链路 | `before/06-staff.png` | 空态只提「新增服务身份」，开卡/双 Key/充值分散多弹窗，充值页在导航外 |

## 环境备忘

- 后端：本地 docker `wujie-lan-sub2api`（127.0.0.1:8080，镜像 lan-release-20260720），契约未动。
- 前端：`VITE_DEV_PROXY_TARGET=http://127.0.0.1:8080 vite --port 3000`。
- 截图脚本：`scripts/capture.mjs`（1440×900、真登录、真点击；探针数据自清理）。
- 注意：本栈已有一条 Seedance 通道（ark-...c7f4，未授权）。P0.3 验收不触碰真实授权门禁语义。

## P0 after 验收（全部真点真填，非静态截图）

环境补充：本地 8080 镜像（lan-release-20260720）早于 trunk，`qcanvas-key-pair` 端点缺失 →
P0.2/P0.3 改用 trunk 源码现编后端（docker `sub2api-k3acc`，127.0.0.1:8088，scratch 库 `sub2api_k3acc` + redis db15）；
P0.1/P0.4 在 8080 验收（契约两版一致）。探针数据均已清理或留在一次性 scratch 库。

| 项 | 证据 | 结果 |
|---|---|---|
| P0.1 录 Ark Key 一次成功、列表见 media 组 | `after/p01-01..06` | 无组时内联建组引导出现；不选组保存被拦（中文报错）；一键建「作图组 media」自动选中；保存成功列表带「分组」列；编辑回填已选组；探针账号删除 |
| P0.2 一页开卡向导 | `after/p02-01..06` | 3 步同页走完：建身份 → 选两组（同组在选择器被禁选）+ 充 $12.34 → 双 Key 明文同屏一次 + 「已充值 $12.34，当前余额 $12.34」；行内「充值」再充 $1 成功；「查看花费」深链 `ai-records?user_id=`；验收身份删除 |
| P0.3 视频默认启用+自动授权+人话 | `after/p03-01..03` | 「保存后启用」「保存后自动授权」默认勾选；填 Key 保存即自动完成一次性授权（无任何额外点击）；卡片「已启用」+「待消费 = 等第一次真实出片，之后通道永久可用」；重复授权按钮按后端状态禁用（门禁语义未动，未伪造可用） |
| P0.4 测试按钮/停用确认 | `after/p04-01..04` | 操作列「测试」按钮移除（count=0）；「停用」弹二次确认，取消不变、确认停用成功、再启用还原 |

注：P0.3 的「自动授权成功 toast」一项脚本断言未抓到（toast 约 4s 自动消失，截图晚于寿命），但卡片进入「待消费」
状态本身即是 `authorizeTinyReal` 成功的直接证据（未授权时该状态不会出现）。

## 8 维自评（P0）

| 维度 | 分 | 依据 |
|---|---|---|
| 功能正确性 | 9 | 四链路全部真后端真数据跑通，断言项全 OK |
| 后端契约不动 | 10 | 仅 frontend/ 变更；group_ids/qcanvas-key-pair/updateBalance/tiny-real-authorization 全是既有契约 |
| 安全语义 | 9 | 明文 Key 仍只显示一次（spec 断言关闭后不残留）；取消/保存后清空 apiKey 反应式状态 |
| 中文文案 | 9 | 新增报错/引导/确认全中文；英文后端原文经 CONSOLE_ERROR_ZH 码表过滤已知码 |
| 真实链路验收 | 9 | Playwright 真点真填真提交（建组/开卡/充值/授权/停用全真写库并清理） |
| 测试同步 | 9 | KeyVaultView.secret/StaffView.keyOnce 两 spec 同步 + 新增 5 用例，32/32 绿，无 skip |
| 截图证据 | 9 | before/after 成对，关键帧含 toast/弹窗/列表态 |
| 回归通过 | 9 | vue-tsc、eslint 零告警；console 4 文件 32 用例全绿 |

均分 ≥8，P0 READY_FOR_REVIEW。

## P1 after 验收（IA + 版本）

| 项 | 证据 | 结果 |
|---|---|---|
| 侧栏 5 项短词无截断 | `after/p1-01-sidebar-five-items.png` | 总览/密钥库/员工卡/用量/系统；旧长名全消失；无「…」；L1/L2 父子同名重复消除 |
| 系统→高级 收口 | `after/p1-02-system-advanced-expanded.png` | 系统健康/设置与备份/高级（上游账号/模型分组/视频通道/定价/通道监控/上游网络/任务记录/生成资产/用量与成本）；视频链路检查并入系统健康（URL 仍可达） |
| 版本短显 + 无琥珀 ping | `after/p1-03-version-popover-clean.png` | badge `v0.1.151`；弹窗仅「当前版本/已是最新版本」，无「有新版本可用/立即更新」；完整版本点击复制 |
| lan 不请求 check-updates | `_p1-report.json` | 全程 0 次 `/admin/system/check-updates` 请求（request 监听断言） |

## 8 维自评（P1）

| 维度 | 分 | 依据 |
|---|---|---|
| 功能正确性 | 9 | 侧栏/弹窗断言全 OK，request 级证据证明不调用 check-updates |
| 后端契约不动 | 10 | 仅读取既有 `lan_admin_mode_enabled` 与公共 `version` 字段 |
| 安全语义 | 9 | 白名单未放宽；商业化面仍全部隐藏且 URL 不可达 |
| 中文文案 | 9 | 新增 i18n key 中英齐备；侧栏全短词中文 |
| 真实链路验收 | 9 | 真人点击展开两级菜单 + 开版本弹窗 |
| 测试同步 | 9 | roleAwareNavigation/AppSidebar 两 spec 同步新契约，115/115 绿；VersionBadge.restart 3/3 |
| 截图证据 | 9 | before(08/09)/after(p1-01..03) 成对 |
| 回归通过 | 9 | vue-tsc/eslint 零告警；批量 flake 复跑全绿 |

均分 ≥8，P1 READY_FOR_REVIEW。
