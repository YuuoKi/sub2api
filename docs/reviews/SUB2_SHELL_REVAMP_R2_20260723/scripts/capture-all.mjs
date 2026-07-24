// 全页面截图采集：桌面 1440×900 15 组 + 移动 390×844 3 张；控制台 JS 错误逐页记录
// 探针员工（视觉审查探针 audit-probe@wujie.local）经 API 造好，截图后不删除（审查统一处理）
import path from 'node:path'
import fs from 'node:fs'
import { chromium } from 'playwright'
import { BASE, ADMIN_EMAIL, ADMIN_PASSWORD, apiToken, api } from './_r2common.mjs'

const outDir = path.resolve(process.argv[2])
fs.mkdirSync(outDir, { recursive: true })

const PROBE_EMAIL = 'audit-probe@wujie.local'
const PROBE_NAME = '视觉审查探针'

// ---- fixture：探针员工（幂等）----
async function ensureProbeStaff() {
  const token = await apiToken()
  const users = await api('GET', '/api/v1/admin/users?page=1&page_size=100', undefined, token)
  let probe = (users.json?.data?.items || []).find((u) => u.email === PROBE_EMAIL)
  if (!probe) {
    const created = await api('POST', '/api/v1/admin/users', { email: PROBE_EMAIL, username: PROBE_NAME, member_type: 'tool', role: 'user' }, token)
    probe = created.json?.data?.user || created.json?.data
    if (!probe?.id) throw new Error('probe user create failed: ' + created.text.slice(0, 200))
    const pair = await api('POST', `/api/v1/admin/users/${probe.id}/qcanvas-key-pair`, { video_group_id: 3, media_group_id: 3 }, token)
    if (pair.status !== 200 && pair.status !== 201) throw new Error('probe key-pair failed: ' + pair.text.slice(0, 200))
    await api('POST', `/api/v1/admin/users/${probe.id}/balance`, { balance: 5, operation: 'add', notes: '视觉审查探针充值' }, token)
  }
  return probe.id
}

// ---- 控制台错误收集 ----
const consoleErrors = {} // pageId -> string[]
let currentPageId = 'boot'
const attachErrorTap = (page) => {
  page.on('console', (msg) => {
    if (msg.type() === 'error') (consoleErrors[currentPageId] ||= []).push(msg.text().slice(0, 500))
  })
  page.on('pageerror', (err) => (consoleErrors[currentPageId] ||= []).push(`pageerror: ${String(err).slice(0, 500)}`))
}

const browser = await chromium.launch()

function newContext(viewport) {
  return browser.newContext({ viewport, locale: 'zh-CN' })
}
async function prepPage(ctx) {
  const page = await ctx.newPage()
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
    if (document.head || document.documentElement) attach()
    else document.addEventListener('DOMContentLoaded', attach, { once: true })
  })
  attachErrorTap(page)
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

const probeId = await ensureProbeStaff()
console.log('probe staff id =', probeId)

// ================= 桌面 1440×900 =================
const ctx = await newContext({ width: 1440, height: 900 })
const page = await prepPage(ctx)
const shot = async (id, name, settle = 800) => {
  currentPageId = id
  await page.waitForTimeout(settle)
  await page.screenshot({ path: path.join(outDir, `${id}-${name}.png`) })
  console.log('shot', id, name)
}
const go = async (id, name, url, settle = 1200) => {
  currentPageId = id
  await page.goto(`${BASE}${url}`, { waitUntil: 'networkidle' })
  await shot(id, name, settle)
}

