import { beforeEach, describe, expect, it, vi } from 'vitest'

const { put } = vi.hoisted(() => ({ put: vi.fn() }))

vi.mock('@/api/client', () => ({
  apiClient: { put },
}))

import { dashboardAPI, updateMonthlyBudget } from '@/api/admin/dashboard'

describe('admin dashboard monthly budget api adapter', () => {
  beforeEach(() => {
    put.mockReset()
  })

  it('updates the company monthly budget via PUT /admin/dashboard/monthly-budget', async () => {
    const response = {
      monthly_budget_cny: 1000,
      monthly_spend_cny: 42.5,
      monthly_budget_usage_percent: 4.25,
    }
    put.mockResolvedValue({ data: response })

    const result = await updateMonthlyBudget(1000)

    expect(put).toHaveBeenCalledWith('/admin/dashboard/monthly-budget', { monthly_budget_cny: 1000 })
    expect(result).toEqual(response)
  })

  it('exposes updateMonthlyBudget on the dashboard barrel', () => {
    expect(dashboardAPI.updateMonthlyBudget).toBe(updateMonthlyBudget)
  })
})
