// r2-02 总览空态：scratch 库零数据 → 中部「三步上手」引导卡
import path from 'node:path'
import { BASE, makeReporter, launch, login } from './_r2common.mjs'

const outDir = path.resolve(process.argv[2])
const { note, save } = makeReporter(outDir, 'r2-02')
const { browser, page } = await launch()
const shot = (n) => page.screenshot({ path: path.join(outDir, `${n}.png`) })

try {
  await login(page)
  await page.goto(`${BASE}/admin/console/overview`, { waitUntil: 'networkidle' })
  await page.waitForTimeout(1500)

  const bodyText = await page.locator('body').innerText()
  note('guide-card-title', bodyText.includes('三步上手'))
  note('guide-step1', bodyText.includes('录入 AI 账号'))
  note('guide-step2', bodyText.includes('给员工开卡'))
  note('guide-step3', bodyText.includes('回到这里看消费'))
  // 空态下不应出现有数据版图表块
  note('no-trend-chart-section', !bodyText.includes('花费趋势'), bodyText.includes('花费趋势') ? 'unexpected trend section' : '')

  // 步骤 1/2 链接可用
  const step1 = page.locator('a[href="/admin/console/key-vault"]').first()
  const step2 = page.locator('a[href="/admin/console/staff"]').first()
  note('step1-link-keyvault', await step1.isVisible())
  note('step2-link-staff', await step2.isVisible())

  await shot('r2-02-overview-empty-guide')
} catch (e) {
  note('fatal', false, e.message)
  await shot('r2-02-zz-fatal')
} finally {
  save()
  await browser.close()
}
