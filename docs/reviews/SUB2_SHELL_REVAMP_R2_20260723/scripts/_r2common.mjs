// R2 验收公共模块：真登录、1440×900、driver.js 遮罩抑制、报告输出
import { chromium } from 'playwright'
import fs from 'node:fs'
import path from 'node:path'

export const BASE = process.env.BASE_URL || 'http://127.0.0.1:3000'
export const API = process.env.API_URL || 'http://127.0.0.1:8089'
export const ADMIN_EMAIL = process.env.ADMIN_EMAIL || 'admin@wujie.local'
export const ADMIN_PASSWORD = process.env.ADMIN_PASSWORD || 'R2accept-20260723'

export const STATE_FILE = path.resolve(process.cwd(), '_probe_state.json')

export function readState() {
  try { return JSON.parse(fs.readFileSync(STATE_FILE, 'utf8')) } catch { return {} }
}
export function writeState(patch) {
  fs.writeFileSync(STATE_FILE, JSON.stringify({ ...readState(), ...patch }, null, 2))
}

export function makeReporter(outDir, tag) {
  fs.mkdirSync(outDir, { recursive: true })
  const report = []
  const note = (name, ok, extra = '') => {
    report.push({ name, ok: !!ok, extra: String(extra) })
    console.log(`${ok ? 'OK  ' : 'FAIL'} ${name}${extra ? ' — ' + extra : ''}`)
  }
  const save = () => fs.writeFileSync(path.join(outDir, `_${tag}-report.json`), JSON.stringify(report, null, 2))
  return { report, note, save }
}

export async function launch() {
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
    if (document.head || document.documentElement) attach()
    else document.addEventListener('DOMContentLoaded', attach, { once: true })
  })
  return { browser, page }
}

export async function login(page) {
  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle' })
  await page.fill('input[type="email"]', ADMIN_EMAIL)
  await page.fill('input[type="password"]', ADMIN_PASSWORD)
  await page.click('button[type="submit"]')
  await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 15000 })
  await page.evaluate(() => {
    Object.keys(localStorage).filter((k) => k.startsWith('onboarding_tour')).forEach((k) => localStorage.setItem(k, 'true'))
  })
}

// ---- API 直连（清理/断言用，不走 UI）----
export async function apiToken() {
  const res = await fetch(`${API}/api/v1/auth/login`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: ADMIN_EMAIL, password: ADMIN_PASSWORD }),
  })
  const data = await res.json()
  if (!data?.data?.access_token) throw new Error('api login failed: ' + JSON.stringify(data).slice(0, 200))
  return data.data.access_token
}

export async function api(method, path, body, token) {
  const res = await fetch(`${API}${path}`, {
    method,
    headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const text = await res.text()
  let json = null
  try { json = JSON.parse(text) } catch {}
  return { status: res.status, json, text: text.slice(0, 600) }
}
