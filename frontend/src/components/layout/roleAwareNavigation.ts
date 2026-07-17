/**
 * Role-aware console information architecture.
 * Pure path/label structure so sidebar wiring and tests share one contract.
 */

export interface RoleNavItem {
  path: string
  label: string
  /** Stable key only; click toggles expand state. */
  expandOnly?: boolean
  /** When true, group starts collapsed (not in expanded set). */
  defaultCollapsed?: boolean
  hideInSimpleMode?: boolean
  featureFlagKey?: 'ops' | 'channelMonitor' | 'riskControl' | 'batchImage' | 'availableChannels'
  children?: RoleNavItem[]
  /** Custom menu target; path already includes /custom/:id */
  customMenuId?: number | string
  iconSvg?: string
}

export interface CustomAdminMenuInput {
  id: number | string
  label: string
  icon_svg?: string
}

export interface BuildAdminRoleNavOptions {
  opsMonitoringEnabled?: boolean
  channelMonitorEnabled?: boolean
  riskControlEnabled?: boolean
  customAdminMenus?: CustomAdminMenuInput[]
  isSimpleMode?: boolean
}

export interface BuildEmployeeRoleNavOptions {
  includeMoreGroup?: boolean
  batchImageEnabled?: boolean
  availableChannelsEnabled?: boolean
  channelMonitorEnabled?: boolean
  isSimpleMode?: boolean
  customUserMenus?: CustomAdminMenuInput[]
}

export const ADMIN_SYSTEM_PATH = '/admin/system'
export const EMPLOYEE_MORE_PATH = '/more'

export const ADMIN_TOP_LEVEL_PATHS = [
  '/admin/console/overview',
  '/admin/console/key-vault',
  '/admin/console/staff',
  '/admin/video/tasks',
  ADMIN_SYSTEM_PATH,
] as const

export const EMPLOYEE_TOP_LEVEL_PATHS = [
  '/dashboard',
  '/video/create',
  '/video/tasks',
  '/keys',
  '/usage',
] as const

function applyFeatureFlags(
  items: RoleNavItem[],
  flags: BuildAdminRoleNavOptions & BuildEmployeeRoleNavOptions,
): RoleNavItem[] {
  const out: RoleNavItem[] = []
  for (const item of items) {
    if (item.featureFlagKey === 'ops' && flags.opsMonitoringEnabled === false) continue
    if (item.featureFlagKey === 'channelMonitor' && flags.channelMonitorEnabled === false) continue
    if (item.featureFlagKey === 'riskControl' && flags.riskControlEnabled === false) continue
    if (item.featureFlagKey === 'batchImage' && flags.batchImageEnabled === false) continue
    if (item.featureFlagKey === 'availableChannels' && flags.availableChannelsEnabled === false) continue

    if (item.children?.length) {
      const children = applyFeatureFlags(item.children, flags)
      if (children.length === 0 && item.expandOnly) continue
      out.push({ ...item, children })
    } else {
      out.push(item)
    }
  }
  return out
}

function applySimpleMode(items: RoleNavItem[], isSimpleMode: boolean): RoleNavItem[] {
  if (!isSimpleMode) return items
  const out: RoleNavItem[] = []
  for (const item of items) {
    if (item.hideInSimpleMode) continue
    if (item.children?.length) {
      const children = applySimpleMode(item.children, true)
      if (children.length === 0 && item.expandOnly) continue
      out.push({ ...item, children })
    } else {
      out.push(item)
    }
  }
  return out
}

export function collectTopLevelPaths(items: RoleNavItem[]): string[] {
  return items.map((item) => item.path)
}

export function collectLeafPaths(item: RoleNavItem | undefined): string[] {
  if (!item) return []
  if (!item.children?.length) return [item.path]
  return item.children.flatMap((child) => collectLeafPaths(child))
}

export function findNavItem(items: RoleNavItem[], path: string): RoleNavItem | undefined {
  for (const item of items) {
    if (item.path === path) return item
    if (item.children?.length) {
      const nested = findNavItem(item.children, path)
      if (nested) return nested
    }
  }
  return undefined
}

export function isDefaultCollapsedGroup(item: RoleNavItem): boolean {
  return item.expandOnly === true && item.defaultCollapsed === true
}

