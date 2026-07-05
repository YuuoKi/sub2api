import { describe, expect, it } from 'vitest'

import {
  DEFAULT_USD_CNY_RATE,
  formatByCurrency,
  formatCny,
  formatUsdAccountAmount,
  normalizeUsdCnyRate
} from '../useDisplayCurrency'

describe('useDisplayCurrency helpers', () => {
  it('formats USD-denominated costs as CNY using the configured rate', () => {
    expect(DEFAULT_USD_CNY_RATE).toBe(7.2)
    expect(formatCny(1.4, 7.2)).toBe('¥10.08')
    expect(formatByCurrency(2, 'USD', 7.35)).toBe('¥14.70')
  })

  it('keeps native CNY values in CNY without double conversion', () => {
    expect(formatByCurrency(5.0094, 'CNY', 7.2)).toBe('¥5.0094')
    expect(formatByCurrency(5.0094, 'cny', 9.9)).toBe('¥5.0094')
  })

  it('falls back to 7.2 for invalid rates', () => {
    expect(normalizeUsdCnyRate(undefined)).toBe(7.2)
    expect(normalizeUsdCnyRate(0)).toBe(7.2)
    expect(normalizeUsdCnyRate(Number.NaN)).toBe(7.2)
    expect(formatCny(1, 0)).toBe('¥7.20')
  })

  it('preserves account balance and quota amounts as USD', () => {
    expect(formatUsdAccountAmount(12.5)).toBe('$12.50')
    expect(formatUsdAccountAmount(0.004)).toBe('$0.0040')
  })
})
