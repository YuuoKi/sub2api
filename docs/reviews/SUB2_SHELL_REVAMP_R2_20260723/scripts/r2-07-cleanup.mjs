// r2-07 清理：删除探针员工（API）+ 校验探针通道/员工均已消失；UI 复核员工列表
import path from 'node:path'
import { BASE, makeReporter, launch, login, apiToken, api, readState } from './_r2common.mjs'

const outDir = path.resolve(process.argv[2])
const { note, save } = makeReporter(outDir, 'r2-07')
const { browser, page } = await launch()
const shot = (n) => page.screenshot({ path: path.join(outDir, `${n}.png`) })

const state = readState()

try {
  const token = await apiToken()

  // 1) 探针员工删除
  const users = await api('GET', '/api/v1/admin/users?page=1&page_size=100', undefined, token)
  const probe = (users.json?.data?.items || []).find((u) => u.email === (state.staffEmail || 'r2-probe-staff@wujie.local'))
  if (probe) {
    const del = await api('DELETE', `/api/v1/admin/users/${probe.id}`, undefined, token)
    note('probe-staff-deleted', del.status === 200 || del.status === 204, `status=${del.status} body=${del.text.slice(0, 200)}`)
  } else {
    note('probe-staff-deleted', true, 'already absent')
  }
  const usersAfter = await api('GET', '/api/v1/admin/users?page=1&page_size=100', undefined, token)
  const stillThere = (usersAfter.json?.data?.items || []).some((u) => u.email === (state.staffEmail || 'r2-probe-staff@wujie.local'))
  note('probe-staff-gone', !stillThere)

  // 2) 探针通道应已在 r2-05 删除；兜底再清一次
  const providers = await api('GET', '/api/v1/admin/video/providers', undefined, token)
  const items = providers.json?.data?.items || providers.json?.data || []
  const leftovers = Array.isArray(items) ? items.filter((p) => String(p.display_name || '').includes('R2探针')) : []
  for (const p of leftovers) {
    await api('DELETE', `/api/v1/admin/video/providers/${p.id}`, undefined, token)
  }
  const providersAfter = await api('GET', '/api/v1/admin/video/providers', undefined, token)
  const itemsAfter = providersAfter.json?.data?.items || providersAfter.json?.data || []
  const remain = Array.isArray(itemsAfter) ? itemsAfter.filter((p) => String(p.display_name || '').includes('R2探针')) : []
  note('no-probe-providers-left', remain.length === 0, `leftovers=${leftovers.length} remain=${remain.length}`)

  // 3) UI 复核员工列表无探针
  await login(page)
  await page.goto(`${BASE}/admin/console/staff`, { waitUntil: 'networkidle' })
  await page.waitForTimeout(1200)
  const bodyText = await page.locator('body').innerText()
  note('ui-staff-list-clean', !bodyText.includes(state.staffEmail || 'r2-probe-staff@wujie.local') && !bodyText.includes('R2探针员工'))
  await shot('r2-07-staff-list-clean')
} catch (e) {
  note('fatal', false, e.message)
  await shot('r2-07-zz-fatal')
} finally {
  save()
  await browser.close()
}
