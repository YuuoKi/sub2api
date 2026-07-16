# K3 Apple-like Frontend Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build and verify the first reviewable Apple-like frontend checkpoint for Sub2API without changing any business contract or paid-provider path.

**Architecture:** Keep the existing Router → View → Store/Composable → API flow intact. Establish visual tokens and shared layout primitives first, then migrate the boss dashboard and video golden path onto those primitives; stop after Phase 1 screenshots and evidence so the user can approve the direction before full-site migration.

**Tech Stack:** Vue 3, TypeScript 5.6, Vite 5, Vue Router 4, Pinia, Tailwind CSS 3, Vue I18n, Vitest 2, Vue Test Utils, PowerShell on Windows.

## Global Constraints

- Work only on branch `codex/k3-apple-ui-experiment-20260717` based on `deb9ff5b8876a5bca1a96f78d94f602a3fd5adb4`.
- Read `00_START_HERE.md`, `01_PROJECT_BASELINE.md`, `02_CURRENT_REALITY_STATUS.md`, `docs/goals/03_CURRENT_GOAL.md`, `PRODUCT_INVARIANTS.md`, `ARCHITECTURE_GUARDRAILS.md`, `CODE_QUALITY_GATE.md`, and `docs/superpowers/specs/2026-07-17-k3-apple-ui-experiment-design.md` before editing.
- Allowed product-code scope is `frontend/**`; outside it, only this task's plan, screenshots, validation logs, and review package may change.
- Do not modify `backend/**`, API contracts, router guards, roles, feature flags, billing, currency semantics, task lifecycle, Idempotency-Key behavior, execution modes, or asset-delivery semantics.
- Do not read `.env`, keys, tokens, cookies, or credentials. Do not invoke a real Provider, push, deploy, merge, delete, reset, clean, or rebase.
- Preserve the Wujie teal brand accent; use status colors only for real semantics.
- Phase 1 ends after the visual checkpoint and review package. Do not begin Phase 2 without explicit user approval.

## File Structure

- `frontend/tailwind.config.js`: source of spacing, radius, shadow, color, and motion tokens.
- `frontend/src/style.css`: shared semantic classes and light/dark surface variables.
- `frontend/src/components/layout/AppLayout.vue`: page canvas and responsive content geometry.
- `frontend/src/components/layout/AppSidebar.vue`: role-aware navigation presentation; existing menu computation remains untouched.
- `frontend/src/components/layout/AppHeader.vue`: current-page and user-tool hierarchy; existing actions remain wired.
- `frontend/src/components/layout/TablePageLayout.vue`: reusable table-page composition.
- `frontend/src/views/admin/DashboardView.vue`: boss/admin information hierarchy checkpoint.
- `frontend/src/views/admin/video/VideoDashboardView.vue`: admin video overview checkpoint.
- `frontend/src/views/admin/video/VideoCreateTaskView.vue`: employee creation golden path.
- `frontend/src/views/admin/video/VideoTasksView.vue`: employee task-status and mobile golden path.
- `frontend/src/**/__tests__`: behavioral and visual-contract regression tests.
- `docs/reviews/K3_APPLE_UI_PHASE1_20260717/`: Phase 1 screenshots, validation logs, and summary.
- `docs/reviews/LATEST_REVIEW_PACKAGE.html`: self-contained latest review entry.

---

### Task 1: Freeze Baseline and Add the Visual Token Contract

**Files:**
- Modify: `frontend/tailwind.config.js`
- Modify: `frontend/src/style.css`
- Create: `frontend/src/__tests__/visualTokens.spec.ts`

**Interfaces:**
- Consumes: existing Tailwind `primary`, `dark`, `shadow-*`, and component classes.
- Produces: CSS variables `--ui-canvas`, `--ui-surface`, `--ui-surface-raised`, `--ui-border`, `--ui-text`, `--ui-text-muted`, `--ui-accent`, `--ui-focus`, plus classes `.ui-page`, `.ui-panel`, `.ui-toolbar`, `.ui-heading`, `.ui-subheading`.

- [ ] **Step 1: Record the immutable baseline**

Run:

```powershell
git status --short
git branch --show-current
git rev-parse HEAD
git diff -- frontend
```

Expected: clean worktree, branch `codex/k3-apple-ui-experiment-20260717`, HEAD containing design commit `f9c6e881`, and no frontend diff.

