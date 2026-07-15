import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

const submitMock = vi.fn()
const listModelsMock = vi.fn()
const listJobsMock = vi.fn()
const listKeysMock = vi.fn()
const getContentMock = vi.fn()
const downloadItemMock = vi.fn()
const showSuccess = vi.fn()

vi.mock('@/api/batchImage', () => ({
  listBatchImageModels: (...args: unknown[]) => listModelsMock(...args),
  submitBatchImageJob: (...args: unknown[]) => submitMock(...args),
  listBatchImageJobs: (...args: unknown[]) => listJobsMock(...args),
  getBatchImageJob: vi.fn(),
  listBatchImageItems: vi.fn().mockResolvedValue({ object: 'list', data: [], has_more: false }),
  cancelBatchImageJob: vi.fn(),
  downloadBatchImageZip: vi.fn(),
  downloadBatchImageItem: (...args: unknown[]) => downloadItemMock(...args),
  getBatchImageItemContent: (...args: unknown[]) => getContentMock(...args),
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
    showSuccess,
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
  default: { props: ['modelValue'], template: '<input />' },
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: { props: ['name'], template: '<span />' },
}))

import BatchImageGuideView from '../BatchImageGuideView.vue'

describe('BatchImageGuideView asset actions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listKeysMock.mockResolvedValue({
      items: [{
        id: 1,
        name: 'key',
        key: 'sk-test',
        status: 'active',
        group: { name: 'Gemini', platform: 'gemini', allow_batch_image_generation: true },
      }],
    })
    listModelsMock.mockResolvedValue({
      data: [{ id: 'gemini-3.1-flash-image', object: 'image.batch.model', provider: 'mock' }],
      execution_capabilities: { mock: true, review_real: false, internal_real: false },
    })
    listJobsMock.mockResolvedValue({ object: 'list', data: [], has_more: false })
    getContentMock.mockResolvedValue(new Blob(['png'], { type: 'image/png' }))
    downloadItemMock.mockResolvedValue(undefined)
    // @ts-expect-error jsdom
    global.URL.createObjectURL = vi.fn(() => 'blob:preview')
    // @ts-expect-error jsdom
    global.URL.revokeObjectURL = vi.fn()
  })

  it('preview download and reuse use server assets not IndexedDB', async () => {
    const wrapper = mount(BatchImageGuideView)
    await flushPromises()
    const vm = wrapper.vm as any
    vm.form.apiKeyId = 1
    vm.selectedBatchApiKeyId = 1
    vm.selectedBatchId = 'imgbatch_1'

    const item = {
      batch_id: 'imgbatch_1',
      custom_id: 'cover_001',
      status: 'succeeded',
      mime_type: 'image/png',
      file_extension: 'png',
      image_count: 1,
      assets: [{ id: 77, image_index: 0, mime_type: 'image/png', byte_size: 12, sha256: 'abc' }],
    }

    expect(vm.primaryAssetId(item)).toBe(77)
    await vm.loadItemPreview(item)
    expect(getContentMock).toHaveBeenCalledWith('sk-test', 'imgbatch_1', 'cover_001', 0)
    await vm.downloadItemImage(item)
    expect(downloadItemMock).toHaveBeenCalled()
    vm.reuseItemAsset(item)
    expect(vm.reusedAssetIds).toContain(77)
    expect(showSuccess).toHaveBeenCalled()
  })
})
