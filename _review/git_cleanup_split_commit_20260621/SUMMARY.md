# G_GIT_CLEANUP_SPLIT_COMMIT — 工作区清理 + C 看板 / D3 分包提交

**日期**：2026-06-21 ｜ **仓**：`D:\sub2api-trunk`（中台仓 linked worktree）｜ **分支**：`wujie/trunk`
**起点 HEAD**：`eca1b65c`（未 push）｜ **执行**：Claude Code (Opus 4.8, ultracode)
**性质**：纯 git 卫生操作——只暂存/提交已审过的现有改动，**未改/未增/未删任何一行代码内容**。

## 结果（一句话）

工作区脏改动已干净拆成两个本地 commit：

| 序 | commit | 说明 | 规模 |
|----|--------|------|------|
| 1 | `e078749a` | **C 只读看板**（护城河快照卡 + sparkline + 样本墙） | 19 files, +719 / −5 |
| 2 | `38df1bcd` | **D3**（脱敏加固 + 保留期 NULL-OUT） | 12 files, +840 / −3 |

`wire_gen.go:188` 逐字未动；主路径（gateway*）零触碰；未跑 wire codegen；未 push；`_review/` 等非产品码全部保持未跟踪。

---

## 1. 四类分类表（逐文件）

### (a) 纯 C 看板 → 全进 Commit 1（whole-file `git add`，15 个）
| 文件 | 性质 |
|------|------|
| `backend/cmd/server/wire_gen.go` | 手工补线：GenerationContent repo+handler（@L244，**不动 L188**） |
| `backend/internal/handler/handler.go` | `AdminHandlers.GenerationContent` 字段 |
| `backend/internal/handler/wire.go` | `ProvideAdminHandlers` 形参 + ProviderSet |
| `backend/internal/repository/wire.go` | `NewGenerationContentRepository` 入 ProviderSet |
| `backend/internal/server/routes/admin.go` | `registerGenerationContentRoutes`（/stats、/samples） |
| `backend/internal/handler/admin/generation_content_handler.go` | **新增** C handler（只调 GetCaptureStats/GetRecent） |
| `frontend/src/api/admin/generation_content.ts` | **新增** C api client |
| `frontend/src/api/admin/index.ts` | barrel 注册 |
| `frontend/src/components/admin/generation-content/CaptureSparkline.vue` | **新增** |
| `frontend/src/components/admin/generation-content/ContentWall.vue` | **新增** |
| `frontend/src/views/admin/GenerationContentView.vue` | **新增** |
| `frontend/src/router/index.ts` | 路由 |
| `frontend/src/components/layout/AppSidebar.vue` | 导航项 |
| `frontend/src/i18n/locales/en.ts` | i18n（仅 generationContent 键） |
| `frontend/src/i18n/locales/zh.ts` | i18n（仅 generationContent 键） |

### (b) 纯 D3 → 全进 Commit 2（whole-file `git add`，8 个）
| 文件 | 性质 |
|------|------|
| `backend/internal/config/config.go` | `ContentRetentionConfig`（dark by default） |
| `backend/internal/service/generation_content_redact.go` | 脱敏 v2：身份证/银行卡/opaque token |
| `backend/internal/service/generation_content_retention_service.go` | **新增** 保留期清理服务 |
| `backend/internal/service/generation_content_retention_service_test.go` | **新增** |
| `backend/internal/repository/generation_content_retention_repo_integration_test.go` | **新增** |
| `backend/internal/service/generation_content_redact_demo_test.go` | **新增** before/after 证据 |
| `backend/internal/service/generation_content_redact_structured_test.go` | **新增** 结构化 PII 用例 |
| `backend/migrations/141_ai_generation_content_retention_index.sql` | **新增** partial index |

