import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post, put } = vi.hoisted(() => ({
  post: vi.fn(),
  put: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { post, put },
}))

import { create, update } from '@/api/admin/users'
import type { UpdateUserRequest } from '@/types'

describe('admin users member_type contract', () => {
  beforeEach(() => {
    post.mockReset()
    put.mockReset()
  })

  it('forwards member_type on user creation (human/tool classification)', async () => {
    post.mockResolvedValue({ data: { user: {}, initial_credential: { temporary_password: 'x', expires_at: '' } } })

    await create({ email: 'tool@wujie.local', member_type: 'tool' })

    expect(post).toHaveBeenCalledWith('/admin/users', { email: 'tool@wujie.local', member_type: 'tool' })
  })

  it('forwards member_type on user update without dropping other fields', async () => {
    put.mockResolvedValue({ data: {} })

    const payload: UpdateUserRequest = { notes: 'n8n bot', member_type: 'tool' }
    await update(9, payload)

    expect(put).toHaveBeenCalledWith('/admin/users/9', payload)
  })
})
