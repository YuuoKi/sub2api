import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import KeyVaultView from '../KeyVaultView.vue'

const mocks = vi.hoisted(() => ({
  accountsList: vi.fn(),
  accountsCreate: vi.fn(),
  accountsUpdate: vi.fn(),
  listProviders: vi.fn(),
  videoContract: vi.fn(),
  createProvider: vi.fn(),
  updateProvider: vi.fn(),
  deleteProvider: vi.fn(),
  authorizeTinyReal: vi.fn(),
  groupsGetAll: vi.fn(),
  groupsCreate: vi.fn(),
  requestConfirmation: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: mocks.accountsList,
      create: mocks.accountsCreate,
      update: mocks.accountsUpdate,
    },
    video: {
      listProviders: mocks.listProviders,
      contract: mocks.videoContract,
      createProvider: mocks.createProvider,
      updateProvider: mocks.updateProvider,
      deleteProvider: mocks.deleteProvider,
      authorizeTinyReal: mocks.authorizeTinyReal,
    },
    groups: {
      getAll: mocks.groupsGetAll,
      create: mocks.groupsCreate,
    },
  },
}))

vi.mock('@/composables/useAppDialog', () => ({
  requestConfirmation: mocks.requestConfirmation,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess: mocks.showSuccess, showError: mocks.showError }),
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => ({ query: {} }),
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }
const IconStub = { template: '<i />' }
const RouterLinkStub = { template: '<a><slot /></a>' }
// GroupSelector/AccountGroupsCell 依赖 i18n 与 GroupBadge，这里只保留 v-model 契约与分组渲染契约
const GroupSelectorStub = {
  props: ['modelValue', 'groups', 'platform'],
  emits: ['update:modelValue'],
  template: `<div data-test="group-selector-stub">
    <label v-for="g in groups.filter((x) => !platform || x.platform === platform)" :key="g.id">
      <input
        type="checkbox"
        :data-test="'group-check-' + g.id"
        :checked="modelValue.includes(g.id)"
        @change="$emit('update:modelValue', $event.target.checked ? [...modelValue, g.id] : modelValue.filter((id) => id !== g.id))"
      />
    </label>
  </div>`,
}
const AccountGroupsCellStub = {
  props: ['groups'],
  template: `<span data-test="groups-cell">{{ (groups || []).map((g) => g.name).join(',') || '-' }}</span>`,
}

const SECRET_KEY = 'sk-super-secret-upstream-key-1234567890'
const ANTHROPIC_GROUP = { id: 7, name: 'media', platform: 'anthropic', status: 'active' }
const HC_ATOM_GROUP = { id: 11, name: 'HC media', platform: 'hc_atom', status: 'active' }

const VIDEO_CONTRACT = {
  provider: 'seedance',
  base_url: 'https://ark.cn-beijing.volces.com',
  default_model: 'seedance-2.0',
  duration_seconds: 4,
  resolution: '720p',
  platforms: [
    { provider: 'seedance', display_name: 'Seedance', default_base_url: 'https://ark.cn-beijing.volces.com', default_model: 'seedance-2.0', adapter_ready: true },
    { provider: 'hc_atom_video_v1', display_name: 'HC-ATOM Video V1', default_base_url: 'https://api-aigc.fzyinghe.com', default_model: 'doubao-seedance-2.0', adapter_ready: true },
    { provider: 'hc_atom_seedance_v3', display_name: 'HC-ATOM Seedance V3', default_base_url: 'https://api-aigc.fzyinghe.com', default_model: 'doubao-seedance-2.0-v3', adapter_ready: true },
    { provider: 'jimeng', display_name: '即梦', default_base_url: '', default_model: '', adapter_ready: false },
  ],
}

