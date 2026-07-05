import { describe, expect, it, vi, beforeEach } from 'vitest'

import { useAdminDisplayCurrencyRate } from '../useAdminDisplayCurrencyRate'

const { getStats } = vi.hoisted(() => ({
  getStats: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    dashboard: {
      getStats
    }
  }
}))

describe('useAdminDisplayCurrencyRate', () => {
  beforeEach(() => {
    getStats.mockReset()
  })

  it('loads the admin USD/CNY rate from dashboard stats', async () => {
    getStats.mockResolvedValue({ usd_cny_rate: 7.5 })
    const { usdCnyRate, loadUsdCnyRate } = useAdminDisplayCurrencyRate()

    await loadUsdCnyRate()

    expect(usdCnyRate.value).toBe(7.5)
  })

  it('falls back to the default rate when dashboard stats cannot provide a valid rate', async () => {
    getStats.mockResolvedValue({ usd_cny_rate: -1 })
    const { usdCnyRate, loadUsdCnyRate } = useAdminDisplayCurrencyRate()

    await loadUsdCnyRate()

    expect(usdCnyRate.value).toBe(7.2)
  })
})
