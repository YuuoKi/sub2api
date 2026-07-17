// K3 Apple-like UI Phase 1.5 — polish evidence capture (WP-D).
// Same fully-mocked approach as capture_k3.mjs: vite preview + Playwright route
// interception. No backend, no credentials, no real Provider, no create submission.
// Targets: TaskProgressRing rows, AnimatedEmptyState variants, ui-skeleton shimmer
// (CSS injection demo — the common Skeleton component currently has NO runtime
// consumer, recorded as a Phase 1.5 risk), Chinese 404, Chinese billing headers.
import { createRequire } from 'node:module'
import fs from 'node:fs/promises'
import path from 'node:path'

const require = createRequire(import.meta.url)
const { chromium } = require('C:/tmp/sub2api-playwright/node_modules/playwright-core')

const baseURL = process.env.K3_BASE_URL || 'http://127.0.0.1:5199'
const outputDir = path.resolve('D:/sub2api-trunk/docs/reviews/K3_APPLE_UI_PHASE1_20260717/screenshots')
const evidencePath = path.resolve('D:/sub2api-trunk/docs/reviews/K3_APPLE_UI_PHASE1_20260717/capture-evidence-phase1.5.json')
const edge = 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe'

// Clearly-fake local session marker (not a credential; API is fully mocked).
const MOCK_TOKEN = 'k3-review-mock-session'

const NOW = '2026-07-17T03:30:00+08:00'
const hoursAgo = (h) => new Date(new Date(NOW).getTime() - h * 3600_000).toISOString()

const adminUser = {
  id: 1,
  username: 'admin',
  email: 'admin@sub2api.local',
  role: 'admin',
  balance: 0,
  concurrency: 10,
  status: 'active',
  allowed_groups: null,
  balance_notify_enabled: false,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  run_mode: 'standard',
}

const employeeUser = {
  id: 9001,
  username: 'employee-k3',
  email: 'employee-g6@sub2api.local',
  role: 'user',
  balance: 100,
  concurrency: 5,
  status: 'active',
  allowed_groups: null,
  balance_notify_enabled: false,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  run_mode: 'standard',
}

const mockProviderAccount = {
  id: 11,
  provider: 'mock',
  display_name: 'Mock 试跑通道',
  enabled: true,
  api_key_configured: true,
  masked_key: 'mock-****',
  base_url: 'https://mock.video-gateway.local',
  default_model: 'mock-video-1',
  rate_limit_per_minute: 30,
  metadata_json: {},
  key_status: 'valid',
  health_status: 'healthy',
  diagnostic_type: '',
  suggested_action: '',
  priority: 1,
  current_inflight: 1,
  today_tasks: 6,
  today_failures: 1,
  last_error: '',
  last_test_at: hoursAgo(3),
  route_available: true,
  route_skip_reason: '',
  created_at: hoursAgo(400),
  updated_at: hoursAgo(3),
}

const task = (over) => ({
  id: 5001,
  provider_account_id: 11,
  provider_account_name: 'Mock 试跑通道',
  provider: 'mock',
  model: 'mock-video-1',
  task_type: 'text_to_video',
  prompt: '【Mock】样例任务',
  status: 'succeeded',
  result_url: '',
  result_url_expires_at: null,
  result_url_expiry_source: 'unknown',
  local_asset_path: '',
  local_asset_saved_at: null,
  local_asset_available: false,
  dispatch_state: 'done',
  settlement_status: 'settled',
  archive_status: 'archived',
  capture_status: 'captured',
  delivery_status: 'deliverable',
  next_action: 'download',
  error_message: '',
  cost_estimate: 0,
  currency: 'USD',
  created_by: 9001,
  created_by_email: 'employee-g6@sub2api.local',
  created_by_name: 'employee-k3',
  created_by_label: 'employee-k3 (employee-g6@sub2api.local)',
  created_at: hoursAgo(2),
  updated_at: hoursAgo(1),
  completed_at: hoursAgo(1),
  negative_prompt: '',
  reference_image_url: '',
  reference_video_url: '',
  aspect_ratio: '16:9',
  duration: 6,
  resolution: '1080p',
  upstream_task_id: 'mock-upstream-5001',
  routing_strategy: 'priority',
  routing_reason: 'mock 试跑通道',
  ...over,
})