async function mountView() {
  const wrapper = mount(KeyVaultView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        Icon: IconStub,
        RouterLink: RouterLinkStub,
        GroupSelector: GroupSelectorStub,
        AccountGroupsCell: AccountGroupsCellStub,
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('KeyVaultView account secret handling', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.accountsList.mockResolvedValue({ items: [], total: 0 })
    mocks.listProviders.mockResolvedValue({ items: [] })
    mocks.groupsGetAll.mockResolvedValue([ANTHROPIC_GROUP])
  })

  it('clears the plaintext API key from form state after the dialog is cancelled', async () => {
    const wrapper = await mountView()

    await wrapper.find('[data-test="open-create-account"]').trigger('click')
    const apiKeyInput = wrapper.find('[data-test="account-api-key"]')
    await apiKeyInput.setValue(SECRET_KEY)
    expect((apiKeyInput.element as HTMLInputElement).value).toBe(SECRET_KEY)

    await wrapper.find('[data-test="cancel-account"]').trigger('click')

    // Re-open the dialog: if the secret had leaked into shared reactive state,
    // it would reappear here even though the user never re-typed it.
    await wrapper.find('[data-test="open-create-account"]').trigger('click')
    const reopenedInput = wrapper.find('[data-test="account-api-key"]')
    expect((reopenedInput.element as HTMLInputElement).value).toBe('')
  })

  it('clears the plaintext API key from form state after a successful save', async () => {
    mocks.accountsCreate.mockResolvedValue({ id: 1 })
    const wrapper = await mountView()

    await wrapper.find('[data-test="open-create-account"]').trigger('click')
    await wrapper.find('input[placeholder="例如：老板的 Claude 主账号"]').setValue('测试账号')
    await wrapper.find('[data-test="group-check-7"]').setValue(true)
    const apiKeyInput = wrapper.find('[data-test="account-api-key"]')
    await apiKeyInput.setValue(SECRET_KEY)

    await wrapper.find('[data-test="account-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.accountsCreate).toHaveBeenCalledTimes(1)
    // Modal is closed post-save; reopening it must show a blank secret field,
    // never the plaintext key that was just submitted.
    await wrapper.find('[data-test="open-create-account"]').trigger('click')
    const reopenedInput = wrapper.find('[data-test="account-api-key"]')
    expect((reopenedInput.element as HTMLInputElement).value).toBe('')
    expect(wrapper.text()).not.toContain(SECRET_KEY)
  })

  it('uses the fixed HC-ATOM endpoint and model builder without echoing the upstream key', async () => {
    mocks.groupsGetAll.mockResolvedValue([HC_ATOM_GROUP])
    mocks.accountsCreate.mockResolvedValue({ id: 11 })
    const wrapper = await mountView()

    await wrapper.find('[data-test="open-create-account"]').trigger('click')
    await wrapper.find('[data-test="account-platform"]').setValue('hc_atom')
    await flushPromises()

    expect(wrapper.find('[data-test="hc-atom-base-url-locked"]').text()).toBe('https://api-aigc.fzyinghe.com')
    const modelDirectory = wrapper.find('[data-test="hc-atom-model-directory"]').text()
    expect(modelDirectory).toContain('gpt-5.6-sol')
    expect(modelDirectory).toContain('gemini-3-flash-preview')
    expect(modelDirectory).toContain('claude-opus-4-6')
    expect(modelDirectory).toContain('seedream-5.0')
    expect(modelDirectory).toContain('doubao-seedream-5.0-pro')
    expect(modelDirectory).toContain('gemini-3.1-flash-image-preview')
    expect(modelDirectory).toContain('gpt-image-2')
    expect(modelDirectory).toContain('s-gpt-image-2')
    expect(wrapper.find('[data-test="hc-atom-model-directory"]').text()).not.toContain('dola-seedream-5.0-pro')

    await wrapper.find('[data-test="account-name"]').setValue('HC 图片主账号')
    await wrapper.find('[data-test="group-check-11"]').setValue(true)
    await wrapper.find('[data-test="account-api-key"]').setValue(SECRET_KEY)
    await wrapper.find('[data-test="account-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.accountsCreate).toHaveBeenCalledWith(expect.objectContaining({
      name: 'HC 图片主账号',
      platform: 'hc_atom',
      type: 'apikey',
      group_ids: [11],
      credentials: {
        api_key: SECRET_KEY,
        protocol: 'hc_atom',
        model_mapping: {
          'gpt-5.6-sol': 'gpt-5.6-sol',
          'gemini-3-flash-preview': 'gemini-3-flash-preview',
          'claude-opus-4-6': 'claude-opus-4-6',
          'seedream-5.0': 'seedream-5.0',
          'doubao-seedream-5.0-pro': 'doubao-seedream-5.0-pro',
          'gemini-3.1-flash-image-preview': 'gemini-3.1-flash-image-preview',
          'gpt-image-2': 'gpt-image-2',
          's-gpt-image-2': 's-gpt-image-2',
        },
      },
    }))
    expect(wrapper.text()).not.toContain(SECRET_KEY)

    await wrapper.find('[data-test="open-create-account"]').trigger('click')
    expect((wrapper.find('[data-test="account-api-key"]').element as HTMLInputElement).value).toBe('')
  })
})

