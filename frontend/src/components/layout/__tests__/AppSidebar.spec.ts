import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

function sourceBetween(startMarker: string, endMarker: string): string {
  const start = componentSource.indexOf(startMarker)
  const end = componentSource.indexOf(endMarker, start)

  expect(start).toBeGreaterThanOrEqual(0)
  expect(end).toBeGreaterThan(start)

  return componentSource.slice(start, end)
}

const selfNavigationSource = sourceBetween('function buildSelfNavItems', 'function finalizeNav')
const adminNavigationSource = sourceBetween('// Admin navigation items', 'function toggleSidebar')
const actualNavigationSource = `${selfNavigationSource}\n${adminNavigationSource}`

describe('AppSidebar navigation contract', () => {
  it('does not expose consumer sales entries in the actual sidebar definitions', () => {
    const removedSalesPaths = [
      '/subscriptions',
      '/purchase',
      '/orders',
      '/payment',
      '/affiliate',
      '/admin/subscriptions',
      '/admin/promo-codes',
      '/admin/affiliates',
      '/admin/orders',
      '/admin/payment'
    ]

    for (const path of removedSalesPaths) {
      expect(actualNavigationSource).not.toContain(`path: '${path}'`)
    }
  })

  it('keeps the complete management surface visible in the sidebar definitions', () => {
    const requiredManagementPaths = [
      '/keys',
      '/usage',
      '/admin/users',
      '/admin/accounts',
      '/admin/groups',
      '/admin/redeem',
      '/admin/settings'
    ]

    for (const path of requiredManagementPaths) {
      expect(actualNavigationSource).toContain(`path: '${path}'`)
    }
  })
})

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
  })
})

describe('AppSidebar scroll position persistence', () => {
  it('binds a template ref to the sidebar nav element', () => {
    expect(componentSource).toContain('ref="sidebarNavRef"')
    expect(componentSource).toContain('sidebar-nav')
  })

  it('declares sidebarNavRef in script setup', () => {
    expect(componentSource).toContain("const sidebarNavRef = ref<HTMLElement | null>(null)")
  })

  it('saves scroll position on beforeUnmount', () => {
    expect(componentSource).toContain('onBeforeUnmount')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('sidebarNavRef.value.scrollTop')
  })

  it('restores scroll position on mount', () => {
    expect(componentSource).toContain('onMounted')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('nextTick')
  })
})

describe('AppSidebar header styles', () => {
  it('does not clip the version badge dropdown', () => {
    const sidebarHeaderBlockMatch = styleSource.match(/\.sidebar-header\s*\{[\s\S]*?\n {2}\}/)
    const sidebarBrandBlockMatch = componentSource.match(/\.sidebar-brand\s*\{[\s\S]*?\n\}/)

    expect(sidebarHeaderBlockMatch).not.toBeNull()
    expect(sidebarBrandBlockMatch).not.toBeNull()
    expect(sidebarHeaderBlockMatch?.[0]).not.toContain('@apply overflow-hidden;')
    expect(sidebarBrandBlockMatch?.[0]).not.toContain('overflow: hidden;')
  })
})
