// P0.1 验收：密钥库分组必选 + 内联建组 + 列表分组列（真人链路：录 Ark Key → media 组）
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
// driver.js 引导遮罩会拦截点击；每次进页面后清掉
async function dismissTour() {
  await page.keyboard.press('Escape').catch(() => {})
  const skip = page.locator('button:has-text("Skip guide"), button:has-text("跳过引导"), .driver-popover-progress-text + button').first()
  if (await skip.count()) await skip.click().catch(() => {})
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(200)
}

try {
  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle' })
  await page.fill('input[type="email"]', process.env.ADMIN_EMAIL || 'admin@wujie.local')
  await page.fill('input[type="password"]', process.env.ADMIN_PASSWORD)
  await page.click('button[type="submit"]')
  await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 15000 })

  await page.goto(`${BASE}/admin/console/key-vault`, { waitUntil: 'networkidle' })
  await dismissTour()
  await page.click('[data-test="open-create-account"]')
  await page.waitForSelector('[data-test="account-form"]')

  // 选 OpenAI（Ark 走 openai 兼容）→ 本地无 openai 组 → 内联建组引导出现
  await page.selectOption('select', 'openai')
  await page.waitForTimeout(400)
  const quickVisible = await page.locator('[data-test="group-quick-create"]').isVisible()
  note('quick-create-visible-for-openai', quickVisible)
  await shot('p01-01-openai-no-group-guide')

  // 不选组直接保存 → 中文报错，不创建
  await page.fill('input[placeholder="例如：老板的 Claude 主账号"]', 'K3验收-Ark作图')
  await page.fill('[data-test="account-api-key"]', 'sk-k3-acceptance-not-a-real-key')
  await page.fill('input[placeholder="留空使用官方默认地址"]', 'https://ark.cn-beijing.volces.com/api/v3')
  await page.click('[data-test="save-account"]')
  await page.waitForTimeout(800)
  const errToast = await page.locator('text=请至少选择一个分组').count()
  note('save-blocked-without-group', errToast > 0)
  await shot('p01-02-save-blocked-no-group')

  // 内联一键建「作图组 media」→ 自动选中
  await page.click('[data-test="quick-create-media"]')
  await page.waitForTimeout(2000)
  const toastOk = await page.locator('text=已创建并选中').count()
  note('quick-create-media-group', toastOk > 0)
  await shot('p01-03-media-group-created-selected')

  // 再保存 → 成功，列表出现且带 media 分组
  await page.click('[data-test="save-account"]')
  await page.waitForTimeout(2500)
  const row = page.locator('tr', { hasText: 'K3验收-Ark作图' }).first()
  const rowExists = await row.count()
  note('account-created-in-list', rowExists > 0)
  const rowText = rowExists ? await row.innerText() : ''
  note('list-shows-media-group', rowText.includes('media'), rowText.replace(/\s+/g, ' ').slice(0, 120))
  await shot('p01-04-list-with-group-column')

  // 编辑回填：打开编辑应看到 media 已勾选
  await row.locator('button:has-text("编辑")').click()
  await page.waitForTimeout(600)
  const checkedBoxes = await page.locator('[data-test="account-form"] input[type="checkbox"]:checked').count()
  note('edit-backfills-group', checkedBoxes > 0, `checked=${checkedBoxes}`)
  await shot('p01-05-edit-group-backfilled')
  await page.click('[data-test="cancel-account"]')
  await page.waitForTimeout(400)

  // 清理：删除验收账号（分组 media/openai 保留，后续链路要用）
  await page.locator('tr', { hasText: 'K3验收-Ark作图' }).locator('button').last().click()
  await page.waitForTimeout(600)
  await page.locator('button:has-text("确认")').last().click()
  await page.waitForTimeout(1500)
  const gone = (await page.locator('tr', { hasText: 'K3验收-Ark作图' }).count()) === 0
  note('cleanup-account-deleted', gone)
  await shot('p01-06-after-cleanup')
} catch (e) {
  note('fatal', false, e.message)
  await shot('p01-zz-fatal')
} finally {
  fs.writeFileSync(path.join(outDir, '_p01-report.json'), JSON.stringify(report, null, 2))
  await browser.close()
}
