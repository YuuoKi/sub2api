import { existsSync, readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../..')
const indexSource = readFileSync(resolve(frontendRoot, 'index.html'), 'utf8')
const mainSource = readFileSync(resolve(frontendRoot, 'src/main.ts'), 'utf8')
const routerSource = readFileSync(resolve(frontendRoot, 'src/router/index.ts'), 'utf8')
const sidebarSource = readFileSync(
  resolve(frontendRoot, 'src/components/layout/AppSidebar.vue'),
  'utf8',
)
const forbiddenDemoMode = ['video', 'gateway', 'demo'].join('_')

describe('无界品牌入口', () => {
  it('使用无界企业 AI 管理中台作为静态页面标题', () => {
    expect(indexSource).toContain('<title>无界 · 企业 AI 管理中台</title>')
    expect(indexSource).not.toContain('<title>Sub2API - AI API Gateway</title>')
  })

  it('启动逻辑不再拼接上游 AI API Gateway 品牌', () => {
    expect(mainSource).not.toContain('AI API Gateway')
  })

  it('侧栏继续展示运行时品牌名与现有 logo', () => {
    expect(sidebarSource).toContain('{{ siteName }}')
    expect(sidebarSource).toContain("siteLogo || '/logo.png'")
  })
})

describe('完整管理后台保护', () => {
  const requiredAdminSurfaces = [
    ['/admin/users', 'src/views/admin/UsersView.vue'],
    ['/admin/accounts', 'src/views/admin/AccountsView.vue'],
    ['/admin/groups', 'src/views/admin/GroupsView.vue'],
    ['/admin/redeem', 'src/views/admin/RedeemView.vue'],
    ['/admin/settings', 'src/views/admin/SettingsView.vue'],
  ] as const

  it.each(requiredAdminSurfaces)('保留 %s 路由与视图', (routePath, viewPath) => {
    expect(routerSource).toContain(`path: '${routePath}'`)
    expect(existsSync(resolve(frontendRoot, viewPath))).toBe(true)
  })

  it('未在路由或侧栏启用 demo 全隐藏模式', () => {
    expect(routerSource).not.toContain(forbiddenDemoMode)
    expect(sidebarSource).not.toContain(forbiddenDemoMode)
  })
})
