// Sub2 管理台壳大改 — before/after 截图与真人链路验收脚本
// 用法: ADMIN_PASSWORD=xxx node capture.mjs <outDir> [--flow keyvault|all]
// 1440x900，真登录、真点击。只读页面状态 + 表单打开，不提交会产生副作用的表单（除非 flow 指定）。
import { chromium } from 'playwright'
import fs from 'node:fs'
import path from 'node:path'

const BASE = process.env.BASE_URL || 'http://127.0.0.1:3000'
const ADMIN_EMAIL = process.env.ADMIN_EMAIL || 'admin@wujie.local'
const ADMIN_PASSWORD = process.env.ADMIN_PASSWORD
const outDir = process.argv[2]
if (!outDir) { console.error('usage: node capture.mjs <outDir>'); process.exit(1) }
if (!ADMIN_PASSWORD) { console.error('ADMIN_PASSWORD env required'); process.exit(1) }
fs.mkdirSync(outDir, { recursive: true })

const report = { steps: [], errors: [] }
const note = (name, ok, extra = '') => { report.steps.push({ name, ok, extra }); console.log(`${ok ? 'OK ' : 'FAIL'} ${name}${extra ? ' — ' + extra : ''}`) }

const browser = await chromium.launch()
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
// driver.js 引导遮罩一出现就移除（等效用户秒点跳过；不挡真人点击）
await page.addInitScript(() => {
  try {
    localStorage.setItem('admin_guide_1_admin_v5_wujie_operator', 'true')
    localStorage.setItem('onboarding_tour_1_admin_v5_wujie_operator', 'true')
  } catch {}
})
await page.addInitScript(() => {
  const style = document.createElement('style')
  style.textContent = '.driver-overlay,.driver-popover,#driver-dummy-element{display:none!important;pointer-events:none!important}'
  const attach = () => (document.head || document.documentElement).appendChild(style)
  if (document.head) attach()
  else new MutationObserver((_, obs) => { if (document.head) { attach(); obs.disconnect() } }).observe(document.documentElement, { childList: true })
})
page.on('pageerror', (e) => report.errors.push(`pageerror: ${e.message}`))
page.on('console', (m) => { if (m.type() === 'error') report.errors.push(`console: ${m.text()}`) })

async function shot(name) {
  await page.screenshot({ path: path.join(outDir, `${name}.png`) })
  note(`shot:${name}`, true)
}

try {
  // 1. 登录页
  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle' })
  await shot('01-login')

  // 2. 真人登录
  await page.fill('input[type="email"]', ADMIN_EMAIL)
  await page.fill('input[type="password"]', ADMIN_PASSWORD)
  await page.click('button[type="submit"]')
  await page.waitForURL((url) => !url.pathname.startsWith('/login'), { timeout: 15000 })
  await page.waitForLoadState('networkidle')
  note('login', true, page.url())

  // 3. 总览
  await page.goto(`${BASE}/admin/console/overview`, { waitUntil: 'networkidle' })
  await shot('02-overview')

  // 4. 密钥库：列表 + 录入弹窗
  await page.goto(`${BASE}/admin/console/key-vault`, { waitUntil: 'networkidle' })
  await shot('03-keyvault-list')
  await page.click('[data-test="open-create-account"]')
  await page.waitForSelector('[data-test="account-form"]', { state: 'visible' })
  await shot('04-keyvault-create-modal')
  // 不选分组直接提交，观察 before 行为（应能保存成功=bug 证据 / after 应被拦）
  await page.fill('[data-test="account-form"] input[placeholder*="Claude"]', 'K3探针-勿留')
  await page.fill('[data-test="account-api-key"]', 'sk-probe-do-not-use-0000')
  await shot('04b-keyvault-create-nogroup-filled')
  // before 阶段点保存会真创建；为不污染环境，仅点击并随后若创建成功则立即删除
  await page.click('[data-test="save-account"]')
  await page.waitForTimeout(2500)
  await shot('04c-keyvault-after-save-nogroup')
  // 清理探针账号（若被创建）
  const probeRow = page.locator('tr', { hasText: 'K3探针-勿留' }).first()
  if (await probeRow.count()) {
    await probeRow.locator('button').last().click() // 删除按钮
    await page.waitForTimeout(600)
    const confirmBtn = page.locator('button:has-text("确认"), button:has-text("删除"), button:has-text("确定")').last()
    if (await confirmBtn.count()) await confirmBtn.click()
    await page.waitForTimeout(1500)
    note('probe-account-cleanup', true)
  } else {
    note('probe-account-cleanup', true, 'no probe row found (nothing created)')
  }

  // 5. 视频 tab
  await page.goto(`${BASE}/admin/console/key-vault?tab=video`, { waitUntil: 'networkidle' })
  await shot('05-keyvault-video-tab')

  // 6. 员工卡
  await page.goto(`${BASE}/admin/console/staff`, { waitUntil: 'networkidle' })
  await shot('06-staff')

  // 7. 视频通道
  await page.goto(`${BASE}/admin/video/providers`, { waitUntil: 'networkidle' })
  await shot('07-video-providers')

  // 8. 侧栏特写（含长名 + 版本号）
  const sidebar = page.locator('aside, nav').first()
  await page.goto(`${BASE}/admin/console/overview`, { waitUntil: 'networkidle' })
  if (await sidebar.count()) {
    await sidebar.screenshot({ path: path.join(outDir, '08-sidebar.png') })
    note('shot:08-sidebar', true)
  }

  // 9. 版本弹窗（点侧栏版本号区域）
  const versionBtn = page.locator('text=/v?[0-9a-f]{6,}/').last()
  if (await versionBtn.count()) {
    await versionBtn.click().catch(() => {})
    await page.waitForTimeout(800)
    await shot('09-version-popover')
  }
} catch (e) {
  note('fatal', false, e.message)
  await page.screenshot({ path: path.join(outDir, 'zz-fatal.png') }).catch(() => {})
} finally {
  fs.writeFileSync(path.join(outDir, '_report.json'), JSON.stringify(report, null, 2))
  await browser.close()
}
