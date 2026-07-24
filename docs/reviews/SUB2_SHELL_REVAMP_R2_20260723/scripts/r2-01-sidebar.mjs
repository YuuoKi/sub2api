// r2-01 侧栏新结构：用量（AI 调用记录+系统健康）；系统下只剩 设置与备份/高级
import path from 'node:path'
import { BASE, makeReporter, launch, login } from './_r2common.mjs'

const outDir = path.resolve(process.argv[2])
const { note, save } = makeReporter(outDir, 'r2-01')
const { browser, page } = await launch()
const shot = (n) => page.screenshot({ path: path.join(outDir, `${n}.png`) })

try {
  await login(page)
  await page.goto(`${BASE}/admin/console/overview`, { waitUntil: 'networkidle' })

  const sidebar = page.locator('aside').first()
  await sidebar.waitFor({ timeout: 10000 })

  // 展开「用量」
  await sidebar.getByText('用量', { exact: true }).first().click()
  await page.waitForTimeout(400)
  // 展开「系统」（默认收起）
  await sidebar.getByText('系统', { exact: true }).first().click()
  await page.waitForTimeout(400)

  const linkCount = async (href) => sidebar.locator(`a[href="${href}"]`).count()
  note('usage-has-ai-records', (await linkCount('/admin/console/ai-records')) === 1)
  note('usage-has-ops-health', (await linkCount('/admin/ops')) === 1)

  // 归属断言：系统健康必须在「用量」组容器内，而不在「系统」组容器内
  const membership = await page.evaluate(() => {
    const aside = document.querySelector('aside')
    const opsLink = aside.querySelector('a[href="/admin/ops"]')
    const aiLink = aside.querySelector('a[href="/admin/console/ai-records"]')
    // 找包含某链接的最近组容器（其内部还有组标题按钮）
    const groupOf = (el) => {
      let node = el?.parentElement
      while (node && node !== aside) {
        if (node.querySelector(':scope > button, :scope > a') && node.querySelector('a[href]') !== el && node.querySelectorAll('a[href]').length >= 1) {
          const labelEl = node.querySelector(':scope > button, :scope > div > button')
          if (labelEl) return labelEl.textContent.trim()
        }
        node = node.parentElement
      }
      return null
    }
    const sysButtons = [...aside.querySelectorAll('button')].map((b) => b.textContent.trim())
    return { opsGroup: groupOf(opsLink), aiGroup: groupOf(aiLink), sysButtons }
  })
  note('ops-under-usage-group', membership.opsGroup === '用量', `opsGroup=${membership.opsGroup}`)
  note('ai-records-under-usage-group', membership.aiGroup === '用量', `aiGroup=${membership.aiGroup}`)

  // 「系统」展开后组内链接集合：只允许 设置与备份 + 高级（及其子项），不得出现 系统健康
  // 结构：分组 button 通过 aria-controls 关联子链接面板 div
  const systemLinks = await page.evaluate(() => {
    const aside = document.querySelector('aside')
    const sysBtn = [...aside.querySelectorAll('button')].find((b) => b.textContent.trim().startsWith('系统'))
    if (!sysBtn) return null
    const panelId = sysBtn.getAttribute('aria-controls')
    const panel = panelId ? document.getElementById(panelId) : null
    if (!panel) return []
    return [...panel.querySelectorAll('a[href]')].map((a) => ({ href: a.getAttribute('href'), text: a.textContent.trim() }))
  })
  const sysHrefs = (systemLinks || []).map((l) => l.href)
  note('system-has-settings-backup', sysHrefs.includes('/admin/settings'), JSON.stringify(systemLinks))
  note('system-no-ops-health', !sysHrefs.includes('/admin/ops'), sysHrefs.join(','))

  // 「高级」按钮在系统面板内；展开后复核子项（含 视频通道），仍不得有 系统健康
  const advancedClicked = await page.evaluate(() => {
    const aside = document.querySelector('aside')
    const sysBtn = [...aside.querySelectorAll('button')].find((b) => b.textContent.trim().startsWith('系统'))
    const panel = document.getElementById(sysBtn?.getAttribute('aria-controls') || '')
    const advBtn = panel ? [...panel.querySelectorAll('button')].find((b) => b.textContent.trim().startsWith('高级')) : null
    if (advBtn) { advBtn.click(); return true }
    return false
  })
  note('system-has-advanced', advancedClicked)
  await page.waitForTimeout(400)
  const systemLinks2 = await page.evaluate(() => {
    const aside = document.querySelector('aside')
    const sysBtn = [...aside.querySelectorAll('button')].find((b) => b.textContent.trim().startsWith('系统'))
    const panel = document.getElementById(sysBtn?.getAttribute('aria-controls') || '')
    if (!panel) return []
    return [...panel.querySelectorAll('a[href]')].map((a) => ({ href: a.getAttribute('href'), text: a.textContent.trim() }))
  })
  const sysHrefs2 = systemLinks2.map((l) => l.href)
  note('advanced-has-video-providers', sysHrefs2.includes('/admin/video/providers'), sysHrefs2.join(','))
  note('system-still-no-ops-health', !sysHrefs2.includes('/admin/ops'), '')

  await shot('r2-01-sidebar-structure')
} catch (e) {
  note('fatal', false, e.message)
  await shot('r2-01-zz-fatal')
} finally {
  save()
  await browser.close()
}
