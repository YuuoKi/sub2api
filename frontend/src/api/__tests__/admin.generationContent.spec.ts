import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { get, post },
}))

import {
  generationContentAPI,
  getSamples,
  getStats,
  getWeeklyReport,
  updateAdoption,
} from '@/api/admin/generation_content'

describe('admin generation-content api adapter', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
  })

  it('reads capture stats from the real backend endpoint', async () => {
    const stats = {
      captured_today: 1,
      captured_week: 2,
      distinct_employees: 1,
      distinct_teams: 1,
      distinct_models: 1,
      total_bytes: 128,
      daily_rate: 0.3,
      daily_series: [{ date: '2026-07-03', count: 1 }],
      is_live: true,
    }
    get.mockResolvedValue({ data: stats })

    const result = await getStats()

    expect(get).toHaveBeenCalledWith('/admin/generation-content/stats', { signal: undefined })
    expect(result).toEqual(stats)
  })

  it('reads recent redacted samples from the real backend endpoint', async () => {
    const response = { samples: [], is_live: false, usd_cny_rate: 7.3 }
    get.mockResolvedValue({ data: response })

    const result = await getSamples()

    expect(get).toHaveBeenCalledWith('/admin/generation-content/samples', { signal: undefined })
    expect(result).toEqual(response)
  })

  it('reads the weekly report from the real backend endpoint', async () => {
    const report = { period_start: '2026-07-01T00:00:00Z', period_end: '2026-07-08T00:00:00Z' }
    get.mockResolvedValue({ data: report })

    const result = await getWeeklyReport()

    expect(get).toHaveBeenCalledWith('/admin/generation-content/weekly-report', { signal: undefined })
    expect(result).toEqual(report)
  })

  it('posts adoption feedback to the per-task real backend endpoint', async () => {
    const response = { enabled: true, saved: true, task_id: 42, adoption_status: 'adopted' }
    post.mockResolvedValue({ data: response })

    const result = await updateAdoption(42, { adoption_status: 'adopted' })

    expect(post).toHaveBeenCalledWith('/admin/generation-content/42/adoption', { adoption_status: 'adopted' })
    expect(result).toEqual(response)
  })

  it('exposes a barrel object matching the module functions', () => {
    expect(generationContentAPI.getStats).toBe(getStats)
    expect(generationContentAPI.getSamples).toBe(getSamples)
    expect(generationContentAPI.getWeeklyReport).toBe(getWeeklyReport)
    expect(generationContentAPI.updateAdoption).toBe(updateAdoption)
  })
})
