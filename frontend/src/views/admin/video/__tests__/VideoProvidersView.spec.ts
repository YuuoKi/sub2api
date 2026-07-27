import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import VideoProvidersView from '../VideoProvidersView.vue'

const mocks = vi.hoisted(() => ({
  authorizeTinyReal: vi.fn(),
  contract: vi.fn(),
  listProviders: vi.fn(),
  listTasks: vi.fn(),
  createProvider: vi.fn(),
  getAllGroups: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    video: {
      contract: mocks.contract,
      listProviders: mocks.listProviders,
      listTasks: mocks.listTasks,
      createProvider: mocks.createProvider,
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
      resolution: '720p',
      platforms: [
        {
          provider: 'seedance',
          display_name: '官方 Ark Seedance',
          default_base_url: 'https://example.invalid',
          default_model: 'seedance-2-0-250428',
          adapter_ready: true
        },
        {
          provider: 'hc_atom_seedance_v3',
          display_name: 'HC-ATOM Seedance 2.0 V3',
          default_base_url: 'https://api-aigc.fzyinghe.com',
          default_model: 'doubao-seedance-2.0',
          adapter_ready: true
        }
      ]
    })
    mocks.listProviders.mockResolvedValue({ items: [provider] })
    mocks.listTasks.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 100, pages: 0 })
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
    expect(dialog?.textContent).toContain('剩余授权次数')
    // 「不可用（后端未提供）」假字段已按改版方案移除：不再渲染无信息占位行
    expect(dialog?.textContent).not.toContain('预算上限')
    expect(dialog?.textContent).not.toContain('不可用（后端未提供）')
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

  it('creates HC-ATOM as a distinct fixed provider without automatic real-call authorization', async () => {
    mocks.createProvider.mockResolvedValue({
      ...provider,
      id: 8,
      provider: 'hc_atom_seedance_v3',
      display_name: 'HC V3'
    })
    mocks.listProviders
      .mockResolvedValueOnce({ items: [provider] })
      .mockResolvedValueOnce({ items: [provider] })

    const wrapper = mount(VideoProvidersView, {
      global: { stubs: { AppLayout: AppLayoutStub, Icon: true } }
    })
    await flushPromises()

    await wrapper.get('#video-provider-kind').setValue('hc_atom_seedance_v3')
    await wrapper.get('#video-provider-name').setValue('HC V3')
    await wrapper.get('#video-provider-group').setValue('3')
    await wrapper.get('#video-provider-secret').setValue('fake-local-secret')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(mocks.createProvider).toHaveBeenCalledWith(expect.objectContaining({
      provider: 'hc_atom_seedance_v3',
      display_name: 'HC V3',
      group_id: 3
    }))
    expect(mocks.authorizeTinyReal).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('https://api-aigc.fzyinghe.com')
  })

  it('shows the latest sanitized task error for the matching provider account', async () => {
    mocks.listTasks.mockResolvedValue({
      items: [{
        provider_account_id: 7,
        provider_error_message: 'upstream provider operation failed',
        error_message: '',
      }],
      total: 1,
      page: 1,
      page_size: 100,
      pages: 1,
    })

    const wrapper = mount(VideoProvidersView, {
      global: { stubs: { AppLayout: AppLayoutStub, Icon: true } }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('最近错误')
    expect(wrapper.text()).toContain('upstream provider operation failed')
  })
})
