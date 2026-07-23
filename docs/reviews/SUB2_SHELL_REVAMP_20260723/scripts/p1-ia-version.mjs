// P1 验收：侧栏 5 项短词无截断 + 版本短显 + lan 模式无更新提示/不请求 check-updates
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
const checkUpdateRequests = []
page.on('request', (req) => { if (req.url().includes('check-updates')) checkUpdateRequests.push(req.url()) })

try {
  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle' })
  await page.fill('input[type="email"]', process.env.ADMIN_EMAIL || 'admin@wujie.local')
  await page.fill('input[type="password"]', process.env.ADMIN_PASSWORD)
  await page.click('button[type="submit"]')
  await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 15000 })
  await page.waitForLoadState('networkidle')

  // 1. 侧栏五项短词
  const sidebar = page.locator('aside').first()
  const sideText = await sidebar.innerText()
  for (const label of ['总览', '密钥库', '员工卡', '用量', '系统']) {
    note(`sidebar-has-${label}`, sideText.includes(label))
  }
  for (const old of ['总览与成本', '上游账号、模型和通道', '员工/API 卡片管理', '调用、任务与资产记录', '系统、健康、备份与']) {
    note(`sidebar-no-old-${old.slice(0, 6)}`, !sideText.includes(old))
  }
  note('sidebar-no-ellipsis', !sideText.includes('…'))
  await shot('p1-01-sidebar-five-items')

  // 2. 展开 系统 → 高级
  await sidebar.locator('text=系统').first().click()
  await page.waitForTimeout(500)
  await sidebar.locator('text=高级').first().click()
  await page.waitForTimeout(500)
  const expandedText = await sidebar.innerText()
  note('system-children', expandedText.includes('系统健康') && expandedText.includes('设置与备份') && expandedText.includes('高级'), expandedText.replace(/\s+/g, ' ').slice(0, 200))
  note('advanced-children', ['上游账号', '模型分组', '视频通道', '任务记录', '用量与成本'].every((x) => expandedText.includes(x)))
  await shot('p1-02-system-advanced-expanded')

  // 3. 版本 badge：短显 + 无琥珀 ping
  const badge = sidebar.locator('div.relative').first()
  const badgeText = (await badge.innerText()).trim()
  note('badge-short-version', !/[0-9a-f]{12,}/i.test(badgeText), badgeText)
  const pingCount = await sidebar.locator('.animate-ping').count()
  note('no-amber-ping', pingCount === 0, `ping=${pingCount}`)

  // 4. 弹窗：无「有新版本可用/立即更新」
  await sidebar.locator('button:has-text("v0.")').first().click()
  await page.waitForTimeout(1200)
  const popText = await page.locator('body').innerText()
  note('no-update-prompt', !/有新版本可用|立即更新|Update Now|new version is available/i.test(popText))
  note('popover-shows-current-version', popText.includes('当前版本'))
  await shot('p1-03-version-popover-clean')

  // 5. lan 模式不请求 check-updates
  note('no-check-updates-request', checkUpdateRequests.length === 0, `${checkUpdateRequests.length} hits`)
} catch (e) {
  note('fatal', false, e.message)
  await shot('p1-zz-fatal')
} finally {
  fs.writeFileSync(path.join(outDir, '_p1-report.json'), JSON.stringify(report, null, 2))
  await browser.close()
}