- [ ] **Step 2: Write the failing token contract test**

Create `frontend/src/__tests__/visualTokens.spec.ts`:

```ts
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const cssPath = fileURLToPath(new URL('../style.css', import.meta.url))
const css = readFileSync(cssPath, 'utf8')

describe('Apple-like visual token contract', () => {
  it.each([
    '--ui-canvas',
    '--ui-surface',
    '--ui-surface-raised',
    '--ui-border',
    '--ui-text',
    '--ui-text-muted',
    '--ui-accent',
    '--ui-focus'
  ])('defines %s', token => {
    expect(css).toContain(token)
  })

  it.each(['.ui-page', '.ui-panel', '.ui-toolbar', '.ui-heading', '.ui-subheading'])(
    'defines %s',
    selector => expect(css).toContain(selector)
  )

  it('supports reduced motion', () => {
    expect(css).toContain('@media (prefers-reduced-motion: reduce)')
  })
})
```

- [ ] **Step 3: Verify the contract test fails**

Run from `frontend`:

```powershell
npx.cmd vitest run src/__tests__/visualTokens.spec.ts --reporter=basic
```

Expected: FAIL because `--ui-canvas` and `.ui-page` are not defined.

- [ ] **Step 4: Implement the minimum shared token layer**

Add light variables under `@layer base` `:root`, dark overrides under `.dark`, and semantic classes under `@layer components`. Use these exact public names:

```css
:root {
  --ui-canvas: 248 250 252;
  --ui-surface: 255 255 255;
  --ui-surface-raised: 255 255 255;
  --ui-border: 15 23 42 / 0.08;
  --ui-text: 15 23 42;
  --ui-text-muted: 100 116 139;
  --ui-accent: 13 148 136;
  --ui-focus: 20 184 166 / 0.35;
}

.dark {
  --ui-canvas: 9 12 17;
  --ui-surface: 17 24 39;
  --ui-surface-raised: 24 33 47;
  --ui-border: 255 255 255 / 0.10;
  --ui-text: 241 245 249;
  --ui-text-muted: 148 163 184;
  --ui-accent: 45 212 191;
  --ui-focus: 45 212 191 / 0.40;
}
```

Implement `.ui-page`, `.ui-panel`, `.ui-toolbar`, `.ui-heading`, and `.ui-subheading` using Tailwind `@apply` plus the variables. Keep existing `.btn`, `.input`, `.card`, `.modal`, and `.sidebar` class names backward compatible. Add reduced-motion rules that set animation duration and transition duration to `0.01ms` without removing focus feedback.

- [ ] **Step 5: Run token tests and static gates**

Run:

```powershell
npx.cmd vitest run src/__tests__/visualTokens.spec.ts --reporter=basic
npx.cmd eslint . --ext .ts,.vue --max-warnings=0
npx.cmd vue-tsc --noEmit
```

Expected: all exit 0.

- [ ] **Step 6: Commit the token contract**

```powershell
git add frontend/tailwind.config.js frontend/src/style.css frontend/src/__tests__/visualTokens.spec.ts
git diff --cached --check
git commit -m "feat(ui): establish apple-like visual tokens"
```

### Task 2: Recompose the Shared Application Shell

**Files:**
- Modify: `frontend/src/components/layout/AppLayout.vue`
- Modify: `frontend/src/components/layout/AppSidebar.vue`
- Modify: `frontend/src/components/layout/AppHeader.vue`
- Modify: `frontend/src/components/layout/TablePageLayout.vue`
- Modify: `frontend/src/components/layout/__tests__/AppSidebar.spec.ts`
- Create: `frontend/src/components/layout/__tests__/AppLayoutVisualContract.spec.ts`

**Interfaces:**
- Consumes: Task 1 semantic classes and existing stores, computed navigation, feature flags, theme state, user menu, and slots.
- Produces: stable shell markers `data-testid="app-shell"`, `data-testid="app-main"`, `data-testid="app-header"`, `data-testid="app-sidebar"`; no change to route or store interfaces.

- [ ] **Step 1: Write shell contract tests**

Create `AppLayoutVisualContract.spec.ts` as a source contract so it does not mock unrelated stores:

