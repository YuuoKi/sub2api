import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import VideoTaskDetailView from '../VideoTaskDetailView.vue'

const mocks = vi.hoisted(() => ({
  getTask: vi.fn(),
  cancelTask: vi.fn(),
  downloadResult: vi.fn(),
  getContract: vi.fn(),
}))

vi.mock('@/api/user/video_simulation', () => ({
  getSimulationTask: mocks.getTask,
  cancelSimulationTask: mocks.cancelTask,
  downloadSimulationResult: mocks.downloadResult,
  getSimulationContract: mocks.getContract,
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => ({ params: { id: '12' }, query: {} }),
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

const queuedTask = {
  id: 12,
  provider: 'mock',
  model: 'mock-video-v1',
  status: 'queued',
  prompt: 'hello',
  duration: 4,
  resolution: '720p',
  cost: 0,
  currency: 'USD',
  pricing_source: 'internal_simulation',
  pricing_version: 'simulation-v1',
  error: '',
  version: 1,
  created_at: '2026-07-18T00:00:00Z',
  updated_at: '2026-07-18T00:00:00Z',
  completed_at: null,
}

describe('VideoTaskDetailView AbortSignal wiring', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.getTask.mockResolvedValue(queuedTask)
    mocks.downloadResult.mockResolvedValue(new Blob(['<svg/>'], { type: 'image/svg+xml' }))
    mocks.getContract.mockResolvedValue({ media_kind: 'image' })
  })

  it('passes an AbortSignal into detail loads', async () => {
    const wrapper = mount(VideoTaskDetailView, GLOBAL_STUBS)
    await flushPromises()

    expect(mocks.getTask.mock.calls[0]?.[1]?.signal).toBeInstanceOf(AbortSignal)
    wrapper.unmount()
  })

  it('aborts the in-flight detail request on unmount', async () => {
    const wrapper = mount(VideoTaskDetailView, GLOBAL_STUBS)
    await flushPromises()

    const signal: AbortSignal = mocks.getTask.mock.calls[0][1].signal
    expect(signal.aborted).toBe(false)

    wrapper.unmount()

    expect(signal.aborted).toBe(true)
  })
})