try {
  // 01 登录页（未登录态）
  currentPageId = '01'
  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle' })
  await shot('01', 'login')

  await login(page)

  await go('02', 'overview', '/admin/console/overview', 1800)
  await go('03', 'keyvault-accounts-tab', '/admin/console/key-vault')

  // 04 录入 AI 账号弹窗（平台下拉可见）
  currentPageId = '04'
  await page.click('[data-test="open-create-account"]')
  await page.waitForSelector('form[data-test="account-form"]', { timeout: 8000 })
  await page.click('form[data-test="account-form"] select') // 展开平台下拉
  await shot('04', 'keyvault-account-modal', 500)
  await page.keyboard.press('Escape')
  await page.click('[data-test="cancel-account"]')
  await page.waitForTimeout(400)

  await go('05', 'keyvault-video-tab', '/admin/console/key-vault?tab=video')

  // 06 视频录入弹窗（平台下拉展开显示置灰项）
  currentPageId = '06'
  await page.click('[data-test="open-create-provider"]')
  await page.waitForSelector('form[data-test="provider-form"]', { timeout: 8000 })
  await page.click('[data-test="provider-platform"]')
  await shot('06', 'keyvault-video-modal-dropdown', 500)
  await page.keyboard.press('Escape')
  await page.click('[data-test="cancel-provider"]')
  await page.waitForTimeout(400)

  // 07 员工卡列表 + 展开「查看卡片」
  currentPageId = '07'
  await page.goto(`${BASE}/admin/console/staff`, { waitUntil: 'networkidle' })
  const probeRow = page.locator('tr', { hasText: PROBE_EMAIL }).first()
  await probeRow.locator('button:has-text("查看卡片")').click()
  await page.waitForTimeout(1500) // 等卡片表加载
  await shot('07', 'staff-list-expanded-card', 600)

  // 08 新增员工弹窗（填好一半）
  currentPageId = '08'
  await page.click('[data-test="create-service-identity"]')
  await page.waitForSelector('form[data-test="service-identity-form"]', { timeout: 8000 })
  await page.locator('form[data-test="service-identity-form"] input').first().fill('示例员工')
  await page.fill('[data-test="service-identity-email"]', 'demo-half@wujie.local')
  await shot('08', 'staff-create-modal-half', 500)
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)
  // 兜底：若 Escape 未关弹窗则点遮罩/取消
  if (await page.locator('form[data-test="service-identity-form"]').isVisible().catch(() => false)) {
    await page.mouse.click(30, 450)
    await page.waitForTimeout(300)
  }

  // 09 行内「充值」弹窗
  currentPageId = '09'
  await probeRow.locator('[data-test="row-recharge"]').click()
  await page.waitForSelector('form[data-test="recharge-form"]', { timeout: 8000 })
  await shot('09', 'staff-recharge-modal', 500)
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)
  if (await page.locator('form[data-test="recharge-form"]').isVisible().catch(() => false)) {
    await page.mouse.click(30, 450)
    await page.waitForTimeout(300)
  }

  // 10 用量→AI 调用记录：两个内部 Tab
  currentPageId = '10a'
  await page.goto(`${BASE}/admin/console/ai-records`, { waitUntil: 'networkidle' })
  await shot('10a', 'ai-records-logs-tab', 1200)
  currentPageId = '10b'
  await page.click('[data-test="tab-prompts"]')
  await page.waitForTimeout(1500)
  await shot('10b', 'ai-records-prompts-tab', 600)

  await go('11', 'ops-health', '/admin/ops', 2000)
  await go('12', 'settings-backup', '/admin/settings', 1500)
  await go('13', 'video-providers-legacy', '/admin/video/providers', 1500)
  await go('14', 'groups', '/admin/groups', 1500)
  await go('15', 'accounts', '/admin/accounts', 1500)
} catch (e) {
  console.error('desktop capture fatal:', e.message)
  try { await page.screenshot({ path: path.join(outDir, 'zz-desktop-fatal.png') }) } catch {}
} finally {
  await ctx.close()
}

// ================= 移动 390×844 =================
const mctx = await newContext({ width: 390, height: 844 })
const mpage = await prepPage(mctx)
try {
  await login(mpage)
  const mgo = async (id, name, url) => {
    currentPageId = id
    await mpage.goto(`${BASE}${url}`, { waitUntil: 'networkidle' })
    await mpage.waitForTimeout(1500)
    await mpage.screenshot({ path: path.join(outDir, `${id}-${name}.png`) })
    console.log('shot', id, name)
  }
  await mgo('m-01', 'overview', '/admin/console/overview')
  await mgo('m-02', 'keyvault', '/admin/console/key-vault')
  await mgo('m-03', 'staff', '/admin/console/staff')
} catch (e) {
  console.error('mobile capture fatal:', e.message)
  try { await mpage.screenshot({ path: path.join(outDir, 'zz-mobile-fatal.png') }) } catch {}
} finally {
  await mctx.close()
  await browser.close()
}

// 去重并落盘
const dedup = {}
for (const [k, v] of Object.entries(consoleErrors)) dedup[k] = [...new Set(v)]
fs.writeFileSync(path.join(outDir, '_console-errors.json'), JSON.stringify(dedup, null, 2))
const pagesWithErrors = Object.entries(dedup).filter(([, v]) => v.length)
console.log(pagesWithErrors.length ? `console errors on: ${pagesWithErrors.map(([k]) => k).join(', ')}` : 'no console errors')
