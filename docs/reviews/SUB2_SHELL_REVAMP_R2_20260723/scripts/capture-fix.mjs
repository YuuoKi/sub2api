// fix 复拍：前端修复后受影响的 7 项页面/状态；断言 + 截图到 audit/（fix- 前缀）
import path from 'node:path'
import fs from 'node:fs'
import { chromium } from 'playwright'
import { BASE, ADMIN_EMAIL, ADMIN_PASSWORD, apiToken, api } from './_r2common.mjs'

const outDir = path.resolve(process.argv[2])
fs.mkdirSync(outDir, { recursive: true })
const report = []
const note = (name, ok, extra = '') => {
  report.push({ name, ok: !!ok, extra: String(extra) })
  console.log(`${ok ? 'OK  ' : 'FAIL'} ${name}${extra ? ' — ' + extra : ''}`)
}

const PROBE_EMAIL = 'audit-probe@wujie.local'

// 探针幂等保证：活跃缺失时复活最近软删的同名员工并补齐 QCanvas 卡；都没有则 API 新建
async function ensureProbe() {
  const token = await apiToken()
  const users = await api('GET', '/api/v1/admin/users?page=1&page_size=100', undefined, token)
  const active = (users.json?.data?.items || []).find((u) => u.email === PROBE_EMAIL)
  if (active) return active.id
  const { execSync } = await import('node:child_process')
  const raw = execSync(
    `docker exec wujie-lan-postgres psql -U sub2api -d sub2api_r2 -tA -c "UPDATE users SET deleted_at=NULL WHERE id=(SELECT MAX(id) FROM users WHERE email='${PROBE_EMAIL}' AND deleted_at IS NOT NULL) RETURNING id"`
  ).toString()
  const restored = (raw.match(/^\d+$/m) || [])[0] || ''
  let probeId
  if (restored) {
    execSync(`docker exec wujie-lan-postgres psql -U sub2api -d sub2api_r2 -c "UPDATE api_keys SET deleted_at=NULL WHERE user_id=${restored} AND deleted_at IS NOT NULL"`)
    probeId = Number(restored)
  } else {
    const created = await api('POST', '/api/v1/admin/users', { email: PROBE_EMAIL, username: '视觉审查探针', member_type: 'tool', role: 'user' }, token)
    const probe = created.json?.data?.user || created.json?.data
    probeId = probe.id
    await api('POST', `/api/v1/admin/users/${probeId}/balance`, { balance: 5, operation: 'add', notes: '视觉审查探针充值' }, token)
  }
  // 保证有卡（复活的分支可能没有）
  const keys = await api('GET', `/api/v1/admin/users/${probeId}/api-keys`, undefined, token)
  const keyItems = keys.json?.data?.items || keys.json?.data || []
  if (!Array.isArray(keyItems) || keyItems.length === 0) {
    await api('POST', `/api/v1/admin/users/${probeId}/qcanvas-key-pair`, { video_group_id: 3, media_group_id: 3 }, token)
  }
  return probeId
}

const browser = await chromium.launch()
const newCtx = (viewport) => browser.newContext({ viewport, locale: 'zh-CN' })
async function prep(ctx) {
  const page = await ctx.newPage()
  await page.addInitScript(() => {
    try {
      localStorage.setItem('admin_guide_1_admin_v5_wujie_operator', 'true')
      localStorage.setItem('onboarding_tour_1_admin_v5_wujie_operator', 'true')
    } catch {}
    const style = document.createElement('style')
    style.textContent = '.driver-overlay,.driver-popover,#driver-dummy-element{display:none!important;pointer-events:none!important}'
    const attach = () => (document.head || document.documentElement).appendChild(style)
    if (document.head || document.documentElement) attach()
    else document.addEventListener('DOMContentLoaded', attach, { once: true })
  })
  return page
}
async function login(page) {
  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle' })
  await page.fill('input[type="email"]', ADMIN_EMAIL)
  await page.fill('input[type="password"]', ADMIN_PASSWORD)
  await page.click('button[type="submit"]')
  await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 15000 })
  await page.evaluate(() => {
    Object.keys(localStorage).filter((k) => k.startsWith('onboarding_tour')).forEach((k) => localStorage.setItem(k, 'true'))
  })
}

