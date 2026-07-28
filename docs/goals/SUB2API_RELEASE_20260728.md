# Sub2API Release 2026-07-28

## 目标

以当前线上 staff console 版本为基线，整合关键计费竞态修复与 HC-ATOM V3 视频/图片接入，形成单一、可回滚、可验证的广州内部 release，并在服务器备份后完成受控热更新。

## 冻结来源

- 线上基线：`fix/staff-console-hotfix-20260726@758f5c4195e60a43b86d5e7f79f9ba8cff8078a4`
- 计费修复：`fix/runtime-billing-blockers-20260725@cadc1b0d8`
- HC-ATOM：`codex/hc-atom-v3-integration-20260727@eb57d5551d8ac0f6aff302f432448b5fe9699b93`
- Release 分支：`codex/sub2api-release-20260728`
- 工作目录：`D:\sub2api-trunk\.worktrees\sub2api-release-20260728`

## 必须保留

- 当前线上 staff 生命周期、开卡、一次性 Key 展示、管理员保护和不可变构建身份。
- 余额及 API Key quota 的原子更新语义，非计费字段更新不得回写计费快照。
- HC-ATOM 固定供应商、固定域名、显式 provider、独立加密域、SSRF 防护、owned-result 归档和一次结算。
- HC 视频真实调度默认关闭；HC 图片不得静默替代 Gemini/Vertex。

## 验收门禁

- 合流无未解决冲突，`git diff --check` 通过。
- 后端：`go test ./... -count=1`、`go build ./...`。
- 前端：全量 Vitest、`vue-tsc --noEmit`、lint、Vite production build。
- 计费竞态、staff console、HC 视频/图片定向测试通过。
- Docker 镜像可构建，loopback `/health` 与嵌入的 `build_commit` 符合 release HEAD。
- secret、品牌和部署产物检查通过。
- `docs/reviews/LATEST_REVIEW_PACKAGE.html` 与 release HEAD、测试和部署状态一致。

## 部署边界

- 不读取、打印、提交或截图密钥、token、cookie、生产连接串。
- 不触发真实付费 provider 调用。
- 不删除数据库卷、备份或用户资产，不公开新增端口。
- 上传前先核对 SSH 主机身份、活动 Compose 目录、现有镜像标签、容器和卷，并创建可恢复备份。
- 失败时恢复部署前镜像标签并重新创建原容器；Git 只用 revert，不使用 reset/clean/rebase。

## 状态

`IN_PROGRESS`。合流、全量门禁、审查包、提交、服务器备份、部署和线上回归全部完成前，不得标记为内部可用或生产 READY。