describe('KeyVaultView group binding (P0)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.accountsList.mockResolvedValue({ items: [], total: 0 })
    mocks.listProviders.mockResolvedValue({ items: [] })
    mocks.groupsGetAll.mockResolvedValue([ANTHROPIC_GROUP])
  })

  it('blocks save when no group is selected and reports a Chinese error', async () => {
    const wrapper = await mountView()

    await wrapper.find('[data-test="open-create-account"]').trigger('click')
    await wrapper.find('input[placeholder="例如：老板的 Claude 主账号"]').setValue('测试账号')
    await wrapper.find('[data-test="account-api-key"]').setValue(SECRET_KEY)
    await wrapper.find('[data-test="account-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.accountsCreate).not.toHaveBeenCalled()
    expect(mocks.showError).toHaveBeenCalledWith(expect.stringContaining('请至少选择一个分组'))
  })

  it('sends group_ids on create when a group is selected', async () => {
    mocks.accountsCreate.mockResolvedValue({ id: 1 })
    const wrapper = await mountView()

    await wrapper.find('[data-test="open-create-account"]').trigger('click')
    await wrapper.find('input[placeholder="例如：老板的 Claude 主账号"]').setValue('Ark 作图')
    await wrapper.find('[data-test="group-check-7"]').setValue(true)
    await wrapper.find('[data-test="account-api-key"]').setValue(SECRET_KEY)
    await wrapper.find('[data-test="account-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.accountsCreate).toHaveBeenCalledTimes(1)
    const payload = mocks.accountsCreate.mock.calls[0][0]
    expect(payload.group_ids).toEqual([7])
  })

  it('quick-creates a preset group inline and auto-selects it when no group exists', async () => {
    mocks.groupsGetAll.mockResolvedValue([])
    mocks.groupsCreate.mockResolvedValue({ id: 9, name: 'media', platform: 'anthropic' })
    mocks.accountsCreate.mockResolvedValue({ id: 2 })
    const wrapper = await mountView()

    await wrapper.find('[data-test="open-create-account"]').trigger('click')
    expect(wrapper.find('[data-test="group-quick-create"]').exists()).toBe(true)

    await wrapper.find('[data-test="quick-create-media"]').trigger('click')
    await flushPromises()

    expect(mocks.groupsCreate).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'media', platform: 'anthropic', allow_image_generation: true }),
    )

    await wrapper.find('input[placeholder="例如：老板的 Claude 主账号"]').setValue('Ark 作图')
    await wrapper.find('[data-test="account-api-key"]').setValue(SECRET_KEY)
    await wrapper.find('[data-test="account-form"]').trigger('submit.prevent')
    await flushPromises()

    const payload = mocks.accountsCreate.mock.calls[0][0]
    expect(payload.group_ids).toEqual([9])
  })

  it('quick-creates an HC media group with the live async model and positive image prices', async () => {
    const created = { id: 12, name: 'media', platform: 'hc_atom', status: 'active' }
    mocks.groupsGetAll.mockReset()
    mocks.groupsGetAll.mockResolvedValueOnce([]).mockResolvedValue([created])
    mocks.groupsCreate.mockResolvedValue(created)
    const wrapper = await mountView()

    await wrapper.find('[data-test="open-create-account"]').trigger('click')
    await wrapper.find('[data-test="account-platform"]').setValue('hc_atom')
    await flushPromises()
    await wrapper.find('[data-test="quick-create-media"]').trigger('click')
    await flushPromises()

    expect(mocks.groupsCreate).toHaveBeenCalledWith(expect.objectContaining({
      name: 'media',
      platform: 'hc_atom',
      allow_image_generation: true,
      allow_batch_image_generation: true,
      image_price_1k: 0.134,
      image_price_2k: 0.201,
      image_price_4k: 0.268,
      video_price_480p: null,
      video_price_720p: null,
      video_price_1080p: null,
      models_list_config: {
        enabled: true,
        models: [
          'gpt-5.6-sol',
          'gemini-3-flash-preview',
          'claude-opus-4-6',
          'seedream-5.0',
          'doubao-seedream-5.0-pro',
          'gemini-3.1-flash-image-preview',
          'gpt-image-2',
          's-gpt-image-2',
        ],
      },
    }))
  })
})

