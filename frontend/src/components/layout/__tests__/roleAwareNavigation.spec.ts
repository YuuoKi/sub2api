import { describe, expect, it } from 'vitest'

import {
  ADMIN_SYSTEM_PATH,
  ADMIN_TOP_LEVEL_PATHS,
  EMPLOYEE_MORE_PATH,
  EMPLOYEE_TOP_LEVEL_PATHS,
  buildAdminRoleNav,
  buildEmployeeRoleNav,
  collectLeafPaths,
  collectTopLevelPaths,
  findNavItem,
  isDefaultCollapsedGroup,
} from '../roleAwareNavigation'

describe('admin role-aware top-level IA', () => {
  it('exposes exactly five top-level entries in brand order', () => {
    const items = buildAdminRoleNav({ opsMonitoringEnabled: true, channelMonitorEnabled: true, riskControlEnabled: true })
    expect(collectTopLevelPaths(items)).toEqual([...ADMIN_TOP_LEVEL_PATHS])
    expect(items.map((item) => item.label)).toEqual([
      '总览',
      '密钥库',
      '成员与开卡',
      '任务记录',
      '系统',
    ])
  })

  it('keeps System expandOnly and collapsed by default', () => {
    const system = findNavItem(buildAdminRoleNav({}), ADMIN_SYSTEM_PATH)
    expect(system?.expandOnly).toBe(true)
    expect(isDefaultCollapsedGroup(system!)).toBe(true)
  })

  it('nests ops/config and advanced/history under System', () => {
    const system = findNavItem(buildAdminRoleNav({ opsMonitoringEnabled: true }), ADMIN_SYSTEM_PATH)
    const groupLabels = (system?.children ?? []).map((child) => child.label)
    expect(groupLabels).toEqual(['运行与配置', '高级与历史'])

    const opsPaths = collectLeafPaths(system?.children?.find((c) => c.label === '运行与配置'))
    const advancedPaths = collectLeafPaths(system?.children?.find((c) => c.label === '高级与历史'))

    for (const path of [
      '/admin/ops',
      '/admin/groups',
      '/admin/video/providers',
      '/admin/video/system-check',
      '/admin/channels/pricing',
      '/admin/channels/monitor',
      '/admin/accounts',
      '/admin/announcements',
      '/admin/proxies',
      '/admin/risk-control',
      '/admin/usage',
      '/admin/settings',
      '/admin/console/ai-records',
      '/admin/generation-content',
      '/admin/dashboard',
      '/admin/users',
    ]) {
      expect(opsPaths).toContain(path)
    }

    for (const path of [
      '/admin/subscriptions',
      '/admin/redeem',
      '/admin/promo-codes',
      '/admin/orders',
      '/admin/orders/dashboard',
      '/admin/orders/plans',
      '/admin/affiliates',
      '/admin/affiliates/invites',
      '/admin/affiliates/rebates',
      '/admin/affiliates/transfers',
    ]) {
      expect(advancedPaths).toContain(path)
    }

    expect(opsPaths).not.toContain('/admin/video/tasks')
    expect(collectTopLevelPaths(buildAdminRoleNav({}))).not.toContain('/admin/users')
  })

  it('places custom admin menus under System ops/config, not top level', () => {
    const items = buildAdminRoleNav({
      customAdminMenus: [{ id: 9, label: '自定义面板', icon_svg: '<svg/>' }],
    })
    expect(collectTopLevelPaths(items)).toEqual([...ADMIN_TOP_LEVEL_PATHS])
    const ops = findNavItem(items, ADMIN_SYSTEM_PATH)?.children?.find((c) => c.label === '运行与配置')
    expect(collectLeafPaths(ops)).toContain('/custom/9')
  })
})

describe('employee role-aware top-level IA', () => {
  it('exposes exactly five top-level entries in brand order', () => {
    const items = buildEmployeeRoleNav({})
    expect(collectTopLevelPaths(items)).toEqual([...EMPLOYEE_TOP_LEVEL_PATHS])
    expect(items.map((item) => item.label)).toEqual([
      '我的工作台',
      '创建任务',
      '任务记录',
      '我的密钥',
      '我的花费',
    ])
  })

  it('keeps legacy employee surfaces out of ordinary top navigation', () => {
    const top = collectTopLevelPaths(buildEmployeeRoleNav({
      batchImageEnabled: true,
      availableChannelsEnabled: true,
      channelMonitorEnabled: true,
    }))
    for (const path of ['/batch-image', '/available-channels', '/monitor', '/redeem', '/profile']) {
      expect(top).not.toContain(path)
    }
  })

  it('opt-in includeMoreGroup can park legacy surfaces, but default production nav stays at five', () => {
    const production = buildEmployeeRoleNav({})
    expect(collectTopLevelPaths(production)).toEqual([...EMPLOYEE_TOP_LEVEL_PATHS])
    expect(findNavItem(production, EMPLOYEE_MORE_PATH)).toBeUndefined()

    const withMore = buildEmployeeRoleNav({
      includeMoreGroup: true,
      batchImageEnabled: true,
      availableChannelsEnabled: true,
      channelMonitorEnabled: true,
    })
    expect(collectTopLevelPaths(withMore).slice(0, 5)).toEqual([...EMPLOYEE_TOP_LEVEL_PATHS])
    const more = findNavItem(withMore, EMPLOYEE_MORE_PATH)
    expect(more?.label).toBe('更多')
    expect(more?.expandOnly).toBe(true)
    expect(isDefaultCollapsedGroup(more!)).toBe(true)
    const moreLeaves = collectLeafPaths(more)
    for (const path of ['/batch-image', '/available-channels', '/monitor', '/redeem']) {
      expect(moreLeaves).toContain(path)
    }
  })
})
