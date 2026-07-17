import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import StaffView from '../StaffView.vue'

const mocks = vi.hoisted(() => ({
  usersList: vi.fn(),
  getBatchUsersUsage: vi.fn(),
  getStats: vi.fn(),
  createApiKeyForUser: vi.fn(),
  getUserApiKeys: vi.fn(),
  getBatchApiKeysUsage: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: {
      list: mocks.usersList,
      getUserApiKeys: mocks.getUserApiKeys,
    },
    dashboard: {
      getBatchUsersUsage: mocks.getBatchUsersUsage,
      getBatchApiKeysUsage: mocks.getBatchApiKeysUsage,
      getStats: mocks.getStats,
    },
    apiKeys: {
      createApiKeyForUser: mocks.createApiKeyForUser,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

const AppLayoutStub = { template: '<div><slot /></div>' }
const IconStub = { template: '<i />' }

const FULL_KEY = 'sk-once-visible-1234567890abcdef'

describe('StaffView key-once issuance', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    window.localStorage.clear()
    mocks.usersList.mockResolvedValue({
      items: [{ id: 1, username: '张三', email: 'zhangsan@example.com', member_type: 'human', status: 'active', notes: '' }],
      total: 1,
    })
    mocks.getBatchUsersUsage.mockResolvedValue({ stats: {} })
    mocks.getStats.mockResolvedValue({ usd_cny_rate: 7.2 })
    // Real backend truth: admin list-keys endpoints always return an empty `key` (apiKeyDTOWithoutSecret).
    mocks.getUserApiKeys.mockResolvedValue({ items: [{ id: 10, name: 'card', key: '', status: 'active', quota: 0, quota_used: 0, last_used_at: null }] })
    mocks.getBatchApiKeysUsage.mockResolvedValue({ stats: {} })
    mocks.createApiKeyForUser.mockResolvedValue({ id: 11, name: 'zhangsan-生产卡', key: FULL_KEY, status: 'active', quota: 0, quota_used: 0 })
  })

  it('shows the full key exactly once right after creation, and never persists it to localStorage', async () => {
    const setItemSpy = vi.spyOn(window.localStorage, 'setItem')
    const wrapper = mount(StaffView, {
      global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub } },
    })
    await flushPromises()

    await wrapper.find('[data-test="issue-card"]').trigger('click')
    await wrapper.find('form[data-test="issue-card-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.createApiKeyForUser).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain(FULL_KEY)

    for (const call of setItemSpy.mock.calls) {
      expect(String(call[1])).not.toContain(FULL_KEY)
    }
  })

  it('masks the key again once the issuance dialog is closed and reopened', async () => {
    const wrapper = mount(StaffView, {
      global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub } },
    })
    await flushPromises()

    await wrapper.find('[data-test="issue-card"]').trigger('click')
    await wrapper.find('form[data-test="issue-card-form"]').trigger('submit.prevent')
    await flushPromises()
    expect(wrapper.text()).toContain(FULL_KEY)

    await wrapper.find('[data-test="issue-card-done"]').trigger('click')
    await wrapper.find('[data-test="issue-card"]').trigger('click')

    expect(wrapper.text()).not.toContain(FULL_KEY)
  })

  it('only ever renders masked key metadata in the staff key list, never the full value', async () => {
    const wrapper = mount(StaffView, {
      global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub } },
    })
    await flushPromises()
    await wrapper.find('[data-test="toggle-expand"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).not.toContain(FULL_KEY)
    expect(wrapper.text()).toContain('不再显示')
  })

  it('fails closed: even if a list DTO unexpectedly carries a non-empty key, no substring of it is ever rendered', async () => {
    // Contract says list endpoints always zero out `key`, but the UI must not
    // trust that and must never render any part of it even if it slips through.
    mocks.getUserApiKeys.mockResolvedValue({
      items: [{ id: 10, name: 'card', key: FULL_KEY, status: 'active', quota: 0, quota_used: 0, last_used_at: null }],
    })

    const wrapper = mount(StaffView, {
      global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub } },
    })
    await flushPromises()
    await wrapper.find('[data-test="toggle-expand"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).not.toContain(FULL_KEY)
    // Not even a prefix/suffix slice of the leaked key should leak into the DOM.
    expect(wrapper.text()).not.toContain(FULL_KEY.slice(0, 8))
    expect(wrapper.text()).not.toContain(FULL_KEY.slice(-4))
    expect(wrapper.text()).toContain('不再显示')
  })
})