```ts
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const read = (name: string) => readFileSync(fileURLToPath(new URL(`../${name}`, import.meta.url)), 'utf8')

describe('shared app shell visual contract', () => {
  it('exposes stable shell landmarks', () => {
    expect(read('AppLayout.vue')).toContain('data-testid="app-shell"')
    expect(read('AppLayout.vue')).toContain('data-testid="app-main"')
    expect(read('AppHeader.vue')).toContain('data-testid="app-header"')
    expect(read('AppSidebar.vue')).toContain('data-testid="app-sidebar"')
  })

  it('keeps layout motion accessible', () => {
    expect(read('AppLayout.vue')).toContain('motion-reduce:transition-none')
  })
})
```

Add to `AppSidebar.spec.ts`:

```ts
it('keeps role and feature-gated navigation builders intact', () => {
  expect(source).toContain('const adminNavItems = computed')
  expect(source).toContain('const userNavItems = computed')
  expect(source).toContain('applyFeatureFlags')
})
```

- [ ] **Step 2: Verify shell tests fail only for new landmarks**

Run:

```powershell
npx.cmd vitest run src/components/layout/__tests__/AppSidebar.spec.ts src/components/layout/__tests__/AppLayoutVisualContract.spec.ts --reporter=basic
```

Expected: existing sidebar tests PASS; new landmark assertions FAIL.

- [ ] **Step 3: Implement the shell without touching navigation computation**

Apply Task 1 classes to the layout. Remove `bg-mesh-gradient` from the default canvas and replace decorative glow with neutral surfaces. Keep these script symbols and their behavior unchanged: `sidebarCollapsed`, `mobileOpen`, `adminNavItems`, `userNavItems`, `personalNavItems`, `customMenuItemsForUser`, `customMenuItemsForAdmin`, `sanitizeSvg`, `toggleTheme`, and stored sidebar scroll restoration.

Use a 256px expanded sidebar, 72px collapsed sidebar, sticky header, and a main content maximum width that can be disabled by data-heavy views. Keep the mobile overlay and drawer close behavior. Add semantic landmarks and accessible labels rather than changing route targets.

- [ ] **Step 4: Run shell and navigation gates**

```powershell
npx.cmd vitest run src/components/layout/__tests__/AppSidebar.spec.ts src/components/layout/__tests__/AppLayoutVisualContract.spec.ts src/__tests__/integration/navigation.spec.ts src/router/__tests__/guards.spec.ts src/router/__tests__/feature-access.spec.ts src/router/__tests__/video-route-access.spec.ts --reporter=basic
npx.cmd eslint . --ext .ts,.vue --max-warnings=0
npx.cmd vue-tsc --noEmit
```

Expected: all exit 0; employee and admin route boundaries remain unchanged.

- [ ] **Step 5: Commit the shell**

```powershell
git add frontend/src/components/layout frontend/src/__tests__/integration/navigation.spec.ts frontend/src/router/__tests__
git diff --cached --check
git commit -m "feat(ui): simplify the shared application shell"
```

### Task 3: Redesign the Boss Dashboard as the Visual North Star

**Files:**
- Modify: `frontend/src/views/admin/DashboardView.vue`
- Modify: `frontend/src/views/admin/__tests__/DashboardView.spec.ts`
- Modify only if required for presentation: `frontend/src/components/common/StatCard.vue`
- Modify only if required for presentation: `frontend/src/components/charts/*.vue`

**Interfaces:**
- Consumes: existing dashboard API calls, default 24-hour range, display currency, reconciliation conclusion, charts, and user ranking.
- Produces: a clear outcome-first dashboard with markers `dashboard-summary`, `dashboard-kpis`, `dashboard-attention`, and `dashboard-trends`.

- [ ] **Step 1: Add outcome-hierarchy assertions**

Add to `DashboardView.spec.ts` after the existing 24-hour test:

```ts
it('renders the boss dashboard in outcome-first order', async () => {
  const wrapper = mount(DashboardView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        LoadingSpinner: true,
        Icon: true,
        DateRangePicker: true,
        Select: true,
        ModelDistributionChart: true,
        TokenUsageTrend: true,
        Line: true
      }
    }
  })
  await flushPromises()

  const ids = ['dashboard-summary', 'dashboard-kpis', 'dashboard-attention', 'dashboard-trends']
  for (const id of ids) expect(wrapper.find(`[data-testid="${id}"]`).exists()).toBe(true)

  const html = wrapper.html()
  expect(html.indexOf('dashboard-summary')).toBeLessThan(html.indexOf('dashboard-kpis'))
  expect(html.indexOf('dashboard-kpis')).toBeLessThan(html.indexOf('dashboard-trends'))
})
```

