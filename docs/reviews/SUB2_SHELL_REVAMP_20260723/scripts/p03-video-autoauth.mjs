// P0.3 验收：视频通道默认启用 + 保存后自动授权 + 「待消费」人话（scratch 后端 8088，探针通道残留可接受）
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

try {
  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle' })
  await page.fill('input[type="email"]', process.env.ADMIN_EMAIL || 'admin@wujie.local')
  await page.fill('input[type="password"]', process.env.ADMIN_PASSWORD)
  await page.click('button[type="submit"]')
  await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 15000 })
  await page.evaluate(() => {
    Object.keys(localStorage).filter((k) => k.startsWith('onboarding_tour')).forEach((k) => localStorage.setItem(k, 'true'))
  })

  await page.goto(`${BASE}/admin/video/providers`, { waitUntil: 'networkidle' })

  // 默认值：保存后启用 ✓、自动授权 ✓（before 为不勾选，老板会漏）
  const enabledChecked = await page.locator('.video-provider-enabled input[type="checkbox"]').isChecked()
  const authChecked = await page.locator('[data-test="authorize-after-save"] input[type="checkbox"]').isChecked()
  note('default-enabled-checked', enabledChecked)
  note('default-authorize-checked', authChecked)
  await shot('p03-01-form-defaults')

  // 真人填写：名称 + 组 + Key → 保存（不点任何授权按钮）
  await page.fill('#video-provider-name', 'K3验收通道')
  const groupOptions = await page.locator('#video-provider-group option').evaluateAll((els) => els.map((e) => e.value).filter((v) => v !== '0'))
  await page.selectOption('#video-provider-group', groupOptions[0])
  await page.fill('#video-provider-secret', 'k3-acceptance-fake-video-key-0000')
  await shot('p03-02-filled')
  await page.click('button:has-text("保存")')
  await page.waitForTimeout(4000)

  const bodyText = await page.locator('body').innerText()
  note('auto-authorize-toast', bodyText.includes('通道已保存并记录单次授权'), bodyText.match(/通道已保存[^。\n]*/)?.[0] || 'no-toast')
  const card = page.locator('article', { hasText: 'K3验收通道' }).first()
  const cardText = (await card.count()) ? await card.innerText() : ''
  note('provider-card-exists', cardText.length > 0)
  note('pending-consume-human-copy', cardText.includes('待消费 = 等第一次真实出片'), cardText.match(/待消费[^\n]*/)?.[0] || cardText.slice(0, 120))
  note('card-enabled', cardText.includes('已启用'))
  await shot('p03-03-after-save-auto-authorized')

  // 授权按钮此时应因「已有待消费授权」禁用，避免重复授权
  const authBtn = card.locator('button:has-text("授权一次最小真实调用")')
  note('reauthorize-disabled', await authBtn.isDisabled())
} catch (e) {
  note('fatal', false, e.message)
  await shot('p03-zz-fatal')
} finally {
  fs.writeFileSync(path.join(outDir, '_p03-report.json'), JSON.stringify(report, null, 2))
  await browser.close()
}
