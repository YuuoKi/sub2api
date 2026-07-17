import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { get },
}))

import videoAPI from '@/api/admin/video'

describe('admin video get-task adapter: pricing provenance and local-asset contract', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('maps a realistic snake_case backend payload through untouched, exposing pricing and local-asset fields', async () => {
    // Realistic backend response shape (Task 2C1/2C2 canonical pricing + local-asset persistence).
    const payload = {
      id: 7,
      api_key_id: 3,
      group_id: 1,
      provider_account_id: 2,
      provider: 'seedance',
      model: 'doubao-seedance-2-0-260128',
      task_type: 'text_to_video',
      prompt: 'a shot of the harbor at dawn',
      status: 'succeeded',
      request_model: 'doubao-seedance-2-0-260128',
      request_duration_seconds: 4,
      request_resolution: '720p',
      upstream_model: 'doubao-seedance-2-0-260128',
      upstream_duration_seconds: 4,
      upstream_resolution: '720p',
      billing_model: 'doubao-seedance-2-0-260128',
      billing_duration_seconds: 4,
      billing_resolution: '720p',
      upstream_task_id: 'up-123',
      result_url: 'https://cdn.example.com/result.mp4',
      last_frame_url: 'https://cdn.example.com/last-frame.jpg',
      error_message: '',
      provider_error_code: '',
      duration_seconds: 4,
      resolution: '720p',
      provider_error_message: '',
      reserved_cost_usd: 0.42,
      reservation_state: 'consumed',
      reserved_at: '2026-07-17T00:00:00Z',
      reservation_window_5h_start: null,
      reservation_window_1d_start: null,
      reservation_window_7d_start: null,
      cost_amount: 0.42,
      provider_actual_cost_usd: 0.4,
      currency: 'USD',
      balance_before_usd: 10,
      balance_after_usd: 9.6,
      balance_delta_usd: -0.4,
      authorization_consumed_at: '2026-07-17T00:00:00Z',
      authorization_consumed_by: 3,
      real_dispatch_count: 1,
      dispatch_state: 'dispatched',
      created_by: 3,
      created_at: '2026-07-16T23:59:00Z',
      updated_at: '2026-07-17T00:00:05Z',
      // Reviewed backend truth (Task 2C1 canonical pricing provenance):
      pricing_source: 'canonical',
      pricing_version: 'v1',
      pricing_cny_per_million_completion_tokens: 12.5,
      pricing_usd_cny_exchange_rate: 7.2,
      pricing_maximum_cny: 999,
      // Reviewed backend truth (Task 2C2 local-asset persistence):
      local_asset_available: true,
      local_asset_download_url: '/admin/video/tasks/7/local-asset',
      local_asset_saved_at: '2026-07-17T00:00:10Z',
    }
    get.mockResolvedValue({ data: payload })

    const result = await videoAPI.getTask(7)

    expect(get).toHaveBeenCalledWith('/admin/video/tasks/7')

    // Pricing provenance
    expect(result.pricing_source).toBe('canonical')
    expect(result.pricing_version).toBe('v1')
    expect(result.pricing_cny_per_million_completion_tokens).toBe(12.5)
    expect(result.pricing_usd_cny_exchange_rate).toBe(7.2)
    expect(result.pricing_maximum_cny).toBe(999)

    // Local-asset persistence
    expect(result.local_asset_available).toBe(true)
    expect(result.local_asset_download_url).toBe('/admin/video/tasks/7/local-asset')
    expect(result.local_asset_saved_at).toBe('2026-07-17T00:00:10Z')
  })

  it('passes through null/absent pricing and local-asset fields without fabricating defaults', async () => {
    const payload = {
      id: 8,
      api_key_id: 3,
      group_id: 1,
      provider_account_id: 2,
      provider: 'seedance',
      model: 'doubao-seedance-2-0-260128',
      task_type: 'text_to_video',
      prompt: 'p',
      status: 'failed',
      request_model: 'm',
      request_duration_seconds: 4,
      request_resolution: '720p',
      upstream_model: null,
      upstream_duration_seconds: null,
      upstream_resolution: null,
      billing_model: null,
      billing_duration_seconds: null,
      billing_resolution: null,
      upstream_task_id: '',
      result_url: '',
      last_frame_url: '',
      error_message: 'upstream timeout',
      provider_error_code: 'TIMEOUT',
      duration_seconds: 4,
      resolution: '720p',
      provider_error_message: 'timed out',
      reserved_cost_usd: 0,
      reservation_state: 'released',
      reserved_at: null,
      reservation_window_5h_start: null,
      reservation_window_1d_start: null,
      reservation_window_7d_start: null,
      cost_amount: 0,
      provider_actual_cost_usd: 0,
      currency: 'USD',
      balance_before_usd: null,
      balance_after_usd: null,
      balance_delta_usd: null,
      authorization_consumed_at: null,
      authorization_consumed_by: null,
      real_dispatch_count: 0,
      dispatch_state: 'failed',
      created_by: 3,
      created_at: '2026-07-16T23:59:00Z',
      updated_at: '2026-07-17T00:00:05Z',
      // No pricing provenance and no local-asset fields at all on this task.
      local_asset_available: false,
      local_asset_download_url: null,
      local_asset_saved_at: null,
    }
    get.mockResolvedValue({ data: payload })

    const result = await videoAPI.getTask(8)

    expect(get).toHaveBeenCalledWith('/admin/video/tasks/8')
    expect(result.pricing_source).toBeUndefined()
    expect(result.pricing_version).toBeUndefined()
    expect(result.local_asset_available).toBe(false)
    expect(result.local_asset_download_url).toBeNull()
    expect(result.local_asset_saved_at).toBeNull()
  })
})
