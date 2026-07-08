import { describe, expect, it } from 'vitest'
import {
  quotaWarningBarClass,
  quotaWarningLevel,
  quotaWarningTextClass,
} from '../consoleUtils'

describe('quotaWarningLevel', () => {
  it('returns none for unlimited or below 80%', () => {
    expect(quotaWarningLevel(100, 0)).toBe('none')
    expect(quotaWarningLevel(7.9, 10)).toBe('none')
  })

  it('returns warn at 80% and critical at 100%', () => {
    expect(quotaWarningLevel(8, 10)).toBe('warn')
    expect(quotaWarningLevel(10, 10)).toBe('critical')
    expect(quotaWarningLevel(12, 10)).toBe('critical')
  })
})

describe('quotaWarning classes', () => {
  it('maps levels to yellow/red tokens', () => {
    expect(quotaWarningTextClass('warn')).toContain('yellow')
    expect(quotaWarningTextClass('critical')).toContain('red')
    expect(quotaWarningBarClass('warn')).toContain('yellow')
    expect(quotaWarningBarClass('critical')).toContain('red')
  })
})
