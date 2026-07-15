import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

const submitMock = vi.fn()
const listModelsMock = vi.fn()
const listJobsMock = vi.fn()
const listKeysMock = vi.fn()

vi.mock('@/api/batchImage', () => ({
  listBatchImageModels: (...args: unknown[]) => listModelsMock(...args),
  submitBatchImageJob: (...args: unknown[]) => submitMock(...args),
  listBatchImageJobs: (...args: unknown[]) => listJobsMock(...args),
  getBatchImageJob: vi.fn(),
  listBatchImageItems: vi.fn(),
  cancelBatchImageJob: vi.fn(),
  downloadBatchImageZip: vi.fn(),
  deleteBatchImageJobRecord: vi.fn(),
  saveBlob: vi.fn(),
}))

vi.mock('@/api', () => ({
  keysAPI: {
    list: (...args: unknown[]) => listKeysMock(...args),
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
    showInfo: vi.fn(),
    fetchPublicSettings: vi.fn().mockResolvedValue(undefined),
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ locale: { value: 'zh-CN' }, t: (k: string) => k }),
  }
})

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copy: vi.fn() }),
}))

vi.mock('@/composables/usePersistedPageSize', () => ({
  getPersistedPageSize: () => 20,
  setPersistedPageSize: vi.fn(),
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: { template: '<div><slot /></div>' },
}))

vi.mock('@/components/layout/TablePageLayout.vue', () => ({
  default: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
}))

vi.mock('@/components/common/DataTable.vue', () => ({
  default: { template: '<div />' },
}))

vi.mock('@/components/common/BaseDialog.vue', () => ({
  default: {
    props: ['show', 'title'],
    template: '<div v-if="show"><slot /><slot name="footer" /></div>',
  },
}))

vi.mock('@/components/common/Select.vue', () => ({
  default: {
    props: ['modelValue', 'options'],
    emits: ['update:modelValue', 'change'],
    template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', ($event.target).value)"><option v-for="o in (options||[])" :key="o.value" :value="o.value">{{ o.label }}</option></select>',
  },
}))

vi.mock('@/components/common/SearchInput.vue', () => ({
  default: { template: '<input />' },
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: { template: '<span />' },
}))

import BatchImageGuideView from '../BatchImageGuideView.vue'

describe('BatchImageGuideView execution_mode', () => {
  beforeEach(() => {
    submitMock.mockReset()
    listModelsMock.mockReset()
    listJobsMock.mockReset()
    listKeysMock.mockReset()
    listKeysMock.mockResolvedValue({
      items: [{ id: 1, name: 'key', key: 'sk-test', group: { name: 'Gemini', platform: 'gemini', allow_batch_image_generation: true } }],
    })
    listJobsMock.mockResolvedValue({ object: 'list', data: [], has_more: false })
    listModelsMock.mockResolvedValue({
      object: 'list',
      data: [{ id: 'gemini-2.5-flash-image', object: 'image.batch.model', provider: 'gemini_api' }],
      execution_capabilities: { mock: true, review_real: false, internal_real: false },
    })
    submitMock.mockResolvedValue({ id: 'imgbatch_1', status: 'queued', model: 'gemini-2.5-flash-image', provider: 'mock' })
  })

  it('defaults to mock and gates review/internal options', async () => {
    const wrapper = mount(BatchImageGuideView)
    await flushPromises()
    const createBtn = wrapper.findAll('button').find((b) => b.text().includes('创建批量任务'))
    expect(createBtn).toBeTruthy()
    await createBtn!.trigger('click')
    await flushPromises()
    const select = wrapper.findAll('select').find((s) => {
      const el = s.element as HTMLSelectElement
      return Array.from(el.options).some((o) => o.value === 'mock')
    })
    expect(select).toBeTruthy()
    expect((select!.element as HTMLSelectElement).value).toBe('mock')
    const reviewOption = Array.from((select!.element as HTMLSelectElement).options).find((o) => o.value === 'review_real')
    const internalOption = Array.from((select!.element as HTMLSelectElement).options).find((o) => o.value === 'internal_real')
    expect(reviewOption?.disabled).toBe(true)
    expect(internalOption?.disabled).toBe(true)
  })
})
