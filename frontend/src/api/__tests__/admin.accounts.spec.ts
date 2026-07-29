import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { post },
}))

import { create } from '@/api/admin/accounts'

describe('admin accounts API adapter', () => {
  beforeEach(() => {
    post.mockReset()
  })

  it('sends the stable UI-session Idempotency-Key required by account creation', async () => {
    const payload = {
      name: 'HC 图片账号',
      platform: 'hc_atom',
      type: 'apikey',
      credentials: { api_key: 'test-only' },
      group_ids: [8],
    }
    post.mockResolvedValue({ data: { id: 10, ...payload } })

    await create(payload, 'account-create-session-1')

    expect(post).toHaveBeenCalledWith(
      '/admin/accounts',
      payload,
      { headers: { 'Idempotency-Key': 'account-create-session-1' } },
    )
  })
})
