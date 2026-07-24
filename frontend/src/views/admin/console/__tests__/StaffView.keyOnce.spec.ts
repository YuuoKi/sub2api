import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import StaffView from '../StaffView.vue'

const mocks = vi.hoisted(() => ({
  usersList: vi.fn(),
  usersCreate: vi.fn(),
  usersUpdateBalance: vi.fn(),
  getBatchUsersUsage: vi.fn(),
  createQCanvasKeyPairForUser: vi.fn(),
  getUserApiKeys: vi.fn(),
  getBatchApiKeysUsage: vi.fn(),
  groupsGetAll: vi.fn(),
  groupsCreate: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  requestConfirmation: vi.fn(),
}))

vi.mock('@/composables/useAppDialog', () => ({
  requestConfirmation: mocks.requestConfirmation,
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
    },
    apiKeys: {
      createQCanvasKeyPairForUser: mocks.createQCanvasKeyPairForUser,
    },
    groups: {
      getAll: mocks.groupsGetAll,
      create: mocks.groupsCreate,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess: mocks.showSuccess, showError: mocks.showError }),
}))

const AppLayoutStub = { template: '<div><slot /></div>' }
const IconStub = { template: '<i />' }
const RouterLinkStub = { template: '<a><slot /></a>' }

const FULL_KEY = 'sk-once-visible-1234567890abcdef'

function mountView() {
  return mount(StaffView, {
    global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub, RouterLink: RouterLinkStub } },
  })
}

