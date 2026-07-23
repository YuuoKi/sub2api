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
  it('uses the five locked lan_admin capability entries with short single-layer labels', () => {
    const items = buildAdminRoleNav({})

    expect(collectTopLevelPaths(items)).toEqual([...ADMIN_TOP_LEVEL_PATHS])
    expect(items.map((item) => item.label)).toEqual([
      '总览',
      '密钥库',
      '员工卡',
      '用量',
      '系统',
    ])
  })

  it('keeps System expandOnly and collapsed by default', () => {
    const system = findNavItem(buildAdminRoleNav({}), ADMIN_SYSTEM_PATH)
    expect(system?.expandOnly).toBe(true)
    expect(isDefaultCollapsedGroup(system!)).toBe(true)
  })

  it('exposes ops/settings directly and tucks legacy surfaces under System > Advanced', () => {
    const system = findNavItem(buildAdminRoleNav({
      customAdminMenus: [{ id: 9, label: '旧入口' }],
      opsMonitoringEnabled: true,
      channelMonitorEnabled: true,
      riskControlEnabled: true,
    }), ADMIN_SYSTEM_PATH)

    expect(system?.children?.map((item) => item.path)).toEqual([
      '/admin/ops',
      '/admin/settings',
      '/admin/system/advanced',
    ])
    const advanced = findNavItem(system?.children ?? [], '/admin/system/advanced')
    expect(collectLeafPaths(advanced)).toEqual([
      '/admin/accounts',
      '/admin/groups',
      '/admin/video/providers',
      '/admin/channels/pricing',
      '/admin/channels/monitor',
      '/admin/proxies',
      '/admin/video/tasks',
      '/admin/generation-content',
      '/admin/usage',
    ])
  })

  it('does not expose legacy commercial, user-management or custom-menu surfaces', () => {
    const items = buildAdminRoleNav({
      customAdminMenus: [{ id: 9, label: '旧入口' }],
      opsMonitoringEnabled: true,
      channelMonitorEnabled: true,
      riskControlEnabled: true,
    })
    const allPaths = items.flatMap((item) => collectLeafPaths(item))

    for (const path of [
      '/admin/dashboard',
      '/admin/users',
      '/admin/subscriptions',
      '/admin/redeem',
      '/admin/promo-codes',
      '/admin/orders',
      '/admin/affiliates',
      '/admin/announcements',
      '/admin/risk-control',
      '/custom/9',
    ]) {
      expect(allPaths, path).not.toContain(path)
    }
  })
})

describe('employee role-aware top-level IA', () => {
  it('keeps the default employee navigation contract for non-lan-admin deployments', () => {
    const items = buildEmployeeRoleNav({})
    expect(collectTopLevelPaths(items)).toEqual([...EMPLOYEE_TOP_LEVEL_PATHS])
  })

  it('returns no employee navigation in the lan_admin product surface', () => {
    expect(buildEmployeeRoleNav({ lanAdminOnly: true })).toEqual([])
  })

  it('keeps optional legacy surfaces out of ordinary top navigation', () => {
    const withMore = buildEmployeeRoleNav({
      includeMoreGroup: true,
      batchImageEnabled: true,
      availableChannelsEnabled: true,
      channelMonitorEnabled: true,
    })
    expect(collectTopLevelPaths(withMore).slice(0, 5)).toEqual([...EMPLOYEE_TOP_LEVEL_PATHS])
    const more = findNavItem(withMore, EMPLOYEE_MORE_PATH)
    expect(more?.expandOnly).toBe(true)
    expect(isDefaultCollapsedGroup(more!)).toBe(true)
  })
})