const ctx = await newCtx({ width: 1440, height: 900 })
const page = await prep(ctx)
const shot = (n) => page.screenshot({ path: path.join(outDir, `${n}.png`) })

try {
  await ensureProbe()
  await login(page)

  // ---------- fix-01 侧栏 系统→高级：视频通道消失，其余 8 项在 ----------
  await page.goto(`${BASE}/admin/console/overview`, { waitUntil: 'networkidle' })
  const sidebar = page.locator('aside').first()
  await sidebar.getByText('系统', { exact: true }).first().click()
  await page.waitForTimeout(400)
  await page.evaluate(() => {
    const aside = document.querySelector('aside')
    const sysBtn = [...aside.querySelectorAll('button')].find((b) => b.textContent.trim().startsWith('系统'))
    const panel = document.getElementById(sysBtn?.getAttribute('aria-controls') || '')
    const advBtn = panel ? [...panel.querySelectorAll('button')].find((b) => b.textContent.trim().startsWith('高级')) : null
    advBtn?.click()
  })
  await page.waitForTimeout(500)
  const advLinks = await page.evaluate(() => {
    const aside = document.querySelector('aside')
    const sysBtn = [...aside.querySelectorAll('button')].find((b) => b.textContent.trim().startsWith('系统'))
    const panel = document.getElementById(sysBtn?.getAttribute('aria-controls') || '')
    if (!panel) return []
    return [...panel.querySelectorAll('a[href]')].map((a) => ({ href: a.getAttribute('href'), text: a.textContent.trim() }))
  })
  const hrefs = advLinks.map((l) => l.href)
  const childLinks = advLinks.filter((l) => l.href !== '/admin/settings')
  note('fix01-advanced-8-items', childLinks.length === 8, JSON.stringify(childLinks.map((l) => l.text)))
  note('fix01-video-providers-gone', !hrefs.includes('/admin/video/providers'), hrefs.join(','))
  note('fix01-tasks-still-there', hrefs.includes('/admin/video/tasks'))
  await shot('fix-01-sidebar-advanced-no-video-providers')

  // ---------- fix-02 员工卡列表：无「备注」列、无「工具」标签 ----------
  await page.goto(`${BASE}/admin/console/staff`, { waitUntil: 'networkidle' })
  await page.waitForTimeout(1000)
  const headerCells = await page.locator('thead th').allInnerTexts()
  note('fix02-no-notes-column', !headerCells.some((t) => t.includes('备注')), JSON.stringify(headerCells))
  const bodyText = await page.locator('body').innerText()
  note('fix02-probe-present', bodyText.includes(PROBE_EMAIL))
  const toolBadges = await page.locator('td span', { hasText: '工具' }).count()
  note('fix02-no-tool-badge', toolBadges === 0, `工具 badge 数=${toolBadges}`)
  await shot('fix-02-staff-list-no-notes-no-tool')

  // ---------- fix-03 展开行卡片表（桌面）----------
  const probeRow = page.locator('tr', { hasText: PROBE_EMAIL }).first()
  await probeRow.locator('button:has-text("查看卡片")').click()
  await page.waitForTimeout(1500)
  const cardTableVisible = await page.locator('td table, div:has(> table)').first().isVisible().catch(() => false)
  note('fix03-card-table-visible', cardTableVisible || (await page.locator('text=QCanvas · video').count()) > 0)
  await shot('fix-03-staff-expanded-card-table')

  // ---------- fix-04 AI 调用记录·提示词采集：英文 ledger 块消失 ----------
  await page.goto(`${BASE}/admin/console/ai-records`, { waitUntil: 'networkidle' })
  await page.click('[data-test="tab-prompts"]')
  await page.waitForTimeout(1800)
  const promptsText = await page.locator('body').innerText()
  note('fix04-no-english-ledger', !promptsText.includes('Weekly Production Ledger'), '')
  note('fix04-weekly-block-hidden', (await page.locator('[data-test="weekly-report"]').count()) === 0, `weekly-report 块数=${await page.locator('[data-test="weekly-report"]').count()}`)
  note('fix04-cn-desc-present', promptsText.includes('提示词与结果已脱敏采集'))
  note('fix04-empty-card-present', promptsText.includes('暂无采集数据'))
  await shot('fix-04-ai-records-prompts-no-ledger')

  // ---------- fix-05 系统健康页头标题 ----------
  await page.goto(`${BASE}/admin/ops`, { waitUntil: 'networkidle' })
  await page.waitForTimeout(1500)
  const opsBody = await page.locator('body').innerText()
  const mainText = await page.locator('main').innerText().catch(() => opsBody)
  const headings = await page.locator('main h1, main h2, header h1, header h2').allInnerTexts().catch(() => [])
  note('fix05-title-system-health', mainText.includes('系统健康') || headings.some((h) => h.includes('系统健康')), `headings=${JSON.stringify(headings.slice(0, 4))}`)
  note('fix05-no-legacy-ops-name', !mainText.includes('运维监控'), mainText.includes('运维监控') ? '仍见「运维监控」' : '')
  await shot('fix-05-ops-system-health-title')

  // ---------- fix-06 控制台噪音：SPA 导航 3 次请求计数 ----------
  const counts = { subscriptions: 0, announcements: 0, adminSettings: 0 }
  const classify = (rawUrl) => {
    try {
      const u = new URL(rawUrl)
      const p = u.pathname // 只看 path，避免 include_subscriptions=false 之类的查询串误判
      if (p.includes('/settings/public')) return null
      if (p.includes('/subscriptions')) return 'subscriptions'
      if (p.includes('/announcements')) return 'announcements'
      if (p.startsWith('/api/v1/admin/settings')) return 'adminSettings'
      return null
    } catch { return null }
  }
  // 刷新首发（基线）
  const boot = { subscriptions: 0, announcements: 0, adminSettings: 0 }
  const bootTap = (req) => { const k = classify(req.url()); if (k) boot[k] += 1 }
  page.on('request', bootTap)
  await page.goto(`${BASE}/admin/console/overview`, { waitUntil: 'networkidle' })
  await page.waitForTimeout(1500)
  page.off('request', bootTap)
  // SPA 导航 3 次：总览→密钥库→员工卡→总览
  const navTap = (req) => { const k = classify(req.url()); if (k) counts[k] += 1 }
  page.on('request', navTap)
  for (const label of ['密钥库', '员工卡', '总览']) {
    await sidebar.getByText(label, { exact: true }).first().click()
    await page.waitForTimeout(1200)
  }
  page.off('request', navTap)
  const navTotal = counts.subscriptions + counts.announcements + counts.adminSettings
  note('fix06-no-refire-on-spa-nav', navTotal === 0, `SPA×3 计数=${JSON.stringify(counts)}；首发基线=${JSON.stringify(boot)}`)
  fs.writeFileSync(path.join(outDir, '_fix-console-requests.json'), JSON.stringify({
    note: 'SPA 导航 3 次（总览→密钥库→员工卡）期间的三类请求计数；boot = 刷新首发基线',
    boot_baseline: boot,
    spa_nav_x3: counts,
    pass: navTotal === 0,
  }, null, 2))
} catch (e) {
  note('fatal-desktop', false, e.message)
  try { await shot('zz-fix-fatal') } catch {}
} finally {
  await ctx.close()
}

