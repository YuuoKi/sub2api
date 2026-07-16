import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import VideoProvidersView from '../VideoProvidersView.vue'

const mocks = vi.hoisted(() => ({
  authorizeTinyReal: vi.fn(),
  contract: vi.fn(),
  listProviders: vi.fn(),
  getAllGroups: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    video: {
      contract: mocks.contract,
      listProviders: mocks.listProviders,
      createProvider: vi.fn(),
      updateProvider: vi.fn(),
      authorizeTinyReal: mocks.authorizeTinyReal
    },
    groups: { getAll: mocks.getAllGroups }
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({ showError: mocks.showError, showSuccess: mocks.showSuccess })
}))

const AppLayoutStub = defineComponent({
  name: 'AppLayout',
  template: '<main><slot /></main>'
})

const provider = {
  id: 7,
  group_id: 3,
  group_name: '制作一组',
  provider: 'seedance' as const,
  display_name: 'Seedance 正式通道',
  enabled: true,
  api_key_configured: true,
  masked_key: 'sk-a••••9z',
  base_url: '',
  default_model: 'seedance-2-0-250428'
}

describe('VideoProvidersView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.contract.mockResolvedValue({
      provider: 'seedance',
      base_url: 'https://example.invalid',
      default_model: 'seedance-2-0-250428',
      duration_seconds: 4,
      resolution: '720p'
    })
    mocks.listProviders.mockResolvedValue({ items: [provider] })
    mocks.getAllGroups.mockResolvedValue([{ id: 3, name: '制作一组', subscription_type: 'standard' }])
    mocks.authorizeTinyReal.mockResolvedValue({ ...provider, tiny_real_authorized_at: '2026-07-16T08:00:00Z' })
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('opens an accessible in-app authorization dialog and cancel restores focus without an API call', async () => {
    const wrapper = mount(VideoProvidersView, {
      attachTo: document.body,
      global: { stubs: { AppLayout: AppLayoutStub, Icon: true } }
    })
    await flushPromises()

    const trigger = wrapper.get('[data-testid="authorize-provider-7"]')
    ;(trigger.element as HTMLButtonElement).focus()
    await trigger.trigger('click')
    await flushPromises()

    const dialog = document.body.querySelector<HTMLElement>('[role="dialog"]')
    expect(dialog).not.toBeNull()
    expect(dialog?.textContent).toContain('4 秒')
    expect(dialog?.textContent).toContain('720p')
    expect(dialog?.textContent).toContain('制作一组')
    expect(dialog?.textContent).toContain('预算上限')
    expect(dialog?.textContent).toContain('剩余授权次数')
    expect(dialog?.textContent).toContain('不可用（后端未提供）')
    expect(dialog?.textContent).not.toContain('当前登录管理员')

    document.body.querySelector<HTMLButtonElement>('[data-testid="cancel-video-authorization"]')?.click()
    await flushPromises()

    expect(mocks.authorizeTinyReal).not.toHaveBeenCalled()
    expect(document.activeElement).toBe(trigger.element)
  })

  it('calls the atomic authorization endpoint only after explicit confirmation', async () => {
    mount(VideoProvidersView, {
      attachTo: document.body,
      global: { stubs: { AppLayout: AppLayoutStub, Icon: true } }
    })
    await flushPromises()

    document.body.querySelector<HTMLButtonElement>('[data-testid="authorize-provider-7"]')?.click()
    await flushPromises()
    document.body.querySelector<HTMLButtonElement>('[data-testid="confirm-video-authorization"]')?.click()
    await flushPromises()

    expect(mocks.authorizeTinyReal).toHaveBeenCalledTimes(1)
    expect(mocks.authorizeTinyReal).toHaveBeenCalledWith(7)
  })

  it('labels every form control and exposes masked saved state plus a disabled reason', async () => {
    mocks.listProviders.mockResolvedValue({
      items: [{ ...provider, enabled: false, api_key_configured: false }]
    })
    const wrapper = mount(VideoProvidersView, {
      global: { stubs: { AppLayout: AppLayoutStub, Icon: true } }
    })
    await flushPromises()

    expect(wrapper.get('label[for="video-provider-name"]').text()).toBe('通道名称')
    expect(wrapper.get('label[for="video-provider-group"]').text()).toContain('员工组')
    expect(wrapper.get('label[for="video-provider-secret"]').text()).toContain('API Key')
    expect(wrapper.text()).toContain('保存后摘要')
    expect(wrapper.text()).toContain(provider.masked_key)
    expect(wrapper.get('[data-testid="authorize-provider-7"]').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('通道已停用；先启用通道。')
  })
})
