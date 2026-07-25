# Task 2 — Sub2 员工复用开卡

工作目录：
`D:\Codex创业任务\QCanvas（无界版）\QCanvas\.worktrees\sub2-guangzhou-hotfix-20260725`

基线：`main@cc6a150c1644915c1576ca8e1263071a5a54e16f`

## 必须实现

1. “员工与开卡”展示全部非管理员 `human`/`tool` 账号，并明确标明类型。
2. 捕获真实扁平 API 错误对象的 `status=409` 与/或 `reason=EMAIL_EXISTS`；不得继续依赖 `error.response.status`。
3. 邮箱以 `trim + 小写` 精确匹配：
   - 已有 active 非管理员 human/tool：复用原 owner，绑定双 Key 与充值
   - admin：中文显式失败
   - disabled：中文显式失败
   - 找不到精确账号：中文显式失败
   - 禁止自动转换 human/tool 身份
4. 创建用户成功但后续签发失败时，保留已创建 owner；重试不得再次创建。
5. 先写真实扁平 409、human/tool 复用、admin/disabled 拒绝、列表展示的失败测试，再最小实现。

## 提交

只完成本任务范围后创建本地提交：

`fix(console): reuse existing employee owners when issuing cards`

不要顺手改弹窗关闭和版本 UI；那些是后续独立提交。

报告写入 `.superpowers/sdd/task-2-sub2-card-report.md`，包含 RED/GREEN、文件、测试和自审。

## 全局约束

- 只在本隔离 worktree 修改；不碰 `D:\sub2api-trunk` 脏主工作区。
- 不读 `.env`、Key、token、cookie；不调用付费 Provider。
- 不 push、不部署、不 reset/clean/rebase。
- 禁止静默兜底和伪数据。
