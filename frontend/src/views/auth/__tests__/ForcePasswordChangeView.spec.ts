import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ForcePasswordChangeView from '@/views/auth/ForcePasswordChangeView.vue'

const { logoutMock, replaceMock, changePasswordMock } = vi.hoisted(() => ({
  logoutMock: vi.fn(),
  replaceMock: vi.fn(),
  changePasswordMock: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ replace: replaceMock }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ logout: logoutMock }),
}))

vi.mock('@/api/user', () => ({
  userAPI: { changePassword: changePasswordMock },
}))

describe('ForcePasswordChangeView', () => {
  beforeEach(() => {
    logoutMock.mockReset()
    replaceMock.mockReset()
    changePasswordMock.mockReset()
    logoutMock.mockResolvedValue(undefined)
    replaceMock.mockResolvedValue(undefined)
  })

  it('offers an explicit logout action that clears the session and returns to login', async () => {
    const wrapper = mount(ForcePasswordChangeView)
    const logoutButton = wrapper
      .findAll('button')
      .find((button) => button.text().trim() === '退出登录')

    expect(logoutButton).toBeDefined()
    expect(logoutButton?.attributes('type')).toBe('button')

    await logoutButton?.trigger('click')
    await flushPromises()

    expect(logoutMock).toHaveBeenCalledTimes(1)
    expect(replaceMock).toHaveBeenCalledWith('/login')
  })
})
