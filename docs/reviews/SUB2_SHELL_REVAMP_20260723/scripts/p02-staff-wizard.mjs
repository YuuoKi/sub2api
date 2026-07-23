// P0.2 验收：员工一页开卡向导（建身份 → 双 Key + 充值 → 明文一次）+ 行内充值/查看花费
import { chromium } from 'playwright'
import fs from 'node:fs'
import path from 'node:path'

const BASE = process.env.BASE_URL || 'http://127.0.0.1:3000'
const API = process.env.API_BASE || 'http://127.0.0.1:8080'
const outDir = process.argv[2]
fs.mkdirSync(outDir, { recursive: true })
const report = []
const note = (name, ok, extra = '') => { report.push({ name, ok, extra }); console.log(`${ok ? 'OK  ' : 'FAIL'} ${name}${extra ? ' — ' + extra : ''}`) }

const browser = await chromium.launch()
const page = await browser.newPage({ viewport: { width: 1440, height: 900 }, locale: 'zh-CN' })
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
const shot = (n) => page.screenshot({ path: path.join(outDir, `${n}.png`) })
async function dismissTour() {
  await page.keyboard.press('Escape').catch(() => {})
  const skip = page.locator('button:has-text("Skip guide"), button:has-text("跳过引导")').first()
  if (await skip.count()) await skip.click().catch(() => {})
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(200)
}

const STAFF_EMAIL = 'k3-accept@wujie.local'
const STAFF_NAME = 'K3验收员工'
let createdUserId = null

