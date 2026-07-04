# Sub2API 2026-07-04 本地验证摘要

执行目录：`D:\sub2api-trunk`  
分支：`wujie/video-capture-moat-20260702`  
状态：内部可用（仅本地测试层面）/ 待复核；本地分组提交已完成（见文末补记）

> **2026-07-04 傍晚补记（提交固化轮）**
>
> - index.lock 阻塞已消失，staging/commit 恢复正常；全部修复已按功能分组提交为 10 个 commit（`17112f8b`..`f176c256`），覆盖 payment、repository、auth、ops WS ticket、gateway/billing、video/drama、page handler、frontend auth/payment/ops，**含全部此前 untracked 的修复文件**（migration 148、ops_ws_ticket*、group_model_scope、sanitizeRedirectPath、oauthFragment、P0 回归测试）。
> - 提交前门禁复跑结果：`go test ./...` exit 0（全包 ok）；`golangci-lint run ./...` 修复 2 个新问题（ops_handler gofmt、affiliate_repo 未使用函数）后 0 issues；`vue-tsc --noEmit` exit 0；`eslint` exit 0；`vitest run` 102 文件 601 测试全过；secret-scan（含 untracked）无高置信度发现。
> - 修复计数修正：经子智能体逐条核对，本轮实际修复为 **7/7 P0 全部修复、32 个 P1 中约 24 个已修、5 个部分修、3 个未修**（P1-019、P1-020、P1-027），下文"Subagent 复查闭环"一节的"2 个前端 P1、3 个后端 P1"为早期保守记录，以本补记为准。

## 本轮边界

- 未 push。
- 未部署。
- 未触发真实 provider。
- 未读取 `.env`、token、cookie、API key 明文。
- 未执行 reset、clean、rebase 或删除历史审计材料。
- 未执行数据库 migration 到真实环境。

## Git 和证据

```powershell
git -c core.quotepath=false status --short
git branch --show-current
git rev-parse --show-toplevel
git diff --check
```

结果：

- 分支：`wujie/video-capture-moat-20260702`
- Git root：`D:/sub2api-trunk`
- `git diff --stat` 最新观测：79 tracked files changed, 1595 insertions(+), 427 deletions(-)
- `git diff --check`：exit 0，仅 `docs/reviews/LATEST_REVIEW_PACKAGE.html` LF 后续可能转 CRLF 的提示。
- `docs/reviews/BUG_AUDIT_FIX_STATUS_20260704.md` 被 `.gitignore:138:docs/*` 忽略，提交时需要 `git add -f`。
- `git add -- payment group paths`：exit 1，linked worktree 外部 Git index 无法创建 `index.lock`。提升权限申请被系统用量限制拒绝，因此本地 staging/commit 未完成。
- 多次 Git 命令提示无法访问 `C:\Users\浩臣移动工作站/.config/git/ignore`，按环境噪声记录，不影响 repo 内 `.gitignore` 判断。

## 后端验证

```powershell
cd D:\sub2api-trunk\backend
$env:GOCACHE='D:\sub2api-trunk\.cache\go-build'
go test ./cmd/server ./internal/service ./internal/handler ./internal/repository ./internal/server/middleware ./internal/server/routes -count=1
```

Exit 0：

- `cmd/server`
- `internal/service`
- `internal/handler`
- `internal/repository`
- `internal/server/middleware`
- `internal/server/routes`

```powershell
go test -tags=unit ./internal/service/ ./internal/handler/ ./internal/repository/ ./internal/server/middleware/
```

Exit 0。

```powershell
go test -tags=integration ./internal/repository -run TestAffiliateRepository_AccrueQuota_SourceOrderDuplicateDoesNotDoubleCredit -count=1
```

Exit 0。

补充窄测均 Exit 0：

- `TestGeminiNativeGroupAllowsModelScope_*`
- `TestAlreadyProcessed_RecentRechargingOrderStillAcksDuplicate`
- `TestAlreadyProcessed_StaleRechargingOrderReturnsRetryableError`
- `TestAffiliateRepository_AccrueQuota_ReturnsCappedAppliedAmount`

## 前端验证

```powershell
cd D:\sub2api-trunk\frontend
node_modules\.bin\vitest.cmd run src/components/payment/__tests__/paymentFlow.spec.ts src/views/user/__tests__/StripePaymentView.spec.ts src/views/user/__tests__/AirwallexPaymentView.spec.ts src/views/user/__tests__/PaymentResultView.spec.ts src/utils/sanitizeRedirectPath.test.ts src/api/admin/__tests__/ops.spec.ts src/api/__tests__/client.spec.ts
```

Exit 0：7 files, 49 tests passed。

```powershell
node_modules\.bin\vitest.cmd run src/views/auth/__tests__/OAuthCallbackView.spec.ts src/views/auth/__tests__/LinuxDoCallbackView.spec.ts src/views/auth/__tests__/OidcCallbackView.spec.ts src/views/auth/__tests__/WechatCallbackView.spec.ts src/views/auth/__tests__/EmailVerifyView.spec.ts
```

Exit 0：5 files, 71 tests passed。

```powershell
node_modules\.bin\vue-tsc.cmd --noEmit
node_modules\.bin\eslint.cmd . --ext .vue,.js,.jsx,.cjs,.mjs,.ts,.tsx,.cts,.mts
pnpm run build:fast
```

结果：

- `vue-tsc --noEmit`：exit 0。
- `eslint`：exit 0。
- `build:fast`：exit 0。
- `build:fast` 非阻断警告：pnpm overrides 位置提示、Node DEP0190、Browserslist 过期、Vite dynamic/static import chunk 警告、部分 chunk 超 500 kB。

## 安全扫描

系统 `python` 不可用，改用 Codex 捆绑 Python：

```powershell
& 'C:\Users\浩臣移动工作站\.cache\codex-runtimes\codex-primary-runtime\dependencies\python\python.exe' tools\secret_scan.py --include-untracked
```

Exit 0：

```text
secret-scan: no high-confidence tracked-plus-untracked findings
```

## Subagent 复查闭环

- Git/证据 reviewer：无硬阻塞；已标注 `git add -f` 和历史审计输入。
- Frontend reviewer：2 个 P1 已修，另补 3 个 P2 测试/边界。
- Backend reviewer：3 个 P1 已修，1 个 P2 affiliate 返回金额已修。
- Final reviewer：已补 P0 对账矩阵、exit code 证据、限定状态和一致回滚说明。

## 已知残余风险

- P1-027：请求前原子 quota reservation 尚未实现。
- P1-019：生产计费刹车依赖配置，默认安全性仍待复核。
- P1-020：真实 charge 属于后续阶段。
- P2/P3：历史审计项仍需拆分处理，不能写为已关闭。
- 没有真实 provider、部署、公网、生产数据验证，因此不能声明产品 READY。

## 回滚

当前未触库、未部署、未 push，且因 Git index 权限/系统用量限制未完成本地提交。提交前仍是未提交 dirty tree；提交后建议用本地 `git revert <commit>` 逐组回退；不使用 `reset --hard`、`clean` 或删除历史审计材料。未来若 migration 已执行，必须先备份数据库并另写兼容回滚脚本。
