import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

import {
  EMPLOYEE_TOP_LEVEL_PATHS,
  buildEmployeeRoleNav,
  collectTopLevelPaths,
} from '../roleAwareNavigation'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')
const navModulePath = resolve(dirname(fileURLToPath(import.meta.url)), '../roleAwareNavigation.ts')
const navModuleSource = readFileSync(navModulePath, 'utf8')

describe('AppSidebar role-aware IA wiring', () => {
  it('uses roleAwareNavigation builders and admin overview homePath', () => {
    expect(componentSource).toContain("from './roleAwareNavigation'")
    expect(componentSource).toContain('buildAdminRoleNav')
    expect(componentSource).toContain('buildEmployeeRoleNav')
    expect(componentSource).toContain("isAdmin.value ? '/admin/console/overview' : '/dashboard'")
  })

  it('wires employee production nav to exactly EMPLOYEE_TOP_LEVEL_PATHS (no 更多)', () => {
    expect(componentSource).toContain('buildEmployeeRoleNav({')
    expect(componentSource).not.toContain('includeMoreGroup: true')
    expect(componentSource).not.toMatch(/includeMoreGroup\s*:/)

    const productionEmployeeNav = buildEmployeeRoleNav({ isSimpleMode: false })
    const topPaths = collectTopLevelPaths(productionEmployeeNav)
    expect(topPaths).toEqual([...EMPLOYEE_TOP_LEVEL_PATHS])
    expect(topPaths).toHaveLength(5)
    expect(topPaths).not.toContain('/more')
    expect(productionEmployeeNav.map((item) => item.label)).not.toContain('更多')
  })

  it('does not keep flat management paths at admin top level definitions', () => {
    expect(navModuleSource).toContain("ADMIN_SYSTEM_PATH = '/admin/system'")
    expect(navModuleSource).toContain('path: ADMIN_SYSTEM_PATH')
    expect(navModuleSource).toContain('运行与配置')
    expect(navModuleSource).toContain('高级与历史')
    expect(navModuleSource).toMatch(/label:\s*'总览'/)
    expect(navModuleSource).toMatch(/label:\s*'我的工作台'/)
  })

  it('keeps nested System rendering helpers for expandOnly groups', () => {
    expect(componentSource).toContain('expandOnly')
    expect(componentSource).toContain('isGroupExpanded')
    expect(componentSource).toContain('handleGroupClick')
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