describe('StaffView one-shot staff issuance', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.requestConfirmation.mockResolvedValue(true)
    window.localStorage.clear()
    mocks.usersList.mockResolvedValue({
      items: [
        {
          id: 1,
          username: 'QCanvas',
          email: 'qcanvas@wujie.local',
          role: 'user',
          member_type: 'tool',
          status: 'active',
          notes: '',
          allowed_groups: null,
          subscriptions: [],
        },
      ],
      total: 1,
    })
    mocks.getBatchUsersUsage.mockResolvedValue({ stats: {} })
    // Real backend truth: admin list-keys endpoints always return an empty `key` (apiKeyDTOWithoutSecret).
    mocks.getUserApiKeys.mockResolvedValue({
      items: [{ id: 10, name: 'card', key: '', status: 'active', quota: 0, quota_used: 0, last_used_at: null }],
    })
    mocks.getBatchApiKeysUsage.mockResolvedValue({ stats: {} })
    mocks.groupsGetAll.mockResolvedValue([
      { id: 7, name: '默认组', status: 'active', platform: 'openai', is_exclusive: false, subscription_type: 'standard' },
      { id: 8, name: '视频组', status: 'active', platform: 'openai', is_exclusive: false, subscription_type: 'standard' },
    ])
    mocks.createQCanvasKeyPairForUser.mockResolvedValue({
      video: { id: 12, name: 'QCanvas · video', key: 'sk-video-one-time', status: 'active', quota: 0, quota_used: 0 },
      media: { id: 13, name: 'QCanvas · media', key: FULL_KEY, status: 'active', quota: 0, quota_used: 0 },
    })
    mocks.usersCreate.mockResolvedValue({
      user: { id: 2, username: '张三', email: 'zhangsan@wujie.local', role: 'user', member_type: 'tool', status: 'active' },
      initial_credential: null,
    })
  })

  it('lists non-admin human and tool employees and shows member type', async () => {
    mocks.usersList.mockResolvedValue({
      items: [
        {
          id: 1,
          username: '工具号',
          email: 'tool@wujie.local',
          role: 'user',
          member_type: 'tool',
          status: 'active',
          notes: '',
          allowed_groups: null,
          subscriptions: [],
        },
        {
          id: 2,
          username: '张三',
          email: 'zhangsan@wujie.local',
          role: 'user',
          member_type: 'human',
          status: 'active',
          notes: '',
          allowed_groups: null,
          subscriptions: [],
        },
        {
          id: 9,
          username: '管理员',
          email: 'admin@wujie.local',
          role: 'admin',
          member_type: 'human',
          status: 'active',
          notes: '',
          allowed_groups: null,
          subscriptions: [],
        },
      ],
      total: 3,
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('工具号')
    expect(wrapper.text()).toContain('张三')
    expect(wrapper.text()).toContain('工具账号')
    expect(wrapper.text()).toContain('员工账号')
    expect(wrapper.text()).not.toContain('admin@wujie.local')
    expect(wrapper.text()).toContain('共 2 个员工')
  })

  it('single modal: creates employee, issues dual keys into the same group, recharges, then shows both plaintext keys once', async () => {
    mocks.usersUpdateBalance.mockResolvedValue({ id: 2, balance: 55 })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('zhangsan@wujie.local')
    await wrapper.find('[data-test="wizard-group"]').setValue('8')
    await wrapper.find('[data-test="wizard-amount"]').setValue(55)
    await wrapper.find('form[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.usersCreate).toHaveBeenCalledWith(expect.objectContaining({
      email: 'zhangsan@wujie.local',
      member_type: 'tool',
      role: 'user',
    }))
    // 后端已放开同组限制：video_group_id 与 media_group_id 传同一个所选组
    expect(mocks.createQCanvasKeyPairForUser).toHaveBeenCalledWith(
      2,
      { video_group_id: 8, media_group_id: 8 },
      expect.any(String),
    )
    expect(mocks.usersUpdateBalance).toHaveBeenCalledWith(2, 55, 'add', expect.any(String))

    expect(wrapper.find('[data-test="wizard-video-key"]').text()).toContain('sk-video-one-time')
    expect(wrapper.find('[data-test="wizard-media-key"]').text()).toContain(FULL_KEY)
    expect(wrapper.find('[data-test="wizard-recharge-result"]').text()).toContain('55.00')

    // 关闭后明文不残留
    await wrapper.find('[data-test="wizard-done"]').trigger('click')
    expect(wrapper.text()).not.toContain('sk-video-one-time')
    expect(wrapper.text()).not.toContain(FULL_KEY)
  })

  it('never persists the one-time plaintext keys to localStorage', async () => {
    const setItemSpy = vi.spyOn(window.localStorage, 'setItem')
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('zhangsan@wujie.local')
    await wrapper.find('form[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(wrapper.text()).toContain(FULL_KEY)
    for (const call of setItemSpy.mock.calls) {
      expect(String(call[1])).not.toContain(FULL_KEY)
      expect(String(call[1])).not.toContain('sk-video-one-time')
    }
  })

  it('reuses the existing active tool employee on flat EMAIL_EXISTS 409 and still issues dual keys', async () => {
    // Axios interceptor rejects with a FLAT object — not error.response.status.
    mocks.usersCreate.mockRejectedValue({ status: 409, reason: 'EMAIL_EXISTS', message: 'email already exists' })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('  QCanvas@wujie.local ')
    await wrapper.find('form[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.usersCreate).toHaveBeenCalledTimes(1)
    expect(mocks.createQCanvasKeyPairForUser).toHaveBeenCalledWith(
      1,
      { video_group_id: 7, media_group_id: 7 },
      expect.any(String),
    )
    expect(wrapper.find('[data-test="wizard-video-key"]').text()).toContain('sk-video-one-time')
    expect(wrapper.find('[data-test="wizard-media-key"]').text()).toContain(FULL_KEY)
  })

  it('reuses an existing active human employee on email conflict without converting member_type', async () => {
    mocks.usersCreate.mockRejectedValue({ status: 409, reason: 'EMAIL_EXISTS', message: 'email already exists' })
    mocks.usersList.mockResolvedValue({
      items: [
        {
          id: 4,
          username: '李四',
          email: 'lisi@wujie.local',
          role: 'user',
          member_type: 'human',
          status: 'active',
          notes: '',
          allowed_groups: null,
          subscriptions: [],
        },
      ],
      total: 1,
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('lisi@wujie.local')
    await wrapper.find('form[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.usersCreate).toHaveBeenCalledTimes(1)
    expect(mocks.createQCanvasKeyPairForUser).toHaveBeenCalledWith(
      4,
      { video_group_id: 7, media_group_id: 7 },
      expect.any(String),
    )
    // Must not attempt to rewrite human → tool on conflict reuse.
    expect(mocks.usersCreate).not.toHaveBeenCalledWith(expect.objectContaining({ member_type: 'human' }))
    expect(wrapper.find('[data-test="wizard-video-key"]').text()).toContain('sk-video-one-time')
  })

  it('rejects email conflict when the exact owner is an admin', async () => {
    mocks.usersCreate.mockRejectedValue({ status: 409, reason: 'EMAIL_EXISTS', message: 'email already exists' })
    mocks.usersList.mockResolvedValue({
      items: [
        {
          id: 3,
          username: 'admin',
          email: 'boss@wujie.local',
          role: 'admin',
          member_type: 'human',
          status: 'active',
          notes: '',
          allowed_groups: null,
          subscriptions: [],
        },
      ],
      total: 1,
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('boss@wujie.local')
    await wrapper.find('form[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.createQCanvasKeyPairForUser).not.toHaveBeenCalled()
    expect(mocks.showError).toHaveBeenCalledWith(expect.stringMatching(/管理员/))
  })

  it('rejects email conflict when the exact owner is disabled', async () => {
    mocks.usersCreate.mockRejectedValue({ status: 409, reason: 'EMAIL_EXISTS', message: 'email already exists' })
    mocks.usersList.mockResolvedValue({
      items: [
        {
          id: 5,
          username: '停用员工',
          email: 'disabled@wujie.local',
          role: 'user',
          member_type: 'tool',
          status: 'disabled',
          notes: '',
          allowed_groups: null,
          subscriptions: [],
        },
      ],
      total: 1,
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('disabled@wujie.local')
    await wrapper.find('form[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.createQCanvasKeyPairForUser).not.toHaveBeenCalled()
    expect(mocks.showError).toHaveBeenCalledWith(expect.stringMatching(/停用|禁用/))
  })

  it('rejects email conflict when no exact non-admin human/tool owner is found', async () => {
    mocks.usersCreate.mockRejectedValue({ status: 409, reason: 'EMAIL_EXISTS', message: 'email already exists' })
    mocks.usersList.mockResolvedValue({
      items: [
        {
          id: 6,
          username: 'other',
          email: 'other@wujie.local',
          role: 'user',
          member_type: 'tool',
          status: 'active',
          notes: '',
          allowed_groups: null,
          subscriptions: [],
        },
      ],
      total: 1,
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('missing@wujie.local')
    await wrapper.find('form[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.createQCanvasKeyPairForUser).not.toHaveBeenCalled()
    expect(mocks.showError).toHaveBeenCalledWith(expect.stringMatching(/找不到|不存在|占用/))
  })

  it('keeps the created owner when key issuance fails so retry does not create again', async () => {
    mocks.createQCanvasKeyPairForUser
      .mockRejectedValueOnce({ status: 500, message: 'key issue failed' })
      .mockResolvedValueOnce({
        video: { id: 12, name: 'QCanvas · video', key: 'sk-video-one-time', status: 'active', quota: 0, quota_used: 0 },
        media: { id: 13, name: 'QCanvas · media', key: FULL_KEY, status: 'active', quota: 0, quota_used: 0 },
      })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('zhangsan@wujie.local')
    await wrapper.find('form[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.usersCreate).toHaveBeenCalledTimes(1)
    expect(mocks.createQCanvasKeyPairForUser).toHaveBeenCalledTimes(1)
    expect(mocks.showError).toHaveBeenCalled()

    await wrapper.find('form[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.usersCreate).toHaveBeenCalledTimes(1)
    expect(mocks.createQCanvasKeyPairForUser).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[data-test="wizard-video-key"]').text()).toContain('sk-video-one-time')
  })

  it('skips recharge when amount is 0', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('zhangsan@wujie.local')
    await wrapper.find('form[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.createQCanvasKeyPairForUser).toHaveBeenCalledTimes(1)
    expect(mocks.usersUpdateBalance).not.toHaveBeenCalled()
    expect(wrapper.find('[data-test="wizard-video-key"]').text()).toContain('sk-video-one-time')
  })

  it('reopens as a blank form after the result dialog closes, with no secret residue', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('zhangsan@wujie.local')
    await wrapper.find('form[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()
    expect(wrapper.text()).toContain(FULL_KEY)

    await wrapper.find('[data-test="wizard-done"]').trigger('click')
    await wrapper.find('[data-test="create-service-identity"]').trigger('click')

    expect(wrapper.text()).not.toContain(FULL_KEY)
    expect(wrapper.text()).not.toContain('sk-video-one-time')
    expect(wrapper.find('form[data-test="service-identity-form"]').exists()).toBe(true)
  })

  it('quick-creates a group inline and selects it for issuance', async () => {
    mocks.groupsGetAll.mockReset()
    mocks.groupsGetAll
      .mockResolvedValueOnce([]) // 首次加载：还没有任何组
      .mockResolvedValue([{ id: 9, name: '后期组', platform: 'openai', status: 'active', is_exclusive: false, subscription_type: 'standard' }])
    mocks.groupsCreate.mockResolvedValue({ id: 9, name: '后期组', platform: 'openai', status: 'active', is_exclusive: false, subscription_type: 'standard' })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="staff-quick-group-name"]').setValue('后期组')
    await wrapper.find('[data-test="staff-quick-group-create"]').trigger('click')
    await flushPromises()

    expect(mocks.groupsCreate).toHaveBeenCalledWith(expect.objectContaining({ name: '后期组', platform: 'openai' }))

    await wrapper.find('[data-test="service-identity-email"]').setValue('zhangsan@wujie.local')
    await wrapper.find('form[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.createQCanvasKeyPairForUser).toHaveBeenCalledWith(
      2,
      { video_group_id: 9, media_group_id: 9 },
      expect.any(String),
    )
  })

  it('does not offer exclusive or subscription groups in the single group select', async () => {
    mocks.groupsGetAll.mockResolvedValue([
      { id: 6, name: '未授权专属组', status: 'active', platform: 'openai', is_exclusive: true, subscription_type: 'standard' },
      { id: 5, name: '订阅组', status: 'active', platform: 'openai', is_exclusive: false, subscription_type: 'subscription' },
      { id: 7, name: '默认组', status: 'active', platform: 'openai', is_exclusive: false, subscription_type: 'standard' },
    ])
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    const options = wrapper.find('[data-test="wizard-group"]').findAll('option')
    expect(options.map((option) => option.text())).not.toContain('未授权专属组')
    expect(options.map((option) => option.text())).not.toContain('订阅组')
    expect(options.map((option) => option.text())).toContain('默认组')
  })

  it('renders the employee total from the user aggregate instead of treating two keys as two balances', async () => {
    mocks.getBatchUsersUsage.mockResolvedValue({
      stats: { 1: { user_id: 1, total_actual_cost: 1.2, today_actual_cost: 0.7, by_platform: [] } },
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('8.64')
  })

  it('only ever renders masked key metadata in the staff key list, never the full value', async () => {
    const wrapper = mountView()
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

    const wrapper = mountView()
    await flushPromises()
    await wrapper.find('[data-test="toggle-expand"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).not.toContain(FULL_KEY)
    // Not even a prefix/suffix slice of the leaked key should leak into the DOM.
    expect(wrapper.text()).not.toContain(FULL_KEY.slice(0, 8))
    expect(wrapper.text()).not.toContain(FULL_KEY.slice(-4))
    expect(wrapper.text()).toContain('不再显示')
  })

  it('does not close the staff modal or clear the form on backdrop click', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('zhangsan@wujie.local')
    await wrapper.find('[data-test="wizard-amount"]').setValue(12)

    const backdrop = wrapper.find('[data-test="staff-modal-backdrop"]')
    expect(backdrop.exists()).toBe(true)
    await backdrop.trigger('click')
    await flushPromises()

    expect(wrapper.find('form[data-test="service-identity-form"]').exists()).toBe(true)
    expect((wrapper.find('[data-test="service-identity-email"]').element as HTMLInputElement).value).toBe(
      'zhangsan@wujie.local',
    )
    expect((wrapper.find('[data-test="wizard-amount"]').element as HTMLInputElement).value).toBe('12')
  })

  it('does not close the recharge modal or clear the amount on backdrop click', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="row-recharge"]').trigger('click')
    await wrapper.find('[data-test="recharge-amount"]').setValue(33)

    const backdrop = wrapper.find('[data-test="recharge-modal-backdrop"]')
    expect(backdrop.exists()).toBe(true)
    await backdrop.trigger('click')
    await flushPromises()

    expect(wrapper.find('form[data-test="recharge-form"]').exists()).toBe(true)
    expect((wrapper.find('[data-test="recharge-amount"]').element as HTMLInputElement).value).toBe('33')
  })

  it('does not close or clear the staff modal on Escape', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('zhangsan@wujie.local')

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await flushPromises()

    expect(wrapper.find('form[data-test="service-identity-form"]').exists()).toBe(true)
    expect((wrapper.find('[data-test="service-identity-email"]').element as HTMLInputElement).value).toBe(
      'zhangsan@wujie.local',
    )
  })

  it('does not close or clear the recharge modal on Escape', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="row-recharge"]').trigger('click')
    await wrapper.find('[data-test="recharge-amount"]').setValue(44)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await flushPromises()

    expect(wrapper.find('form[data-test="recharge-form"]').exists()).toBe(true)
    expect((wrapper.find('[data-test="recharge-amount"]').element as HTMLInputElement).value).toBe('44')
  })

  it('after keys are shown, only 「我已安全保存，完成」 closes the modal; Escape and backdrop do not', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('zhangsan@wujie.local')
    await wrapper.find('form[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(wrapper.find('[data-test="wizard-video-key"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="wizard-done"]').text()).toContain('我已安全保存，完成')
    expect(wrapper.find('[data-test="wizard-cancel"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="staff-modal-close-x"]').exists()).toBe(false)

    await wrapper.find('[data-test="staff-modal-backdrop"]').trigger('click')
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await flushPromises()

    expect(wrapper.find('[data-test="wizard-video-key"]').exists()).toBe(true)
    expect(wrapper.text()).toContain(FULL_KEY)

    await wrapper.find('[data-test="wizard-done"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-test="wizard-video-key"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain(FULL_KEY)
  })

  it('asks for Chinese confirmation before discarding a filled staff form on Cancel', async () => {
    mocks.requestConfirmation.mockResolvedValueOnce(false)
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('zhangsan@wujie.local')
    await wrapper.find('[data-test="wizard-cancel"]').trigger('click')
    await flushPromises()

    expect(mocks.requestConfirmation).toHaveBeenCalledWith(
      expect.objectContaining({ message: expect.stringMatching(/清空|取消|未保存|已填写/) }),
    )
    expect(wrapper.find('form[data-test="service-identity-form"]').exists()).toBe(true)
    expect((wrapper.find('[data-test="service-identity-email"]').element as HTMLInputElement).value).toBe(
      'zhangsan@wujie.local',
    )

    mocks.requestConfirmation.mockResolvedValueOnce(true)
    await wrapper.find('[data-test="wizard-cancel"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('form[data-test="service-identity-form"]').exists()).toBe(false)
  })

  it('reuses one stable idempotency key for all submit retries in the same open session', async () => {
    const uuidSpy = vi.spyOn(crypto, 'randomUUID')
      .mockReturnValueOnce('idem-stable-aaa')
      .mockReturnValueOnce('idem-after-reopen')

    mocks.createQCanvasKeyPairForUser
      .mockRejectedValueOnce({ status: 500, message: 'key issue failed' })
      .mockResolvedValueOnce({
        video: { id: 12, name: 'QCanvas · video', key: 'sk-video-one-time', status: 'active', quota: 0, quota_used: 0 },
        media: { id: 13, name: 'QCanvas · media', key: FULL_KEY, status: 'active', quota: 0, quota_used: 0 },
      })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('zhangsan@wujie.local')
    await wrapper.find('form[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()
    await wrapper.find('form[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.createQCanvasKeyPairForUser).toHaveBeenNthCalledWith(
      1,
      2,
      { video_group_id: 7, media_group_id: 7 },
      'idem-stable-aaa',
    )
    expect(mocks.createQCanvasKeyPairForUser).toHaveBeenNthCalledWith(
      2,
      2,
      { video_group_id: 7, media_group_id: 7 },
      'idem-stable-aaa',
    )

    await wrapper.find('[data-test="wizard-done"]').trigger('click')
    await wrapper.find('[data-test="create-service-identity"]').trigger('click')
    await wrapper.find('[data-test="service-identity-email"]').setValue('lisi@wujie.local')
    mocks.createQCanvasKeyPairForUser.mockResolvedValueOnce({
      video: { id: 22, name: 'QCanvas · video', key: 'sk-video-2', status: 'active', quota: 0, quota_used: 0 },
      media: { id: 23, name: 'QCanvas · media', key: 'sk-media-2', status: 'active', quota: 0, quota_used: 0 },
    })
    mocks.usersCreate.mockResolvedValueOnce({
      user: { id: 3, username: '李四', email: 'lisi@wujie.local', role: 'user', member_type: 'tool', status: 'active' },
      initial_credential: null,
    })
    await wrapper.find('form[data-test="service-identity-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.createQCanvasKeyPairForUser).toHaveBeenLastCalledWith(
      3,
      { video_group_id: 7, media_group_id: 7 },
      'idem-after-reopen',
    )
    uuidSpy.mockRestore()
  })
})