export function buildAdminRoleNav(options: BuildAdminRoleNavOptions = {}): RoleNavItem[] {
  const opsChildren: RoleNavItem[] = [
    { path: '/admin/ops', label: '运维监控', featureFlagKey: 'ops' },
    { path: '/admin/groups', label: '分组管理', hideInSimpleMode: true },
    { path: '/admin/video/providers', label: '生成通道' },
    { path: '/admin/video/system-check', label: '系统检查' },
    { path: '/admin/channels/pricing', label: '渠道定价', hideInSimpleMode: true },
    { path: '/admin/channels/monitor', label: '渠道监控', featureFlagKey: 'channelMonitor' },
    { path: '/admin/accounts', label: '账号管理' },
    { path: '/admin/announcements', label: '公告' },
    { path: '/admin/proxies', label: 'IP管理' },
    { path: '/admin/risk-control', label: '风控中心', hideInSimpleMode: true, featureFlagKey: 'riskControl' },
    { path: '/admin/usage', label: '全局用量' },
    { path: '/admin/settings', label: '系统设置' },
    { path: '/admin/console/ai-records', label: 'AI 记录' },
    { path: '/admin/generation-content', label: '生成内容' },
    { path: '/admin/dashboard', label: '技术仪表盘' },
    { path: '/admin/users', label: '用户管理', hideInSimpleMode: true },
    ...(options.customAdminMenus ?? []).map((menu): RoleNavItem => ({
      path: `/custom/${menu.id}`,
      label: menu.label,
      customMenuId: menu.id,
      iconSvg: menu.icon_svg,
    })),
  ]

  const advancedChildren: RoleNavItem[] = [
    { path: '/admin/subscriptions', label: '订阅管理', hideInSimpleMode: true },
    { path: '/admin/redeem', label: '兑换码', hideInSimpleMode: true },
    { path: '/admin/promo-codes', label: '优惠码', hideInSimpleMode: true },
    { path: '/admin/orders/dashboard', label: '支付概览' },
    { path: '/admin/orders', label: '订单管理' },
    { path: '/admin/orders/plans', label: '订阅套餐' },
    { path: '/admin/affiliates', label: '邀请返利' },
    { path: '/admin/affiliates/invites', label: '邀请记录' },
    { path: '/admin/affiliates/rebates', label: '返利记录' },
    { path: '/admin/affiliates/transfers', label: '提取记录' },
  ]

  const items: RoleNavItem[] = [
    { path: '/admin/console/overview', label: '总览' },
    { path: '/admin/console/key-vault', label: '密钥库' },
    { path: '/admin/console/staff', label: '成员与开卡' },
    { path: '/admin/video/tasks', label: '任务记录' },
    {
      path: ADMIN_SYSTEM_PATH,
      label: '系统',
      expandOnly: true,
      defaultCollapsed: true,
      children: [
        {
          path: '/admin/system/ops-config',
          label: '运行与配置',
          expandOnly: true,
          defaultCollapsed: true,
          children: opsChildren,
        },
        {
          path: '/admin/system/advanced',
          label: '高级与历史',
          expandOnly: true,
          defaultCollapsed: true,
          children: advancedChildren,
        },
      ],
    },
  ]

  return applySimpleMode(applyFeatureFlags(items, options), options.isSimpleMode === true)
}

export function buildEmployeeRoleNav(options: BuildEmployeeRoleNavOptions = {}): RoleNavItem[] {
  const items: RoleNavItem[] = [
    { path: '/dashboard', label: '我的工作台' },
    { path: '/video/create', label: '创建任务' },
    { path: '/video/tasks', label: '任务记录' },
    { path: '/keys', label: '我的密钥' },
    { path: '/usage', label: '我的花费', hideInSimpleMode: true },
  ]

  if (options.includeMoreGroup) {
    items.push({
      path: EMPLOYEE_MORE_PATH,
      label: '更多',
      expandOnly: true,
      defaultCollapsed: true,
      children: [
        { path: '/batch-image', label: '批量生图', hideInSimpleMode: true, featureFlagKey: 'batchImage' },
        { path: '/available-channels', label: '可用渠道', hideInSimpleMode: true, featureFlagKey: 'availableChannels' },
        { path: '/monitor', label: '渠道状态', featureFlagKey: 'channelMonitor' },
        { path: '/redeem', label: '兑换', hideInSimpleMode: true },
        ...(options.customUserMenus ?? []).map((menu): RoleNavItem => ({
          path: `/custom/${menu.id}`,
          label: menu.label,
          customMenuId: menu.id,
          iconSvg: menu.icon_svg,
        })),
      ],
    })
  }

  return applySimpleMode(applyFeatureFlags(items, options), options.isSimpleMode === true)
}