- [ ] **Step 2: Verify the new dashboard test fails**

```powershell
npx.cmd vitest run src/views/admin/__tests__/DashboardView.spec.ts --reporter=basic
```

Expected: FAIL because the four test IDs are absent; all pre-existing assertions remain green.

- [ ] **Step 3: Implement the outcome-first dashboard**

Keep every existing fetch, computed value, time-range default, refresh action, currency formatter, and chart data mapping. Reorder only the presentation:

1. reconciliation/operating conclusion and its honest pending state;
2. no more than four primary KPIs, with secondary metrics visually quieter;
3. an attention section for exceptions and next actions;
4. trends and rankings below the decision layer.

Use neutral panels, tabular numerals, concise labels, one teal primary action, and explicit empty states. Do not translate `$` to CNY or claim a Provider invoice was uploaded when it was not.

- [ ] **Step 4: Run dashboard and shared gates**

```powershell
npx.cmd vitest run src/views/admin/__tests__/DashboardView.spec.ts src/components/__tests__/Dashboard.spec.ts --reporter=basic
npx.cmd eslint . --ext .ts,.vue --max-warnings=0
npx.cmd vue-tsc --noEmit
```

Expected: all exit 0.

- [ ] **Step 5: Commit the dashboard**

```powershell
git add frontend/src/views/admin/DashboardView.vue frontend/src/views/admin/__tests__/DashboardView.spec.ts frontend/src/components/common/StatCard.vue frontend/src/components/charts
git diff --cached --check
git commit -m "feat(ui): focus the boss dashboard on decisions"
```

### Task 4: Redesign the Video Golden Path Without Changing Lifecycle Semantics

**Files:**
- Modify: `frontend/src/views/admin/video/VideoDashboardView.vue`
- Modify: `frontend/src/views/admin/video/VideoCreateTaskView.vue`
- Modify: `frontend/src/views/admin/video/VideoTasksView.vue`
- Modify: `frontend/src/views/admin/video/__tests__/VideoCreateTaskExecutionMode.spec.ts`
- Modify: `frontend/src/views/admin/video/__tests__/VideoTasksView.spec.ts`
- Keep passing: all other files in `frontend/src/views/admin/video/__tests__/`

**Interfaces:**
- Consumes: existing execution-mode capability gates, stable Idempotency-Key, create API, task lifecycle polling, local/remote asset distinctions, filters, pagination, and 390px mobile list.
- Produces: markers `video-primary-action`, `video-execution-mode`, `video-task-filters`, and existing `video-task-mobile-list` / `video-task-desktop-table`.

- [ ] **Step 1: Add presentation-only regression assertions**

In `VideoCreateTaskExecutionMode.spec.ts`, add:

```ts
it('keeps execution mode visible beside the primary task action', async () => {
  const wrapper = mount(VideoCreateTaskView)
  await flushPromises()
  expect(wrapper.find('[data-testid="video-execution-mode"]').exists()).toBe(true)
  expect(wrapper.find('[data-testid="video-primary-action"]').exists()).toBe(true)
})
```

In `VideoTasksView.spec.ts`, add:

```ts
it('keeps task filters separate from task results', async () => {
  wrapper = mount(VideoTasksView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        Icon: true
      }
    }
  })
  await flushPromises()
  expect(wrapper.find('[data-testid="video-task-filters"]').exists()).toBe(true)
  expect(wrapper.find('[data-testid="video-task-desktop-table"]').exists()).toBe(true)
  expect(wrapper.find('[data-testid="video-task-mobile-list"]').exists()).toBe(true)
})
```

- [ ] **Step 2: Verify only the new presentation assertions fail**

```powershell
npx.cmd vitest run src/views/admin/video/__tests__/VideoCreateTaskExecutionMode.spec.ts src/views/admin/video/__tests__/VideoTasksView.spec.ts --reporter=basic
```

Expected: FAIL for missing test IDs; default mock, idempotency, polling, delivery, and mobile assertions remain green.

- [ ] **Step 3: Implement the three-screen hierarchy**