### (c) 混合文件 → 按 hunk 拆（4 个，C 部分进 1，D3 部分进 2）
| 文件 | C 部分 | D3 部分 |
|------|--------|---------|
| `backend/internal/repository/generation_content_repo.go` | `GetCaptureStats` + `GetRecent`（+114） | `"time"` import + `PurgeExpiredContent`（+49） |
| `backend/internal/service/generation_content.go` | 接口注释改 + `GetCaptureStats`/`GetRecent` 接口方法 + 3 个 struct（DailyPoint/Stats/Sample）（+37/−1） | `PurgeExpiredContent` 接口方法 4 行（+4） |
| `backend/internal/service/generation_content_collector_test.go` | `GetCaptureStats`/`GetRecent` mock（+9） | `"time"` import + `PurgeExpiredContent` mock（+26） |
| `backend/internal/service/generation_content_collector_panic_test.go` | `GetCaptureStats`/`GetRecent` mock（+9） | `"time"` import + `PurgeExpiredContent` mock（+5） |

> 拆分对账：4 个混合文件每个 `staged(C) + unstaged(D3) == 原 full diff` 全部相等。
> 8 个删除行精确劈成 **5（C）+ 3（D3）**；27 个去重文件 = 19 + 12 − 4(混合)。

### (d) 不入库（保持未跟踪/忽略，未动未删）
- `_review/**`（全部审查包 / evidence / harness / throwaway `.sql` / mock `.go` / `*.pid` 等）
- 无 `*.exe`、无根级/docs 报告、无 `_NIGHT_RUN*` 异物（现场未发现异物）。

---

## 2. 混合文件 hunk 归属说明（如何拆）

对 4 个混合文件用「等价 patch 方式」拆分（非 `git add .`/`-A`）：
1. `git diff <file>` 导出完整 patch 到仓外临时目录；
2. 按行号删除 D3 行段（tab-safe，不碰 Edit 易错的制表符匹配），生成 `*.c.patch`；
3. `git apply --cached --recount` 把 **C-only** hunk 暂存进 index；
4. 提交 Commit 1 后，对同一文件 `git add <file>` 自动把「index(已含C) ↔ 工作树(C+D3)」的差 = **D3 余量**暂存，进 Commit 2。

D3 行段（被从 C patch 中剔除的部分）：
- **repo.go**：hunk1 整段（`+"time"` import）；hunk2 中 `PurgeExpiredContent` 函数块（含其与 GetRecent 间的 1 行空行）。保留 GetCaptureStats↔GetRecent 之间恰好 1 行空行（gofmt 合规）。
- **generation_content.go**：接口体内 `PurgeExpiredContent` 注释×3 + 签名×1（共 4 行）。接口注释改归 C（"并为只读看板提供聚合/样本读取" 描述的是读方法）。
- **ctest / ptest**：hunk1 整段（`+"time"` import）；hunk2 中 `PurgeExpiredContent` mock 块（含 1 行空行）。

C patch 应用前已 `git apply --cached --check --recount` 全部干跑通过；应用后 `git diff --cached | grep D3标记` = **空**。

---

## 3. Step 3 — Commit 1（C）验证

```
git show --stat HEAD  → 19 files（仅 C：wire_gen/handler/wire/repo(只读)/routes/handler.go
                         + 新增 handler + 4 个 frontend 新件 + i18n/router/sidebar/barrel），+719/−5
```

**wire_gen.go:188 逐字未动**（C 补线落在 L244，在 188 之下，不移位）：
```
eca1b65c:188 = \tgatewayService.SetGenerationContentCollector(service.NewGenerationContentCollector(repository.NewGenerationContentRepository(db), configConfig))
HEAD    :188 = (同上)   >>> IDENTICAL <<<
```

**真·C 孤立编译验证**（`git archive HEAD | tar -x` 导出 C-commit 树到仓外，**所有 D3 文件缺席**，零触碰任何 worktree）：
```
confirmed: no retention/structured/demo D3 files in C tree
confirmed: migration 141 absent from C tree
go build ./...                                  → exit 0   ✅
go vet ./internal/service/... ./repository/... ./handler/...  → exit 0   ✅
```
> 即 C commit 在「D3 全不存在」时也能独立 build+vet（C-only mock 满足 C-only 接口）——分包边界自洽，非仅靠工作树整体编译。

---

## 4. Step 5 — Commit 2（D3）验证（全工作树 = C+D3）