// In-progress rows first so the progress ring is above the fold.
const videoTasks = [
  task({ id: 5006, prompt: '【Mock】城市夜景航拍，霓虹倒影在雨后街道', status: 'queued', delivery_status: 'processing', next_action: 'poll', completed_at: null, created_at: hoursAgo(0.1), updated_at: hoursAgo(0.1) }),
  task({ id: 5005, prompt: '【Mock】产品宣传片：茶杯在晨光中旋转', status: 'running', delivery_status: 'processing', next_action: 'poll', completed_at: null, created_at: hoursAgo(0.4), updated_at: hoursAgo(0.2) }),
  task({ id: 5004, prompt: '【Mock】山间云海延时摄影', status: 'failed', delivery_status: 'delivery_failed', next_action: 'review_delivery', error_message: '上游返回超时（mock 模拟失败样例）', completed_at: hoursAgo(3), created_at: hoursAgo(3.5), updated_at: hoursAgo(3) }),
  task({ id: 5003, prompt: '【Mock】办公室场景：团队协作白板讨论', status: 'succeeded', delivery_status: 'deliverable', next_action: 'download', local_asset_available: true, local_asset_path: '/data/video_assets/mock/5003.mp4', local_asset_saved_at: hoursAgo(5), result_url: 'https://files.example.com/mock/5003.mp4', created_at: hoursAgo(6), updated_at: hoursAgo(5), completed_at: hoursAgo(5) }),
  task({ id: 5002, prompt: '【Mock】海边日落，浪花在礁石上破碎', status: 'succeeded', delivery_status: 'deliverable', next_action: 'download', result_url: 'https://files.example.com/mock/5002-cover.svg?expires=demo', result_url_expires_at: hoursAgo(-20), result_url_expiry_source: 'url_query', created_at: hoursAgo(9), updated_at: hoursAgo(8), completed_at: hoursAgo(8) }),
]

const emptyVideoDashboard = {
  today_tasks: 0,
  success_rate: 0,
  failed_tasks: 0,
  queued_tasks: 0,
  running_tasks: 0,
  provider_status: [],
  health_diagnostics: [],
  recent_failures: [],
  recent_successes: [],
  usage_overview: [],
}

const billingImports = {
  items: [
    {
      id: 31,
      provider: 'seedance',
      provider_account_id: 'mock-acct-01',
      billing_period_start: '2026-06-01',
      billing_period_end: '2026-06-30',
      timezone: 'UTC',
      original_currency: 'USD',
      source_type: 'csv',
      invoice_number: 'MOCK-INV-202606',
      file_sha256: 'mocksha25600000000000000000000000000000000000000000000000000demo',
      storage_key: 'mock/31.csv',
      original_filename: 'mock-statement-202606.csv',
      byte_size: 2048,
      status: 'imported',
      line_count: 42,
      created_at: hoursAgo(48),
    },
  ],
}

const billingPeriodSummary = {
  items: [
    {
      provider: 'seedance',
      provider_account_id: 'mock-acct-01',
      billing_period_start: '2026-06-01',
      billing_period_end: '2026-06-30',
      import_count: 1,
      matched: 40,
      has_diff: 2,
      provider_only: 0,
      internal_only: 0,
      conclusion: 'has_diff',
    },
  ],
}

const publicSettings = {
  registration_enabled: false,
  email_verify_enabled: false,
  force_email_on_third_party_signup: false,
  registration_email_suffix_whitelist: [],
  promo_code_enabled: false,
  password_reset_enabled: true,
  invitation_code_enabled: false,
  turnstile_enabled: false,
  turnstile_site_key: '',
  site_name: 'Sub2API',
  site_logo: '',
  site_subtitle: 'K3 Review Mock',
  api_base_url: '',
  contact_info: '',
  doc_url: '',
  home_content: '',
  hide_ccs_import_button: false,
  payment_enabled: false,
  risk_control_enabled: false,
  table_default_page_size: 20,
  table_page_size_options: [10, 20, 50, 100],
  custom_menu_items: [],
  custom_endpoints: [],
  linuxdo_oauth_enabled: false,
  wechat_oauth_enabled: false,
  oidc_oauth_enabled: false,
  oidc_oauth_provider_name: '',
  github_oauth_enabled: false,
  google_oauth_enabled: false,
  backend_mode_enabled: false,
  version: 'k3-review-mock',
}

