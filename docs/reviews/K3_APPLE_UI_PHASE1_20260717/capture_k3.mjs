// K3 Apple-like UI Phase 1 — visual matrix capture (Task 5, Step 3).
// Fully mock: no backend, no credentials, no real Provider, no create submission.
// Serves the production build via `vite preview` and intercepts every API call
// with local fixtures. Evidence is written to capture-evidence.json.
import { createRequire } from 'node:module'
import fs from 'node:fs/promises'
import path from 'node:path'

const require = createRequire(import.meta.url)
const { chromium } = require('C:/tmp/sub2api-playwright/node_modules/playwright-core')

const baseURL = process.env.K3_BASE_URL || 'http://127.0.0.1:5199'
const outputDir = path.resolve('D:/sub2api-trunk/docs/reviews/K3_APPLE_UI_PHASE1_20260717/screenshots')
const evidencePath = path.resolve('D:/sub2api-trunk/docs/reviews/K3_APPLE_UI_PHASE1_20260717/capture-evidence.json')
const edge = 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe'

// Clearly-fake local session marker (not a credential; API is fully mocked).
const MOCK_TOKEN = 'k3-review-mock-session'

const NOW = '2026-07-17T01:55:00+08:00'
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

const trendPoint = (h, requests, tokens, cost) => ({
  date: hoursAgo(h),
  requests,
  input_tokens: Math.round(tokens * 0.42),
  output_tokens: Math.round(tokens * 0.5),
  cache_creation_tokens: Math.round(tokens * 0.03),
  cache_read_tokens: Math.round(tokens * 0.05),
  total_tokens: tokens,
  cost,
  actual_cost: Math.round(cost * 0.93 * 100) / 100,
})

const snapshotV2 = {
  generated_at: NOW,
  start_date: hoursAgo(24),
  end_date: NOW,
  granularity: 'hour',
  stats: {
    total_users: 128,
    today_new_users: 3,
    active_users: 42,
    hourly_active_users: 9,
    stats_updated_at: NOW,
    stats_stale: false,
    total_api_keys: 86,
    active_api_keys: 74,
    total_accounts: 12,
    normal_accounts: 10,
    error_accounts: 1,
    ratelimit_accounts: 1,
    overload_accounts: 0,
    total_requests: 1284502,
    total_input_tokens: 52100000,
    total_output_tokens: 63800000,
    total_cache_creation_tokens: 3100000,
    total_cache_read_tokens: 6200000,
    total_tokens: 125200000,
    total_cost: 12480.55,
    total_actual_cost: 11902.31,
    total_account_cost: 10311.4,
    usd_cny_rate: 7.2,
    today_requests: 18432,
    today_input_tokens: 3880000,
    today_output_tokens: 4760000,
    today_cache_creation_tokens: 210000,
    today_cache_read_tokens: 460000,
    today_tokens: 9310000,
    today_cost: 462.13,
    today_actual_cost: 431.88,
    today_account_cost: 388.2,
    average_duration_ms: 842,
    uptime: 1234567,
    rpm: 14,
    tpm: 68400,
  },
  trend: Array.from({ length: 24 }, (_, i) =>
    trendPoint(23 - i, 520 + ((i * 137) % 380), 260000 + ((i * 97777) % 220000), 12.4 + ((i * 37) % 90) / 10)
  ),
  models: [
    { model: 'claude-sonnet-4-5', requests: 9210, input_tokens: 1980000, output_tokens: 2410000, cache_creation_tokens: 98000, cache_read_tokens: 210000, total_tokens: 4698000, cost: 236.4, actual_cost: 219.6, account_cost: 198.2 },
    { model: 'gpt-4o', requests: 5830, input_tokens: 1210000, output_tokens: 1490000, cache_creation_tokens: 64000, cache_read_tokens: 150000, total_tokens: 2914000, cost: 141.2, actual_cost: 132.9, account_cost: 118.4 },
    { model: 'deepseek-v3.2', requests: 3392, input_tokens: 690000, output_tokens: 860000, cache_creation_tokens: 48000, cache_read_tokens: 100000, total_tokens: 1698000, cost: 84.53, actual_cost: 79.38, account_cost: 71.6 },
  ],
}

const usersTrend = {
  trend: Array.from({ length: 12 }, (_, i) => ({
    date: hoursAgo(23 - i * 2),
    user_id: 9001 + (i % 4),
    email: `user${(i % 4) + 1}@sub2api.local`,
    username: `user${(i % 4) + 1}`,
    requests: 120 + ((i * 53) % 160),
    tokens: 98000 + ((i * 31337) % 240000),
    cost: 4.2 + ((i * 17) % 60) / 10,
    actual_cost: 3.9 + ((i * 13) % 55) / 10,
  })),
  start_date: hoursAgo(24),
  end_date: NOW,
  granularity: 'hour',
}