```
git show --stat HEAD  → 12 files（仅 D3：config/redact/repo(Purge)/generation_content.go(Purge接口)
                         /2 collector test(Purge mock) + 5 新件 + migration 141），+840/−3
go build ./...   → exit 0   ✅
go vet ./...     → exit 0   ✅
go test ./internal/service/...  → ok  github.com/Wei-Shaw/sub2api/internal/service  48.278s  → exit 0   ✅
```

D3 关键单测（`-run 'Redact|Retention|Purge|Structured|CNID|Card|Opaque' -v`）全 PASS：
```
--- PASS: TestRedactGenerationStructuredPII_CNIDCardRedacted
--- PASS: TestRedactGenerationStructuredPII_BankCardRedacted
--- PASS: TestRedactGenerationStructuredPII_OpaqueTokenRedacted
--- PASS: TestRedactGenerationStructuredPII_IdentifiersUntouched
--- PASS: TestRedactGenerationStructuredPII_InvalidIDNotRedacted
--- PASS: TestRedactGenerationStructuredPII_NormalChinesePreserved
--- PASS: TestRedactGenerationHardening_BeforeAfterDemo
--- PASS: TestRedactGenerationPrompt_IDAndCardRedacted
--- PASS: TestNewGenerationContentRetentionService_{UsesConfig,Defaults,ClampsRetentionDays}
--- PASS: TestGenerationContentRetention_RunOnceDryRun_NoSideEffects
--- PASS: TestGenerationContentRetention_RunOncePurgesOnlyExpired
--- PASS: TestGenerationContentRetention_RunOnceDrainsMultipleBatches
--- PASS: TestGenerationContentRetention_RunOnceFailOpenOnRepoError
--- PASS: TestGenerationContentRetention_NilSafe
(exit 0)
```

---

## 5. Step 6 — 最终核验

```
[1] git log --oneline -3
    38df1bcd feat(retention): D3 脱敏加固...
    e078749a feat(dashboard): 价值出口 C 只读看板...
    eca1b65c feat(collector): M1-B 主路径...            ← 两新 commit 落在 eca1b65c 之上 ✅

[2] git status → 工作区干净，唯一未跟踪 = _review（产品码全已提交）✅
    tracked unstaged = 0 ; staged uncommitted = 0 ; untracked 非_review 产品文件 = 0

[3] git diff --name-only eca1b65c HEAD | grep -i gateway  → 空（主路径零触碰）✅

[4] git diff eca1b65c HEAD --stat → 27 files changed, 1559 insertions(+), 8 deletions(-)
    = C(719/5) + D3(840/3)，删除 8 = 5+3，去重文件 27 = 19+12−4 → 两 commit 之和 == 提交前工作区产品码 ✅

[5] wire_gen.go eca1b65c..HEAD = 仅 L244 一处手工补线 hunk（无整文件重排）→ 未跑 wire codegen ✅
    wire_gen.go:188 HEAD vs eca1b65c → IDENTICAL ✅
```

---

## 6. 禁止项逐条确认（§5）

- ⛔ 不 push — **未 push**（两 commit 仅本地）✅
- ⛔ 不改代码逻辑 — **仅暂存/提交既有改动，零增删改代码内容**✅
- ⛔ 不跑 wire codegen / 不动 188 — **未跑；188 逐字在；wire_gen 仅 L244 单 hunk**✅
- ⛔ 不提交 `_review`/evidence/文档/.exe — **未提交，全留未跟踪**✅
- ⛔ `git add .`/`-A` — **未用；全程显式 `git add <file>` 或 `git apply --cached`**✅
- ⛔ `reset --hard`/`clean -fd`/`rebase` — **未用**✅
- ⛔ 不碰 main/旧路径/其它 worktree、不翻 flag、不连真库/真 key — **未碰**（C 孤立验证用 `git archive` 导出仓外临时目录，非 worktree）✅

## 停止条件

✅ 两干净 commit 完成 + §3/§5 验证齐 + 工作区干净（仅 `_review` 未跟踪）→ **停，等迁移**。