describe('KeyVaultView video provider management', () => {
  const PROVIDER = {
    id: 3,
    group_id: 7,
    group_name: 'media',
    provider: 'seedance',
    display_name: 'ark 一号',
    enabled: true,
    api_key_configured: true,
    masked_key: 'sk-...c7f4',
    base_url: '',
    default_model: 'seedance-2.0',
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mocks.accountsList.mockResolvedValue({ items: [], total: 0 })
    mocks.listProviders.mockResolvedValue({ items: [PROVIDER] })
    mocks.videoContract.mockResolvedValue(VIDEO_CONTRACT)
    mocks.groupsGetAll.mockResolvedValue([ANTHROPIC_GROUP])
    mocks.requestConfirmation.mockResolvedValue(true)
    mocks.createProvider.mockResolvedValue({ ...PROVIDER, id: 4 })
    mocks.updateProvider.mockResolvedValue({ ...PROVIDER, enabled: false })
    mocks.deleteProvider.mockResolvedValue({ message: 'ok' })
    mocks.authorizeTinyReal.mockResolvedValue({ ...PROVIDER, tiny_real_authorized_at: '2026-07-23T00:00:00Z' })
  })

  async function switchToVideoTab(wrapper: ReturnType<typeof mount>) {
    const tab = wrapper.findAll('button').find((button) => button.text() === '视频通道')
    expect(tab).toBeTruthy()
    await tab!.trigger('click')
  }

  it('creates a provider in-page but keeps real-call authorization opt-in and off by default', async () => {
    const wrapper = await mountView()
    await switchToVideoTab(wrapper)

    await wrapper.find('[data-test="open-create-provider"]').trigger('click')
    expect((wrapper.find('[data-test="provider-authorize-after-save"]').element as HTMLInputElement).checked).toBe(false)
    // 未就绪平台置灰并标注「即将接入」
    const platformOptions = wrapper.find('[data-test="provider-platform"]').findAll('option')
    const jimeng = platformOptions.find((option) => option.attributes('value') === 'jimeng')
    expect(jimeng?.attributes('disabled')).toBeDefined()
    expect(jimeng?.text()).toContain('即将接入')

    await wrapper.find('[data-test="provider-name"]').setValue('ark 一号')
    await wrapper.find('[data-test="group-check-7"]').setValue(true)
    await wrapper.find('[data-test="provider-api-key"]').setValue(SECRET_KEY)
    await wrapper.find('form[data-test="provider-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.createProvider).toHaveBeenCalledTimes(1)
    expect(mocks.createProvider).toHaveBeenCalledWith(expect.objectContaining({
      group_id: 7,
      provider: 'seedance',
      display_name: 'ark 一号',
      api_key: SECRET_KEY,
      enabled: true,
      default_model: 'seedance-2.0',
    }))
    expect(mocks.authorizeTinyReal).not.toHaveBeenCalled()

    // 保存后密钥不回显：重开弹窗是空密码框
    await wrapper.find('[data-test="open-create-provider"]').trigger('click')
    expect((wrapper.find('[data-test="provider-api-key"]').element as HTMLInputElement).value).toBe('')
    expect(wrapper.text()).not.toContain(SECRET_KEY)
  })

  it('keeps an HC V1 provider quick group on the V1 public alias', async () => {
    const created = { id: 12, name: 'video', platform: 'hc_atom', status: 'active' }
    mocks.groupsGetAll.mockReset()
    mocks.groupsGetAll.mockResolvedValueOnce([]).mockResolvedValue([created])
    mocks.groupsCreate.mockResolvedValue(created)
    mocks.listProviders.mockResolvedValue({ items: [] })
    const wrapper = await mountView()
    await switchToVideoTab(wrapper)

    await wrapper.find('[data-test="open-create-provider"]').trigger('click')
    await wrapper.find('[data-test="provider-platform"]').setValue('hc_atom_video_v1')
    await wrapper.find('[data-test="provider-quick-create-video"]').trigger('click')
    await flushPromises()

    expect(mocks.groupsCreate).toHaveBeenCalledWith(expect.objectContaining({
      platform: 'hc_atom',
      models_list_config: {
        enabled: true,
        models: ['doubao-seedance-2.0'],
      },
    }))
  })

  it('keeps the HC V3 group allowlist and provider default model on the V3 public alias', async () => {
    const created = { id: 13, name: 'video', platform: 'hc_atom', status: 'active' }
    mocks.groupsGetAll.mockReset()
    mocks.groupsGetAll.mockResolvedValueOnce([]).mockResolvedValue([created])
    mocks.groupsCreate.mockResolvedValue(created)
    mocks.listProviders.mockResolvedValue({ items: [] })
    const wrapper = await mountView()
    await switchToVideoTab(wrapper)

    await wrapper.find('[data-test="open-create-provider"]').trigger('click')
    await wrapper.find('[data-test="provider-platform"]').setValue('hc_atom_seedance_v3')
    await flushPromises()
    await wrapper.find('[data-test="provider-quick-create-video"]').trigger('click')
    await flushPromises()

    expect(mocks.groupsCreate).toHaveBeenCalledWith(expect.objectContaining({
      name: 'video',
      platform: 'hc_atom',
      allow_image_generation: false,
      allow_batch_image_generation: false,
      image_price_1k: null,
      image_price_2k: null,
      image_price_4k: null,
      video_price_480p: 0.05,
      video_price_720p: 0.07,
      video_price_1080p: 0.25,
      models_list_config: {
        enabled: true,
        models: ['doubao-seedance-2.0-v3'],
      },
    }))

    await wrapper.find('[data-test="provider-name"]').setValue('HC V3 一号')
    await wrapper.find('[data-test="provider-api-key"]').setValue(SECRET_KEY)
    await wrapper.find('form[data-test="provider-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.createProvider).toHaveBeenCalledWith(expect.objectContaining({
      group_id: 13,
      provider: 'hc_atom_seedance_v3',
      default_model: 'doubao-seedance-2.0-v3',
    }))
  })

  it('blocks provider save when no group is selected', async () => {
    const wrapper = await mountView()
    await switchToVideoTab(wrapper)

    await wrapper.find('[data-test="open-create-provider"]').trigger('click')
    await wrapper.find('[data-test="provider-name"]').setValue('ark 一号')
    await wrapper.find('[data-test="provider-api-key"]').setValue(SECRET_KEY)
    await wrapper.find('form[data-test="provider-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.createProvider).not.toHaveBeenCalled()
    expect(mocks.showError).toHaveBeenCalledWith(expect.stringContaining('请至少选择一个分组'))
  })

  it('falls back to a static seedance-only platform list when the contract request fails', async () => {
    mocks.videoContract.mockRejectedValue(new Error('network down'))
    const wrapper = await mountView()
    await switchToVideoTab(wrapper)

    await wrapper.find('[data-test="open-create-provider"]').trigger('click')
    await flushPromises()
    const platformOptions = wrapper.find('[data-test="provider-platform"]').findAll('option')
    expect(platformOptions).toHaveLength(1)
    expect(platformOptions[0].attributes('value')).toBe('seedance')
  })

  it('disables a provider only after a danger confirmation', async () => {
    const wrapper = await mountView()
    await switchToVideoTab(wrapper)

    mocks.requestConfirmation.mockResolvedValueOnce(false)
    await wrapper.find('[data-test="toggle-provider-3"]').trigger('click')
    expect(mocks.updateProvider).not.toHaveBeenCalled()

    await wrapper.find('[data-test="toggle-provider-3"]').trigger('click')
    await flushPromises()
    expect(mocks.requestConfirmation).toHaveBeenCalled()
    expect(mocks.updateProvider).toHaveBeenCalledWith(3, { enabled: false })
  })

  it('deletes a provider only after a danger confirmation', async () => {
    const wrapper = await mountView()
    await switchToVideoTab(wrapper)

    await wrapper.find('[data-test="remove-provider-3"]').trigger('click')
    await flushPromises()
    expect(mocks.requestConfirmation).toHaveBeenCalled()
    expect(mocks.deleteProvider).toHaveBeenCalledWith(3)
  })
})
