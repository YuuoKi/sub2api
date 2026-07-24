// r2-03 密钥库视频 Tab：平台下拉 — Seedance 2.0 可选；即梦/Veo 3.1/快乐小马 置灰「即将接入」
import path from 'node:path'
import { BASE, makeReporter, launch, login } from './_r2common.mjs'

const outDir = path.resolve(process.argv[2])
const { note, save } = makeReporter(outDir, 'r2-03')
const { browser, page } = await launch()
const shot = (n) => page.screenshot({ path: path.join(outDir, `${n}.png`) })

try {
  await login(page)
  await page.goto(`${BASE}/admin/console/key-vault?tab=video`, { waitUntil: 'networkidle' })

  // 契约层断言：platforms 数组来自后端（请求监听）
  const contractPromise = page.waitForResponse((r) => r.url().includes('/api/v1/admin/video/contract'), { timeout: 10000 }).catch(() => null)
  await page.reload({ waitUntil: 'networkidle' })
  const contractRes = await contractPromise
  if (contractRes) {
    const body = await contractRes.json().catch(() => null)
    const platforms = body?.data?.platforms || []
    note('contract-has-platforms', platforms.length === 4, JSON.stringify(platforms.map((p) => `${p.provider}:${p.adapter_ready}`)))
  } else {
    note('contract-has-platforms', false, 'no contract response captured')
  }

  await page.click('[data-test="open-create-provider"]')
  await page.waitForSelector('form[data-test="provider-form"]', { timeout: 8000 })

  // DOM 断言：四个平台的 disabled 状态与文案
  const options = await page.locator('[data-test="provider-platform"] option').evaluateAll((els) =>
    els.map((e) => ({ value: e.value, text: e.textContent.trim(), disabled: e.disabled }))
  )
  const find = (v) => options.find((o) => o.value === v)
  note('seedance-selectable', find('seedance') && !find('seedance').disabled, JSON.stringify(find('seedance')))
  note('jimeng-disabled-coming', find('jimeng')?.disabled && find('jimeng')?.text.includes('即将接入'), JSON.stringify(find('jimeng')))
  note('veo-disabled-coming', find('veo')?.disabled && find('veo')?.text.includes('即将接入'), JSON.stringify(find('veo')))
  note('kling-disabled-coming', find('kling')?.disabled && find('kling')?.text.includes('即将接入'), JSON.stringify(find('kling')))

  // 尝试打开原生下拉截图（headless Chromium 的 listbox 是内部弹层，可被截图捕获）
  await page.click('[data-test="provider-platform"]')
  await page.waitForTimeout(500)
  await shot('r2-03-platform-dropdown-open')

  // 关掉下拉（Esc），补一张弹窗全景
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)
  await shot('r2-03-provider-modal')
} catch (e) {
  note('fatal', false, e.message)
  await shot('r2-03-zz-fatal')
} finally {
  save()
  await browser.close()
}
