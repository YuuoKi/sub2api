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
  lanAdminOnly?: boolean
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
  const items: RoleNavItem[] = [
    { path: '/admin/console/overview', label: '总览与成本' },
    { path: '/admin/console/key-vault', label: '上游账号、模型和通道' },
    { path: '/admin/console/staff', label: '员工/API 卡片管理' },
    { path: '/admin/video/tasks', label: '调用、任务与资产记录' },
    {
      path: ADMIN_SYSTEM_PATH,
      label: '系统、健康、备份与恢复',
      expandOnly: true,
      defaultCollapsed: true,
      children: [
        {
          path: '/admin/system/upstream',
          label: '上游账号、模型和通道',
          expandOnly: true,
          defaultCollapsed: true,
          children: [
            { path: '/admin/accounts', label: '上游账号' },
            { path: '/admin/groups', label: '模型分组', hideInSimpleMode: true },
            { path: '/admin/video/providers', label: '视频通道' },
            { path: '/admin/channels/pricing', label: '模型与通道定价', hideInSimpleMode: true },
            { path: '/admin/channels/monitor', label: '通道监控', featureFlagKey: 'channelMonitor' },
            { path: '/admin/proxies', label: '上游网络' },
          ],
        },
        {
          path: '/admin/system/records-assets',
          label: '调用、任务与资产记录',
          expandOnly: true,
          defaultCollapsed: true,
          children: [
            { path: '/admin/console/ai-records', label: '调用记录' },
            { path: '/admin/generation-content', label: '生成资产' },
            { path: '/admin/usage', label: '用量与成本' },
          ],
        },
        {
          path: '/admin/system/health-recovery',
          label: '系统、健康、备份与恢复',
          expandOnly: true,
          defaultCollapsed: true,
          children: [
            { path: '/admin/ops', label: '系统健康', featureFlagKey: 'ops' },
            { path: '/admin/video/system-check', label: '视频链路检查' },
            { path: '/admin/settings', label: '系统、备份与恢复' },
          ],
        },
      ],
    },
  ]

  return applySimpleMode(applyFeatureFlags(items, options), options.isSimpleMode === true)
}

export function buildEmployeeRoleNav(options: BuildEmployeeRoleNavOptions = {}): RoleNavItem[] {
  if (options.lanAdminOnly === true) return []

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