const secretPatterns = [
  /Authorization\s*[:=]/i,
  /api[_-]?key\s*[:=]\s*['"]?[A-Za-z0-9_\-]{16,}/i,
  /password\s*[:=]/i,
  /[?&](X-Amz-Signature|Signature|token)=/i,
  /sk-[A-Za-z0-9]{20,}/,
]

const evidence = {
  baseURL,
  capturedAt: new Date().toISOString(),
  mode: 'fully-mocked (vite preview + route interception); no backend, no credentials, no real provider, no create submission',
  captures: [],
  apiCalls: [],
  unknownEndpoints: [],
  consoleErrors: [],
  secretHits: 0,
  notes: [
    'phase1.5-skeleton-shimmer: ui-skeleton demo nodes were INJECTED into the page DOM (CSS verification only). The common Skeleton.vue component has no runtime consumer in the app; recorded as a Phase 1.5 risk.',
  ],
}

function scanSecrets(text) {
  for (const re of secretPatterns) {
    if (re.test(text || '')) evidence.secretHits += 1
  }
}

async function installMocks(page, role, item) {
  const user = role === 'admin' ? adminUser : employeeUser
  const previewSvg = await fs.readFile(
    path.resolve('D:/sub2api-trunk/docs/reviews/K3_APPLE_UI_PHASE1_20260717/assets/mock-remote-preview.svg')
  )
  await page.route('**/files.example.com/**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'image/svg+xml', body: previewSvg })
  })
  await page.route(/\/api\/v1\//, async (route) => {
    const req = route.request()
    const url = new URL(req.url())
    const key = `${req.method()} ${url.pathname.replace(/^\/api\/v1/, '')}`
    evidence.apiCalls.push({ capture: item.name, role, key })
    let payload
    if (key === 'GET /auth/me') payload = user
    else if (key === 'GET /settings/public') payload = publicSettings
    else if (key === 'GET /admin/compliance') payload = { required: false, accepted: true }
    else if (key === 'GET /announcements') payload = []
    else if (key === 'GET /subscriptions/active') payload = []
    else if (key === 'GET /admin/settings') payload = { ops_monitoring_enabled: true, ops_realtime_monitoring_enabled: true, ops_query_mode_default: 'auto', custom_menu_items: [] }
    else if (key === 'GET /admin/payment/config') payload = { enabled: false }
    else if (key === 'GET /admin/dashboard/snapshot-v2' && item.failDashboardStats) {
      await route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ code: 500, message: 'mock stats failure' }) })
      return
    }
    else if (key === 'GET /admin/dashboard/snapshot-v2') payload = { stats: null, trend: [], models: [] }
    else if (key === 'GET /admin/dashboard/users-trend') payload = { trend: [] }
    else if (key === 'GET /admin/dashboard/users-ranking') payload = { ranking: [] }
    else if (key === 'GET /admin/provider-billing/boss-conclusions') payload = { items: [{ provider: 'seedance', conclusion: 'not_uploaded' }] }
    else if (key === 'GET /admin/provider-billing/imports') payload = billingImports
    else if (key === 'GET /admin/provider-billing/period-summary') payload = billingPeriodSummary
    else if (key === 'GET /admin/video/dashboard') payload = item.emptyDashboard ? emptyVideoDashboard : { ...emptyVideoDashboard, provider_status: [{ ...mockProviderAccount, today_tasks: 6, running_tasks: 1, failed_tasks: 1, last_test_at: hoursAgo(3), updated_at: hoursAgo(1) }], today_tasks: 6, success_rate: 83.3, failed_tasks: 1, queued_tasks: 1, running_tasks: 1 }
    else if (key === 'GET /admin/video/providers') payload = { items: [mockProviderAccount] }
    else if (key === 'GET /video/providers') payload = { items: [mockProviderAccount], execution_capabilities: { mock: true, review_real: false, internal_real: false } }
    else if (key === 'GET /video/tasks') payload = item.emptyTasks ? { items: [], total: 0, page: 1, page_size: 20, pages: 0 } : { items: videoTasks, total: videoTasks.length, page: 1, page_size: 20, pages: 1 }
    else if (/^GET \/video\/tasks\/\d+$/.test(key)) payload = videoTasks[0]
    else {
      evidence.unknownEndpoints.push({ capture: item.name, role, key })
      payload = {}
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, message: 'ok', data: payload }),
    })
  })
  await page.route('**/setup/status**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, message: 'ok', data: { needs_setup: false, step: 'done' } }),
    })
  })
}

