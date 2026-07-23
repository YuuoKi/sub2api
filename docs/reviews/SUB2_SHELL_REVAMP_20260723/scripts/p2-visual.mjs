// P2 验收：Auth 去光斑/渐变、总览去环/饼、密钥库/员工卡收敛
import { chromium } from 'playwright'
import fs from 'node:fs'
import path from 'node:path'

const BASE = process.env.BASE_URL || 'http://127.0.0.1:3001'
const outDir = process.argv[2]
fs.mkdirSync(outDir, { recursive: true })
const report = []
const note = (name, ok, extra = '') => { report.push({ name, ok, extra }); console.log(`${ok ? 'OK  ' : 'FAIL'} ${name}${extra ? ' — ' + extra : ''}`) }

const browser = await chromium.launch()
const page = await browser.newPage({ viewport: { width: 1440, height: 900 }, locale: 'zh-CN' })
await page.addInitScript(() => {
  try {
    localStorage.setItem('admin_guide_1_admin_v5_wujie_operator', 'true')
    localStorage.setItem('onboarding_tour_1_admin_v5_wujie_operator', 'true')
  } catch {}
})
const shot = (n) => page.screenshot({ path: path.join(outDir, `${n}.png`) })

try {
  // 1. 登录页：无 blur-3xl 光斑、无渐变标题、「登录」而非「欢迎回来」
  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle' })
  const blurCount = await page.locator('.blur-3xl').count()
  note('auth-no-blob', blurCount === 0, `blur-3xl=${blurCount}`)
  const gradTitle = await page.locator('h1.bg-gradient-to-r, h1[class*="bg-clip-text"]').count()
  note('auth-no-gradient-title', gradTitle === 0)
  const bodyText0 = await page.locator('body').innerText()
  note('auth-title-login', bodyText0.includes('登录') && !bodyText0.includes('欢迎回来'))
  await shot('p2-01-login-clean')

  await page.fill('input[type="email"]', process.env.ADMIN_EMAIL || 'admin@wujie.local')
  await page.fill('input[type="password"]', process.env.ADMIN_PASSWORD)
  await page.click('button[type="submit"]')
  await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 15000 })

  // 2. 总览：无圆环 svg、无饼图 canvas、副标改「消费与调用概览」、异常上游卡存在
  await page.goto(`${BASE}/admin/console/overview`, { waitUntil: 'networkidle' })
  await page.waitForTimeout(1200)
  const ringSvg = await page.locator('section svg[viewBox="0 0 48 48"]').count()
  note('overview-no-rings', ringSvg === 0, `rings=${ringSvg}`)
  const doughnutCanvas = await page.locator('section canvas').count()
  // 趋势线图 canvas 保留 1 个；饼图没了 → canvas ≤1
  note('overview-no-doughnut', doughnutCanvas <= 1, `canvas=${doughnutCanvas}`)
  const ovText = await page.locator('body').innerText()
  note('overview-subtitle', ovText.includes('消费与调用概览') && !ovText.includes('一眼看完'))
  note('overview-error-upstream-card', ovText.includes('异常上游'))
  await shot('p2-02-overview-clean')

  // 3. 密钥库：无口语文案、平台列 muted 文本无底色 badge
  await page.goto(`${BASE}/admin/console/key-vault`, { waitUntil: 'networkidle' })
  const kvText = await page.locator('body').innerText()
  note('keyvault-no-colloquial', !kvText.includes('把老板手上的密钥统一收进来'))
  const colorBadge = await page.locator('tbody span.bg-orange-50, tbody span.bg-blue-50, tbody span.bg-violet-50').count()
  note('keyvault-platform-muted', colorBadge === 0, `colorBadges=${colorBadge}`)
  await shot('p2-03-keyvault-clean')

  // 4. 员工卡：无 sky/violet 类型 badge
  await page.goto(`${BASE}/admin/console/staff`, { waitUntil: 'networkidle' })
  const staffBadges = await page.locator('tbody span.bg-sky-100, tbody span.bg-violet-100').count()
  note('staff-no-type-badges', staffBadges === 0, `typeBadges=${staffBadges}`)
  await shot('p2-04-staff-clean')
} catch (e) {
  note('fatal', false, e.message)
  await shot('p2-zz-fatal')
} finally {
  fs.writeFileSync(path.join(outDir, '_p2-report.json'), JSON.stringify(report, null, 2))
  await browser.close()
}
