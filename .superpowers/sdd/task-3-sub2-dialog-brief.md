# Task 3 — Sub2 无损开卡弹窗与稳定幂等键

工作目录：
`D:\Codex创业任务\QCanvas（无界版）\QCanvas\.worktrees\sub2-guangzhou-hotfix-20260725`

## 必须实现

1. 开卡弹窗不可通过遮罩、拖选释放、Escape 关闭或清空表单。
2. Key 成功显示后，只能通过“我已安全保存，完成”关闭。
3. 关闭前不得因遮罩、Escape、右上角误操作丢失一次性双 Key。
4. 每次显式打开弹窗生成稳定 idempotency key：
   - 同一次打开中的失败重试沿用
   - 只有显式取消/完成后，下次打开才生成新 key
5. 创建 owner 成功、签发失败时重试沿用同一个 owner 与 idempotency key。
6. 先写遮罩、拖选、Escape、一次性 Key、稳定幂等键失败测试，再最小实现。

## 提交

完成后创建本地提交：

`fix(console): make employee card dialogs lossless`

报告写入 `.superpowers/sdd/task-3-sub2-dialog-report.md`。

## 全局约束

沿用 Task 2 约束；不修改版本 UI，不部署、不 push。