const matrix = [
  { name: 'phase1.5-video-tasks-progress-ring-light-1440', role: 'employee', route: '/admin/video/tasks', width: 1440, height: 1000, theme: 'light', fullPage: true },
  { name: 'phase1.5-video-tasks-progress-ring-light-390', role: 'employee', route: '/admin/video/tasks', width: 390, height: 844, theme: 'light', fullPage: false },
  { name: 'phase1.5-video-tasks-empty-light-1440', role: 'employee', route: '/admin/video/tasks', width: 1440, height: 1000, theme: 'light', fullPage: true, emptyTasks: true },
  { name: 'phase1.5-video-dashboard-empty-light-1440', role: 'admin', route: '/admin/video', width: 1440, height: 1000, theme: 'light', fullPage: true, emptyDashboard: true },
  { name: 'phase1.5-skeleton-shimmer-light-1440', role: 'admin', route: '/admin/video', width: 1440, height: 1000, theme: 'light', fullPage: false, injectSkeleton: true },
  { name: 'phase1.5-notfound-zh-light-1440', role: 'admin', route: '/k3-review-definitely-not-a-page', width: 1440, height: 1000, theme: 'light', fullPage: false },
  { name: 'phase1.5-provider-billing-zh-light-1440', role: 'admin', route: '/admin/provider-billing', width: 1440, height: 1000, theme: 'light', fullPage: true },
  { name: 'phase1.5-boss-dashboard-empty-light-1440', role: 'admin', route: '/admin/dashboard', width: 1440, height: 1000, theme: 'light', fullPage: true, failDashboardStats: true },
]

// Optional single-capture filter for re-runs against a different build (e.g.
// VITE_PRODUCT_MODE=standard for the operations-branch video dashboard).
const only = (process.env.K3_ONLY || '').trim()
const runMatrix = only ? matrix.filter((m) => m.name === only) : matrix
if (!runMatrix.length) throw new Error(`K3_ONLY=${only} matched no capture`)

await fs.mkdir(outputDir, { recursive: true })
const browser = await chromium.launch({ executablePath: edge, headless: true })

