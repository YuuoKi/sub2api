import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import KeyVaultView from '../KeyVaultView.vue'

const mocks = vi.hoisted(() => ({
  accountsList: vi.fn(),
  accountsCreate: vi.fn(),
  accountsUpdate: vi.fn(),
  listProviders: vi.fn(),
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
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
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

const SECRET_KEY = 'sk-super-secret-upstream-key-1234567890'

async function mountView() {
  const wrapper = mount(KeyVaultView, {
    global: { stubs: { AppLayout: AppLayoutStub, Icon: IconStub, RouterLink: RouterLinkStub } },
  })
  await flushPromises()
  return wrapper
}

describe('KeyVaultView account secret handling', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.accountsList.mockResolvedValue({ items: [], total: 0 })
    mocks.listProviders.mockResolvedValue({ items: [] })
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
