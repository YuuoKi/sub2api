/**
 * 控制台共享工具：金额 / 数字 / 日期格式化与时间范围计算。
 */
import { formatCny, formatUsdAccountAmount } from '../../../composables/useDisplayCurrency'

export function formatLocalDate(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

export type ConsoleRangeKey = '7d' | '30d' | 'month'

export function getConsoleRange(key: ConsoleRangeKey): { start: string; end: string } {
  const end = new Date()
  let start: Date
  if (key === 'month') {
    start = new Date(end.getFullYear(), end.getMonth(), 1)
  } else {
    const days = key === '7d' ? 7 : 30
    start = new Date(end.getTime() - days * 24 * 60 * 60 * 1000)
  }
  return { start: formatLocalDate(start), end: formatLocalDate(end) }
}

export function formatMoney(value?: number | null, usdCnyRate?: number | null): string {
  return formatCny(value, usdCnyRate)
}

export function formatAccountUsd(value?: number | null): string {
  return formatUsdAccountAmount(value)
}

export function formatCount(value?: number | null): string {
  const n = Number(value ?? 0)
  if (!Number.isFinite(n)) return '0'
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 10_000) return `${(n / 1_000).toFixed(1)}K`
  return n.toLocaleString()
}

export function formatTokens(value?: number | null): string {
  const n = Number(value ?? 0)
  if (!Number.isFinite(n)) return '0'
  if (n >= 1_000_000_000) return `${(n / 1_000_000_000).toFixed(2)}B`
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return n.toLocaleString()
}

export function formatDateTime(value?: string | null): string {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '—'
  return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

export function formatDuration(ms?: number | null): string {
  const n = Number(ms ?? 0)
  if (!Number.isFinite(n) || n <= 0) return '—'
  if (n >= 60_000) return `${(n / 60_000).toFixed(1)} 分钟`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)} 秒`
  return `${Math.round(n)} 毫秒`
}

/** 员工显示名：优先用户名，其次邮箱前缀。 */
export function staffDisplayName(username?: string | null, email?: string | null): string {
  const name = (username || '').trim()
  if (name) return name
  const mail = (email || '').trim()
  if (mail.includes('@')) return mail.split('@')[0]
  return mail || '未知员工'
}

/**
 * 控制台中文错误码表：后端 reason → 人话。
 * 高管控制台禁止英文后端原文直出 toast；命中表的走译文，未命中的回退到后端 message。
 */
export const CONSOLE_ERROR_ZH: Record<string, string> = {
  probe_invalid_baseurl: '此接口地址不支持简易连通测试，请直接使用或改用完整「上游账号」页诊断',
  internal_error: '服务器内部错误，请稍后重试；反复出现请去「系统」页看健康检查',
  ADMIN_COMPLIANCE_ACK_REQUIRED: '需要先在页面上完成管理员合规确认',
  CANNOT_DISABLE_ADMIN_USER: '不能停用管理员账号（避免把自己锁出管理端）',
  CANNOT_DELETE_ADMIN_USER: '不能删除管理员账号（避免把自己锁出管理端）',
  CANNOT_DEMOTE_LAST_ADMIN: '不能降级最后一个管理员账号（避免锁出管理端）',
  CANNOT_ACTIVATE_QUOTA_EXHAUSTED: '额度仍用尽，请先重置额度再启用',
  CANNOT_ACTIVATE_EXPIRED: '密钥已过期，请先延长有效期再启用',
}

export type QuotaWarningLevel = 'none' | 'warn' | 'critical'

/** Derive quota warning level from used/limit (mirrors backend QuotaUsagePercent). */
export function quotaWarningLevel(used?: number | null, limit?: number | null): QuotaWarningLevel {
  const quota = Number(limit ?? 0)
  if (!Number.isFinite(quota) || quota <= 0) return 'none'
  const usedAmt = Number(used ?? 0)
  const safeUsed = Number.isFinite(usedAmt) && usedAmt > 0 ? usedAmt : 0
  const ratio = safeUsed / quota
  if (ratio >= 1) return 'critical'
  if (ratio >= 0.8) return 'warn'
  return 'none'
}

/** Text color classes for quota usage (reuse KeysView yellow/red tokens). */
export function quotaWarningTextClass(level: QuotaWarningLevel): string {
  switch (level) {
    case 'critical':
      return 'text-red-500'
    case 'warn':
      return 'text-yellow-500'
    case 'none':
      return 'text-gray-600 dark:text-gray-300'
    default: {
      const _exhaustive: never = level
      throw new Error(`Unhandled QuotaWarningLevel: ${String(_exhaustive)}`)
    }
  }
}

export function quotaWarningBarClass(level: QuotaWarningLevel): string {
  switch (level) {
    case 'critical':
      return 'bg-red-500'
    case 'warn':
      return 'bg-yellow-500'
    case 'none':
      return 'bg-ui-accent'
    default: {
      const _exhaustive: never = level
      throw new Error(`Unhandled QuotaWarningLevel: ${String(_exhaustive)}`)
    }
  }
}

/** Parse /admin/console/ai-records query for overview drill-down. */
export function parseAiRecordsQuery(query: Record<string, unknown> | { [key: string]: unknown }): {
  userId: number
  model: string
  tab: 'logs' | 'prompts'
} {
  const rawUser = query.user_id
  const userStr = Array.isArray(rawUser) ? String(rawUser[0] ?? '') : String(rawUser ?? '')
  const parsed = Number.parseInt(userStr, 10)
  const userId = Number.isFinite(parsed) && parsed > 0 ? parsed : 0

  const rawModel = query.model
  const model = (Array.isArray(rawModel) ? String(rawModel[0] ?? '') : String(rawModel ?? '')).trim()

  const rawTab = query.tab
  const tabStr = Array.isArray(rawTab) ? String(rawTab[0] ?? '') : String(rawTab ?? '')
  const tab = tabStr === 'prompts' ? 'prompts' : 'logs'

  return { userId, model, tab }
}
