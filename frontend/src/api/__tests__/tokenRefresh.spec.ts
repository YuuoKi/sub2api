import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import axios from 'axios'
import { setActivePinia, createPinia } from 'pinia'
import { refreshAccessTokenOnce, resetTokenRefreshStateForTests } from '@/api/tokenRefresh'

vi.mock('@/i18n', () => ({
  getLocale: () => 'zh-CN',
}))

const refreshPayload = {
  code: 0,
  data: {
    access_token: 'new-access-token',
    refresh_token: 'new-refresh-token',
    expires_in: 3600,
  },
}

describe('refreshAccessTokenOnce', () => {
  beforeEach(() => {
    localStorage.clear()
    resetTokenRefreshStateForTests()
    vi.clearAllMocks()
  })

  afterEach(() => {
    resetTokenRefreshStateForTests()
    vi.restoreAllMocks()
  })

  it('deduplicates concurrent refresh calls into a single POST /auth/refresh', async () => {
    localStorage.setItem('refresh_token', 'old-refresh-token')

    const postSpy = vi.spyOn(axios, 'post').mockImplementation(async () => {
      await new Promise((resolve) => setTimeout(resolve, 20))
      return { data: refreshPayload }
    })

    const [first, second] = await Promise.all([
      refreshAccessTokenOnce(),
      refreshAccessTokenOnce(),
    ])

    expect(postSpy).toHaveBeenCalledTimes(1)
    expect(first).toEqual(second)
    expect(first.access_token).toBe('new-access-token')
    expect(localStorage.getItem('auth_token')).toBe('new-access-token')
    expect(localStorage.getItem('refresh_token')).toBe('new-refresh-token')
  })

  it('syncs Pinia auth store after a successful refresh', async () => {
    localStorage.setItem('refresh_token', 'old-refresh-token')

    setActivePinia(createPinia())
    const { useAuthStore } = await import('@/stores/auth')
    const store = useAuthStore()
    store.$patch({
      token: 'old-access-token',
      user: { id: 1, username: 'user' } as never,
    })

    vi.spyOn(axios, 'post').mockResolvedValue({ data: refreshPayload })

    await refreshAccessTokenOnce()

    expect(store.token).toBe('new-access-token')
    expect(localStorage.getItem('auth_token')).toBe('new-access-token')
    expect(localStorage.getItem('refresh_token')).toBe('new-refresh-token')
  })
})