const usersRanking = {
  ranking: [
    { user_id: 9001, email: 'employee-g6@sub2api.local', actual_cost: 96.4, requests: 4120, tokens: 2080000 },
    { user_id: 9002, email: 'user2@sub2api.local', actual_cost: 81.2, requests: 3610, tokens: 1740000 },
    { user_id: 9003, email: 'user3@sub2api.local', actual_cost: 66.9, requests: 2980, tokens: 1420000 },
    { user_id: 9004, email: 'user4@sub2api.local', actual_cost: 44.1, requests: 2010, tokens: 980000 },
    { user_id: 9005, email: 'user5@sub2api.local', actual_cost: 28.7, requests: 1320, tokens: 660000 },
  ],
  total_actual_cost: 431.88,
  total_requests: 18432,
  total_tokens: 9310000,
  start_date: hoursAgo(24),
  end_date: NOW,
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

const videoTasks = [
  task({ id: 5006, prompt: '【Mock】城市夜景航拍，霓虹倒影在雨后街道', status: 'queued', delivery_status: 'processing', next_action: 'poll', result_url: '', completed_at: null, created_at: hoursAgo(0.1), updated_at: hoursAgo(0.1) }),
  task({ id: 5005, prompt: '【Mock】产品宣传片：茶杯在晨光中旋转', status: 'running', delivery_status: 'processing', next_action: 'poll', completed_at: null, created_at: hoursAgo(0.4), updated_at: hoursAgo(0.2) }),
  task({ id: 5004, prompt: '【Mock】山间云海延时摄影', status: 'failed', delivery_status: 'delivery_failed', next_action: 'review_delivery', error_message: '上游返回超时（mock 模拟失败样例）', completed_at: hoursAgo(3), created_at: hoursAgo(3.5), updated_at: hoursAgo(3) }),
  task({ id: 5003, prompt: '【Mock】办公室场景：团队协作白板讨论', status: 'succeeded', delivery_status: 'deliverable', next_action: 'download', local_asset_available: true, local_asset_path: '/data/video_assets/mock/5003.mp4', local_asset_saved_at: hoursAgo(5), result_url: 'https://files.example.com/mock/5003.mp4', created_at: hoursAgo(6), updated_at: hoursAgo(5), completed_at: hoursAgo(5) }),
  task({ id: 5002, prompt: '【Mock】海边日落，浪花在礁石上破碎', status: 'succeeded', delivery_status: 'deliverable', next_action: 'download', result_url: 'https://files.example.com/mock/5002-cover.svg?expires=demo', result_url_expires_at: hoursAgo(-20), result_url_expiry_source: 'url_query', created_at: hoursAgo(9), updated_at: hoursAgo(8), completed_at: hoursAgo(8) }),
  task({ id: 5001, prompt: '【Mock】片头动画：几何图形渐变组合', status: 'succeeded', delivery_status: 'deliverable', next_action: 'download', local_asset_available: true, local_asset_path: '/data/video_assets/mock/5001.mp4', local_asset_saved_at: hoursAgo(12), created_at: hoursAgo(14), updated_at: hoursAgo(12), completed_at: hoursAgo(12) }),
]

const videoDashboard = {
  today_tasks: 6,
  success_rate: 83.3,
  failed_tasks: 1,
  queued_tasks: 1,
  running_tasks: 1,
  provider_status: [
    {
      provider: 'mock',
      display_name: 'Mock 试跑通道',
      enabled: true,
      api_key_configured: true,
      masked_key: 'mock-****',
      default_model: 'mock-video-1',
      key_status: 'valid',
      health_status: 'healthy',
      diagnostic_type: '',
      suggested_action: '',
      route_available: true,
      route_skip_reason: '',
      priority: 1,
      current_inflight: 1,
      last_error: '',
      last_test_at: hoursAgo(3),
      updated_at: hoursAgo(1),
      today_tasks: 6,
      running_tasks: 1,
      failed_tasks: 1,
    },
  ],
  health_diagnostics: [],
  recent_failures: [videoTasks[2]],
  recent_successes: [videoTasks[3], videoTasks[4]],
  usage_overview: [
    { provider: 'mock', model: 'mock-video-1', status: 'succeeded', count: 3, cost_estimate: 0, duration: 18, currency: 'USD', pricing_source: 'mock', pricing_version: 'v1' },
    { provider: 'mock', model: 'mock-video-1', status: 'failed', count: 1, cost_estimate: 0, duration: 0, currency: 'USD', pricing_source: 'mock', pricing_version: 'v1' },
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

const fixtures = {
  'GET /auth/me': () => null, // special-cased per role below
  'GET /settings/public': () => publicSettings,
  'GET /admin/compliance': () => ({ required: false, accepted: true }),
  'GET /announcements': () => [],
  'GET /subscriptions/active': () => [],
  'GET /admin/settings': () => ({ ops_monitoring_enabled: true, ops_realtime_monitoring_enabled: true, ops_query_mode_default: 'auto', custom_menu_items: [] }),
  'GET /admin/payment/config': () => ({ enabled: false }),
  'GET /admin/dashboard/snapshot-v2': () => snapshotV2,
  'GET /admin/dashboard/users-trend': () => usersTrend,
  'GET /admin/dashboard/users-ranking': () => usersRanking,
  'GET /admin/provider-billing/boss-conclusions': () => ({ items: [{ provider: 'seedance', conclusion: 'not_uploaded' }] }),
  'GET /admin/video/dashboard': () => videoDashboard,
  'GET /admin/video/providers': () => ({ items: [mockProviderAccount] }),
  'GET /video/providers': () => ({ items: [mockProviderAccount], execution_capabilities: { mock: true, review_real: false, internal_real: false } }),
  'GET /video/tasks': () => ({ items: videoTasks, total: videoTasks.length, page: 1, page_size: 20, pages: 1 }),
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
}

function scanSecrets(text) {
  for (const re of secretPatterns) {
    if (re.test(text || '')) evidence.secretHits += 1
  }
}

async function installMocks(page, role) {
  const user = role === 'admin' ? adminUser : employeeUser
  // Mock CDN: the remote-deliverable preview fixture is served locally so no
  // outbound network call is ever attempted.
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
    evidence.apiCalls.push({ role, key })
    let payload
    if (key === 'GET /auth/me') payload = user
    else if (fixtures[key]) payload = fixtures[key]()
    else if (/^GET \/video\/tasks\/\d+$/.test(key)) payload = videoTasks[0]
    else {
      evidence.unknownEndpoints.push({ role, key })
      payload = Array.isArray(fixtures[key]) ? [] : {}
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
  { name: 'boss-dashboard-light-1440', role: 'admin', route: '/admin/dashboard', width: 1440, height: 1000, theme: 'light', fullPage: true },
  { name: 'boss-dashboard-dark-1440', role: 'admin', route: '/admin/dashboard', width: 1440, height: 1000, theme: 'dark', fullPage: true },
  { name: 'admin-video-dashboard-light-1440', role: 'admin', route: '/admin/video', width: 1440, height: 1000, theme: 'light', fullPage: true },
  { name: 'employee-video-create-light-1440', role: 'employee', route: '/admin/video/create', width: 1440, height: 1000, theme: 'light', fullPage: true },
  { name: 'employee-video-tasks-light-1440', role: 'employee', route: '/admin/video/tasks', width: 1440, height: 1000, theme: 'light', fullPage: true },
  { name: 'employee-video-tasks-dark-1440', role: 'employee', route: '/admin/video/tasks', width: 1440, height: 1000, theme: 'dark', fullPage: true },
  { name: 'employee-video-tasks-light-390', role: 'employee', route: '/admin/video/tasks', width: 390, height: 844, theme: 'light', fullPage: false },
  { name: 'employee-video-tasks-dark-390', role: 'employee', route: '/admin/video/tasks', width: 390, height: 844, theme: 'dark', fullPage: false },
]

await fs.mkdir(outputDir, { recursive: true })
const browser = await chromium.launch({ executablePath: edge, headless: true })

try {
  for (const item of matrix) {
    // Pre-capture checklist: role/route/viewport/theme/mock-mode are fixed per matrix entry;
    // no credentials are used anywhere (API fully mocked); no create will be submitted.
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
    await installMocks(page, item.role)

    await page.goto(`${baseURL}${item.route}`, { waitUntil: 'networkidle', timeout: 60000 })
    await page.waitForSelector('[data-testid="app-main"]', { timeout: 15000 }).catch(() => {})
    await page.waitForTimeout(1500)

    // Sanity assertions before capture: correct route, no redirect to /login.
    const currentUrl = page.url()
    if (currentUrl.includes('/login')) {
      throw new Error(`capture ${item.name} redirected to login: ${currentUrl}`)
    }
    const html = await page.content()
    scanSecrets(html)

    const shot = path.join(outputDir, `${item.name}.png`)
    await page.screenshot({ path: shot, fullPage: item.fullPage })
    evidence.captures.push({
      name: item.name,
      role: item.role,
      route: item.route,
      viewport: `${item.width}x${item.height}`,
      theme: item.theme,
      mock: true,
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
await fs.writeFile(evidencePath, JSON.stringify(evidence, null, 2), 'utf8')
console.log(JSON.stringify(evidence.summary, null, 2))
if (evidence.secretHits > 0) process.exit(2)
if (evidence.captures.length !== matrix.length) process.exit(3)