// ---------- fix-03m 移动 390×844：展开卡片表横向可滚 ----------
const mctx = await newCtx({ width: 390, height: 844 })
const mpage = await prep(mctx)
try {
  await login(mpage)
  await mpage.goto(`${BASE}/admin/console/staff`, { waitUntil: 'networkidle' })
  await mpage.waitForTimeout(1000)
  const mRow = mpage.locator('tr', { hasText: PROBE_EMAIL }).first()
  await mRow.locator('button:has-text("查看卡片")').click()
  await mpage.waitForTimeout(1500)
  // 横向可滚断言：展开区卡片表被 overflow-x:auto/scroll 容器包裹（内容超出即可滚；未超出则完整可见不被裁剪）
  const scrollInfo = await mpage.evaluate(() => {
    const rows = [...document.querySelectorAll('tr')]
    const expanded = rows.find((r) => r.textContent.includes('卡名'))
    if (!expanded) return null
    let wrapper = null
    expanded.querySelectorAll('*').forEach((n) => {
      const ov = getComputedStyle(n).overflowX
      if ((ov === 'auto' || ov === 'scroll') && n.querySelector('table')) wrapper = wrapper || n
    })
    if (!wrapper) return { wrapperFound: false }
    const table = wrapper.querySelector('table')
    return {
      wrapperFound: true,
      overflowX: getComputedStyle(wrapper).overflowX,
      wrapperClient: wrapper.clientWidth,
      wrapperScroll: wrapper.scrollWidth,
      tableWidth: table ? table.scrollWidth : null,
      viewportW: window.innerWidth,
      bodyScrollW: document.documentElement.scrollWidth,
    }
  })
  const reachable = scrollInfo && scrollInfo.wrapperFound && (
    scrollInfo.wrapperScroll > scrollInfo.wrapperClient + 4 || // 超出：可滚
    (scrollInfo.tableWidth ?? 0) <= scrollInfo.wrapperClient + 4 // 未超出：完整可见
  )
  note('fix03m-horizontal-scrollable', !!reachable, JSON.stringify(scrollInfo))
  await mpage.screenshot({ path: path.join(outDir, 'fix-03m-staff-expanded-card-mobile.png') })
} catch (e) {
  note('fatal-mobile', false, e.message)
  try { await mpage.screenshot({ path: path.join(outDir, 'zz-fix-mobile-fatal.png') }) } catch {}
} finally {
  await mctx.close()
}

