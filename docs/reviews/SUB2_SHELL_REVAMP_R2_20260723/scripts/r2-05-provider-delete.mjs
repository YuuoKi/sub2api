// r2-05 删除探针通道：二次确认 → 成功 → 列表消失（契约④ DELETE /admin/video/providers/:id）
import path from 'node:path'
import { BASE, makeReporter, launch, login, readState } from './_r2common.mjs'

const outDir = path.resolve(process.argv[2])
const { note, save } = makeReporter(outDir, 'r2-05')
const { browser, page } = await launch()
const shot = (n) => page.screenshot({ path: path.join(outDir, `${n}.png`) })

const state = readState()
const PROVIDER_ID = state.providerId
const PROBE_NAME = state.providerName || 'R2探针-中转通道'

try {
  if (!PROVIDER_ID) throw new Error('state 缺 providerId，先跑 r2-04')

  await login(page)
  await page.goto(`${BASE}/admin/console/key-vault?tab=video`, { waitUntil: 'networkidle' })

  note('probe-present-before-delete', (await page.locator(`[data-test="remove-provider-${PROVIDER_ID}"]`).count()) === 1)

  // 抓 DELETE 请求/响应
  const deletePromise = page.waitForResponse(
    (r) => r.url().includes(`/api/v1/admin/video/providers/${PROVIDER_ID}`) && r.request().method() === 'DELETE',
    { timeout: 15000 }
  )
  await page.click(`[data-test="remove-provider-${PROVIDER_ID}"]`)

  // 二次确认弹窗出现（截图取证），点「确认」执行删除
  const confirmDialog = page.locator('text=确定删除通道').first()
  await confirmDialog.waitFor({ timeout: 8000 })
  await shot('r2-05a-delete-confirm-dialog')

  await page.click('button:has-text("确认")')
  const deleteRes = await deletePromise
  note('delete-http-ok', deleteRes.status() === 200 || deleteRes.status() === 204, `status=${deleteRes.status()} body=${(await deleteRes.text()).slice(0, 200)}`)

  await page.waitForTimeout(1500)
  const bodyText = await page.locator('body').innerText()
  note('delete-success-toast', bodyText.includes('通道已删除'), bodyText.match(/通道已删除/)?.[0] || 'no-toast')
  note('probe-gone-from-list', !bodyText.includes(PROBE_NAME))
  await shot('r2-05b-provider-deleted-list')
} catch (e) {
  note('fatal', false, e.message)
  await shot('r2-05-zz-fatal')
} finally {
  save()
  await browser.close()
}
