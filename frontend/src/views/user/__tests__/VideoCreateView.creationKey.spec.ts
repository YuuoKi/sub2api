import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import VideoCreateView from '../VideoCreateView.vue'

const mocks = vi.hoisted(() => ({
  getContract: vi.fn(),
  createTask: vi.fn(),
  listKeys: vi.fn(),
}))

vi.mock('@/api/user/video_simulation', () => ({
  getSimulationContract: mocks.getContract,
  createSimulationTask: mocks.createTask,
}))

vi.mock('@/api/keys', () => ({
  keysAPI: {
    list: mocks.listKeys,
  },
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => ({ params: {}, query: {} }),
  }
})

const GLOBAL_STUBS = {
  global: {
    stubs: {
      AppLayout: { template: '<div><slot /></div>' },
      RouterLink: { template: '<a><slot /></a>' },
    },
  },
}

describe('VideoCreateView creation_key idempotency', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.getContract.mockResolvedValue({
      provider: 'mock',
      model: 'mock-video-v1',
      label: '模拟视频结果',
      media_kind: 'image',
      duration_seconds: 4,
      resolution: '720p',
      currency: 'USD',
      pricing_source: 'internal_simulation',
      pricing_version: 'simulation-v1',
      cost_amount: 0,
      network: false,
      billing: false,
    })
    mocks.listKeys.mockResolvedValue({
      items: [{ id: 3, name: 'key-a', status: 'active' }],
      total: 1,
    })
    mocks.createTask.mockResolvedValue({ id: 99, status: 'queued' })
  })

  it('includes a creation_key UUID in the create payload for each Create click', async () => {
    const wrapper = mount(VideoCreateView, GLOBAL_STUBS)
    await flushPromises()

    await wrapper.find('#sim-api-key').setValue('3')
    await wrapper.find('#sim-prompt').setValue('hello simulation')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.createTask).toHaveBeenCalledTimes(1)
    const payload = mocks.createTask.mock.calls[0][0]
    expect(payload).toMatchObject({
      api_key_id: 3,
      prompt: 'hello simulation',
    })
    expect(typeof payload.creation_key).toBe('string')
    expect(payload.creation_key.length).toBeGreaterThan(8)

    wrapper.unmount()
  })

  it('generates a new creation_key for a subsequent intentional Create click after failure', async () => {
    mocks.createTask.mockRejectedValueOnce(new Error('create failed'))
    mocks.createTask.mockResolvedValueOnce({ id: 100, status: 'queued' })

    const wrapper = mount(VideoCreateView, GLOBAL_STUBS)
    await flushPromises()

    await wrapper.find('#sim-api-key').setValue('3')
    await wrapper.find('#sim-prompt').setValue('retry me')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    const firstKey = mocks.createTask.mock.calls[0][0].creation_key

    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    const secondKey = mocks.createTask.mock.calls[1][0].creation_key
    expect(secondKey).not.toBe(firstKey)

    wrapper.unmount()
  })
})