// ---------- fix-07 清理探针员工 + 空态 ----------
try {
  const token = await apiToken()
  const users = await api('GET', '/api/v1/admin/users?page=1&page_size=100', undefined, token)
  const probe = (users.json?.data?.items || []).find((u) => u.email === PROBE_EMAIL)
  if (probe) {
    const del = await api('DELETE', `/api/v1/admin/users/${probe.id}`, undefined, token)
    note('fix07-probe-deleted', del.status === 200 || del.status === 204, `status=${del.status}`)
  } else {
    note('fix07-probe-deleted', true, 'already absent')
  }
  const ctx2 = await newCtx({ width: 1440, height: 900 })
  const page2 = await prep(ctx2)
  await login(page2)
  await page2.goto(`${BASE}/admin/console/staff`, { waitUntil: 'networkidle' })
  await page2.waitForTimeout(1200)
  const text2 = await page2.locator('body').innerText()
  note('fix07-staff-empty-state', !text2.includes(PROBE_EMAIL) && text2.includes('还没有员工'))
  await page2.screenshot({ path: path.join(outDir, 'fix-07-staff-clean.png') })
  await ctx2.close()
} catch (e) {
  note('fatal-cleanup', false, e.message)
}

fs.writeFileSync(path.join(outDir, '_fix-report.json'), JSON.stringify(report, null, 2))
await browser.close()
const fails = report.filter((r) => !r.ok)
console.log(fails.length ? `FAILURES: ${fails.map((f) => f.name).join(', ')}` : 'ALL FIX CHECKS OK')
