// r2-04 视频通道页内创建：名称+分组+假 Key+自定义接口地址；取消勾选「保存后自动授权」；保存成功列表出现
// 附带 API 级断言：base_url 透传到列表；default_model 透传（API 探针，建后即删）
import path from 'node:path'
import { BASE, makeReporter, launch, login, apiToken, api, writeState } from './_r2common.mjs'

const outDir = path.resolve(process.argv[2])
const { note, save } = makeReporter(outDir, 'r2-04')
const { browser, page } = await launch()
const shot = (n) => page.screenshot({ path: path.join(outDir, `${n}.png`) })

const PROBE_NAME = 'R2探针-中转通道'
const PROBE_KEY = 'r2-probe-fake-key-20260723'
const PROBE_BASE_URL = 'http://relay.example.com/v3'

try {
  // 预清理：历史探针通道先删（同组唯一槽位约束会让重跑 409）
  const token0 = await apiToken()
  const pre = await api('GET', '/api/v1/admin/video/providers', undefined, token0)
  const preItems = pre.json?.data?.items || pre.json?.data || []
  for (const p of Array.isArray(preItems) ? preItems.filter((x) => String(x.display_name || '').includes('R2探针')) : []) {
    await api('DELETE', `/api/v1/admin/video/providers/${p.id}`, undefined, token0)
  }

  await login(page)
  await page.goto(`${BASE}/admin/console/key-vault?tab=video`, { waitUntil: 'networkidle' })

  await page.click('[data-test="open-create-provider"]')
  await page.waitForSelector('form[data-test="provider-form"]', { timeout: 8000 })

  // 真人填写
  await page.fill('[data-test="provider-name"]', PROBE_NAME)
  // 分组 checkbox：选「video」组
  const groupLabel = page.locator('form[data-test="provider-form"] label', { hasText: 'video' }).first()
  await groupLabel.click()
  await page.fill('[data-test="provider-api-key"]', PROBE_KEY)
  await page.fill('[data-test="provider-base-url"]', PROBE_BASE_URL)

  // 取消勾选「保存后自动授权」（默认勾选；避免真实上游调用）；保留「保存后启用」
  const authBox = page.locator('[data-test="provider-authorize-after-save"]')
  note('authorize-default-checked', await authBox.isChecked())
  if (await authBox.isChecked()) await authBox.click()
  note('authorize-unchecked-by-user', !(await authBox.isChecked()))
  const enabledBox = page.locator('[data-test="provider-enabled"]')
  note('enabled-still-checked', await enabledBox.isChecked())

  await shot('r2-04a-provider-form-filled')

  // 抓保存请求/响应；同时监听是否发生 tiny-real 授权调用（取消勾选后应为 0 次）
  let tinyRealCalls = 0
  page.on('request', (req) => { if (req.url().includes('/tiny-real-authorization')) tinyRealCalls += 1 })
  const createPromise = page.waitForResponse(
    (r) => r.url().includes('/api/v1/admin/video/providers') && r.request().method() === 'POST',
    { timeout: 15000 }
  )
  await page.click('[data-test="save-provider"]')
  const createRes = await createPromise
  const createBody = await createRes.json().catch(() => null)
  note('create-http-ok', createRes.status() === 200 || createRes.status() === 201, `status=${createRes.status()} body=${JSON.stringify(createBody)?.slice(0, 300)}`)
  const createReqBody = JSON.parse(createRes.request().postData() || '{}')
  note('create-request-carries-base-url', createReqBody.base_url === PROBE_BASE_URL, JSON.stringify(createReqBody))

  await page.waitForTimeout(2000)
  await shot('r2-04b-provider-created-list')

  const bodyText = await page.locator('body').innerText()
  note('provider-in-list', bodyText.includes(PROBE_NAME))
  note('no-autoauth-call', tinyRealCalls === 0, `tiny-real-authorization 请求数=${tinyRealCalls}`)

  // API 级断言：记录已建且 enabled；base_url 按契约不落响应（json:"-"，与密钥同级保密），改为 DB 直查证明持久化
  const token = await apiToken()
  const list = await api('GET', '/api/v1/admin/video/providers', undefined, token)
  const items = list.json?.data?.items || list.json?.data || []
  const mine = Array.isArray(items) ? items.find((p) => p.display_name === PROBE_NAME) : null
  note('api-list-has-provider', !!mine, JSON.stringify(mine)?.slice(0, 400))
  note('api-list-enabled', mine?.enabled === true, `enabled=${mine?.enabled}`)
  if (mine?.id) writeState({ providerId: mine.id, providerName: PROBE_NAME })

  // DB 直查：base_url 透传持久化（响应 DTO 不回显 base_url 是既有保密设计）
  const { execSync } = await import('node:child_process')
  const dbRow = execSync(
    `docker exec wujie-lan-postgres psql -U sub2api -d sub2api_r2 -tA -c "SELECT display_name||'|'||base_url||'|'||default_model FROM video_provider_accounts WHERE display_name='${PROBE_NAME}'"`
  ).toString().trim()
  note('db-base-url-persisted', dbRow.includes(`|${PROBE_BASE_URL}|`), dbRow)

  // API 探针：default_model 透传（UI 无此字段，契约③要求透传；用 media 组避开同组默认模型槽位约束；建后即删）
  const probe = await api('POST', '/api/v1/admin/video/providers', {
    group_id: 2, provider: 'seedance', display_name: 'R2探针-default-model透传',
    api_key: PROBE_KEY, base_url: PROBE_BASE_URL, default_model: 'doubao-seedance-2-0-probe', enabled: false,
  }, token)
  const probeId = probe.json?.data?.id
  note('probe-create-ok', !!probeId, `status=${probe.status} ${probe.text.slice(0, 200)}`)
  note('default-model-passthrough', probe.json?.data?.default_model === 'doubao-seedance-2-0-probe', JSON.stringify(probe.json?.data)?.slice(0, 400))
  note('probe-base-url-echo-none', probe.json?.data?.base_url === undefined, 'base_url 不回显为预期设计')
  if (probeId) {
    const probeDb = execSync(
      `docker exec wujie-lan-postgres psql -U sub2api -d sub2api_r2 -tA -c "SELECT base_url||'|'||default_model FROM video_provider_accounts WHERE id=${probeId}"`
    ).toString().trim()
    note('db-probe-custom-model-persisted', probeDb === `${PROBE_BASE_URL}|doubao-seedance-2-0-probe`, probeDb)
    await api('DELETE', `/api/v1/admin/video/providers/${probeId}`, undefined, token)
  }
} catch (e) {
  note('fatal', false, e.message)
  await shot('r2-04-zz-fatal')
} finally {
  save()
  await browser.close()
}
