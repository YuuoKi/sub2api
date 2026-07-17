import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { mount, flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import TablePageLayout from '../TablePageLayout.vue'

const dir = dirname(fileURLToPath(import.meta.url))
const read = (name: string) => readFileSync(resolve(dir, `../${name}`), 'utf8')

describe('responsive semantics (~390px shell contract)', () => {
  afterEach(() => {
    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      writable: true,
      value: 1024
    })
  })

  it('AppLayout exposes shell landmarks and mobile-first page padding', () => {
    const layout = read('AppLayout.vue')
    expect(layout).toContain('data-testid="app-shell"')
    expect(layout).toContain('data-testid="app-main"')
    expect(layout).toContain('<AppHeader />')
    expect(layout).toContain('<AppSidebar />')
    // Mobile-first padding: p-4 base, then md/lg steps
    expect(layout).toMatch(/class="p-4 md:p-6 lg:p-8"/)
    expect(layout).toContain('bg-ui-canvas')
  })

  it('TablePageLayout keeps horizontal table scroll and 390px mobile mode', async () => {
    const source = read('TablePageLayout.vue')
    expect(source).toContain('table-scroll-container')
    expect(source).toContain('overflow-x-auto')
    expect(source).toContain('window.innerWidth < 1024')
    expect(source).toContain("'mobile-mode': isMobile")
    expect(source).toMatch(/\.table-page-layout\.mobile-mode[\s\S]*overflow-visible/)

    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      writable: true,
      value: 390
    })

    const wrapper = mount(TablePageLayout, {
      slots: {
        table: '<table class="table-wrapper"><tbody><tr><td>cell</td></tr></tbody></table>'
      }
    })
    await flushPromises()

    expect(wrapper.classes()).toContain('mobile-mode')
    expect(wrapper.find('.table-scroll-container').exists()).toBe(true)
  })

  it('AppSidebar / AppHeader keep landmark testids for narrow shells', () => {
    expect(read('AppSidebar.vue')).toContain('data-testid="app-sidebar"')
    expect(read('AppSidebar.vue')).toContain('aria-label="Primary navigation"')
    expect(read('AppHeader.vue')).toContain('data-testid="app-header"')
    expect(read('AppHeader.vue')).toContain('app-header-control')
    expect(read('AppSidebar.vue')).toContain('sidebar-link')
  })
})
