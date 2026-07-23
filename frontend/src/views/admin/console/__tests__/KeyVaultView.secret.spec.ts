import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import KeyVaultView from '../KeyVaultView.vue'

const mocks = vi.hoisted(() => ({
  accountsList: vi.fn(),
  accountsCreate: vi.fn(),
  accountsUpdate: vi.fn(),
  listProviders: vi.fn(),
  groupsGetAll: vi.fn(),
  groupsCreate: vi.fn(),
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
})
