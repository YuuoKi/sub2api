import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import KeysView from '../KeysView.vue'

const {
  keysCreate,
  keysList,
  getAvailableGroups,
  getUserGroupRates,
  getPublicSettings,
  showError,
  showSuccess,
} = vi.hoisted(() => ({
  keysCreate: vi.fn(),
  keysList: vi.fn(),
  getAvailableGroups: vi.fn(),
  getUserGroupRates: vi.fn(),
  getPublicSettings: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/utils/productMode', () => ({
  isVideoGatewayDemoMode: false,
}))

vi.mock('@/api', () => ({
  keysAPI: {
    create: (...args: unknown[]) => keysCreate(...args),
    list: (...args: unknown[]) => keysList(...args),
    update: vi.fn(),
    delete: vi.fn(),
    toggleStatus: vi.fn(),
  },
  userGroupsAPI: {
    getAvailable: (...args: unknown[]) => getAvailableGroups(...args),
    getUserGroupRates: (...args: unknown[]) => getUserGroupRates(...args),
  },
  authAPI: {
    getPublicSettings: (...args: unknown[]) => getPublicSettings(...args),
  },
  usageAPI: {
    getDashboardApiKeysUsage: vi.fn().mockResolvedValue({ stats: {} }),
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
  }),
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({
    isCurrentStep: () => false,
    nextStep: vi.fn(),
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }
const TablePageLayoutStub = {
  template: '<div><slot name="actions" /><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>',
}
const BaseDialogStub = {
  props: ['show'],
  template: `
    <div v-if="show" class="base-dialog-stub">
      <slot />
      <slot name="footer" />
    </div>
  `,
}
const SelectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  template: `
    <select
      data-testid="group-select"
      :value="modelValue ?? ''"
      @change="$emit('update:modelValue', $event.target.value === '' ? null : Number($event.target.value))"
    >
      <option value="">--</option>
      <option v-for="opt in options" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
    </select>
  `,
}

function mountKeysView() {
  return mount(KeysView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        TablePageLayout: TablePageLayoutStub,
        BaseDialog: BaseDialogStub,
        DataTable: true,
        Pagination: true,
        EmptyState: true,
        Select: SelectStub,
        SearchInput: true,
        Icon: true,
        ConfirmDialog: true,
        UseKeyModal: true,
        EndpointPopover: true,
        GroupBadge: true,
        GroupOptionItem: true,
        Teleport: true,
      },
    },
  })
}

describe('KeysView handleSubmit errors', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    keysList.mockResolvedValue({ items: [], total: 0, pages: 0 })
    getAvailableGroups.mockResolvedValue([
      {
        id: 1,
        name: 'Default Group',
        description: null,
        platform: 'openai',
        rate_multiplier: 1,
        subscription_type: 'standard',
      },
    ])
    getUserGroupRates.mockResolvedValue({})
    getPublicSettings.mockResolvedValue({ api_base_url: 'https://api.example.com' })
  })

  it('shows interceptor error message when create key fails', async () => {
    keysCreate.mockRejectedValue({
      status: 400,
      code: 'API_KEY_LIMIT_REACHED',
      message: 'Maximum API keys reached',
    })

    const wrapper = mountKeysView()
    await flushPromises()

    await wrapper.find('[data-tour="keys-create-btn"]').trigger('click')
    await flushPromises()

    const form = wrapper.find('#key-form')
    expect(form.exists()).toBe(true)

    await form.find('[data-tour="key-form-name"]').setValue('Automation Key')
    const groupSelect = form.find('[data-testid="group-select"]')
    await groupSelect.setValue('1')
    await groupSelect.trigger('change')
    await nextTick()
    await form.trigger('submit')
    await flushPromises()

    expect(keysCreate).toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('Maximum API keys reached')
    expect(showError).not.toHaveBeenCalledWith('keys.failedToSave')
  })
})
