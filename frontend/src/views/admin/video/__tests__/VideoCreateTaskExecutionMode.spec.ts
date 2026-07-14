import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

const createMock = vi.fn()
const listProvidersMock = vi.fn()

vi.mock('@/api/admin/video', () => ({
  videoTaskAPI: {
    listProviders: (...args: unknown[]) => listProvidersMock(...args),
    create: (...args: unknown[]) => createMock(...args),
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
    showInfo: vi.fn(),
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isAdmin: false,
  }),
}))

vi.mock('@/utils/productMode', () => ({
  isVideoGatewayDemoMode: true,
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
  RouterLink: { template: '<a><slot /></a>' },
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: { template: '<div><slot /></div>' },
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: { template: '<span />' },
}))

import VideoCreateTaskView from '../VideoCreateTaskView.vue'

describe('VideoCreateTaskView execution_mode', () => {
  beforeEach(() => {
    createMock.mockReset()
    listProvidersMock.mockReset()
    listProvidersMock.mockResolvedValue({
      items: [],
      execution_capabilities: { mock: true, review_real: true, internal_real: false },
    })
    createMock.mockResolvedValue({ id: 12 })
  })

  it('defaults to mock and never sends provider_account_id=0', async () => {
    const wrapper = mount(VideoCreateTaskView)
    await flushPromises()
    const select = wrapper.find('select')
    expect((select.element as HTMLSelectElement).value).toBe('mock')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()
    expect(createMock).toHaveBeenCalled()
    const payload = createMock.mock.calls[0][0]
    expect(payload.execution_mode).toBe('mock')
    expect(payload.provider_account_id).toBeUndefined()
  })

  it('reuses stable Idempotency-Key while in-flight double submit', async () => {
    let resolveCreate: (value: unknown) => void = () => undefined
    createMock.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveCreate = resolve
        }),
    )
    const wrapper = mount(VideoCreateTaskView)
    await flushPromises()
    const form = wrapper.find('form')
    void form.trigger('submit.prevent')
    await flushPromises()
    void form.trigger('submit.prevent')
    await flushPromises()
    expect(createMock).toHaveBeenCalledTimes(1)
    const key1 = createMock.mock.calls[0][1]
    expect(typeof key1).toBe('string')
    expect(key1.length).toBeGreaterThan(8)
    resolveCreate({ id: 99 })
    await flushPromises()
  })
})