try {
  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle' })
  await page.fill('input[type="email"]', process.env.ADMIN_EMAIL || 'admin@wujie.local')
  await page.fill('input[type="password"]', process.env.ADMIN_PASSWORD)
  await page.click('button[type="submit"]')
  await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 15000 })
  // 关掉 driver.js 新手引导（否则遮罩拦截真实点击）；等效于用户点「跳过引导」
  await page.evaluate(() => {
    Object.keys(localStorage)
      .filter((k) => k.startsWith('onboarding_tour'))
      .forEach((k) => localStorage.setItem(k, 'true'))
    localStorage.setItem('onboarding_tour_1_admin_v5_wujie_operator', 'true')
  })

  await page.goto(`${BASE}/admin/console/staff`, { waitUntil: 'networkidle' })
  await dismissTour()
  await shot('p02-00-staff-before')

  // 若上次残留先清掉（界面里不可见也无妨，后面按 API 清）
  await page.click('[data-test="create-service-identity"]')
  await page.waitForSelector('[data-test="service-identity-form"]')
  await page.fill('input[placeholder="例如：QCanvas 批量出图"]', STAFF_NAME)
  await page.fill('[data-test="service-identity-email"]', STAFF_EMAIL)
  await shot('p02-01-wizard-step1')
  await page.click('[data-test="wizard-step1-next"]')
  console.log('DEBUG clicked step1-next at', new Date().toISOString())
  const createResp = await page.waitForResponse((r) => r.url().includes('/api/v1/admin/users') && r.request().method() === 'POST', { timeout: 20000 }).catch(() => null)
  console.log('DEBUG users.create status:', createResp ? createResp.status() : 'none', createResp ? (await createResp.text()).slice(0, 200) : '')
  await page.waitForSelector('[data-test="wizard-pair-form"]', { timeout: 30000 })
  note('wizard-step1-creates-identity', true)

  // 第 2 步：选两个不同组 + 充值金额
  const videoSelect = page.locator('[data-test="wizard-video-group"]')
  const mediaSelect = page.locator('[data-test="wizard-media-group"]')
  const groupOptions = await videoSelect.locator('option').allTextContents()
  note('wizard-has-2plus-groups', groupOptions.length >= 3, groupOptions.join('/'))
  // 取前两个可选组 id
  const ids = await videoSelect.locator('option').evaluateAll((els) => els.map((e) => e.value).filter((v) => v !== '0'))
  await videoSelect.selectOption(ids[0])
  await page.waitForTimeout(200)
  // 同组在 media 选择器被禁用
  const mediaDisabled = await mediaSelect.locator(`option[value="${ids[0]}"]`).getAttribute('disabled')
  note('same-group-disabled-in-media-selector', mediaDisabled !== null)
  await mediaSelect.selectOption(ids[1])
  await page.fill('[data-test="wizard-amount"]', '12.34')
  await shot('p02-02-wizard-step2')
  await dismissTour()
  await page.click('[data-test="wizard-submit"]')
  await page.waitForTimeout(4000)
  const toastText = await page.locator('.fixed.top-4, [role="alert"], .toast').allTextContents().catch(() => [])
  console.log('DEBUG toasts:', JSON.stringify(toastText).slice(0, 400))
  console.log('DEBUG page text snippet:', (await page.locator('body').innerText()).replace(/\s+/g, ' ').slice(0, 500))
  await shot('p02-02b-after-submit-debug')
  await page.waitForSelector('[data-test="wizard-video-key"]', { timeout: 20000 })
  note('wizard-step2-issues-pair', true)

  // 第 3 步：双 Key 明文 + 充值结果
  const videoKey = await page.locator('[data-test="wizard-video-key"]').innerText()
  const mediaKey = await page.locator('[data-test="wizard-media-key"]').innerText()
  note('plaintext-dual-keys-shown', videoKey.startsWith('sk-') && mediaKey.startsWith('sk-'), `${videoKey.slice(0, 6)}…/${mediaKey.slice(0, 6)}…`)
  const rechargeText = await page.locator('[data-test="wizard-recharge-result"]').innerText()
  note('recharge-result-shown', rechargeText.includes('12.34'), rechargeText)
  await shot('p02-03-wizard-step3-keys')
  await page.click('[data-test="wizard-done"]')
  await page.waitForTimeout(1500)

  // 列表出现新身份，行内有 充值/查看花费
  const row = page.locator('tr', { hasText: STAFF_NAME }).first()
  note('staff-row-appears', (await row.count()) > 0)
  const spendLink = row.locator('[data-test="row-view-spend"]')
  const href = await spendLink.getAttribute('href')
  note('view-spend-deeplink', !!href && href.includes('user_id='), href || '')
  await shot('p02-04-staff-row-actions')

  // 行内充值再走一笔
  await dismissTour()
  await row.locator('[data-test="row-recharge"]').click()
  await page.waitForSelector('[data-test="recharge-form"]')
  await page.fill('[data-test="recharge-amount"]', '1')
  await shot('p02-05-row-recharge-modal')
  await page.click('[data-test="recharge-submit"]')
  await page.waitForTimeout(2000)
  const toast = await page.locator('text=已充值').count()
  note('row-recharge-works', toast > 0)
  await shot('p02-06-row-recharge-done')

  // 取 user id 便于清理
  createdUserId = href ? Number(new URLSearchParams(href.split('?')[1]).get('user_id')) : null
} catch (e) {
  note('fatal', false, e.message)
  await shot('p02-zz-fatal')
} finally {
  fs.writeFileSync(path.join(outDir, '_p02-report.json'), JSON.stringify(report, null, 2))
  await browser.close()
  // 清理：删除验收身份（级联清卡）
  if (createdUserId) {
    const loginRes = await fetch(`${API}/api/v1/auth/login`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: process.env.ADMIN_EMAIL || 'admin@wujie.local', password: process.env.ADMIN_PASSWORD }),
    }).then((r) => r.json())
    const token = loginRes.data.access_token
    const del = await fetch(`${API}/api/v1/admin/users/${createdUserId}`, { method: 'DELETE', headers: { Authorization: `Bearer ${token}` } })
    console.log(`cleanup user ${createdUserId}: HTTP ${del.status}`)
  }
}
