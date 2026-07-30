# Task 4 — Sub2 不可变构建身份与自更新移除

工作目录：
`D:\Codex创业任务\QCanvas（无界版）\QCanvas\.worktrees\sub2-guangzhou-hotfix-20260725`

## 必须实现

1. 版本组件只读显示“当前部署版本”，彻底移除：
   - 最新版本
   - 立即更新
   - 在线回滚
   - 刷新上游版本
   - 对应前端请求与状态
2. 后端不再挂载：
   - `check-updates`
   - `update`
   - `rollback-versions`
   - `rollback`
   保留只读版本接口；运维更新只走受控部署。
3. 统一 BuildInfo，让页面 HTML 注入、公共设置、管理员版本接口、CLI `--version` 来自同一对象。
4. 显示格式固定：
   - `广州内部版 YYYY.MM.DD-rN`
   - 详情展示最终完整 Sub2 SHA 与构建时间
   - 不给 SHA 强加 `v`
5. 公共版本契约增加 `build_commit`、`build_date`。
6. Network 中更新检查请求必须为 0。
7. 先写失败测试覆盖 UI 无按钮/无请求、路由不挂载、四个版本真相一致，再最小实现。

## 提交

完成后创建本地提交：

`fix(console): replace self-update UI with immutable build identity`

报告写入 `.superpowers/sdd/task-4-sub2-version-report.md`。

## 全局约束

- 不通过启用 `lan_admin` 掩盖更新 UI。
- 不读取 `.env` 或密钥。
- 不 push、不部署、不调用外部 Provider。
- 不保留自更新兼容双轨。
