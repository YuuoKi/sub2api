import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import StaffView from '../StaffView.vue'

const mocks = vi.hoisted(() => ({
  usersList: vi.fn(),
  usersCreate: vi.fn(),
  usersUpdateBalance: vi.fn(),
  getBatchUsersUsage: vi.fn(),
  getStats: vi.fn(),
  createApiKeyForUser: vi.fn(),
	createQCanvasKeyPairForUser: vi.fn(),
  getUserApiKeys: vi.fn(),
  getBatchApiKeysUsage: vi.fn(),
  groupsGetAll: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: {
      list: mocks.usersList,
      create: mocks.usersCreate,
      updateBalance: mocks.usersUpdateBalance,
      getUserApiKeys: mocks.getUserApiKeys,
    },
    dashboard: {
      getBatchUsersUsage: mocks.getBatchUsersUsage,
      getBatchApiKeysUsage: mocks.getBatchApiKeysUsage,
      getStats: mocks.getStats,
    },
    apiKeys: {
      createApiKeyForUser: mocks.createApiKeyForUser,
		createQCanvasKeyPairForUser: mocks.createQCanvasKeyPairForUser,
    },
    groups: {
      getAll: mocks.groupsGetAll,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

const AppLayoutStub = { template: '<div><slot /></div>' }
const IconStub = { template: '<i />' }
const RouterLinkStub = { template: '<a><slot /></a>' }

const FULL_KEY = 'sk-once-visible-1234567890abcdef'

describe('StaffView key-once issuance', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    window.localStorage.clear()
    mocks.usersList.mockResolvedValue({
      items: [{ id: 1, username: 'QCanvas', email: 'qcanvas@wujie.local', member_type: 'tool', status: 'active', notes: '', allowed_groups: null, subscriptions: [] }],
      total: 1,
    })
    mocks.getBatchUsersUsage.mockResolvedValue({ stats: {} })
    mocks.getStats.mockResolvedValue({ usd_cny_rate: 7.2 })
    // Real backend truth: admin list-keys endpoints always return an empty `key` (apiKeyDTOWithoutSecret).
    mocks.getUserApiKeys.mockResolvedValue({ items: [{ id: 10, name: 'card', key: '', status: 'active', quota: 0, quota_used: 0, last_used_at: null }] })
    mocks.getBatchApiKeysUsage.mockResolvedValue({ stats: {} })
    mocks.groupsGetAll.mockResolvedValue([
      { id: 7, name: '默认组', status: 'active', platform: 'openai', is_exclusive: false, subscription_type: 'standard' },
		{ id: 8, name: '视频组', status: 'active', platform: 'openai', is_exclusive: false, subscription_type: 'standard' },
    ])
    mocks.createApiKeyForUser.mockResolvedValue({ id: 11, name: 'zhangsan-生产卡', key: FULL_KEY, status: 'active', quota: 0, quota_used: 0 })
		mocks.createQCanvasKeyPairForUser.mockResolvedValue({
			video: { id: 12, name: 'QCanvas · video', key: 'sk-video-one-time', status: 'active', quota: 0, quota_used: 0 },
			media: { id: 13, name: 'QCanvas · media', key: 'sk-media-one-time', status: 'active', quota: 0, quota_used: 0 },
		})
    mocks.usersCreate.mockResolvedValue({
      user: { id: 2, username: 'Batch worker', email: 'batch@wujie.local', member_type: 'tool', status: 'active' },
      initial_credential: null,
    })
  })

  it('shows the full key exactly once right after creation, and never persists it to localStorage', async () => {
    const setItemSpy = vi.spyOn(window.localStorage, 'setItem')
    const wrapper = mount(StaffView, {
      global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub, RouterLink: RouterLinkStub } },
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

	it('issues QCanvas video and media keys atomically with two distinct group selections', async () => {
		const wrapper = mount(StaffView, {
			global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub, RouterLink: RouterLinkStub } },
		})
		await flushPromises()

		await wrapper.find('[data-test="qcanvas-key-pair"]').trigger('click')
		await wrapper.find('[data-test="qcanvas-video-group"]').setValue('8')
		await wrapper.find('[data-test="qcanvas-media-group"]').setValue('7')
		await wrapper.find('form[data-test="qcanvas-key-pair-form"]').trigger('submit.prevent')
		await flushPromises()

		expect(mocks.createQCanvasKeyPairForUser).toHaveBeenCalledWith(
			1,
			{ video_group_id: 8, media_group_id: 7 },
			expect.any(String),
		)
		expect(wrapper.text()).toContain('sk-video-one-time')
		expect(wrapper.text()).toContain('sk-media-one-time')
	})

	it('forgets both QCanvas secrets after the result dialog closes', async () => {
		const wrapper = mount(StaffView, {
			global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub, RouterLink: RouterLinkStub } },
		})
		await flushPromises()

		await wrapper.find('[data-test="qcanvas-key-pair"]').trigger('click')
		await wrapper.find('form[data-test="qcanvas-key-pair-form"]').trigger('submit.prevent')
		await flushPromises()
		expect(wrapper.text()).toContain('sk-video-one-time')
		expect(wrapper.text()).toContain('sk-media-one-time')

		await wrapper.find('[data-test="qcanvas-key-pair-done"]').trigger('click')
		await wrapper.find('[data-test="qcanvas-key-pair"]').trigger('click')
		expect(wrapper.text()).not.toContain('sk-video-one-time')
		expect(wrapper.text()).not.toContain('sk-media-one-time')
	})

	it('renders the employee total from the user aggregate instead of treating two keys as two balances', async () => {
		mocks.getBatchUsersUsage.mockResolvedValue({
			stats: { 1: { user_id: 1, total_actual_cost: 1.2, today_actual_cost: 0.7, by_platform: [] } },
		})
		const wrapper = mount(StaffView, {
			global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub, RouterLink: RouterLinkStub } },
		})
		await flushPromises()

		expect(wrapper.text()).toContain('8.64')
	})

  it('binds a newly issued service card to an active group', async () => {
    const wrapper = mount(StaffView, {
      global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub, RouterLink: RouterLinkStub } },
    })
    await flushPromises()

    await wrapper.find('[data-test="issue-card"]').trigger('click')
    await wrapper.find('form[data-test="issue-card-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.createApiKeyForUser).toHaveBeenCalledWith(
      1,
      expect.objectContaining({ group_id: 7 }),
      expect.any(String),
    )
  })

  it('creates only a tool service identity and never opens a temporary credential flow', async () => {
    const wrapper = mount(StaffView, {
      global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub, RouterLink: RouterLinkStub } },
    })
    await flushPromises()

    expect(wrapper.find('[data-test="create-human-employee"]').exists()).toBe(false)
    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('batch@wujie.local')
    await wrapper.find('[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.usersCreate).toHaveBeenCalledWith(expect.objectContaining({
      email: 'batch@wujie.local',
      member_type: 'tool',
      role: 'user',
    }))
    expect(wrapper.findComponent({ name: 'InitialCredentialDialog' }).exists()).toBe(false)
  })

  it('removes local quota and expiry inputs and always issues an unlimited API card', async () => {
    const wrapper = mount(StaffView, {
      global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub, RouterLink: RouterLinkStub } },
    })
    await flushPromises()

    await wrapper.find('[data-test="issue-card"]').trigger('click')
    expect(wrapper.find('[data-test="issue-card-quota"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('有效期（天）')
    await wrapper.find('form[data-test="issue-card-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.createApiKeyForUser).toHaveBeenCalledWith(
      1,
      expect.objectContaining({ quota: 0 }),
      expect.any(String),
    )
    expect(mocks.createApiKeyForUser.mock.calls[0]?.[1]).not.toHaveProperty('expires_in_days')
  })

  it('selects the first eligible group instead of an ineligible exclusive group', async () => {
    mocks.groupsGetAll.mockResolvedValue([
      { id: 6, name: '未授权专属组', status: 'active', platform: 'openai', is_exclusive: true, subscription_type: 'standard' },
      { id: 7, name: '默认组', status: 'active', platform: 'openai', is_exclusive: false, subscription_type: 'standard' },
    ])

    const wrapper = mount(StaffView, {
      global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub, RouterLink: RouterLinkStub } },
    })
    await flushPromises()

    await wrapper.find('[data-test="issue-card"]').trigger('click')
    const options = wrapper.find('[data-test="issue-card-group"]').findAll('option')
    expect(options.map((option) => option.text())).not.toContain('未授权专属组')
    await wrapper.find('form[data-test="issue-card-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.createApiKeyForUser).toHaveBeenCalledWith(
      1,
      expect.objectContaining({ group_id: 7 }),
      expect.any(String),
    )
  })

  it('fills an eligible default when groups finish loading after the dialog opens', async () => {
    let resolveGroups!: (groups: Array<Record<string, unknown>>) => void
    mocks.groupsGetAll.mockReturnValue(new Promise((resolve) => { resolveGroups = resolve }))

    const wrapper = mount(StaffView, {
      global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub, RouterLink: RouterLinkStub } },
    })
    await flushPromises()

    await wrapper.find('[data-test="issue-card"]').trigger('click')
    expect(wrapper.text()).toContain('正在加载可用组')
    expect(wrapper.text()).not.toContain('当前没有可绑定的启用组')

    resolveGroups([
      { id: 7, name: '默认组', status: 'active', platform: 'openai', is_exclusive: false, subscription_type: 'standard' },
    ])
    await flushPromises()
    await wrapper.find('form[data-test="issue-card-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.createApiKeyForUser).toHaveBeenCalledWith(
      1,
      expect.objectContaining({ group_id: 7 }),
      expect.any(String),
    )
  })

  it('masks the key again once the issuance dialog is closed and reopened', async () => {
    const wrapper = mount(StaffView, {
      global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub, RouterLink: RouterLinkStub } },
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

  it('one-page wizard: creates identity, issues dual keys with recharge, then shows both plaintext keys once', async () => {
    mocks.usersUpdateBalance.mockResolvedValue({ id: 2, balance: 55 })
    const wrapper = mount(StaffView, {
      global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub, RouterLink: RouterLinkStub } },
    })
    await flushPromises()

    // 第 1 步：建服务身份
    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('batch@wujie.local')
    await wrapper.find('form[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()
    expect(mocks.usersCreate).toHaveBeenCalledWith(expect.objectContaining({ email: 'batch@wujie.local', member_type: 'tool' }))

    // 第 2 步：两个不同的组 + 充值金额；同组选项在选择器里被禁用
    expect(wrapper.find('form[data-test="wizard-pair-form"]').exists()).toBe(true)
    await wrapper.find('[data-test="wizard-video-group"]').setValue('8')
    await wrapper.find('[data-test="wizard-media-group"]').setValue('7')
    const mediaOptions = wrapper.find('[data-test="wizard-media-group"]').findAll('option')
    expect(mediaOptions.find((o) => o.attributes('value') === '8')?.attributes('disabled')).toBeDefined()
    await wrapper.find('[data-test="wizard-amount"]').setValue(55)
    await wrapper.find('form[data-test="wizard-pair-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.createQCanvasKeyPairForUser).toHaveBeenCalledWith(
      2,
      { video_group_id: 8, media_group_id: 7 },
      expect.any(String),
    )
    expect(mocks.usersUpdateBalance).toHaveBeenCalledWith(2, 55, 'add', expect.any(String))

    // 第 3 步：明文双 Key 同屏展示一次
    expect(wrapper.find('[data-test="wizard-video-key"]').text()).toContain('sk-video-one-time')
    expect(wrapper.find('[data-test="wizard-media-key"]').text()).toContain('sk-media-one-time')
    expect(wrapper.find('[data-test="wizard-recharge-result"]').text()).toContain('55.00')

    // 关闭后明文不残留
    await wrapper.find('[data-test="wizard-done"]').trigger('click')
    expect(wrapper.text()).not.toContain('sk-video-one-time')
    expect(wrapper.text()).not.toContain('sk-media-one-time')
  })

  it('one-page wizard: skips recharge when amount is 0', async () => {
    const wrapper = mount(StaffView, {
      global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub, RouterLink: RouterLinkStub } },
    })
    await flushPromises()

    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('batch@wujie.local')
    await wrapper.find('form[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()

    await wrapper.find('[data-test="wizard-video-group"]').setValue('8')
    await wrapper.find('[data-test="wizard-media-group"]').setValue('7')
    await wrapper.find('form[data-test="wizard-pair-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.createQCanvasKeyPairForUser).toHaveBeenCalledTimes(1)
    expect(mocks.usersUpdateBalance).not.toHaveBeenCalled()
  })

  it('only ever renders masked key metadata in the staff key list, never the full value', async () => {
    const wrapper = mount(StaffView, {
      global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub, RouterLink: RouterLinkStub } },
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
      global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub, RouterLink: RouterLinkStub } },
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