try {
  for (const item of runMatrix) {
    // Pre-capture checklist: role/route/viewport/theme/mock-mode fixed per entry;
    // no credentials anywhere (API fully mocked); no create will be submitted.
    const user = item.role === 'admin' ? adminUser : employeeUser
    const context = await browser.newContext({
      viewport: { width: item.width, height: item.height },
      locale: 'zh-CN',
      deviceScaleFactor: 1,
    })
    await context.addInitScript(
      ({ token, userJson, theme }) => {
        localStorage.setItem('auth_token', token)
        localStorage.setItem('auth_user', userJson)
        localStorage.setItem('theme', theme)
        localStorage.setItem('sub2api_locale', 'zh-CN')
        localStorage.setItem('admin_guide_1_admin_v4_interactive', 'true')
        localStorage.setItem('user_guide_2_user_v4_interactive', 'true')
      },
      { token: MOCK_TOKEN, userJson: JSON.stringify(user), theme: item.theme }
    )
    const page = await context.newPage()
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        const text = msg.text().slice(0, 300)
        scanSecrets(text)
        evidence.consoleErrors.push({ capture: item.name, text })
      }
    })
    await installMocks(page, item.role, item)

    await page.goto(`${baseURL}${item.route}`, { waitUntil: 'networkidle', timeout: 60000 })
    if (!item.route.startsWith('/k3-review-definitely')) {
      await page.waitForSelector('[data-testid="app-main"]', { timeout: 15000 }).catch(() => {})
    }
    await page.waitForTimeout(1500)

    const currentUrl = page.url()
    if (currentUrl.includes('/login')) {
      throw new Error(`capture ${item.name} redirected to login: ${currentUrl}`)
    }
    const html = await page.content()
    scanSecrets(html)

    const shot = path.join(outputDir, `${item.name}.png`)
    if (item.injectSkeleton) {
      // CSS verification ONLY: the common Skeleton.vue (ui-skeleton shimmer) has no
      // runtime consumer, so demo nodes are injected to photograph the shimmer style.
      const handle = await page.evaluateHandle(() => {
        const host = document.querySelector('[data-testid="app-main"]') || document.body
        const demo = document.createElement('section')
        demo.setAttribute('data-testid', 'k3-shimmer-demo')
        demo.style.cssText = 'margin:24px;padding:24px;border-radius:12px;background:var(--ui-surface,#fff);display:block;'
        demo.innerHTML = `
          <h2 style="margin:0 0 16px;font-size:16px;font-weight:600;color:var(--ui-text,#111)">shimmer 骨架样式验证（注入演示节点）</h2>
          <div style="display:grid;gap:12px;max-width:640px;">
            <div class="ui-skeleton rounded-lg" style="height:20px;width:40%;"></div>
            <div class="ui-skeleton rounded-lg" style="height:16px;width:100%;"></div>
            <div class="ui-skeleton rounded-lg" style="height:16px;width:75%;"></div>
            <div class="ui-skeleton rounded-lg" style="height:16px;width:83%;"></div>
            <div style="display:flex;gap:12px;align-items:center;">
              <div class="ui-skeleton rounded-full" style="height:48px;width:48px;"></div>
              <div style="flex:1;display:grid;gap:8px;">
                <div class="ui-skeleton rounded-lg" style="height:14px;width:60%;"></div>
                <div class="ui-skeleton rounded-lg" style="height:14px;width:35%;"></div>
              </div>
            </div>
          </div>`
        host.prepend(demo)
        return demo
      })
      await page.waitForTimeout(400)
      await handle.screenshot({ path: shot })
    } else {
      await page.screenshot({ path: shot, fullPage: item.fullPage })
    }
    evidence.captures.push({
      name: item.name,
      role: item.role,
      route: item.route,
      viewport: `${item.width}x${item.height}`,
      theme: item.theme,
      mock: true,
      injectedDemoNodes: Boolean(item.injectSkeleton),
      emptyFixture: Boolean(item.emptyTasks || item.emptyDashboard),
      secretsUsed: false,
      finalUrl: currentUrl,
      screenshot: `screenshots/${item.name}.png`,
    })
    await context.close()
  }
} finally {
  await browser.close()
}

evidence.unknownEndpoints = [...new Map(evidence.unknownEndpoints.map((e) => [e.key, e])).values()]
evidence.summary = {
  captures: evidence.captures.length,
  apiCalls: evidence.apiCalls.length,
  unknownEndpoints: evidence.unknownEndpoints.length,
  consoleErrors: evidence.consoleErrors.length,
  secretHits: evidence.secretHits,
}
// Merge with previous run's evidence so multi-build captures accumulate.
let merged = { captures: evidence.captures, apiCalls: evidence.apiCalls, consoleErrors: evidence.consoleErrors, unknownEndpoints: evidence.unknownEndpoints, secretHits: evidence.secretHits, runs: 1 }
try {
  const prev = JSON.parse(await fs.readFile(evidencePath, 'utf8'))
  const fresh = new Set(evidence.captures.map((c) => c.name))
  merged.captures = [...(prev.captures || []).filter((c) => !fresh.has(c.name)), ...evidence.captures]
  merged.apiCalls = [...(prev.apiCalls || []), ...evidence.apiCalls]
  merged.consoleErrors = [...(prev.consoleErrors || []), ...evidence.consoleErrors]
  const uk = new Map([...(prev.unknownEndpoints || []), ...evidence.unknownEndpoints].map((e) => [e.key, e]))
  merged.unknownEndpoints = [...uk.values()]
  merged.secretHits = (prev.secretHits || 0) + evidence.secretHits
  merged.runs = (prev.runs || 1) + 1
} catch { /* first run */ }
const out = {
  ...evidence,
  ...merged,
  summary: {
    captures: merged.captures.length,
    apiCalls: merged.apiCalls.length,
    unknownEndpoints: merged.unknownEndpoints.length,
    consoleErrors: merged.consoleErrors.length,
    secretHits: merged.secretHits,
    runs: merged.runs,
  },
}
await fs.writeFile(evidencePath, JSON.stringify(out, null, 2), 'utf8')
console.log(JSON.stringify(out.summary, null, 2))
if (evidence.secretHits > 0) process.exit(2)
if (evidence.captures.length !== runMatrix.length) process.exit(3)
