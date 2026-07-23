// P0.4 验收：密钥库隐藏「测试」按钮 + 停用二次确认 + 恢复（8080 共享后端，状态最终还原）
import { chromium } from 'playwright'
import fs from 'node:fs'
import path from 'node:path'

const BASE = process.env.BASE_URL || 'http://127.0.0.1:3000'
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
  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle' })
  await page.fill('input[type="email"]', process.env.ADMIN_EMAIL || 'admin@wujie.local')
  await page.fill('input[type="password"]', process.env.ADMIN_PASSWORD)
  await page.click('button[type="submit"]')
  await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 15000 })

  await page.goto(`${BASE}/admin/console/key-vault`, { waitUntil: 'networkidle' })

  // 1. 操作列不再有「测试」
  const testBtnCount = await page.locator('tbody button:has-text("测试")').count()
  note('test-button-hidden', testBtnCount === 0, `count=${testBtnCount}`)
  await shot('p04-01-no-test-button')

  // 2. 点「停用」→ 二次确认弹窗 → 取消 → 状态不变
  const row = page.locator('tbody tr').first()
  await row.locator('button:has-text("停用")').click()
  await page.waitForTimeout(800)
  const confirmDlg = page.locator('text=确定停用账号')
  note('disable-needs-confirmation', (await confirmDlg.count()) > 0)
  await shot('p04-02-disable-confirm-dialog')
  await page.locator('button:has-text("取消")').last().click()
  await page.waitForTimeout(600)
  note('cancel-keeps-active', (await row.innerText()).includes('正常'))

  // 3. 再点「停用」→ 确认 → toast 已停用 → 行变已停用
  await row.locator('button:has-text("停用")').click()
  await page.waitForTimeout(500)
  await page.locator('button:has-text("确认")').last().click()
  await page.waitForTimeout(2000)
  const rowText = await page.locator('tbody tr').first().innerText()
  note('disable-confirmed-works', rowText.includes('已停用'), rowText.replace(/\s+/g, ' ').slice(0, 80))
  await shot('p04-03-disabled-row')

  // 4. 还原：点「启用」
  await page.locator('tbody tr').first().locator('button:has-text("启用")').click()
  await page.waitForTimeout(2000)
  const restored = await page.locator('tbody tr').first().innerText()
  note('re-enabled-restore', restored.includes('正常'))
  await shot('p04-04-restored')
} catch (e) {
  note('fatal', false, e.message)
  await shot('p04-zz-fatal')
} finally {
  fs.writeFileSync(path.join(outDir, '_p04-report.json'), JSON.stringify(report, null, 2))
  await browser.close()
}
