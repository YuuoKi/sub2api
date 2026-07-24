// r2-06 员工开卡单弹窗：姓名/邮箱/所在分组（单组）/充值 $1 → 同屏双 Key 明文（契约①：同组双 Key 后端放行）
import path from 'node:path'
import { BASE, makeReporter, launch, login, apiToken, api, writeState } from './_r2common.mjs'

const outDir = path.resolve(process.argv[2])
const { note, save } = makeReporter(outDir, 'r2-06')
const { browser, page } = await launch()
const shot = (n) => page.screenshot({ path: path.join(outDir, `${n}.png`) })

const PROBE_EMAIL = 'r2-probe-staff@wujie.local'
const PROBE_NAME = 'R2探针员工'

try {
  // 预清理：同名探针员工若存在先删（保证脚本可重跑）
  const token = await apiToken()
  const users = await api('GET', '/api/v1/admin/users?page=1&page_size=100', undefined, token)
  const stale = (users.json?.data?.items || []).find((u) => u.email === PROBE_EMAIL)
  if (stale) await api('DELETE', `/api/v1/admin/users/${stale.id}`, undefined, token)

  await login(page)
  await page.goto(`${BASE}/admin/console/staff`, { waitUntil: 'networkidle' })
  note('page-title-staff-and-card', (await page.locator('body').innerText()).includes('员工与开卡'))

  await page.click('[data-test="create-service-identity"]')
  await page.waitForSelector('form[data-test="service-identity-form"]', { timeout: 8000 })

  // 真人填写：姓名 → 邮箱 → 所在分组（单组）→ 充值 $1
  await page.locator('form[data-test="service-identity-form"] input').first().fill(PROBE_NAME)
  await page.fill('[data-test="service-identity-email"]', PROBE_EMAIL)
  const groupOptions = await page.locator('[data-test="wizard-group"] option').evaluateAll((els) =>
    els.map((e) => ({ value: e.value, text: e.textContent.trim() })).filter((o) => o.value !== '0')
  )
  const videoGroup = groupOptions.find((o) => o.text === 'video') || groupOptions[0]
  await page.selectOption('[data-test="wizard-group"]', videoGroup.value)
  note('single-group-selected', !!videoGroup, JSON.stringify(groupOptions))
  await page.fill('[data-test="wizard-amount"]', '1')
  await shot('r2-06a-staff-form-filled')

  // 抓 qcanvas-key-pair 请求/响应（契约①证据：同组双 Key 放行）
  const pairPromise = page.waitForResponse(
    (r) => r.url().includes('/qcanvas-key-pair') && r.request().method() === 'POST',
    { timeout: 20000 }
  )
  await page.click('[data-test="wizard-submit"]')
  const pairRes = await pairPromise
  const pairReq = JSON.parse(pairRes.request().postData() || '{}')
  const pairBody = await pairRes.json().catch(() => null)
  note('pair-request-same-group', pairReq.video_group_id === pairReq.media_group_id && Number(pairReq.video_group_id) > 0, JSON.stringify(pairReq))
  note('pair-http-ok', pairRes.status() === 200 || pairRes.status() === 201, `status=${pairRes.status()} body=${JSON.stringify(pairBody)?.slice(0, 200)}`)

  // 同屏双 Key 明文
  await page.waitForSelector('[data-test="wizard-video-key"]', { timeout: 15000 })
  const videoKey = (await page.locator('[data-test="wizard-video-key"]').innerText()).trim()
  const mediaKey = (await page.locator('[data-test="wizard-media-key"]').innerText()).trim()
  note('video-key-plaintext-shown', videoKey.startsWith('sk-'), `${videoKey.slice(0, 10)}…`)
  note('media-key-plaintext-shown', mediaKey.startsWith('sk-'), `${mediaKey.slice(0, 10)}…`)
  note('two-keys-distinct', videoKey !== mediaKey)

  const rechargeText = await page.locator('[data-test="wizard-recharge-result"]').innerText().catch(() => '')
  note('recharge-1-usd-shown', rechargeText.includes('已充值 $1.00'), rechargeText)
  await shot('r2-06b-dual-keys-plaintext')

  // 记录探针员工 id 供清理
  const users2 = await api('GET', '/api/v1/admin/users?page=1&page_size=100', undefined, token)
  const mine = (users2.json?.data?.items || []).find((u) => u.email === PROBE_EMAIL)
  if (mine) {
    writeState({ staffUserId: mine.id, staffEmail: PROBE_EMAIL })
    const detail = await api('GET', `/api/v1/admin/users/${mine.id}`, undefined, token)
    note('balance-1-usd', Number(detail.json?.data?.balance ?? mine.balance) === 1, `balance=${detail.json?.data?.balance ?? mine.balance}`)
  } else {
    note('staff-user-created', false, 'user not found after wizard')
  }
} catch (e) {
  note('fatal', false, e.message)
  await shot('r2-06-zz-fatal')
} finally {
  save()
  await browser.close()
}
