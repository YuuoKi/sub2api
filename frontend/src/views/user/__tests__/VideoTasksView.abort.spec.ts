import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import VideoTasksView from '../VideoTasksView.vue'

const mocks = vi.hoisted(() => ({
  listTasks: vi.fn(),
  cancelTask: vi.fn(),
  downloadResult: vi.fn(),
  getContract: vi.fn(),
}))

vi.mock('@/api/user/video_simulation', () => ({
  listSimulationTasks: mocks.listTasks,
  cancelSimulationTask: mocks.cancelTask,
  downloadSimulationResult: mocks.downloadResult,
  getSimulationContract: mocks.getContract,
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
      AnimatedEmptyState: { template: '<div />' },
      TaskProgressRing: { template: '<div />' },
      RouterLink: { template: '<a><slot /></a>' },
    },
  },
}

describe('VideoTasksView AbortSignal wiring', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.listTasks.mockResolvedValue({ items: [] })
    mocks.downloadResult.mockResolvedValue(new Blob(['<svg/>'], { type: 'image/svg+xml' }))
    mocks.getContract.mockResolvedValue({ media_kind: 'image' })
  })

  it('passes an AbortSignal into list polling loads', async () => {
    const wrapper = mount(VideoTasksView, GLOBAL_STUBS)
    await flushPromises()

    expect(mocks.listTasks.mock.calls[0]?.[0]?.signal).toBeInstanceOf(AbortSignal)
    wrapper.unmount()
  })

  it('aborts the in-flight list request on unmount', async () => {
    const wrapper = mount(VideoTasksView, GLOBAL_STUBS)
    await flushPromises()

    const signal: AbortSignal = mocks.listTasks.mock.calls[0][0].signal
    expect(signal.aborted).toBe(false)

    wrapper.unmount()

    expect(signal.aborted).toBe(true)
  })

  it('aborts the previous load when a new load starts', async () => {
    const wrapper = mount(VideoTasksView, GLOBAL_STUBS)
    await flushPromises()

    const firstSignal: AbortSignal = mocks.listTasks.mock.calls[0][0].signal
    expect(firstSignal.aborted).toBe(false)

    await wrapper.find('button.btn-secondary').trigger('click')
    await flushPromises()

    expect(firstSignal.aborted).toBe(true)
    expect(mocks.listTasks.mock.calls[1]?.[0]?.signal).toBeInstanceOf(AbortSignal)
    expect(mocks.listTasks.mock.calls[1][0].signal).not.toBe(firstSignal)

    wrapper.unmount()
  })
})