- Video dashboard: show system status, one recommended next action, then supporting counts and links.
- Create task: keep prompt and execution mode in the primary flow; put advanced parameters behind the existing disclosure; keep real-call confirmation impossible to miss.
- Task list: simplify filters, keep statuses and failure reasons explicit, preserve the separate 390px card list, and keep remote deliverable distinct from locally archived asset.

Do not rename or replace API calls. Do not change `execution_mode`, `provider_account_id`, `Idempotency-Key`, lifecycle polling conditions, `hasUsableRemoteAsset`, `local_asset_available`, result expiration, or copy-to-create behavior.

- [ ] **Step 4: Run the complete video regression suite**

```powershell
npx.cmd vitest run src/views/admin/video/__tests__ src/router/__tests__/video-route-access.spec.ts --reporter=basic
npx.cmd eslint . --ext .ts,.vue --max-warnings=0
npx.cmd vue-tsc --noEmit
```

Expected: all exit 0; no skipped lifecycle tests introduced.

- [ ] **Step 5: Commit the video golden path**

```powershell
git add frontend/src/views/admin/video
git diff --cached --check
git commit -m "feat(ui): clarify the video production journey"
```

### Task 5: Verify Phase 1 and Produce the Review Checkpoint

**Files:**
- Create: `docs/reviews/K3_APPLE_UI_PHASE1_20260717/SUMMARY.md`
- Create: `docs/reviews/K3_APPLE_UI_PHASE1_20260717/validation.log`
- Create: `docs/reviews/K3_APPLE_UI_PHASE1_20260717/screenshots/*.png`
- Modify: `docs/reviews/LATEST_REVIEW_PACKAGE.html`

**Interfaces:**
- Consumes: Tasks 1–4 and the baseline screenshots under `docs/reviews/assets/real-product-readiness-20260715/`.
- Produces: a self-contained Phase 1 decision package; no production-readiness status change.

- [ ] **Step 1: Run all frontend gates from `frontend`**

```powershell
npx.cmd eslint . --ext .ts,.vue --max-warnings=0
npx.cmd vue-tsc --noEmit
npx.cmd vitest run --reporter=basic
pnpm run build
```

Expected: four commands exit 0. Record exact command, exit code, test count, build output summary, and timestamp in `validation.log`; do not summarize a failed or skipped gate as PASS.

- [ ] **Step 2: Run repository hygiene checks**

```powershell
Set-Location ..
git diff --check
git status --short
git diff --stat wujie/video-capture-moat-20260702...HEAD
git diff --name-only wujie/video-capture-moat-20260702...HEAD
```

Expected: no whitespace errors; product-code changes stay within `frontend/**`; additional changes are only task docs and evidence.

- [ ] **Step 3: Capture the visual matrix using only mock or no-paid data**

Capture PNG screenshots for:

```text
boss-dashboard-light-1440.png
boss-dashboard-dark-1440.png
admin-video-dashboard-light-1440.png
employee-video-create-light-1440.png
employee-video-tasks-light-1440.png
employee-video-tasks-dark-1440.png
employee-video-tasks-light-390.png
employee-video-tasks-dark-390.png
```

Before every capture, confirm the role, route, viewport, theme, mock/no-paid mode, and absence of secrets. Do not trigger a real create. Compare against `01-boss-dashboard.png`, `02-admin-video-dashboard.png`, `06-employee-video-create.png`, `07-employee-video-tasks.png`, and `09-employee-tasks-mobile.png`.

- [ ] **Step 4: Write the self-contained review package**

`SUMMARY.md` and `LATEST_REVIEW_PACKAGE.html` must contain:

```text
目标与设计原则
执行分支、baseline HEAD、review HEAD
精确变更清单
前后对比截图
三角色与四种视图矩阵
验证命令、退出码和结果
未验证或跳过项
业务不变量复核
风险与回滚
当前状态：可演示 或 待复核
Phase 2 建议与明确停止点
文件索引
可复制后续提示词
```

Do not write “内部可用” or production READY based on this visual checkpoint.

- [ ] **Step 5: Commit the Phase 1 checkpoint and stop**

```powershell
git add -f docs/reviews/K3_APPLE_UI_PHASE1_20260717 docs/reviews/LATEST_REVIEW_PACKAGE.html
git diff --cached --check
git commit -m "docs(review): package K3 apple-like phase 1"
git status --short
git log --oneline --decorate -6
```

Expected: commit succeeds, worktree is clean, and execution stops for user review. Do not begin Phase 2, push, merge, or deploy.
