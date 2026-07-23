// P2 补拍：8080 有数据环境下的总览单色占比条（对照 before/02-overview 的饼图+圆环）
import { chromium } from 'playwright'
import path from 'node:path'
import fs from 'node:fs'

const BASE = 'http://127.0.0.1:3000'
const outDir = process.argv[2]
fs.mkdirSync(outDir, { recursive: true })
const browser = await chromium.launch()
const page = await browser.newPage({ viewport: { width: 1440, height: 900 }, locale: 'zh-CN' })
await page.addInitScript(() => {
  try {
    localStorage.setItem('admin_guide_1_admin_v5_wujie_operator', 'true')
    localStorage.setItem('onboarding_tour_1_admin_v5_wujie_operator', 'true')
  } catch {}
})
await page.goto(`${BASE}/login`, { waitUntil: 'networkidle' })
await page.fill('input[type="email"]', process.env.ADMIN_EMAIL || 'admin@wujie.local')
await page.fill('input[type="password"]', process.env.ADMIN_PASSWORD)
await page.click('button[type="submit"]')
await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 15000 })
await page.goto(`${BASE}/admin/console/overview`, { waitUntil: 'networkidle' })
await page.waitForTimeout(1500)
await page.screenshot({ path: path.join(outDir, 'p2-02b-overview-with-data.png') })
const hasBar = await page.locator('.bg-ui-accent').count()
const text = await page.locator('body').innerText()
console.log('ui-accent bars:', hasBar, '| 异常上游:', text.includes('异常上游'), '| gemini row:', text.includes('gemini-2.5-flash'))
await browser.close()
