# CODEX_TASK_S3_G2 — S3 收口：文档入库 + 人民币漏网之鱼 + 浏览器验收清单（新对话任务书）

> **执行模式：单会话顺序 loop。** 你是 Sub2API 仓执行代理（Codex）。
> **背景**：S3 loop 核心已 commit（`31af790e` + `a3adde7b`）：扣费自愈、admin production gate、console/usage/generation/video 人民币展示。
> 三方验证：**代码可信**，但仍有缺口：文档未入库、Groups 等页仍显示 `$`、无浏览器走查记录。
>
> **本任务目标**：收口 git 文档、补齐老板常看页面的 ¥ 漏网、产出可执行的浏览器验收步骤与结果记录。
> **禁止**：push、deploy、读 `.env`/密钥、真实付费 provider。

---

## 0. 读序

1. 本文件
2. `docs/superpowers/codex-handoff/deliverables/2026-07-06-S3-LOOP-review.md`
3. 桌面 `S3循环人民币任务全部执行过程总结.md`

**基线：**

```powershell
cd D:\sub2api-trunk
git log -3 --oneline    # 应有 a3adde7b, 31af790e
git status --short      # 预期：M CODEX_START_HERE.md, ?? CODEX_TASK_S3_LOOP_CNY.md
```

---

## Phase G2-0 · 文档 commit

```powershell
git add docs/superpowers/codex-handoff/CODEX_TASK_S3_LOOP_CNY.md
git add docs/superpowers/codex-handoff/CODEX_START_HERE.md
git commit -m "docs(s3): add S3 loop task book and update codex entry"
```

不得改动 `CODEX_START_HERE.md` 里与当前任务指向无关的其它段落，除非一并修正 S3 G2 入口。

---

## Phase G2-1 · 人民币漏网页面修补

验证发现：**console 三页（总览/成员/AI 记录）已 ¥**，但老板可能点到的页面仍 `$`：

| 优先级 | 文件 | 问题 |
|--------|------|------|
| P0 | `frontend/src/views/admin/GroupsView.vue` | 用量列 `${{ formatCost(...) }}`，无汇率 |
| P1 | `frontend/src/components/admin/affiliate/AdminAffiliateRecordsTable.vue`（若存在） | `'$' + formatAmount` |
| P2 | 订单/兑换页 | 部分 `$` 可能是 **USD 余额语义**（充值/兑换），**不要改**；只改「花费/用量统计」类列 |

**实现要求：**
- 复用已有 `useAdminDisplayCurrencyRate` 或 `useDisplayCurrency` 的 `formatCny` / `formatByCurrency`。
- 从 dashboard stats 或页面已有 API 读 `usd_cny_rate`，失败回退 7.2。
- **余额、quota、充值金额** 保持 USD 展示 + 中文标注「账户币种 USD」——不要误改。
- Groups 的 `daily_limit_usd` 等字段名含 usd，标签可改为「日限额（USD）」并旁注 ≈¥ 换算，**输入仍 USD**。

**测试：**
- 若有对应 spec，更新断言；至少跑：
```powershell
cd frontend
npx.cmd eslint . --ext .ts,.vue --max-warnings=0
npx.cmd vue-tsc --noEmit
npx.cmd vitest run src/views/admin/__tests__/GroupsView.spec.ts --reporter=basic
```
（无 Groups spec 则跑相关 admin 用例 + 新增最小 spec 1 条 format 断言）

Commit：`feat(console): extend CNY display to groups and affiliate usage columns`

---

## Phase G2-2 · 浏览器验收记录（人工步骤 + Codex 填表）

**Codex 不启动 dev server 也可以完成本节**：写出 `BROWSER_VERIFY_S3.md` 给老板/你自己按表打勾。

若本机可起 dev（`http://127.0.0.1:18081` 或项目默认端口），Codex 可尝试启动并记录；**起不来则只写步骤，标「待人工执行」**。

验收步骤（必须写入 deliverable）：

1. 登录管理后台（demo 模式六导航）
2. 设置页把 `usd_cny_rate` 改为 `7.5` 保存
3. 打开 **总览** → 总花费数字应变化（×7.5 口径）
4. 打开 **成员与开卡** → 今日/累计花费为 ¥
5. 打开 **任务记录 → AI 调用记录** → 金额为 ¥
6. 打开 **视频任务** → Seedance 行显示 ¥ 原价（不 ×7.5）；mock USD 行 ×7.5
7. 打开 **生成内容/提示词采集** → CNY 样本如 `¥5.01` 不是 `¥37.57`
8. 打开 **Groups**（若 G2-1 已做）→ 用量列为 ¥
9. 确认顶栏/个人余额仍为 USD 或明确标注

产出：`docs/superpowers/codex-handoff/deliverables/2026-07-07-S3-BROWSER-VERIFY.md`

每项：pass / fail / skip + 备注。不得伪造截图路径。

---

## Phase G2-3 · 全量门禁复跑

```powershell
cd D:\sub2api-trunk\backend
$env:GOCACHE='D:\sub2api-trunk\tmp\gocache'
go test ./...
golangci-lint run ./...

cd ..\frontend
npx.cmd eslint . --ext .ts,.vue --max-warnings=0
npx.cmd vue-tsc --noEmit
npx.cmd vitest run --reporter=basic
```

记录：退出码、测试数。失败 → 修（≤3 轮）或 blocked。

---

## Phase G2-4 · 审查包更新

更新或追加：
- `deliverables/2026-07-06-S3-LOOP-review.md` 追加 **G2 附录**，或新建 `deliverables/2026-07-07-S3-G2-closeout.md`

必须含：
- G2-0～G2-3 验收表
- 新增 commit hash
- 明确状态：**内部可用 / 待人工浏览器复核** → 若 G2-2 全 pass 可升为 **内部可用 / 可演示**（仍不宣称生产就绪）
- 未 push

Commit：`docs(s3): add G2 closeout and browser verify checklist`

---

## 停止条件

- 改 Groups 引发大范围回归 → 回滚该文件，标 blocked
- 需要改 users.balance 存储语义 → 停止，老板决策

---

## 新对话开场白（复制即用）

```text
你是 D:\sub2api-trunk 的执行代理（Codex）。
完整阅读并按 loop 执行：docs/superpowers/codex-handoff/CODEX_TASK_S3_G2_CLOSEOUT.md
先 G2-0 文档 commit，再 G2-1 Groups 等漏网 ¥，再 G2-2 浏览器验收表，再 G2-3 全量门禁，最后 G2-4 审查包。
Windows 前端门禁用 npx.cmd eslint / vue-tsc / vitest，不要用 pnpm run（缺 sh）。
禁止 push、禁止读 .env、禁止真实付费调用。
做完汇报审查包路径和 commit 列表。
```
