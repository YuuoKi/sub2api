import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))

vi.mock('@/api/client', () => ({ apiClient: { post } }))

import { create, resetPassword } from '@/api/admin/users'

describe('admin temporary credential api', () => {
  beforeEach(() => post.mockReset())

  it('creates a user without accepting a client-supplied password', async () => {
    const response = {
      user: { id: 9, email: 'employee@example.com' },
      initial_credential: { temporary_password: 'secret', expires_at: '2026-07-17T00:00:00Z' }
    }
    post.mockResolvedValue({ data: response })

    const result = await create({ email: 'employee@example.com', username: 'Employee' })

    expect(post).toHaveBeenCalledWith('/admin/users', {
      email: 'employee@example.com',
      username: 'Employee'
    })
    expect(result).toEqual(response)
  })

  it('returns a one-time credential when an administrator resets a password', async () => {
    const response = {
      initial_credential: { temporary_password: 'new-secret', expires_at: '2026-07-17T00:00:00Z' }
    }
    post.mockResolvedValue({ data: response })

    const result = await resetPassword(9)

    expect(post).toHaveBeenCalledWith('/admin/users/9/reset-password')
    expect(result).toEqual(response.initial_credential)
  })
})
