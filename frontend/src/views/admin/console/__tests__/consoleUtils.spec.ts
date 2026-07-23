import { describe, expect, it } from 'vitest'
import {
  parseAiRecordsQuery,
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

  it('handles the none variant explicitly (exhaustive switch, no string fallthrough)', () => {
    expect(quotaWarningTextClass('none')).toContain('gray')
    // 正常态走全站统一 accent（bg-ui-accent），不再单独使用 teal
    expect(quotaWarningBarClass('none')).toContain('ui-accent')
  })
})

describe('parseAiRecordsQuery', () => {
  it('parses user_id model and prompts tab', () => {
    expect(parseAiRecordsQuery({ user_id: '42', model: ' seedance ', tab: 'prompts' })).toEqual({
      userId: 42,
      model: 'seedance',
      tab: 'prompts',
    })
  })

  it('defaults invalid values', () => {
    expect(parseAiRecordsQuery({ user_id: '0', tab: 'other' })).toEqual({
      userId: 0,
      model: '',
      tab: 'logs',
    })
  })
})
