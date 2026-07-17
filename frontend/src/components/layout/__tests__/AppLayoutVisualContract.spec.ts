import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const read = (name: string) =>
  readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), `../${name}`), 'utf8')

describe('shared app shell visual contract', () => {
  it('exposes stable shell landmarks', () => {
    expect(read('AppLayout.vue')).toContain('data-testid="app-shell"')
    expect(read('AppLayout.vue')).toContain('data-testid="app-main"')
    expect(read('AppHeader.vue')).toContain('data-testid="app-header"')
    expect(read('AppSidebar.vue')).toContain('data-testid="app-sidebar"')
  })

  it('keeps layout motion accessible', () => {
    expect(read('AppLayout.vue')).toContain('motion-reduce:transition-none')
  })

  it('applies focus-visible rings via --ui-focus on shell controls', () => {
    const style = readFileSync(
      resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css'),
      'utf8'
    )
    expect(style).toMatch(/\.sidebar-link:focus-visible\s*\{[\s\S]*--ui-focus/)
    expect(style).toMatch(/\.app-header-control:focus-visible\s*\{[\s\S]*--ui-focus/)
    expect(read('AppHeader.vue')).toContain('app-header-control')
  })

  it('preserves Task 3 console navigation entries', () => {
    const sidebar = read('AppSidebar.vue')
    expect(sidebar).toContain("path: '/admin/console/overview'")
    expect(sidebar).toContain("path: '/admin/console/key-vault'")
    expect(sidebar).toContain("path: '/admin/console/staff'")
    expect(sidebar).toContain("path: '/admin/console/ai-records'")
    expect(sidebar).toContain("path: '/admin/generation-content'")
    expect(sidebar).toContain('filterAdminNavigationForMode')
  })
})
