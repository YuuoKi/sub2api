import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

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

describe('AppSidebar navigation builders', () => {
  it('keeps role and feature-gated navigation builders intact', () => {
    expect(componentSource).toContain('const adminNavItems = computed')
    expect(componentSource).toContain('const userNavItems = computed')
    expect(componentSource).toContain('applyFeatureFlags')
  })
})

describe('AppSidebar video navigation', () => {
  it('makes video trial and task records discoverable for regular users', () => {
    const selfNavBlock = componentSource.match(/function buildSelfNavItems[\s\S]*?return items\n}/)?.[0]

    expect(selfNavBlock).toContain("path: '/admin/video/create', label: '视频试跑'")
    expect(selfNavBlock).toContain("path: '/admin/video/tasks', label: '任务记录'")
  })

  it('uses business Chinese for admin video navigation and groups technical pages under system', () => {
    expect(componentSource).toContain("path: '/admin/video',\n      label: '视频生产'")
    expect(componentSource).toContain("label: '系统',")
    expect(componentSource).toContain("{ path: '/admin/video/providers', label: '生成通道'")
    expect(componentSource).toContain("{ path: '/admin/video/system-check', label: '系统检查'")
    expect(componentSource).not.toContain("{ path: '/admin/video/create', label: 'Create Task'")
  })
})
