# Sub2API 当前现实状态

更新时间：2026-07-15
状态：**READY_FOR_USER_REAL_TEST / 待复核**

边界：无付费收口已完成；用户真实点击、真实账单与人眼验收未完成，故不能写“内部可用”。

## 已有证据

- 共享 RealCreateGuard 已进入产品视频/图片 create chokepoint；默认关闭，启用需绝对 state path；fail-closed。
- 执行模式 `mock | review_real | internal_real` 已落地；API-key Seedance 真实创建排除 `review_only` 并强制 `internal_real` 策略预留。
- Gemini 脱敏 fixture 产品 DB 恢复证明 Submit=0；图片资产可预览/下载/再次引用。
- Provider 账单导入/规范化/匹配对合成 CSV/XLSX 可用；差异不自动改余额。
- 无付费代码证据 tip `8296c2a6` → 镜像 `sub2api:real-readiness-8296c2a6`（`sha256:ad432520f38c60fe67e85aaab7878ab47c901e12847fb5b2eacb9d972e1864fb`）healthy，UID 1000，绑定 `127.0.0.1:18080`；G7 审查包纳入提交 `a2344431`。
- 三角色 mock 浏览器：9 截图、79 API 2xx、secretHits 0。
- 历史真实：Gemini 图片 1/4、Seedance 视频 2/4、累计预留 ¥20；Seedance 产品恢复链与内部账务对齐证据仍成立。

## 尚未证明 / 留给用户

- 员工 UI 新真实图片 create + **人眼**：预览内容、下载文件、再次引用。
- 员工 UI 新真实视频 create=1 + **人眼**：播放、时长/画幅、下载、再次引用。
- **人眼业务文案**：真实确认弹窗计费/额度说明、列表与详情状态一致、币种一致、失败文案无密钥泄露（mock 审查包已见部分中英混排与 `$`/CNY 混用，记债不阻塞真实 create）。
- 真实 Provider 账单明细上传后的匹配/差异结论。
- 正式非 `review_only` 通道齐备后，是否启用 `internal_real`。
- 测试结束后禁用临时账号并废弃密钥。

## 事实边界

mock 成功不等于生产可用；内部 usage/价目表/账本一致不等于 Provider 正式账单一致；`result_url` 存在不等于资产已持久交付。
