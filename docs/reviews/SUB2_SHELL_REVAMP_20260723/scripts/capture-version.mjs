// 补拍：版本弹窗 before（点侧栏顶部版本 badge）
import { chromium } from 'playwright'
import path from 'node:path'

const BASE = process.env.BASE_URL || 'http://127.0.0.1:3000'
const outDir = process.argv[2]
const browser = await chromium.launch()
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
await page.goto(`${BASE}/login`, { waitUntil: 'networkidle' })
await page.fill('input[type="email"]', process.env.ADMIN_EMAIL || 'admin@wujie.local')
await page.fill('input[type="password"]', process.env.ADMIN_PASSWORD)
await page.click('button[type="submit"]')
await page.waitForURL((url) => !url.pathname.startsWith('/login'), { timeout: 15000 })
await page.waitForLoadState('networkidle')
// 侧栏顶部版本 badge（品牌区下方的小字版本号）
const badge = page.locator('aside').first().locator('div.relative').first()
await badge.click()
await page.waitForTimeout(1200)
await page.screenshot({ path: path.join(outDir, '09-version-popover.png') })
console.log('done')
await browser.close()
